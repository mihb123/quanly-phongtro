package thuchi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

type PGStore struct {
	db *sql.DB
}

func NewPGStore(db *sql.DB) *PGStore {
	return &PGStore{db: db}
}

type pgTx struct {
	tx    *sql.Tx
	owner string
	now   time.Time
}

func (s *PGStore) RunOwner(ctx context.Context, ownerID string, fn func(ctx context.Context, tx Tx) error) error {
	if !ValidID(ownerID) {
		return ErrInvalid
	}
	sqlTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = sqlTx.Rollback() }()
	if _, err := sqlTx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('thuchi:' || $1, 0))`, ownerID); err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}
	if _, err := sqlTx.ExecContext(ctx, `INSERT INTO thuchi_sync_state (owner_id) VALUES ($1) ON CONFLICT DO NOTHING`, ownerID); err != nil {
		return fmt.Errorf("init sync state: %w", err)
	}
	if err := fn(ctx, &pgTx{tx: sqlTx, owner: ownerID, now: time.Now().UTC()}); err != nil {
		return err
	}
	return sqlTx.Commit()
}

var syncedTables = []string{"thuchi_accounts", "thuchi_categories", "thuchi_transactions", "thuchi_rules", "thuchi_budgets", "thuchi_recurrences", "thuchi_raw_events"}

func (s *PGStore) PurgeTombstones(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	rows, err := s.db.QueryContext(ctx, `SELECT owner_id FROM thuchi_sync_state`)
	if err != nil {
		return err
	}
	var owners []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		owners = append(owners, id)
	}
	rows.Close()
	for _, owner := range owners {
		err := s.RunOwner(ctx, owner, func(ctx context.Context, tx Tx) error {
			p := tx.(*pgTx)
			var maxSeq int64
			for _, table := range syncedTables {
				var seq *int64
				if err := p.tx.QueryRowContext(ctx, `WITH d AS (DELETE FROM `+table+` WHERE owner_id = $1 AND deleted_at IS NOT NULL AND deleted_at < $2 RETURNING change_seq) SELECT MAX(change_seq) FROM d`, owner, cutoff).Scan(&seq); err != nil {
					return err
				}
				if seq != nil && *seq > maxSeq {
					maxSeq = *seq
				}
			}
			if maxSeq > 0 {
				_, err := p.tx.ExecContext(ctx, `UPDATE thuchi_sync_state SET purged_seq = GREATEST(purged_seq, $2) WHERE owner_id = $1`, owner, maxSeq)
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *pgTx) PurgedSeq(ctx context.Context) (int64, error) {
	var seq int64
	err := t.tx.QueryRowContext(ctx, `SELECT purged_seq FROM thuchi_sync_state WHERE owner_id = $1`, t.owner).Scan(&seq)
	return seq, err
}

func (s *PGStore) OwnerExists(ctx context.Context, ownerID string) (bool, error) {
	if !ValidID(ownerID) {
		return false, nil
	}
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND is_activated)`, ownerID).Scan(&exists)
	return exists, err
}

func (s *PGStore) PendingRawEvents(ctx context.Context, limit int) ([]*RawEvent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+strings.Join(rawCols, ", ")+` FROM thuchi_raw_events WHERE status = 'pending' AND attempts < 5 ORDER BY received_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*RawEvent
	for rows.Next() {
		e := &RawEvent{}
		if err := rows.Scan(rawPtrs(e)...); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

var tokenCols = "id, owner_id, kind, name, secret_enc, config, last_used_at, created_at, revoked_at"

func scanToken(row interface{ Scan(...any) error }) (*IngestToken, error) {
	t := &IngestToken{}
	var cfg []byte
	if err := row.Scan(&t.ID, &t.OwnerID, &t.Kind, &t.Name, &t.SecretEnc, &cfg, &t.LastUsedAt, &t.CreatedAt, &t.RevokedAt); err != nil {
		return nil, err
	}
	t.Config = map[string]string{}
	_ = scanJSON(cfg, &t.Config)
	return t, nil
}

func (s *PGStore) FindIngestToken(ctx context.Context, id string) (*IngestToken, error) {
	if !ValidID(id) {
		return nil, ErrNotFound
	}
	t, err := scanToken(s.db.QueryRowContext(ctx, `SELECT `+tokenCols+` FROM thuchi_ingest_tokens WHERE id = $1 AND revoked_at IS NULL`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (s *PGStore) TouchIngestToken(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE thuchi_ingest_tokens SET last_used_at = NOW() WHERE id = $1`, id)
	return err
}

