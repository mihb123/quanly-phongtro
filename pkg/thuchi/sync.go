package thuchi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

const maxClockSkew = time.Minute

type Mutation struct {
	MID     string                     `json:"mid"`
	Entity  string                     `json:"entity"`
	Op      string                     `json:"op"`
	ID      string                     `json:"id"`
	Changes map[string]json.RawMessage `json:"changes"`
	TS      int64                      `json:"ts"`
	Action  string                     `json:"action"`
	Args    json.RawMessage            `json:"args"`
}

type MutationResult struct {
	MID   string `json:"mid"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type PushResult struct {
	Results []MutationResult `json:"results"`
	Seq     int64            `json:"seq"`
}

type PullResult struct {
	Reset   bool     `json:"reset,omitempty"`
	Seq     int64    `json:"seq"`
	HasMore bool     `json:"has_more"`
	Changes *Changes `json:"changes"`
}

var writable = map[string]map[string]bool{
	"accounts":     set("name", "kind", "bank_code", "account_no_last4", "holder_name", "include_in_stats", "sort", "balance"),
	"categories":   set("name", "icon", "color", "kind", "parent_id", "sort"),
	"transactions": set("category_id", "note", "lat", "lng", "place_name", "excluded", "kind", "pair_id", "pair_status", "refund_of_category_id", "needs_review", "party"),
	"rules":        set("pattern", "match", "category_id", "merchant_name", "origin"),
	"budgets":      set("category_id", "month", "limit_amount"),
	"recurrences":  set("account_id", "direction", "amount", "category_id", "merchant", "note", "freq", "interval", "start_at", "end_at", "active"),
}

var manualTxFields = set("account_id", "direction", "amount", "occurred_at", "raw_content", "merchant", "counterparty_name")

func set(keys ...string) map[string]bool {
	m := make(map[string]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m
}

func (s *Service) Pull(ctx context.Context, ownerID string, since int64, limit int) (*PullResult, error) {
	var out *PullResult
	err := s.Store.RunOwner(ctx, ownerID, func(ctx context.Context, tx Tx) error {
		if err := s.ensureSeeded(ctx, tx); err != nil {
			return err
		}
		if err := tx.Savepoint(ctx, func() error { return s.generateRecurrences(ctx, tx) }); err != nil {
			if ctx.Err() != nil {
				return err
			}
			log.Printf("thuchi: generate recurrences for %s: %v", tx.Owner(), err)
		}
		reset := false
		if since > 0 {
			purged, err := tx.PurgedSeq(ctx)
			if err != nil {
				return err
			}
			if since < purged {
				reset, since = true, 0
			}
		}
		ch, seq, more, err := tx.Changes(ctx, since, limit)
		if err != nil {
			return err
		}
		out = &PullResult{Reset: reset, Seq: seq, HasMore: more, Changes: ch}
		return nil
	})
	return out, err
}

func (s *Service) ensureSeeded(ctx context.Context, tx Tx) error {
	seeded, err := tx.Seeded(ctx)
	if err != nil || seeded {
		return err
	}
	if err := s.addDefaultCategories(ctx, tx); err != nil {
		return err
	}
	cash := &Account{ID: NewID(), Name: "Tiền mặt", Kind: "cash", Connector: ConnectorManual, IncludeInStats: true, Sort: 100}
	if err := tx.SaveAccount(ctx, cash); err != nil {
		return err
	}
	return tx.MarkSeeded(ctx)
}

func (s *Service) addDefaultCategories(ctx context.Context, tx Tx) error {
	existing, err := tx.ListCategories(ctx)
	if err != nil {
		return err
	}
	have := map[string]string{}
	for _, c := range existing {
		if c.Key != "" {
			have[c.Key+"/"+c.Kind] = c.ID
		}
	}
	p := s.preset()
	for i, d := range p.Categories {
		if _, ok := have[d.Key+"/"+d.Kind]; ok {
			continue
		}
		c := &Category{ID: NewID(), Key: d.Key, Name: d.Name, Icon: d.Icon, Color: d.Color, Kind: d.Kind, Sort: i}
		if parent, ok := have[p.Parents[d.Key]+"/"+d.Kind]; ok {
			c.ParentID = &parent
		}
		if err := tx.SaveCategory(ctx, c); err != nil {
			return err
		}
		have[d.Key+"/"+d.Kind] = c.ID
	}
	return nil
}

func (s *Service) Push(ctx context.Context, ownerID string, muts []Mutation) (*PushResult, error) {
	out := &PushResult{Results: make([]MutationResult, 0, len(muts))}
	err := s.Store.RunOwner(ctx, ownerID, func(ctx context.Context, tx Tx) error {
		if err := s.ensureSeeded(ctx, tx); err != nil {
			return err
		}
		for i := range muts {
			m := &muts[i]
			err := tx.Savepoint(ctx, func() error { return s.applyMutation(ctx, tx, m) })
			res := MutationResult{MID: m.MID, OK: err == nil}
			if err != nil {
				if !errors.Is(err, ErrInvalid) && !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrForbidden) {
					return err
				}
				res.Error = err.Error()
			}
			out.Results = append(out.Results, res)
		}
		seq, err := tx.CurrentSeq(ctx)
		out.Seq = seq
		return err
	})
	if err != nil {
		return nil, err
	}
	s.Kick()
	return out, nil
}

func (s *Service) clampTS(ts int64) int64 {
	now := s.now().UnixMilli()
	if ts <= 0 || ts > now+maxClockSkew.Milliseconds() {
		return now
	}
	return ts
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

func (s *Service) applyMutation(ctx context.Context, tx Tx, m *Mutation) error {
	ts := s.clampTS(m.TS)
	if m.Op == "action" {
		return s.applyAction(ctx, tx, m.Action, m.Args, ts)
	}
	if !ValidID(m.ID) {
		return invalid("id")
	}
	switch m.Entity {
	case "accounts":
		cur, err := tx.GetAccount(ctx, m.ID)
		isNew := errors.Is(err, ErrNotFound)
		if err != nil && !isNew {
			return err
		}
		if isNew {
			if m.Op != "upsert" {
				return ErrNotFound
			}
			cur = &Account{ID: m.ID, Kind: "bank", Connector: ConnectorManual, IncludeInStats: true}
		}
		allowed := writable["accounts"]
		if !isNew || cur.Connector != ConnectorManual {
			allowed = without(allowed, "balance")
		}
		if err := s.patch(cur, &cur.Meta, m, allowed, ts); err != nil {
			return err
		}
		if raw, ok := m.Changes["account_no"]; ok && m.Op != "delete" {
			var no string
			if err := json.Unmarshal(raw, &no); err != nil {
				return invalid("account_no")
			}
			if digits := OnlyDigits(no); len(digits) >= 6 {
				cur.AccountNoHash = AccountHash(tx.Owner(), digits)
				cur.AccountNoLast4 = Last4(digits)
			} else if strings.TrimSpace(no) == "" && cur.Connector == ConnectorManual {
				cur.AccountNoHash = ""
			} else {
				return invalid("account_no")
			}
		}
		if err := validateAccount(cur); err != nil {
			return err
		}
		return tx.SaveAccount(ctx, cur)
	case "categories":
		cur, err := tx.GetCategory(ctx, m.ID)
		isNew := errors.Is(err, ErrNotFound)
		if err != nil && !isNew {
			return err
		}
		if isNew {
			if m.Op != "upsert" {
				return ErrNotFound
			}
			cur = &Category{ID: m.ID, Kind: "expense"}
		}
		if err := s.patch(cur, &cur.Meta, m, writable["categories"], ts); err != nil {
			return err
		}
		if m.Op == "delete" {
			if err := tx.SaveCategory(ctx, cur); err != nil {
				return err
			}
			return detachChildren(ctx, tx, cur.ID, ts)
		}
		if err := validateCategory(ctx, tx, cur); err != nil {
			return err
		}
		return tx.SaveCategory(ctx, cur)
	case "transactions":
		cur, err := tx.GetTransaction(ctx, m.ID)
		isNew := errors.Is(err, ErrNotFound)
		if err != nil && !isNew {
			return err
		}
		if isNew {
			if m.Op != "upsert" {
				return ErrNotFound
			}
			cur = &Transaction{ID: m.ID, Kind: KindNormal, Sources: StringList{SourceManual}}
		}
		allowed := writable["transactions"]
		if cur.ManualOnly() {
			allowed = union(allowed, manualTxFields)
		}
		if !isNew {
			out, in, ok, err := s.manualTransfer(ctx, tx, cur)
			if err != nil {
				return err
			}
			if ok {
				if m.Op == "delete" {
					return s.deleteTransfer(ctx, tx, out, in, ts)
				}
				allowed = without(allowed, "account_id", "direction", "amount", "occurred_at", "kind", "pair_id", "pair_status", "category_id", "refund_of_category_id")
			}
		}
		if err := s.patch(cur, &cur.Meta, m, allowed, ts); err != nil {
			return err
		}
		if err := validateTransaction(ctx, tx, cur); err != nil {
			return err
		}
		if isNew {
			cur.Merchant = strings.TrimSpace(cur.Merchant)
		}
		return s.saveTx(ctx, tx, cur)
	case "rules":
		cur, err := tx.GetRule(ctx, m.ID)
		isNew := errors.Is(err, ErrNotFound)
		if err != nil && !isNew {
			return err
		}
		if isNew {
			if m.Op != "upsert" {
				return ErrNotFound
			}
			cur = &Rule{ID: m.ID, Match: MatchContains, Origin: OriginManual}
		}
		if err := s.patch(cur, &cur.Meta, m, writable["rules"], ts); err != nil {
			return err
		}
		if err := validateRule(cur); err != nil {
			return err
		}
		return tx.SaveRule(ctx, cur)
	case "budgets":
		cur, err := tx.GetBudget(ctx, m.ID)
		isNew := errors.Is(err, ErrNotFound)
		if err != nil && !isNew {
			return err
		}
		if isNew {
			if m.Op != "upsert" {
				return ErrNotFound
			}
			cur = &Budget{ID: m.ID}
		}
		if err := s.patch(cur, &cur.Meta, m, writable["budgets"], ts); err != nil {
			return err
		}
		if !monthPattern.MatchString(cur.Month) || cur.LimitAmount < 0 {
			return invalid("budget")
		}
		return tx.SaveBudget(ctx, cur)
	case "recurrences":
		return s.applyRecurrence(ctx, tx, m, ts)
	default:
		return invalid("entity %q", m.Entity)
	}
}

func without(m map[string]bool, keys ...string) map[string]bool {
	out := make(map[string]bool, len(m))
	for k, v := range m {
		out[k] = v
	}
	for _, k := range keys {
		delete(out, k)
	}
	return out
}

func union(a, b map[string]bool) map[string]bool {
	out := make(map[string]bool, len(a)+len(b))
	for k := range a {
		out[k] = true
	}
	for k := range b {
		out[k] = true
	}
	return out
}

func (s *Service) patch(entity any, meta *Meta, m *Mutation, allowed map[string]bool, ts int64) error {
	if meta.FieldTS == nil {
		meta.FieldTS = FieldTS{}
	}
	if m.Op == "delete" {
		if prev, ok := meta.FieldTS["deleted_at"]; ok && prev > ts {
			return nil
		}
		now := s.now().UTC()
		meta.DeletedAt = &now
		meta.FieldTS["deleted_at"] = ts
		return nil
	}
	if m.Op != "upsert" && m.Op != "patch" {
		return invalid("op %q", m.Op)
	}
	raw, err := json.Marshal(entity)
	if err != nil {
		return err
	}
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	changed := false
	for k, v := range m.Changes {
		if !allowed[k] {
			continue
		}
		if prev, ok := meta.FieldTS[k]; ok && prev > ts {
			continue
		}
		fields[k] = v
		meta.FieldTS[k] = ts
		changed = true
	}
	if !changed {
		return nil
	}
	merged, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(merged, entity); err != nil {
		return invalid("payload: %v", err)
	}
	return nil
}

var monthPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

func oneOf(v string, opts ...string) bool {
	for _, o := range opts {
		if v == o {
			return true
		}
	}
	return false
}

func validateAccount(a *Account) error {
	a.Name = strings.TrimSpace(a.Name)
	if a.Name == "" || len([]rune(a.Name)) > 100 {
		return invalid("account name")
	}
	if !oneOf(a.Kind, "bank", "ewallet", "credit", "cash") {
		return invalid("account kind")
	}
	if len(a.AccountNoLast4) > 4 {
		a.AccountNoLast4 = Last4(a.AccountNoLast4)
	}
	return nil
}

func validateCategory(ctx context.Context, tx Tx, c *Category) error {
	c.Name = strings.TrimSpace(c.Name)
	if c.Name == "" || len([]rune(c.Name)) > 60 {
		return invalid("category name")
	}
	if !oneOf(c.Kind, "expense", "income") {
		return invalid("category kind")
	}
	if c.ParentID != nil {
		if *c.ParentID == c.ID {
			return invalid("parent_id")
		}
		p, err := tx.GetCategory(ctx, *c.ParentID)
		if errors.Is(err, ErrNotFound) {
			return invalid("parent_id")
		}
		if err != nil {
			return err
		}
		if p.DeletedAt != nil || p.ParentID != nil || p.Kind != c.Kind {
			return invalid("parent_id")
		}
	}
	all, err := tx.ListCategories(ctx)
	if err != nil {
		return err
	}
	for _, o := range all {
		if o.ID != c.ID && o.ParentID != nil && *o.ParentID == c.ID && (c.ParentID != nil || o.Kind != c.Kind) {
			return invalid("parent_id")
		}
	}
	return nil
}

func detachChildren(ctx context.Context, tx Tx, parentID string, ts int64) error {
	all, err := tx.ListCategories(ctx)
	if err != nil {
		return err
	}
	for _, c := range all {
		if c.ParentID == nil || *c.ParentID != parentID {
			continue
		}
		c.ParentID = nil
		if c.FieldTS == nil {
			c.FieldTS = FieldTS{}
		}
		c.FieldTS["parent_id"] = ts
		if err := tx.SaveCategory(ctx, c); err != nil {
			return err
		}
	}
	return nil
}

func validateTransaction(ctx context.Context, tx Tx, t *Transaction) error {
	if !oneOf(t.Direction, DirectionIn, DirectionOut) {
		return invalid("direction")
	}
	if t.Amount <= 0 || t.Amount > 1e13 {
		return invalid("amount")
	}
	if t.OccurredAt.IsZero() {
		return invalid("occurred_at")
	}
	if !oneOf(t.Kind, KindNormal, KindTransfer, KindRefund, KindCardPayment, KindAdjustment, KindBorrow, KindLend, KindRepay, KindCollect) {
		return invalid("kind")
	}
	if !oneOf(t.PairStatus, "", PairAuto, PairSuggested, PairConfirmed, PairRejected) {
		return invalid("pair_status")
	}
	if len([]rune(t.Note)) > 2000 || len([]rune(t.PlaceName)) > 300 || len([]rune(t.RawContent)) > 2000 {
		return invalid("text too long")
	}
	if t.ManualOnly() && len([]rune(strings.TrimSpace(t.Merchant))) > 200 {
		return invalid("merchant")
	}
	t.Party = strings.TrimSpace(t.Party)
	if len([]rune(t.Party)) > 100 {
		return invalid("party")
	}
	if dir, ok := debtDirection[t.Kind]; ok && dir != t.Direction {
		return invalid("kind")
	}
	if t.AccountID != nil {
		if _, err := tx.GetAccount(ctx, *t.AccountID); err != nil {
			return invalid("account_id")
		}
	}
	for _, id := range []*string{t.CategoryID, t.RefundOfCategoryID} {
		if id != nil {
			if _, err := tx.GetCategory(ctx, *id); err != nil {
				return invalid("category_id")
			}
		}
	}
	if t.PairID != nil && !ValidID(*t.PairID) {
		return invalid("pair_id")
	}
	if (t.Lat == nil) != (t.Lng == nil) {
		return invalid("lat/lng")
	}
	return nil
}

func validateRule(r *Rule) error {
	r.Pattern = strings.TrimSpace(r.Pattern)
	if r.Pattern == "" || len([]rune(r.Pattern)) > 200 {
		return invalid("pattern")
	}
	if !oneOf(r.Match, MatchContains, MatchRegex, MatchMerchant, MatchOwnAccount) {
		return invalid("match")
	}
	if !oneOf(r.Origin, OriginManual, OriginLearned) {
		return invalid("origin")
	}
	if r.Match == MatchRegex {
		if _, err := regexp.Compile("(?i)" + r.Pattern); err != nil {
			return invalid("regex")
		}
	}
	if r.Match != MatchOwnAccount && r.CategoryID == nil {
		return invalid("category_id")
	}
	return nil
}
