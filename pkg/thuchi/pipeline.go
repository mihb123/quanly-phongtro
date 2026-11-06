package thuchi

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
)

const (
	mergeWindow    = 10 * time.Minute
	transferWindow = 15 * time.Minute
	maxTransferFee = 11000
	refundLookback = 60 * 24 * time.Hour
)

var sourcePriority = map[string]int{SourceSePay: 3, SourceNotify: 2, SourceEmail: 2, SourceManual: 0, SourceDerived: 0}

func (s *Service) ProcessRawEvent(ctx context.Context, ownerID, id string) error {
	return s.Store.RunOwner(ctx, ownerID, func(ctx context.Context, tx Tx) error {
		ev, err := tx.GetRawEvent(ctx, id)
		if err != nil {
			return err
		}
		if ev.Status != RawPending {
			return nil
		}
		ev.Attempts++
		var ins []*Incoming
		var parseErr error
		switch ev.Source {
		case SourceSePay:
			var in *Incoming
			in, parseErr = ParseSePay([]byte(ev.Payload))
			ins = []*Incoming{in}
		case SourceNotify:
			var in *Incoming
			in, _, parseErr = ParseNotify([]byte(ev.Payload), ev.ReceivedAt)
			ins = []*Incoming{in}
		case SourceEmail:
			ins, _, parseErr = ParseEmailEvents([]byte(ev.Payload))
		default:
			parseErr = errors.New("nguồn không hỗ trợ")
		}
		if parseErr != nil {
			ev.Status = RawUnparsed
			ev.Error = parseErr.Error()
			return tx.SaveRawEvent(ctx, ev)
		}
		if err := s.ensureSeeded(ctx, tx); err != nil {
			return err
		}
		var firstID string
		allDup := true
		for _, in := range ins {
			txID, dup, err := s.Ingest(ctx, tx, in, &ev.ID)
			if err != nil {
				return err
			}
			if firstID == "" {
				firstID = txID
			}
			allDup = allDup && dup
		}
		ev.Status = RawParsed
		if allDup {
			ev.Status = RawDuplicate
		}
		ev.Error = ""
		ev.TransactionID = &firstID
		return tx.SaveRawEvent(ctx, ev)
	})
}

type ingestCtx struct {
	owner      string
	accounts   []*Account
	categories map[string]*Category
	byKey      map[string]string
	rules      []*Rule
}

func (s *Service) loadIngestCtx(ctx context.Context, tx Tx) (*ingestCtx, error) {
	accounts, err := tx.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	cats, err := tx.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := tx.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	ic := &ingestCtx{owner: tx.Owner(), accounts: accounts, categories: map[string]*Category{}, byKey: map[string]string{}, rules: rules}
	for _, c := range cats {
		ic.categories[c.ID] = c
		if c.Key != "" {
			if _, ok := ic.byKey[c.Key]; !ok || c.Kind == "expense" {
				ic.byKey[c.Key] = c.ID
			}
		}
	}
	return ic, nil
}

func (ic *ingestCtx) account(id *string) *Account {
	if id == nil {
		return nil
	}
	for _, a := range ic.accounts {
		if a.ID == *id {
			return a
		}
	}
	return nil
}

func (ic *ingestCtx) catID(key string) *string {
	if id, ok := ic.byKey[key]; ok {
		return &id
	}
	return nil
}