func (s *PGStore) ListIngestTokens(ctx context.Context, ownerID string) ([]*IngestToken, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+tokenCols+` FROM thuchi_ingest_tokens WHERE owner_id = $1 AND revoked_at IS NULL ORDER BY created_at`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*IngestToken
	for rows.Next() {
		t, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *PGStore) CreateIngestToken(ctx context.Context, t *IngestToken) error {
	cfg, err := StringMapValue(t.Config)
	if err != nil {
		return err
	}
	return s.db.QueryRowContext(ctx, `INSERT INTO thuchi_ingest_tokens (id, owner_id, kind, name, secret_enc, config) VALUES ($1, $2, $3, $4, $5, $6) RETURNING created_at`,
		t.ID, t.OwnerID, t.Kind, t.Name, t.SecretEnc, cfg).Scan(&t.CreatedAt)
}

func (s *PGStore) RevokeIngestToken(ctx context.Context, ownerID, id string) error {
	if !ValidID(id) {
		return ErrNotFound
	}
	res, err := s.db.ExecContext(ctx, `UPDATE thuchi_ingest_tokens SET revoked_at = NOW() WHERE id = $1 AND owner_id = $2 AND revoked_at IS NULL`, id, ownerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func StringMapValue(m map[string]string) (string, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	return string(b), err
}

func (t *pgTx) Owner() string { return t.owner }

func (t *pgTx) NextSeq(ctx context.Context) (int64, error) {
	var seq int64
	err := t.tx.QueryRowContext(ctx, `UPDATE thuchi_sync_state SET current_seq = current_seq + 1 WHERE owner_id = $1 RETURNING current_seq`, t.owner).Scan(&seq)
	return seq, err
}

func (t *pgTx) CurrentSeq(ctx context.Context) (int64, error) {
	var seq int64
	err := t.tx.QueryRowContext(ctx, `SELECT current_seq FROM thuchi_sync_state WHERE owner_id = $1`, t.owner).Scan(&seq)
	return seq, err
}

func (t *pgTx) Seeded(ctx context.Context) (bool, error) {
	var at *time.Time
	err := t.tx.QueryRowContext(ctx, `SELECT seeded_at FROM thuchi_sync_state WHERE owner_id = $1`, t.owner).Scan(&at)
	return at != nil, err
}

func (t *pgTx) MarkSeeded(ctx context.Context) error {
	_, err := t.tx.ExecContext(ctx, `UPDATE thuchi_sync_state SET seeded_at = NOW() WHERE owner_id = $1`, t.owner)
	return err
}

func (t *pgTx) Savepoint(ctx context.Context, fn func() error) error {
	if _, err := t.tx.ExecContext(ctx, `SAVEPOINT thuchi_sp`); err != nil {
		return err
	}
	if err := fn(); err != nil {
		if _, rbErr := t.tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT thuchi_sp`); rbErr != nil {
			return fmt.Errorf("rollback savepoint: %w", rbErr)
		}
		if _, relErr := t.tx.ExecContext(ctx, `RELEASE SAVEPOINT thuchi_sp`); relErr != nil {
			return fmt.Errorf("release savepoint: %w", relErr)
		}
		return err
	}
	_, err := t.tx.ExecContext(ctx, `RELEASE SAVEPOINT thuchi_sp`)
	return err
}

func (t *pgTx) stamp(ctx context.Context, m *Meta) error {
	seq, err := t.NextSeq(ctx)
	if err != nil {
		return err
	}
	m.OwnerID = t.owner
	m.ChangeSeq = seq
	m.Version++
	m.UpdatedAt = t.now
	if m.CreatedAt.IsZero() {
		m.CreatedAt = t.now
	}
	if m.FieldTS == nil {
		m.FieldTS = FieldTS{}
	}
	return nil
}

func placeholders(n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprintf("$%d", i+1)
	}
	return strings.Join(parts, ", ")
}

func (t *pgTx) upsert(ctx context.Context, table string, cols []string, vals []any) error {
	sets := make([]string, 0, len(cols))
	for _, c := range cols {
		if c == "id" || c == "owner_id" || c == "created_at" {
			continue
		}
		sets = append(sets, c+" = EXCLUDED."+c)
	}
	query := `INSERT INTO ` + table + ` (` + strings.Join(cols, ", ") + `) VALUES (` + placeholders(len(cols)) + `) ON CONFLICT (id) DO UPDATE SET ` +
		strings.Join(sets, ", ") + ` WHERE ` + table + `.owner_id = EXCLUDED.owner_id`
	res, err := t.tx.ExecContext(ctx, query, vals...)
	if err != nil {
		return fmt.Errorf("upsert %s: %w", table, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrForbidden
	}
	return nil
}

func (t *pgTx) getOne(ctx context.Context, table string, cols []string, id string, ptrs []any) error {
	if !ValidID(id) {
		return ErrNotFound
	}
	err := t.tx.QueryRowContext(ctx, `SELECT `+strings.Join(cols, ", ")+` FROM `+table+` WHERE owner_id = $1 AND id = $2`, t.owner, id).Scan(ptrs...)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (t *pgTx) queryRows(ctx context.Context, query string, args []any, scan func(*sql.Rows) error) error {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

var metaCols = []string{"owner_id", "field_ts", "version", "change_seq", "created_at", "updated_at", "deleted_at"}

func metaPtrs(m *Meta) []any {
	return []any{&m.OwnerID, &m.FieldTS, &m.Version, &m.ChangeSeq, &m.CreatedAt, &m.UpdatedAt, &m.DeletedAt}
}

func metaVals(m *Meta) []any {
	return []any{m.OwnerID, m.FieldTS, m.Version, m.ChangeSeq, m.CreatedAt, m.UpdatedAt, m.DeletedAt}
}

var accountCols = append([]string{"id", "name", "kind", "bank_code", "account_no_last4", "account_no_hash", "holder_name", "connector", "balance", "balance_at", "include_in_stats", "sort"}, metaCols...)

func accountPtrs(a *Account) []any {
	return append([]any{&a.ID, &a.Name, &a.Kind, &a.BankCode, &a.AccountNoLast4, &a.AccountNoHash, &a.HolderName, &a.Connector, &a.Balance, &a.BalanceAt, &a.IncludeInStats, &a.Sort}, metaPtrs(&a.Meta)...)
}

func accountVals(a *Account) []any {
	return append([]any{a.ID, a.Name, a.Kind, a.BankCode, a.AccountNoLast4, a.AccountNoHash, a.HolderName, a.Connector, a.Balance, a.BalanceAt, a.IncludeInStats, a.Sort}, metaVals(&a.Meta)...)
}

func (t *pgTx) GetAccount(ctx context.Context, id string) (*Account, error) {
	a := &Account{}
	if err := t.getOne(ctx, "thuchi_accounts", accountCols, id, accountPtrs(a)); err != nil {
		return nil, err
	}
	return a, nil
}

func (t *pgTx) ListAccounts(ctx context.Context) ([]*Account, error) {
	var out []*Account
	err := t.queryRows(ctx, `SELECT `+strings.Join(accountCols, ", ")+` FROM thuchi_accounts WHERE owner_id = $1 AND deleted_at IS NULL ORDER BY sort, created_at`, []any{t.owner}, func(r *sql.Rows) error {
		a := &Account{}
		if err := r.Scan(accountPtrs(a)...); err != nil {
			return err
		}
		out = append(out, a)
		return nil
	})
	return out, err
}

func (t *pgTx) SaveAccount(ctx context.Context, a *Account) error {
	if err := t.stamp(ctx, &a.Meta); err != nil {
		return err
	}
	return t.upsert(ctx, "thuchi_accounts", accountCols, accountVals(a))
}

var categoryCols = append([]string{"id", "key", "name", "icon", "color", "kind", "parent_id", "sort"}, metaCols...)

func categoryPtrs(c *Category) []any {
	return append([]any{&c.ID, &c.Key, &c.Name, &c.Icon, &c.Color, &c.Kind, &c.ParentID, &c.Sort}, metaPtrs(&c.Meta)...)
}

func categoryVals(c *Category) []any {
	return append([]any{c.ID, c.Key, c.Name, c.Icon, c.Color, c.Kind, c.ParentID, c.Sort}, metaVals(&c.Meta)...)
}

func (t *pgTx) GetCategory(ctx context.Context, id string) (*Category, error) {
	c := &Category{}
	if err := t.getOne(ctx, "thuchi_categories", categoryCols, id, categoryPtrs(c)); err != nil {
		return nil, err
	}
	return c, nil
}

func (t *pgTx) ListCategories(ctx context.Context) ([]*Category, error) {
	var out []*Category
	err := t.queryRows(ctx, `SELECT `+strings.Join(categoryCols, ", ")+` FROM thuchi_categories WHERE owner_id = $1 AND deleted_at IS NULL ORDER BY sort`, []any{t.owner}, func(r *sql.Rows) error {
		c := &Category{}
		if err := r.Scan(categoryPtrs(c)...); err != nil {
			return err
		}
		out = append(out, c)
		return nil
	})
	return out, err
}

func (t *pgTx) SaveCategory(ctx context.Context, c *Category) error {
	if err := t.stamp(ctx, &c.Meta); err != nil {
		return err
	}
	return t.upsert(ctx, "thuchi_categories", categoryCols, categoryVals(c))
}

var txCols = append([]string{"id", "account_id", "direction", "amount", "occurred_at", "raw_content", "merchant", "counterparty_name", "counterparty_account", "balance_after", "category_id", "note", "lat", "lng", "place_name", "kind", "pair_id", "pair_status", "refund_of_category_id", "excluded", "fingerprint", "sources", "reconciled", "needs_review", "party", "recurrence_id"}, metaCols...)

func txPtrs(x *Transaction) []any {
	return append([]any{&x.ID, &x.AccountID, &x.Direction, &x.Amount, &x.OccurredAt, &x.RawContent, &x.Merchant, &x.CounterpartyName, &x.CounterpartyAccount, &x.BalanceAfter, &x.CategoryID, &x.Note, &x.Lat, &x.Lng, &x.PlaceName, &x.Kind, &x.PairID, &x.PairStatus, &x.RefundOfCategoryID, &x.Excluded, &x.Fingerprint, &x.Sources, &x.Reconciled, &x.NeedsReview, &x.Party, &x.RecurrenceID}, metaPtrs(&x.Meta)...)
}

func txVals(x *Transaction) []any {
	return append([]any{x.ID, x.AccountID, x.Direction, x.Amount, x.OccurredAt, x.RawContent, x.Merchant, x.CounterpartyName, x.CounterpartyAccount, x.BalanceAfter, x.CategoryID, x.Note, x.Lat, x.Lng, x.PlaceName, x.Kind, x.PairID, x.PairStatus, x.RefundOfCategoryID, x.Excluded, x.Fingerprint, x.Sources, x.Reconciled, x.NeedsReview, x.Party, x.RecurrenceID}, metaVals(&x.Meta)...)
}

func (t *pgTx) GetTransaction(ctx context.Context, id string) (*Transaction, error) {
	x := &Transaction{}
	if err := t.getOne(ctx, "thuchi_transactions", txCols, id, txPtrs(x)); err != nil {
		return nil, err
	}
	return x, nil
}

func (t *pgTx) FindTransactions(ctx context.Context, q TxQuery) ([]*Transaction, error) {
	where := []string{"owner_id = $1", "deleted_at IS NULL"}
	args := []any{t.owner}
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, strings.ReplaceAll(cond, "?", fmt.Sprintf("$%d", len(args))))
	}
	if q.AccountID != "" {
		add("account_id = ?", q.AccountID)
	}
	if q.NotAccount != "" {
		add("account_id IS DISTINCT FROM ?", q.NotAccount)
	}
	if q.Direction != "" {
		add("direction = ?", q.Direction)
	}
	if q.AmountMin > 0 {
		add("amount >= ?", q.AmountMin)
	}
	if q.AmountMax > 0 {
		add("amount <= ?", q.AmountMax)
	}
	if !q.From.IsZero() {
		add("occurred_at >= ?", q.From)
	}
	if !q.To.IsZero() {
		add("occurred_at <= ?", q.To)
	}
	if q.ExcludeID != "" {
		add("id <> ?", q.ExcludeID)
	}
	if q.Merchant != "" {
		add("merchant = ?", q.Merchant)
	}
	if q.NoCategory {
		where = append(where, "category_id IS NULL", "NOT jsonb_exists(field_ts, 'category_id')")
	}
	limit := q.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT ` + strings.Join(txCols, ", ") + ` FROM thuchi_transactions WHERE ` + strings.Join(where, " AND ") + fmt.Sprintf(` ORDER BY occurred_at DESC LIMIT %d`, limit)
	var out []*Transaction
	err := t.queryRows(ctx, query, args, func(r *sql.Rows) error {
		x := &Transaction{}
		if err := r.Scan(txPtrs(x)...); err != nil {
			return err
		}
		out = append(out, x)
		return nil
	})
	return out, err
}

func (t *pgTx) SaveTransaction(ctx context.Context, x *Transaction) error {
	if err := t.stamp(ctx, &x.Meta); err != nil {
		return err
	}
	if x.Sources == nil {
		x.Sources = StringList{}
	}
	return t.upsert(ctx, "thuchi_transactions", txCols, txVals(x))
}

func (t *pgTx) FindSource(ctx context.Context, source, sourceRef string) (string, error) {
	var id string
	err := t.tx.QueryRowContext(ctx, `SELECT transaction_id FROM thuchi_tx_sources WHERE owner_id = $1 AND source = $2 AND source_ref = $3`, t.owner, source, sourceRef).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func (t *pgTx) AddSource(ctx context.Context, s TxSource) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO thuchi_tx_sources (owner_id, transaction_id, source, source_ref, raw_event_id) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (owner_id, source, source_ref) DO NOTHING`,
		t.owner, s.TransactionID, s.Source, s.SourceRef, s.RawEventID)
	return err
}

func (t *pgTx) ReplaceSource(ctx context.Context, s TxSource) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO thuchi_tx_sources (owner_id, transaction_id, source, source_ref, raw_event_id) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (owner_id, source, source_ref) DO UPDATE SET transaction_id = EXCLUDED.transaction_id`,
		t.owner, s.TransactionID, s.Source, s.SourceRef, s.RawEventID)
	return err
}

var ruleCols = append([]string{"id", "pattern", "match", "category_id", "merchant_name", "origin", "hits"}, metaCols...)

func rulePtrs(r *Rule) []any {
	return append([]any{&r.ID, &r.Pattern, &r.Match, &r.CategoryID, &r.MerchantName, &r.Origin, &r.Hits}, metaPtrs(&r.Meta)...)
}

func ruleVals(r *Rule) []any {
	return append([]any{r.ID, r.Pattern, r.Match, r.CategoryID, r.MerchantName, r.Origin, r.Hits}, metaVals(&r.Meta)...)
}

func (t *pgTx) GetRule(ctx context.Context, id string) (*Rule, error) {
	r := &Rule{}
	if err := t.getOne(ctx, "thuchi_rules", ruleCols, id, rulePtrs(r)); err != nil {
		return nil, err
	}
	return r, nil
}

func (t *pgTx) ListRules(ctx context.Context) ([]*Rule, error) {
	var out []*Rule
	err := t.queryRows(ctx, `SELECT `+strings.Join(ruleCols, ", ")+` FROM thuchi_rules WHERE owner_id = $1 AND deleted_at IS NULL ORDER BY created_at`, []any{t.owner}, func(rows *sql.Rows) error {
		r := &Rule{}
		if err := rows.Scan(rulePtrs(r)...); err != nil {
			return err
		}
		out = append(out, r)
		return nil
	})
	return out, err
}

func (t *pgTx) SaveRule(ctx context.Context, r *Rule) error {
	if err := t.stamp(ctx, &r.Meta); err != nil {
		return err
	}
	return t.upsert(ctx, "thuchi_rules", ruleCols, ruleVals(r))
}

var budgetCols = append([]string{"id", "category_id", "month", "limit_amount"}, metaCols...)

func budgetPtrs(b *Budget) []any {
	return append([]any{&b.ID, &b.CategoryID, &b.Month, &b.LimitAmount}, metaPtrs(&b.Meta)...)
}

func budgetVals(b *Budget) []any {
	return append([]any{b.ID, b.CategoryID, b.Month, b.LimitAmount}, metaVals(&b.Meta)...)
}

func (t *pgTx) GetBudget(ctx context.Context, id string) (*Budget, error) {
	b := &Budget{}
	if err := t.getOne(ctx, "thuchi_budgets", budgetCols, id, budgetPtrs(b)); err != nil {
		return nil, err
	}
	return b, nil
}

func (t *pgTx) SaveBudget(ctx context.Context, b *Budget) error {
	if err := t.stamp(ctx, &b.Meta); err != nil {
		return err
	}
	return t.upsert(ctx, "thuchi_budgets", budgetCols, budgetVals(b))
}

var recurrenceCols = append([]string{"id", "account_id", "direction", "amount", "category_id", "merchant", "note", "freq", "freq_interval", "start_at", "end_at", "next_at", "active"}, metaCols...)

func recurrencePtrs(r *Recurrence) []any {
	return append([]any{&r.ID, &r.AccountID, &r.Direction, &r.Amount, &r.CategoryID, &r.Merchant, &r.Note, &r.Freq, &r.Interval, &r.StartAt, &r.EndAt, &r.NextAt, &r.Active}, metaPtrs(&r.Meta)...)
}

func recurrenceVals(r *Recurrence) []any {
	return append([]any{r.ID, r.AccountID, r.Direction, r.Amount, r.CategoryID, r.Merchant, r.Note, r.Freq, r.Interval, r.StartAt, r.EndAt, r.NextAt, r.Active}, metaVals(&r.Meta)...)
}

func (t *pgTx) GetRecurrence(ctx context.Context, id string) (*Recurrence, error) {
	r := &Recurrence{}
	if err := t.getOne(ctx, "thuchi_recurrences", recurrenceCols, id, recurrencePtrs(r)); err != nil {
		return nil, err
	}
	return r, nil
}

func (t *pgTx) DueRecurrences(ctx context.Context, now time.Time, limit int) ([]*Recurrence, error) {
	var out []*Recurrence
	err := t.queryRows(ctx, `SELECT `+strings.Join(recurrenceCols, ", ")+` FROM thuchi_recurrences WHERE owner_id = $1 AND active AND deleted_at IS NULL AND next_at <= $2 ORDER BY next_at LIMIT $3`, []any{t.owner, now, limit}, func(rows *sql.Rows) error {
		r := &Recurrence{}
		if err := rows.Scan(recurrencePtrs(r)...); err != nil {
			return err
		}
		out = append(out, r)
		return nil
	})
	return out, err
}

func (t *pgTx) SaveRecurrence(ctx context.Context, r *Recurrence) error {
	if err := t.stamp(ctx, &r.Meta); err != nil {
		return err
	}
	return t.upsert(ctx, "thuchi_recurrences", recurrenceCols, recurrenceVals(r))
}

func (s *PGStore) DueRecurrenceOwners(ctx context.Context, now time.Time, limit int) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT owner_id FROM thuchi_recurrences WHERE active AND deleted_at IS NULL AND next_at <= $1 LIMIT $2`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

var rawCols = []string{"id", "owner_id", "source", "source_key", "payload", "received_at", "status", "error", "transaction_id", "attempts", "version", "change_seq", "updated_at", "deleted_at"}

func rawPtrs(e *RawEvent) []any {
	return []any{&e.ID, &e.OwnerID, &e.Source, &e.SourceKey, &e.Payload, &e.ReceivedAt, &e.Status, &e.Error, &e.TransactionID, &e.Attempts, &e.Version, &e.ChangeSeq, &e.UpdatedAt, &e.DeletedAt}
}

func (t *pgTx) GetRawEvent(ctx context.Context, id string) (*RawEvent, error) {
	e := &RawEvent{}
	if err := t.getOne(ctx, "thuchi_raw_events", rawCols, id, rawPtrs(e)); err != nil {
		return nil, err
	}
	return e, nil
}

func (t *pgTx) SaveRawEvent(ctx context.Context, e *RawEvent) error {
	seq, err := t.NextSeq(ctx)
	if err != nil {
		return err
	}
	e.OwnerID = t.owner
	e.ChangeSeq = seq
	e.Version++
	e.UpdatedAt = t.now
	if e.ReceivedAt.IsZero() {
		e.ReceivedAt = t.now
	}
	if e.ID == "" {
		e.ID = NewID()
	}
	vals := []any{e.ID, e.OwnerID, e.Source, e.SourceKey, e.Payload, e.ReceivedAt, e.Status, e.Error, e.TransactionID, e.Attempts, e.Version, e.ChangeSeq, e.UpdatedAt, e.DeletedAt}
	cols := make([]string, 0, len(rawCols))
	cols = append(cols, rawCols...)
	sets := []string{}
	for _, c := range cols {
		if c == "id" || c == "owner_id" || c == "received_at" || c == "payload" || c == "source" {
			continue
		}
		sets = append(sets, c+" = EXCLUDED."+c)
	}
	res, err := t.tx.ExecContext(ctx, `INSERT INTO thuchi_raw_events (`+strings.Join(cols, ", ")+`) VALUES (`+placeholders(len(cols))+`) ON CONFLICT (id) DO UPDATE SET `+strings.Join(sets, ", ")+` WHERE thuchi_raw_events.owner_id = EXCLUDED.owner_id`, vals...)
	if err != nil {
		return fmt.Errorf("save raw event: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrForbidden
	}
	return nil
}

const rawPayloadPreview = 8192

func (t *pgTx) Changes(ctx context.Context, since int64, limit int) (*Changes, int64, bool, error) {
	if limit <= 0 {
		limit = 500
	}
	ch := &Changes{Accounts: []*Account{}, Categories: []*Category{}, Transactions: []*Transaction{}, Rules: []*Rule{}, Budgets: []*Budget{}, Recurrences: []*Recurrence{}, RawEvents: []*RawEvent{}}
	collect := func(table string, cols []string, scan func(*sql.Rows) (int64, error)) ([]int64, error) {
		var seqs []int64
		err := t.queryRows(ctx, `SELECT `+strings.Join(cols, ", ")+` FROM `+table+` WHERE owner_id = $1 AND change_seq > $2 ORDER BY change_seq LIMIT $3`, []any{t.owner, since, limit + 1}, func(r *sql.Rows) error {
			seq, err := scan(r)
			seqs = append(seqs, seq)
			return err
		})
		return seqs, err
	}
	var all []int64
	add := func(seqs []int64, err error) error {
		all = append(all, seqs...)
		return err
	}
	if err := add(collect("thuchi_accounts", accountCols, func(r *sql.Rows) (int64, error) {
		a := &Account{}
		err := r.Scan(accountPtrs(a)...)
		ch.Accounts = append(ch.Accounts, a)
		return a.ChangeSeq, err
	})); err != nil {
		return nil, 0, false, err
	}
	if err := add(collect("thuchi_categories", categoryCols, func(r *sql.Rows) (int64, error) {
		c := &Category{}
		err := r.Scan(categoryPtrs(c)...)
		ch.Categories = append(ch.Categories, c)
		return c.ChangeSeq, err
	})); err != nil {
		return nil, 0, false, err
	}
	if err := add(collect("thuchi_transactions", txCols, func(r *sql.Rows) (int64, error) {
		x := &Transaction{}
		err := r.Scan(txPtrs(x)...)
		ch.Transactions = append(ch.Transactions, x)
		return x.ChangeSeq, err
	})); err != nil {
		return nil, 0, false, err
	}
	if err := add(collect("thuchi_rules", ruleCols, func(r *sql.Rows) (int64, error) {
		x := &Rule{}
		err := r.Scan(rulePtrs(x)...)
		ch.Rules = append(ch.Rules, x)
		return x.ChangeSeq, err
	})); err != nil {
		return nil, 0, false, err
	}
	if err := add(collect("thuchi_budgets", budgetCols, func(r *sql.Rows) (int64, error) {
		x := &Budget{}
		err := r.Scan(budgetPtrs(x)...)
		ch.Budgets = append(ch.Budgets, x)
		return x.ChangeSeq, err
	})); err != nil {
		return nil, 0, false, err
	}
	if err := add(collect("thuchi_recurrences", recurrenceCols, func(r *sql.Rows) (int64, error) {
		x := &Recurrence{}
		err := r.Scan(recurrencePtrs(x)...)
		ch.Recurrences = append(ch.Recurrences, x)
		return x.ChangeSeq, err
	})); err != nil {
		return nil, 0, false, err
	}
	if err := add(collect("thuchi_raw_events", rawCols, func(r *sql.Rows) (int64, error) {
		x := &RawEvent{}
		err := r.Scan(rawPtrs(x)...)
		if x.Status == RawParsed || x.Status == RawDuplicate || x.Status == RawIgnored {
			x.Payload = ""
		} else if len(x.Payload) > rawPayloadPreview {
			x.Payload = x.Payload[:rawPayloadPreview]
		}
		ch.RawEvents = append(ch.RawEvents, x)
		return x.ChangeSeq, err
	})); err != nil {
		return nil, 0, false, err
	}

	current, err := t.CurrentSeq(ctx)
	if err != nil {
		return nil, 0, false, err
	}
	if len(all) <= limit {
		return ch, current, false, nil
	}
	slices.Sort(all)
	cutoff := all[limit-1]
	ch.trim(cutoff)
	return ch, cutoff, true, nil
}

func (c *Changes) trim(cutoff int64) {
	c.Accounts = filterSeq(c.Accounts, cutoff, func(x *Account) int64 { return x.ChangeSeq })
	c.Categories = filterSeq(c.Categories, cutoff, func(x *Category) int64 { return x.ChangeSeq })
	c.Transactions = filterSeq(c.Transactions, cutoff, func(x *Transaction) int64 { return x.ChangeSeq })
	c.Rules = filterSeq(c.Rules, cutoff, func(x *Rule) int64 { return x.ChangeSeq })
	c.Budgets = filterSeq(c.Budgets, cutoff, func(x *Budget) int64 { return x.ChangeSeq })
	c.Recurrences = filterSeq(c.Recurrences, cutoff, func(x *Recurrence) int64 { return x.ChangeSeq })
	c.RawEvents = filterSeq(c.RawEvents, cutoff, func(x *RawEvent) int64 { return x.ChangeSeq })
}

func filterSeq[T any](in []T, cutoff int64, seq func(T) int64) []T {
	out := in[:0]
	for _, x := range in {
		if seq(x) <= cutoff {
			out = append(out, x)
		}
	}
	return out
}
