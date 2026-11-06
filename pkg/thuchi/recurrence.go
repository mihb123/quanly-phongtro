package thuchi

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"
)

const (
	FreqDaily   = "daily"
	FreqWeekly  = "weekly"
	FreqMonthly = "monthly"
	FreqYearly  = "yearly"

	maxRecurrenceBatch    = 60
	maxRecurrenceBackfill = 12
)

var recurrenceZone = time.FixedZone("ICT", 7*3600)

var debtDirection = map[string]string{
	KindBorrow:  DirectionIn,
	KindLend:    DirectionOut,
	KindRepay:   DirectionOut,
	KindCollect: DirectionIn,
}

func clampDate(start time.Time, year int, month time.Month) time.Time {
	day := start.Day()
	if last := time.Date(year, month+1, 0, 0, 0, 0, 0, recurrenceZone).Day(); day > last {
		day = last
	}
	return time.Date(year, month, day, start.Hour(), start.Minute(), start.Second(), start.Nanosecond(), recurrenceZone).UTC()
}

func occurrence(r *Recurrence, n int) time.Time {
	start := r.StartAt.In(recurrenceZone)
	step := n * max(r.Interval, 1)
	switch r.Freq {
	case FreqDaily:
		return start.AddDate(0, 0, step).UTC()
	case FreqWeekly:
		return start.AddDate(0, 0, 7*step).UTC()
	case FreqYearly:
		return clampDate(start, start.Year()+step, start.Month())
	default:
		months := int(start.Month()) - 1 + step
		return clampDate(start, start.Year()+months/12, time.Month(months%12+1))
	}
}

func occurrenceIndex(r *Recurrence, t time.Time) int {
	if !t.After(r.StartAt) {
		return 0
	}
	step := max(r.Interval, 1)
	a, b := r.StartAt.In(recurrenceZone), t.In(recurrenceZone)
	var n int
	switch r.Freq {
	case FreqDaily:
		n = int(b.Sub(a).Hours()/24) / step
	case FreqWeekly:
		n = int(b.Sub(a).Hours()/(24*7)) / step
	case FreqYearly:
		n = (b.Year() - a.Year()) / step
	default:
		n = ((b.Year()-a.Year())*12 + int(b.Month()) - int(a.Month())) / step
	}
	n = max(n, 0)
	for occurrence(r, n).Before(t) {
		n++
	}
	for n > 0 && !occurrence(r, n-1).Before(t) {
		n--
	}
	return n
}

func (s *Service) applyRecurrence(ctx context.Context, tx Tx, m *Mutation, ts int64) error {
	cur, err := tx.GetRecurrence(ctx, m.ID)
	isNew := errors.Is(err, ErrNotFound)
	if err != nil && !isNew {
		return err
	}
	if isNew {
		if m.Op != "upsert" {
			return ErrNotFound
		}
		cur = &Recurrence{ID: m.ID, Direction: DirectionOut, Freq: FreqMonthly, Interval: 1, Active: true}
	}
	prev := *cur
	if cur.EndAt != nil {
		end := *cur.EndAt
		prev.EndAt = &end
	}
	if err := s.patch(cur, &cur.Meta, m, writable["recurrences"], ts); err != nil {
		return err
	}
	if m.Op == "delete" {
		return tx.SaveRecurrence(ctx, cur)
	}
	if err := validateRecurrence(ctx, tx, cur); err != nil {
		return err
	}
	expired := !isNew && !prev.Active && prev.EndAt != nil && prev.NextAt.After(*prev.EndAt)
	if _, sent := m.Changes["active"]; expired && !sent && (cur.EndAt == nil || !prev.NextAt.After(*cur.EndAt)) {
		cur.Active = true
		cur.FieldTS["active"] = ts
	}
	switch {
	case isNew:
		cur.NextAt = occurrence(cur, 0)
	case cur.Active && !prev.Active:
		base := s.now().UTC().Add(time.Microsecond)
		if prev.NextAt.After(base) {
			base = prev.NextAt
		}
		cur.NextAt = occurrence(cur, occurrenceIndex(cur, base))
	case !cur.StartAt.Equal(prev.StartAt) || cur.Freq != prev.Freq || cur.Interval != prev.Interval:
		cur.NextAt = occurrence(cur, occurrenceIndex(cur, prev.NextAt))
	}
	return tx.SaveRecurrence(ctx, cur)
}