func (s *Service) Ingest(ctx context.Context, tx Tx, in *Incoming, rawEventID *string) (string, bool, error) {
	if existing, err := tx.FindSource(ctx, in.Source, in.SourceRef); err != nil {
		return "", false, err
	} else if existing != "" {
		return existing, true, nil
	}
	ic, err := s.loadIngestCtx(ctx, tx)
	if err != nil {
		return "", false, err
	}
	acct, err := s.resolveAccount(ctx, tx, ic, in)
	if err != nil {
		return "", false, err
	}
	if in.BalanceAfter != nil && (acct.BalanceAt == nil || !in.OccurredAt.Before(*acct.BalanceAt)) {
		acct.Balance = *in.BalanceAfter
		at := in.OccurredAt
		acct.BalanceAt = &at
		if err := tx.SaveAccount(ctx, acct); err != nil {
			return "", false, err
		}
	}

	if merged, err := s.mergeInto(ctx, tx, acct, in); err != nil {
		return "", false, err
	} else if merged != nil {
		if err := tx.AddSource(ctx, TxSource{TransactionID: merged.ID, Source: in.Source, SourceRef: in.SourceRef, RawEventID: rawEventID}); err != nil {
			return "", false, err
		}
		return merged.ID, false, nil
	}

	t := &Transaction{
		ID:                  NewID(),
		AccountID:           &acct.ID,
		Direction:           in.Direction,
		Amount:              in.Amount,
		OccurredAt:          in.OccurredAt,
		RawContent:          in.Content,
		CounterpartyName:    in.CounterpartyName,
		CounterpartyAccount: in.CounterpartyAccount,
		BalanceAfter:        in.BalanceAfter,
		Lat:                 in.Lat,
		Lng:                 in.Lng,
		Kind:                KindNormal,
		Sources:             StringList{in.Source},
	}
	s.detectKind(ctx, tx, ic, t)
	if t.CategoryID == nil && t.Kind == KindNormal {
		s.classify(ctx, tx, ic, t)
	}
	if err := tx.SaveTransaction(ctx, t); err != nil {
		return "", false, err
	}
	if err := tx.AddSource(ctx, TxSource{TransactionID: t.ID, Source: in.Source, SourceRef: in.SourceRef, RawEventID: rawEventID}); err != nil {
		return "", false, err
	}
	if t.Kind == KindNormal || (t.Kind == KindTransfer && t.PairID == nil) {
		if err := s.matchTransfer(ctx, tx, ic, t); err != nil {
			return "", false, err
		}
	}
	return t.ID, false, nil
}

func (s *Service) resolveAccount(ctx context.Context, tx Tx, ic *ingestCtx, in *Incoming) (*Account, error) {
	hash := AccountHash(tx.Owner(), in.AccountNumber)
	if in.AccountNumber == "" {
		hash = AccountHash(tx.Owner(), "connector:"+in.Source+":"+in.BankCode)
	}
	for _, a := range ic.accounts {
		if hash != "" && a.AccountNoHash == hash {
			if a.Connector == ConnectorManual && a.Kind != "cash" {
				a.Connector = in.Source
				if a.BankCode == "" {
					a.BankCode = in.BankCode
				}
				if err := tx.SaveAccount(ctx, a); err != nil {
					return nil, err
				}
			}
			return a, nil
		}
	}
	if in.AccountNumber == "" {
		var match *Account
		for _, a := range ic.accounts {
			if a.BankCode == in.BankCode && a.Connector != ConnectorManual {
				if match != nil {
					match = nil
					break
				}
				match = a
			}
		}
		if match != nil {
			return match, nil
		}
	}
	kind := "bank"
	if _, ok := walletKeywords[in.BankCode]; ok {
		kind = "ewallet"
	}
	name := firstNonEmpty(BankNames[in.BankCode], in.BankName, in.BankCode)
	last4 := Last4(in.AccountNumber)
	if last4 != "" {
		name += " ·" + last4
	}
	a := &Account{
		ID:             NewID(),
		Name:           name,
		Kind:           kind,
		BankCode:       in.BankCode,
		AccountNoLast4: last4,
		AccountNoHash:  hash,
		Connector:      in.Source,
		IncludeInStats: true,
		Sort:           len(ic.accounts),
	}
	if err := tx.SaveAccount(ctx, a); err != nil {
		return nil, err
	}
	ic.accounts = append(ic.accounts, a)
	return a, nil
}

