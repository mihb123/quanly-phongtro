package thuchi

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type actionArgs struct {
	ID         string  `json:"id"`
	A          string  `json:"a"`
	B          string  `json:"b"`
	CategoryID *string `json:"category_id"`
	Learn      bool    `json:"learn"`
}

func (s *Service) applyAction(ctx context.Context, tx Tx, action string, raw json.RawMessage, ts int64) error {
	var args actionArgs
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &args); err != nil {
			return invalid("args")
		}
	}
	switch action {
	case "set_category":
		return s.actSetCategory(ctx, tx, args, ts)
	case "confirm_transfer":
		return s.actConfirmTransfer(ctx, tx, args, ts)
	case "reject_transfer":
		return s.actRejectTransfer(ctx, tx, args, ts)
	case "mark_transfer":
		return s.actMarkTransfer(ctx, tx, args, ts)
	case "unmark_transfer":
		return s.actUnmarkTransfer(ctx, tx, args, ts)
	case "create_transfer", "update_transfer":
		var ta transferArgs
		if err := json.Unmarshal(raw, &ta); err != nil {
			return invalid("args")
		}
		return s.actSaveTransfer(ctx, tx, ta, ts)
	case "delete_transfer":
		t, err := tx.GetTransaction(ctx, firstNonEmpty(args.ID, args.A))
		if err != nil {
			return err
		}
		out, in, ok, err := s.manualTransfer(ctx, tx, t)
		if err != nil {
			return err
		}
		if !ok {
			return invalid("không phải chuyển tiền tay")
		}
		return s.deleteTransfer(ctx, tx, out, in, ts)
	case "adjust_balance":
		var aa adjustArgs
		if err := json.Unmarshal(raw, &aa); err != nil {
			return invalid("args")
		}
		return s.actAdjustBalance(ctx, tx, aa, ts)
	case "reparse_raw", "dismiss_raw":
		ev, err := tx.GetRawEvent(ctx, args.ID)
		if err != nil {
			return err
		}
		if action == "dismiss_raw" {
			ev.Status = RawDismissed
		} else {
			if ev.Status == RawParsed || ev.Status == RawDuplicate {
				return nil
			}
			ev.Status = RawPending
			ev.Attempts = 0
			ev.Error = ""
		}
		return tx.SaveRawEvent(ctx, ev)
	default:
		return invalid("action %q", action)
	}
}

func setField(t *Transaction, field string, ts int64) {
	if t.FieldTS == nil {
		t.FieldTS = FieldTS{}
	}
	if prev, ok := t.FieldTS[field]; !ok || ts >= prev {
		t.FieldTS[field] = ts
	}
}

