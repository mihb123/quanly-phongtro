package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/uptrace/bun"
)

type RevenueSummaryRepository struct {
	db *bun.DB
}

func NewRevenueSummaryRepository(db *bun.DB) *RevenueSummaryRepository {
	return &RevenueSummaryRepository{db: db}
}

// Upsert creates or updates a revenue summary record
func (r *RevenueSummaryRepository) Upsert(ctx context.Context, summary *model.HouseRevenueSummary) error {
	_, err := r.db.NewInsert().
		Model(summary).
		On("CONFLICT (house_id, period) DO UPDATE").
		Set("total_revenue = EXCLUDED.total_revenue").
		Set("total_cost = EXCLUDED.total_cost").
		Set("profit = EXCLUDED.profit").
		Set("updated_at = NOW()").
		Returning("id, updated_at").
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("upsert revenue summary: %w", err)
	}
	return nil
}

// GetByHouseAndPeriod fetches a summary by house and period
func (r *RevenueSummaryRepository) GetByHouseAndPeriod(ctx context.Context, houseID, period string) (*model.HouseRevenueSummary, error) {
	var summary model.HouseRevenueSummary
	err := r.db.NewSelect().
		Model(&summary).
		Where("house_id = ?", houseID).
		Where("period = ?", period).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("revenue summary not found")
		}
		return nil, fmt.Errorf("get revenue summary: %w", err)
	}
	return &summary, nil
}

// ListByHouseIDs fetches summaries for multiple houses in a specific period
func (r *RevenueSummaryRepository) ListByHouseIDs(ctx context.Context, houseIDs []string, period string) ([]model.HouseRevenueSummary, error) {
	var summaries []model.HouseRevenueSummary
	if len(houseIDs) == 0 {
		return summaries, nil
	}

	err := r.db.NewSelect().
		Model(&summaries).
		Where("house_id IN (?)", bun.In(houseIDs)).
		Where("period = ?", period).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("list revenue summaries: %w", err)
	}
	return summaries, nil
}

// CalculateRevenue dynamically calculates the total revenue from paid invoices
// for a given house and period.
func (r *RevenueSummaryRepository) CalculateRevenue(ctx context.Context, houseID, period string) (float64, error) {
	var totalRevenue sql.NullFloat64

	// Join invoices with rooms to filter by house_id
	err := r.db.NewSelect().
		TableExpr("invoices AS i").
		ColumnExpr("SUM(i.total_amount)").
		Join("JOIN rooms AS r ON i.room_id = r.id").
		Where("r.house_id = ?", houseID).
		Where("i.period = ?", period).
		Where("i.status = ?", "PAID").
		Scan(ctx, &totalRevenue)

	if err != nil {
		return 0, fmt.Errorf("calculate total revenue: %w", err)
	}

	if totalRevenue.Valid {
		return totalRevenue.Float64, nil
	}
	return 0, nil
}