func (s *Service) mergeInto(ctx context.Context, tx Tx, acct *Account, in *Incoming) (*Transaction, error) {
	cands, err := tx.FindTransactions(ctx, TxQuery{
		AccountID: acct.ID,
		Direction: in.Direction,
		AmountMin: in.Amount,
		AmountMax: in.Amount,
		From:      in.OccurredAt.Add(-mergeWindow),
		To:        in.OccurredAt.Add(mergeWindow),
		Limit:     20,
	})
	if err != nil {
		return nil, err
	}
	var best *Transaction
	var bestScore time.Duration
	for _, c := range cands {
		if c.Sources.Has(in.Source) || c.Sources.Has(SourceDerived) {
			continue
		}
		if c.BalanceAfter != nil && in.BalanceAfter != nil && *c.BalanceAfter != *in.BalanceAfter {
			continue
		}
		score := absDur(c.OccurredAt.Sub(in.OccurredAt))
		if c.BalanceAfter != nil && in.BalanceAfter != nil {
			score -= time.Hour
		}
		if best == nil || score < bestScore {
			best, bestScore = c, score
		}
	}
	if best == nil {
		return nil, nil
	}
	better := true
	for _, src := range best.Sources {
		if sourcePriority[src] >= sourcePriority[in.Source] {
			better = false
		}
	}
	if better {
		best.OccurredAt = in.OccurredAt
		if in.Content != "" {
			best.RawContent = in.Content
		}
		if in.CounterpartyName != "" {
			best.CounterpartyName = in.CounterpartyName
		}
		if in.CounterpartyAccount != "" {
			best.CounterpartyAccount = in.CounterpartyAccount
		}
	}
	if best.BalanceAfter == nil {
		best.BalanceAfter = in.BalanceAfter
	}
	if best.Lat == nil && in.Lat != nil && in.Lng != nil {
		best.Lat, best.Lng = in.Lat, in.Lng
	}
	if best.RawContent == "" {
		best.RawContent = in.Content
	}
	best.Sources = append(best.Sources, in.Source)
	best.Reconciled = true
	if err := tx.SaveTransaction(ctx, best); err != nil {
		return nil, err
	}
	return best, nil
}

