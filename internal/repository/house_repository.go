package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/uptrace/bun"
)

type HouseRepository struct {
	db *bun.DB
}

func NewHouseRepository(db *bun.DB) *HouseRepository {
	return &HouseRepository{db: db}
}

func (r *HouseRepository) CreateHouse(ctx context.Context, h *model.House) error {
	_, err := r.db.NewInsert().
		Model(h).
		Column("manager_id", "name", "address", "default_electricity_price", "default_water_price", "default_wifi_price", "default_parking_price", "default_service_price").
		Returning("id, created_at, updated_at").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("create house: %w", err)
	}
	return nil
}

func (r *HouseRepository) GetByID(ctx context.Context, id, managerID string) (*model.House, error) {
	var h model.House
	err := r.db.NewSelect().
		Model(&h).
		Where("id = ? AND manager_id = ?", id, managerID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrHouseNotFound
		}
		return nil, fmt.Errorf("get house by id: %w", err)
	}
	return &h, nil
}

func (r *HouseRepository) ListHouseByManagerID(ctx context.Context, managerID string, limit, offset int, search string) ([]model.House, error) {
	var houses []model.House
	q := r.db.NewSelect().
		Model(&houses).
		Where("manager_id = ?", managerID).
		OrderExpr("created_at DESC, name ASC").
		Limit(limit).
		Offset(offset)

	if search != "" {
		q.Where("(name ILIKE ? OR address ILIKE ?)", "%"+search+"%", "%"+search+"%")
	}

	err := q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list houses: %w", err)
	}
	return houses, nil
}

func (r *HouseRepository) UpdateHouse(ctx context.Context, id, managerID string, params model.UpdateHouseParams) (*model.House, error) {
	q := r.db.NewUpdate().
		Model((*model.House)(nil)).
		Where("id = ? AND manager_id = ?", id, managerID).
		Returning("id, manager_id, name, address, default_electricity_price, default_water_price, default_wifi_price, default_parking_price, default_service_price, created_at, updated_at")

	updated := false
	if params.Name != nil {
		q.Set("name = ?", *params.Name)
		updated = true
	}
	if params.Address != nil {
		q.Set("address = ?", *params.Address)
		updated = true
	}
	if params.DefaultElectricityPrice != nil {
		q.Set("default_electricity_price = ?", *params.DefaultElectricityPrice)
		updated = true
	}
	if params.DefaultWaterPrice != nil {
		q.Set("default_water_price = ?", *params.DefaultWaterPrice)
		updated = true
	}
	if params.DefaultWifiPrice != nil {
		q.Set("default_wifi_price = ?", *params.DefaultWifiPrice)
		updated = true
	}
	if params.DefaultParkingPrice != nil {
		q.Set("default_parking_price = ?", *params.DefaultParkingPrice)
		updated = true
	}
	if params.DefaultServicePrice != nil {
		q.Set("default_service_price = ?", *params.DefaultServicePrice)
		updated = true
	}

	if !updated {
		return nil, fmt.Errorf("update house: no fields to update")
	}

	q.Set("updated_at = NOW()")

	var h model.House
	err := q.Scan(ctx, &h.ID, &h.ManagerID, &h.Name, &h.Address, &h.DefaultElectricityPrice, &h.DefaultWaterPrice, &h.DefaultWifiPrice, &h.DefaultParkingPrice, &h.DefaultServicePrice, &h.CreatedAt, &h.UpdatedAt)
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrHouseNotFound
		}
		return nil, fmt.Errorf("update house: %w", err)
	}
	return &h, nil
}

func (r *HouseRepository) DeleteHouse(ctx context.Context, id, managerID string) error {
	res, err := r.db.NewDelete().
		Model((*model.House)(nil)).
		Where("id = ? AND manager_id = ?", id, managerID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete house: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete house rows affected: %w", err)
	}
	if rows == 0 {
		return model.ErrHouseNotFound
	}
	return nil
}

func (r *HouseRepository) IsHouseOwnedBy(ctx context.Context, houseID, managerID string) (bool, error) {
	var exists int
	err := r.db.NewSelect().
		ColumnExpr("1").
		Model((*model.House)(nil)).
		Where("id = ? AND manager_id = ?", houseID, managerID).
		Scan(ctx, &exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("is house owned by: %w", err)
	}
	return true, nil
}