func (s *Service) actSetCategory(ctx context.Context, tx Tx, a actionArgs, ts int64) error {
	t, err := tx.GetTransaction(ctx, a.ID)
	if err != nil {
		return err
	}
	if _, _, ok, err := s.manualTransfer(ctx, tx, t); err != nil {
		return err
	} else if ok {
		return invalid("chuyển tiền không có danh mục")
	}
	if a.CategoryID != nil {
		if _, err := tx.GetCategory(ctx, *a.CategoryID); err != nil {
			return invalid("category_id")
		}
	}
	t.CategoryID = a.CategoryID
	t.NeedsReview = false
	setField(t, "category_id", ts)
	setField(t, "needs_review", ts)
	if err := tx.SaveTransaction(ctx, t); err != nil {
		return err
	}
	if !a.Learn || a.CategoryID == nil {
		return nil
	}
	key := MerchantKey(t)
	if key == "" {
		return nil
	}
	rules, err := tx.ListRules(ctx)
	if err != nil {
		return err
	}
	var rule *Rule
	for _, r := range rules {
		if r.Match == MatchMerchant && Normalize(r.Pattern) == key {
			rule = r
			break
		}
	}
	if rule == nil {
		rule = &Rule{ID: NewID(), Pattern: firstNonEmpty(t.Merchant, t.CounterpartyName), Match: MatchMerchant, Origin: OriginLearned}
	}
	rule.CategoryID = a.CategoryID
	if err := tx.SaveRule(ctx, rule); err != nil {
		return err
	}
	similar, err := tx.FindTransactions(ctx, TxQuery{Direction: t.Direction, ExcludeID: t.ID, Limit: 500})
	if err != nil {
		return err
	}
	for _, o := range similar {
		if o.Kind != KindNormal || MerchantKey(o) != key {
			continue
		}
		if _, userSet := o.FieldTS["category_id"]; userSet {
			continue
		}
		o.CategoryID = a.CategoryID
		o.NeedsReview = false
		if err := tx.SaveTransaction(ctx, o); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) loadPair(ctx context.Context, tx Tx, a actionArgs) (*Transaction, *Transaction, error) {
	x, err := tx.GetTransaction(ctx, a.A)
	if err != nil {
		return nil, nil, err
	}
	id := a.B
	if id == "" && x.PairID != nil {
		id = *x.PairID
	}
	if id == "" {
		return x, nil, nil
	}
	y, err := tx.GetTransaction(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if x.Direction == y.Direction {
		return nil, nil, invalid("hai giao dịch cùng chiều")
	}
	if x.Direction == DirectionIn {
		x, y = y, x
	}
	return x, y, nil
}

func (s *Service) actConfirmTransfer(ctx context.Context, tx Tx, a actionArgs, ts int64) error {
	out, in, err := s.loadPair(ctx, tx, a)
	if err != nil {
		return err
	}
	if in == nil {
		return s.actMarkTransfer(ctx, tx, actionArgs{ID: out.ID}, ts)
	}
	ic, err := s.loadIngestCtx(ctx, tx)
	if err != nil {
		return err
	}
	for _, t := range []*Transaction{out, in} {
		for _, f := range []string{"kind", "pair_id", "pair_status", "category_id", "needs_review"} {
			setField(t, f, ts)
		}
	}
	return s.pair(ctx, tx, ic, out, in, PairConfirmed)
}

func (s *Service) actRejectTransfer(ctx context.Context, tx Tx, a actionArgs, ts int64) error {
	x, y, err := s.loadPair(ctx, tx, a)
	if err != nil {
		return err
	}
	ic, err := s.loadIngestCtx(ctx, tx)
	if err != nil {
		return err
	}
	for _, t := range []*Transaction{x, y} {
		if t == nil {
			continue
		}
		t.PairStatus = PairRejected
		t.Kind = KindNormal
		t.NeedsReview = false
		for _, f := range []string{"kind", "pair_status", "needs_review"} {
			setField(t, f, ts)
		}
		if t.CategoryID == nil {
			s.classify(ctx, tx, ic, t)
		}
		if err := tx.SaveTransaction(ctx, t); err != nil {
			return err
		}
	}
	return s.removeFee(ctx, tx, x.ID)
}

func (s *Service) actMarkTransfer(ctx context.Context, tx Tx, a actionArgs, ts int64) error {
	t, err := tx.GetTransaction(ctx, firstNonEmpty(a.ID, a.A))
	if err != nil {
		return err
	}
	t.Kind = KindTransfer
	t.PairStatus = PairConfirmed
	t.NeedsReview = false
	t.CategoryID = nil
	for _, f := range []string{"kind", "pair_status", "needs_review", "category_id"} {
		setField(t, f, ts)
	}
	if err := tx.SaveTransaction(ctx, t); err != nil {
		return err
	}
	pattern := OnlyDigits(t.CounterpartyAccount)
	if len(pattern) < 6 {
		pattern = Normalize(t.CounterpartyName)
	}
	if pattern == "" {
		return nil
	}
	rules, err := tx.ListRules(ctx)
	if err != nil {
		return err
	}
	for _, r := range rules {
		if r.Match == MatchOwnAccount && Normalize(r.Pattern) == pattern {
			return nil
		}
	}
	return tx.SaveRule(ctx, &Rule{ID: NewID(), Pattern: pattern, Match: MatchOwnAccount, Origin: OriginLearned})
}

func (s *Service) actUnmarkTransfer(ctx context.Context, tx Tx, a actionArgs, ts int64) error {
	t, err := tx.GetTransaction(ctx, firstNonEmpty(a.ID, a.A))
	if err != nil {
		return err
	}
	if t.PairID != nil {
		return s.actRejectTransfer(ctx, tx, actionArgs{A: t.ID}, ts)
	}
	ic, err := s.loadIngestCtx(ctx, tx)
	if err != nil {
		return err
	}
	t.Kind = KindNormal
	t.PairStatus = ""
	for _, f := range []string{"kind", "pair_status"} {
		setField(t, f, ts)
	}
	s.classify(ctx, tx, ic, t)
	return tx.SaveTransaction(ctx, t)
}

type transferArgs struct {
	OutID         string    `json:"out_id"`
	InID          string    `json:"in_id"`
	FromAccountID string    `json:"from_account_id"`
	ToAccountID   string    `json:"to_account_id"`
	Amount        int64     `json:"amount"`
	Fee           int64     `json:"fee"`
	FeeID         string    `json:"fee_id"`
	OccurredAt    time.Time `json:"occurred_at"`
	Note          string    `json:"note"`
}

type adjustArgs struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	Balance   int64  `json:"balance"`
}

func balanceEffect(t *Transaction) (string, int64) {
	if t == nil || t.DeletedAt != nil || t.AccountID == nil || !t.ManualOnly() {
		return "", 0
	}
	if t.Direction == DirectionIn {
		return *t.AccountID, t.Amount
	}
	return *t.AccountID, -t.Amount
}

func (s *Service) saveTx(ctx context.Context, tx Tx, t *Transaction) error {
	prev, err := tx.GetTransaction(ctx, t.ID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	beforeAcct, beforeAmt := balanceEffect(prev)
	if err := tx.SaveTransaction(ctx, t); err != nil {
		return err
	}
	afterAcct, afterAmt := balanceEffect(t)
	if beforeAcct == afterAcct {
		return s.shiftBalance(ctx, tx, afterAcct, afterAmt-beforeAmt)
	}
	if err := s.shiftBalance(ctx, tx, beforeAcct, -beforeAmt); err != nil {
		return err
	}
	return s.shiftBalance(ctx, tx, afterAcct, afterAmt)
}

func (s *Service) shiftBalance(ctx context.Context, tx Tx, accountID string, delta int64) error {
	if accountID == "" {
		return nil
	}
	a, err := tx.GetAccount(ctx, accountID)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if a.Connector != ConnectorManual {
		return nil
	}
	a.Balance += delta
	return tx.SaveAccount(ctx, a)
}

func (s *Service) manualTransfer(ctx context.Context, tx Tx, t *Transaction) (*Transaction, *Transaction, bool, error) {
	if t.Kind != KindTransfer || t.PairID == nil || t.DeletedAt != nil || !t.ManualOnly() {
		return nil, nil, false, nil
	}
	p, err := tx.GetTransaction(ctx, *t.PairID)
	if errors.Is(err, ErrNotFound) {
		return nil, nil, false, nil
	}
	if err != nil {
		return nil, nil, false, err
	}
	if p.DeletedAt != nil || p.Kind != KindTransfer || !p.ManualOnly() || p.PairID == nil || *p.PairID != t.ID || p.Direction == t.Direction {
		return nil, nil, false, nil
	}
	if t.Direction == DirectionIn {
		return p, t, true, nil
	}
	return t, p, true, nil
}

func (s *Service) transferSide(ctx context.Context, tx Tx, id, pairID string) (*Transaction, error) {
	if !ValidID(id) {
		return nil, invalid("id")
	}
	t, err := tx.GetTransaction(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return &Transaction{ID: id, Sources: StringList{SourceManual}}, nil
	}
	if err != nil {
		return nil, err
	}
	if t.DeletedAt != nil {
		return nil, ErrNotFound
	}
	if !t.ManualOnly() {
		return nil, invalid("chỉ sửa được giao dịch nhập tay")
	}
	if t.PairID != nil && *t.PairID != pairID {
		return nil, invalid("pair_id")
	}
	return t, nil
}

func lwwField(t *Transaction, field string, ts int64) bool {
	if t.FieldTS == nil {
		t.FieldTS = FieldTS{}
	}
	if prev, ok := t.FieldTS[field]; ok && prev > ts {
		return false
	}
	t.FieldTS[field] = ts
	return true
}

func (s *Service) feeCandidate(ctx context.Context, tx Tx, id, outID, inID string, ts int64) (*Transaction, bool, error) {
	if !ValidID(id) || id == outID || id == inID {
		return nil, false, nil
	}
	t, err := tx.GetTransaction(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return &Transaction{ID: id}, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !t.ManualOnly() || t.Kind != KindNormal || t.PairID == nil || *t.PairID != outID || t.FieldTS["amount"] > ts || t.FieldTS["deleted_at"] > ts {
		return nil, false, nil
	}
	return t, false, nil
}

func (s *Service) actSaveTransfer(ctx context.Context, tx Tx, a transferArgs, ts int64) error {
	if a.OutID == a.InID {
		return invalid("id")
	}
	if a.FromAccountID == a.ToAccountID {
		return invalid("tài khoản nguồn và đích trùng nhau")
	}
	for _, id := range []string{a.FromAccountID, a.ToAccountID} {
		acct, err := tx.GetAccount(ctx, id)
		if err != nil || acct.DeletedAt != nil {
			return invalid("account_id")
		}
	}
	if a.Amount <= 0 || a.Amount > 1e13 || a.Fee < 0 || a.Fee > 1e13 {
		return invalid("amount")
	}
	if a.OccurredAt.IsZero() {
		return invalid("occurred_at")
	}
	a.Note = strings.TrimSpace(a.Note)
	if len([]rune(a.Note)) > 2000 {
		return invalid("text too long")
	}
	out, err := s.transferSide(ctx, tx, a.OutID, a.InID)
	if err != nil {
		return err
	}
	in, err := s.transferSide(ctx, tx, a.InID, a.OutID)
	if err != nil {
		return err
	}
	feeRef := "fee:" + out.ID
	linkedID, err := tx.FindSource(ctx, SourceDerived, feeRef)
	if err != nil {
		return err
	}
	var linked *Transaction
	if linkedID != "" {
		linked, err = tx.GetTransaction(ctx, linkedID)
		if errors.Is(err, ErrNotFound) {
			linked = nil
		} else if err != nil {
			return err
		}
	}
	feeStale := linked != nil && (linked.FieldTS["amount"] > ts || linked.FieldTS["deleted_at"] > ts)

	at := a.OccurredAt.UTC()
	sides := []struct {
		t       *Transaction
		account string
		dir     string
		pair    string
	}{{out, a.FromAccountID, DirectionOut, in.ID}, {in, a.ToAccountID, DirectionIn, out.ID}}
	for _, side := range sides {
		t := side.t
		account, pair := side.account, side.pair
		if lwwField(t, "account_id", ts) || t.AccountID == nil {
			t.AccountID = &account
		}
		if lwwField(t, "amount", ts) || t.Amount <= 0 {
			t.Amount = a.Amount
		}
		if lwwField(t, "occurred_at", ts) || t.OccurredAt.IsZero() {
			t.OccurredAt = at
		}
		if lwwField(t, "note", ts) {
			t.Note = a.Note
		}
		t.Direction = side.dir
		t.Kind = KindTransfer
		t.PairID = &pair
		t.PairStatus = PairConfirmed
		t.CategoryID = nil
		t.RefundOfCategoryID = nil
		t.NeedsReview = false
		for _, f := range []string{"direction", "kind", "pair_id", "pair_status", "category_id", "refund_of_category_id", "needs_review"} {
			setField(t, f, ts)
		}
	}
	for _, side := range sides {
		if err := s.saveTx(ctx, tx, side.t); err != nil {
			return err
		}
	}
	if feeStale {
		return nil
	}

	if a.Fee == 0 {
		if linked == nil || linked.DeletedAt != nil {
			return nil
		}
		now := s.now().UTC()
		linked.DeletedAt = &now
		setField(linked, "deleted_at", ts)
		return s.saveTx(ctx, tx, linked)
	}

	fee, isNew := linked, false
	if a.FeeID != "" && (linked == nil || a.FeeID != linked.ID) {
		cand, fresh, err := s.feeCandidate(ctx, tx, a.FeeID, out.ID, in.ID, ts)
		if err != nil {
			return err
		}
		if cand != nil {
			fee, isNew = cand, fresh
		}
	}
	if fee == nil {
		fee, isNew = &Transaction{ID: NewID()}, true
	}
	if isNew {
		ic, err := s.loadIngestCtx(ctx, tx)
		if err != nil {
			return err
		}
		fee.Kind = KindNormal
		fee.Sources = StringList{SourceManual}
		fee.Merchant = "Phí chuyển khoản"
		fee.CategoryID = ic.catID(CatPhiNganHang)
	}
	from, outID := a.FromAccountID, out.ID
	fee.AccountID = &from
	fee.PairID = &outID
	fee.Direction = DirectionOut
	fee.Amount = a.Fee
	fee.OccurredAt = at
	fee.DeletedAt = nil
	for _, f := range []string{"account_id", "direction", "amount", "occurred_at", "pair_id", "deleted_at"} {
		setField(fee, f, ts)
	}
	if err := s.saveTx(ctx, tx, fee); err != nil {
		return err
	}
	if linked != nil && linked.ID == fee.ID {
		return nil
	}
	if linked != nil && linked.DeletedAt == nil {
		now := s.now().UTC()
		linked.DeletedAt = &now
		setField(linked, "deleted_at", ts)
		if err := s.saveTx(ctx, tx, linked); err != nil {
			return err
		}
	}
	return tx.ReplaceSource(ctx, TxSource{TransactionID: fee.ID, Source: SourceDerived, SourceRef: feeRef})
}

func (s *Service) deleteTransfer(ctx context.Context, tx Tx, out, in *Transaction, ts int64) error {
	now := s.now().UTC()
	for _, t := range []*Transaction{out, in} {
		t.DeletedAt = &now
		setField(t, "deleted_at", ts)
		if err := s.saveTx(ctx, tx, t); err != nil {
			return err
		}
	}
	return s.removeFee(ctx, tx, out.ID)
}

func (s *Service) actAdjustBalance(ctx context.Context, tx Tx, a adjustArgs, ts int64) error {
	if !ValidID(a.ID) {
		return invalid("id")
	}
	if _, err := tx.GetTransaction(ctx, a.ID); err == nil {
		return nil
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	acct, err := tx.GetAccount(ctx, a.AccountID)
	if err != nil || acct.DeletedAt != nil {
		return invalid("account_id")
	}
	if acct.Connector != ConnectorManual {
		return invalid("tài khoản kết nối ngân hàng tự cập nhật số dư")
	}
	if a.Balance < -1e13 || a.Balance > 1e13 {
		return invalid("balance")
	}
	diff := a.Balance - acct.Balance
	if diff == 0 {
		now := s.now().UTC()
		marker := &Transaction{
			ID:         a.ID,
			AccountID:  &acct.ID,
			Direction:  DirectionIn,
			OccurredAt: time.UnixMilli(ts).UTC(),
			Kind:       KindAdjustment,
			Sources:    StringList{SourceManual},
		}
		marker.DeletedAt = &now
		if err := tx.SaveTransaction(ctx, marker); err != nil {
			return err
		}
		return tx.SaveAccount(ctx, acct)
	}
	t := &Transaction{
		ID:         a.ID,
		AccountID:  &acct.ID,
		Direction:  DirectionIn,
		Amount:     diff,
		OccurredAt: time.UnixMilli(ts).UTC(),
		Kind:       KindAdjustment,
		Sources:    StringList{SourceManual},
	}
	if diff < 0 {
		t.Direction = DirectionOut
		t.Amount = -diff
	}
	if t.Amount > 1e13 {
		return invalid("amount")
	}
	return s.saveTx(ctx, tx, t)
}