func absDur(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func hasAny(text string, keywords []string) bool {
	for _, k := range keywords {
		if containsWord(text, k) {
			return true
		}
	}
	return false
}

func (s *Service) detectKind(ctx context.Context, tx Tx, ic *ingestCtx, t *Transaction) {
	text := Normalize(t.RawContent + " " + t.CounterpartyName)
	if t.Direction == DirectionOut && hasAny(text, cardPaymentKeywords) {
		t.Kind = KindCardPayment
		return
	}
	if t.Direction == DirectionIn && hasAny(text, refundKeywords) {
		t.Kind = KindRefund
		merchant, catKey := lookupMerchant(text)
		t.Merchant = merchant
		if merchant != "" {
			orig, err := tx.FindTransactions(ctx, TxQuery{Direction: DirectionOut, Merchant: merchant, From: t.OccurredAt.Add(-refundLookback), To: t.OccurredAt, Limit: 5})
			if err == nil {
				for _, o := range orig {
					if o.CategoryID != nil {
						t.RefundOfCategoryID = o.CategoryID
						break
					}
				}
			}
		}
		if t.RefundOfCategoryID == nil && catKey != "" {
			t.RefundOfCategoryID = ic.catID(catKey)
		}
		t.CategoryID = t.RefundOfCategoryID
		t.NeedsReview = t.CategoryID == nil
		return
	}
	if s.ownCounterparty(ic, t, nil) {
		t.Kind = KindTransfer
		t.PairStatus = PairAuto
	}
}

func lookupMerchant(normalized string) (string, string) {
	for _, m := range MerchantDictionary {
		for _, k := range m.Keywords {
			if containsWord(normalized, k) {
				return m.Name, m.Category
			}
		}
	}
	return "", ""
}

func MerchantKey(t *Transaction) string {
	return Normalize(firstNonEmpty(t.Merchant, t.CounterpartyName))
}

func (s *Service) classify(ctx context.Context, tx Tx, ic *ingestCtx, t *Transaction) {
	text := Normalize(t.RawContent + " " + t.CounterpartyName)
	merchant, catKey := lookupMerchant(text)
	if merchant != "" {
		t.Merchant = merchant
	} else if t.Merchant == "" {
		t.Merchant = firstNonEmpty(strings.TrimSpace(t.CounterpartyName), DeriveMerchant(t.RawContent))
	}
	key := MerchantKey(t)
	var hit *Rule
	for _, r := range ic.rules {
		if r.Match == MatchMerchant && r.Origin == OriginLearned && r.CategoryID != nil && key != "" && Normalize(r.Pattern) == key {
			hit = r
			break
		}
	}
	if hit == nil {
		for _, r := range ic.rules {
			if r.CategoryID == nil || r.Match == MatchOwnAccount {
				continue
			}
			if ruleMatches(r, text, key) {
				hit = r
				break
			}
		}
	}
	if hit != nil {
		if _, ok := ic.categories[*hit.CategoryID]; ok {
			t.CategoryID = hit.CategoryID
			if hit.MerchantName != "" {
				t.Merchant = hit.MerchantName
			}
			hit.Hits++
			_ = tx.SaveRule(ctx, hit)
			return
		}
	}
	if catKey != "" {
		if id := ic.catID(catKey); id != nil {
			t.CategoryID = id
			return
		}
	}
	if t.Direction == DirectionIn {
		t.CategoryID = ic.catID(CatDuocChuyen)
		if t.CategoryID != nil {
			return
		}
	}
	t.NeedsReview = true
}

func ruleMatches(r *Rule, text, merchantKey string) bool {
	switch r.Match {
	case MatchContains:
		p := Normalize(r.Pattern)
		return p != "" && strings.Contains(text, p)
	case MatchMerchant:
		return merchantKey != "" && Normalize(r.Pattern) == merchantKey
	case MatchRegex:
		re, err := regexp.Compile("(?i)" + r.Pattern)
		return err == nil && re.MatchString(Fold(text))
	}
	return false
}

func (s *Service) ownCounterparty(ic *ingestCtx, t *Transaction, target *Account) bool {
	digits := DigitRuns(t.RawContent + " " + t.CounterpartyAccount)
	for _, d := range digits {
		h := AccountHash(ic.owner, d)
		for _, a := range ic.accounts {
			if a.AccountNoHash == "" || (t.AccountID != nil && a.ID == *t.AccountID) {
				continue
			}
			if target != nil && a.ID != target.ID {
				continue
			}
			if a.AccountNoHash == h {
				return true
			}
		}
	}
	cp := Normalize(t.CounterpartyAccount + " " + t.CounterpartyName)
	content := Normalize(t.RawContent)
	for _, r := range ic.rules {
		if r.Match != MatchOwnAccount {
			continue
		}
		p := Normalize(r.Pattern)
		if p == "" {
			continue
		}
		if OnlyDigits(r.Pattern) == r.Pattern && len(p) >= 6 {
			for _, d := range digits {
				if d == p {
					return true
				}
			}
			continue
		}
		if containsWord(cp, p) || containsWord(content, p) {
			return true
		}
	}
	return false
}

func (s *Service) strongPairSignal(ic *ingestCtx, x *Transaction, other *Account) bool {
	if s.ownCounterparty(ic, x, other) {
		return true
	}
	text := Normalize(x.RawContent + " " + x.CounterpartyName)
	if kws, ok := walletKeywords[other.BankCode]; ok && hasAny(text, kws) {
		return true
	}
	if name := Normalize(BankNames[other.BankCode]); name != "" {
		self := ic.account(x.AccountID)
		if self != nil && self.Kind == "ewallet" && containsWord(text, name) {
			return true
		}
	}
	if holder := Normalize(other.HolderName); strings.Count(holder, " ") >= 1 && containsWord(text, holder) {
		return true
	}
	return false
}

func (s *Service) matchTransfer(ctx context.Context, tx Tx, ic *ingestCtx, x *Transaction) error {
	if x.AccountID == nil || x.PairStatus == PairRejected {
		return nil
	}
	q := TxQuery{NotAccount: *x.AccountID, From: x.OccurredAt.Add(-transferWindow), To: x.OccurredAt.Add(transferWindow), ExcludeID: x.ID, Limit: 20}
	if x.Direction == DirectionOut {
		q.Direction, q.AmountMin, q.AmountMax = DirectionIn, maxInt64(1, x.Amount-maxTransferFee), x.Amount
	} else {
		q.Direction, q.AmountMin, q.AmountMax = DirectionOut, x.Amount, x.Amount+maxTransferFee
	}
	cands, err := tx.FindTransactions(ctx, q)
	if err != nil {
		return err
	}
	var best *Transaction
	bestStrong := false
	for _, c := range cands {
		if c.AccountID == nil || ic.account(c.AccountID) == nil || c.PairID != nil {
			continue
		}
		if !(c.Kind == KindNormal || (c.Kind == KindTransfer && c.PairStatus == PairAuto)) {
			continue
		}
		out, in := x, c
		if x.Direction == DirectionIn {
			out, in = c, x
		}
		strong := s.strongPairSignal(ic, out, ic.account(in.AccountID)) || s.strongPairSignal(ic, in, ic.account(out.AccountID))
		if !strong && (c.Kind == KindTransfer || x.Kind == KindTransfer) {
			continue
		}
		closer := best == nil || absDur(c.OccurredAt.Sub(x.OccurredAt)) < absDur(best.OccurredAt.Sub(x.OccurredAt))
		if (strong && !bestStrong) || (strong == bestStrong && closer) {
			best, bestStrong = c, strong
		}
	}
	if best == nil {
		return nil
	}
	out, in := x, best
	if x.Direction == DirectionIn {
		out, in = best, x
	}
	status := PairSuggested
	if bestStrong {
		status = PairAuto
	}
	return s.pair(ctx, tx, ic, out, in, status)
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func (s *Service) pair(ctx context.Context, tx Tx, ic *ingestCtx, out, in *Transaction, status string) error {
	for _, t := range []*Transaction{out, in} {
		other := in
		if t == in {
			other = out
		}
		id := other.ID
		t.PairID = &id
		t.PairStatus = status
		if status == PairSuggested {
			t.NeedsReview = true
		} else {
			t.Kind = KindTransfer
			t.NeedsReview = false
			t.CategoryID = nil
		}
		if err := tx.SaveTransaction(ctx, t); err != nil {
			return err
		}
	}
	if status != PairSuggested && out.Amount > in.Amount {
		return s.createFee(ctx, tx, ic, out, out.Amount-in.Amount)
	}
	return nil
}

func (s *Service) createFee(ctx context.Context, tx Tx, ic *ingestCtx, out *Transaction, fee int64) error {
	ref := "fee:" + out.ID
	if existing, err := tx.FindSource(ctx, SourceDerived, ref); err != nil || existing != "" {
		if existing != "" {
			if t, err := tx.GetTransaction(ctx, existing); err == nil && t.DeletedAt != nil {
				t.DeletedAt = nil
				t.Amount = fee
				return s.saveTx(ctx, tx, t)
			}
		}
		return err
	}
	f := &Transaction{
		ID:         NewID(),
		AccountID:  out.AccountID,
		Direction:  DirectionOut,
		Amount:     fee,
		OccurredAt: out.OccurredAt,
		RawContent: "Phí chuyển khoản",
		Merchant:   "Phí chuyển khoản",
		CategoryID: ic.catID(CatPhiNganHang),
		Kind:       KindNormal,
		Sources:    StringList{SourceDerived},
	}
	if err := tx.SaveTransaction(ctx, f); err != nil {
		return err
	}
	return tx.AddSource(ctx, TxSource{TransactionID: f.ID, Source: SourceDerived, SourceRef: ref})
}

func (s *Service) removeFee(ctx context.Context, tx Tx, outID string) error {
	existing, err := tx.FindSource(ctx, SourceDerived, "fee:"+outID)
	if err != nil || existing == "" {
		return err
	}
	f, err := tx.GetTransaction(ctx, existing)
	if err != nil || f.DeletedAt != nil {
		return nil
	}
	now := s.now().UTC()
	f.DeletedAt = &now
	return s.saveTx(ctx, tx, f)
}