func validateRecurrence(ctx context.Context, tx Tx, r *Recurrence) error {
	if !oneOf(r.Direction, DirectionIn, DirectionOut) {
		return invalid("direction")
	}
	if r.Amount <= 0 || r.Amount > 1e13 {
		return invalid("amount")
	}
	if !oneOf(r.Freq, FreqDaily, FreqWeekly, FreqMonthly, FreqYearly) {
		return invalid("freq")
	}
	if r.Interval < 1 || r.Interval > 1000 {
		return invalid("interval")
	}
	if r.StartAt.IsZero() || r.StartAt.Year() < 2000 || r.StartAt.Year() > 2200 {
		return invalid("start_at")
	}
	r.StartAt = r.StartAt.UTC()
	if r.EndAt != nil {
		if r.EndAt.Before(r.StartAt) {
			return invalid("end_at")
		}
		end := r.EndAt.UTC()
		r.EndAt = &end
	}
	r.Merchant = strings.TrimSpace(r.Merchant)
	if len([]rune(r.Merchant)) > 200 || len([]rune(r.Note)) > 2000 {
		return invalid("text too long")
	}
	if r.AccountID != nil {
		a, err := tx.GetAccount(ctx, *r.AccountID)
		if err != nil || a.DeletedAt != nil {
			return invalid("account_id")
		}
	}
	if r.CategoryID != nil {
		c, err := tx.GetCategory(ctx, *r.CategoryID)
		if err != nil || c.DeletedAt != nil {
			return invalid("category_id")
		}
	}
	return nil
}

func (s *Service) generateRecurrences(ctx context.Context, tx Tx) error {
	now := s.now().UTC()
	due, err := tx.DueRecurrences(ctx, now, maxRecurrenceBatch)
	if err != nil {
		return err
	}
	budget := maxRecurrenceBatch
	for _, r := range due {
		if budget == 0 {
			break
		}
		accountID, categoryID, err := recurrenceRefs(ctx, tx, r)
		if err != nil {
			return err
		}
		n := occurrenceIndex(r, r.NextAt)
		if cutoff := now.AddDate(0, -maxRecurrenceBackfill, 0); r.NextAt.Before(cutoff) {
			n = max(n, occurrenceIndex(r, cutoff))
		}
		for budget > 0 {
			at := occurrence(r, n)
			if at.After(now) || (r.EndAt != nil && at.After(*r.EndAt)) {
				break
			}
			recID := r.ID
			t := &Transaction{
				ID:           NewID(),
				AccountID:    accountID,
				Direction:    r.Direction,
				Amount:       r.Amount,
				OccurredAt:   at,
				Merchant:     r.Merchant,
				Note:         r.Note,
				CategoryID:   categoryID,
				Kind:         KindNormal,
				Sources:      StringList{SourceManual},
				RecurrenceID: &recID,
			}
			if err := s.saveTx(ctx, tx, t); err != nil {
				return err
			}
			budget--
			n++
		}
		r.NextAt = occurrence(r, n)
		if r.EndAt != nil && r.NextAt.After(*r.EndAt) {
			r.Active = false
		}
		if err := tx.SaveRecurrence(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func recurrenceRefs(ctx context.Context, tx Tx, r *Recurrence) (*string, *string, error) {
	var accountID, categoryID *string
	if r.AccountID != nil {
		a, err := tx.GetAccount(ctx, *r.AccountID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, nil, err
		}
		if err == nil && a.DeletedAt == nil {
			accountID = &a.ID
		}
	}
	if r.CategoryID != nil {
		c, err := tx.GetCategory(ctx, *r.CategoryID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, nil, err
		}
		if err == nil && c.DeletedAt == nil {
			categoryID = &c.ID
		}
	}
	return accountID, categoryID, nil
}

func (s *Service) ProcessRecurrences(ctx context.Context) {
	owners, err := s.Store.DueRecurrenceOwners(ctx, s.now().UTC(), 100)
	if err != nil {
		log.Printf("thuchi: list due recurrences: %v", err)
		return
	}
	for _, owner := range owners {
		if ctx.Err() != nil {
			return
		}
		if err := s.Store.RunOwner(ctx, owner, s.generateRecurrences); err != nil {
			log.Printf("thuchi: generate recurrences for %s: %v", owner, err)
		}
	}
}
