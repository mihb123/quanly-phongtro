package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/uptrace/bun"
)

type HouseCostRepository struct {
	db *bun.DB
}

func NewHouseCostRepository(db *bun.DB) *HouseCostRepository {
	return &HouseCostRepository{db: db}
}

// Create inserts a new house cost record
func (r *HouseCostRepository) Create(ctx context.Context, cost *model.HouseCost) error {
	_, err := r.db.NewInsert().
		Model(cost).
		ExcludeColumn("created_at", "updated_at").
		Returning("id, created_at, updated_at").
		Exec(ctx)

	if err != nil {
		if strings.Contains(err.Error(), "uq_house_costs_house_period") || strings.Contains(err.Error(), "duplicate key value") {
			return errors.New("house cost record already exists for this period")
		}
		return fmt.Errorf("create house cost: %w", err)
	}
	return nil
}

// GetByHouseAndPeriod fetches a house cost record by house ID and period
func (r *HouseCostRepository) GetByHouseAndPeriod(ctx context.Context, houseID, period string) (*model.HouseCost, error) {
	var cost model.HouseCost
	err := r.db.NewSelect().
		Model(&cost).
		Where("house_id = ?", houseID).
		Where("period = ?", period).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("house cost not found")
		}
		return nil, fmt.Errorf("get house cost by house and period: %w", err)
	}
	return &cost, nil
}

// GetLatestByHouseID fetches the most recent house cost record
func (r *HouseCostRepository) GetLatestByHouseID(ctx context.Context, houseID string) (*model.HouseCost, error) {
	var cost model.HouseCost
	err := r.db.NewSelect().
		Model(&cost).
		Where("house_id = ?", houseID).
		Order("period DESC").
		Limit(1).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("house cost not found")
		}
		return nil, fmt.Errorf("get latest house cost: %w", err)
	}
	return &cost, nil
}

// ListByHouseIDs fetches house costs for multiple houses in a specific period
func (r *HouseCostRepository) ListByHouseIDs(ctx context.Context, houseIDs []string, period string) ([]model.HouseCost, error) {
	var costs []model.HouseCost
	if len(houseIDs) == 0 {
		return costs, nil
	}

	err := r.db.NewSelect().
		Model(&costs).
		Where("house_id IN (?)", bun.In(houseIDs)).
		Where("period = ?", period).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("list house costs: %w", err)
	}
	return costs, nil
}

// Update completely updates an existing house cost record.
// Note: Verification of ownership (managerID) should be done by the service layer.
func (r *HouseCostRepository) Update(ctx context.Context, cost *model.HouseCost) error {
	res, err := r.db.NewUpdate().
		Model(cost).
		ExcludeColumn("created_at").
		Where("id = ?", cost.ID).
		Where("house_id = ?", cost.HouseID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("update house cost: %w", err)
	}
	rows, err := res.RowsAffected()
	if err == nil && rows == 0 {
		return errors.New("house cost not found")
	}
	return nil
}

// Delete removes a house cost record.
func (r *HouseCostRepository) Delete(ctx context.Context, id, houseID string) error {
	res, err := r.db.NewDelete().
		Model((*model.HouseCost)(nil)).
		Where("id = ?", id).
		Where("house_id = ?", houseID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("delete house cost: %w", err)
	}

	rows, err := res.RowsAffected()
	if err == nil && rows == 0 {
		return errors.New("house cost not found")
	}
	return nil
}
