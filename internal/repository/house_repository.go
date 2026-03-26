package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

type HouseRepository struct {
	db *sql.DB
}

func NewHouseRepository(db *sql.DB) *HouseRepository {
	return &HouseRepository{db: db}
}

func (r *HouseRepository) CreateHouse(ctx context.Context, h *model.House) error {
	query := `INSERT INTO houses (manager_id, name, address, default_electricity_price, default_water_price, default_wifi_price, default_parking_price, default_service_price)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			  RETURNING id, created_at, updated_at`
	err := r.db.QueryRowContext(ctx, query, h.ManagerID, h.Name, h.Address, h.DefaultElectricityPrice, h.DefaultWaterPrice, h.DefaultWifiPrice, h.DefaultParkingPrice, h.DefaultServicePrice).Scan(
		&h.ID,
		&h.CreatedAt,
		&h.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create house: %w", err)
	}
	return nil
}

func (r *HouseRepository) GetByID(ctx context.Context, id, managerID string) (*model.House, error) {
	const query = `
		SELECT id, manager_id, name, address, default_electricity_price, default_water_price,
		       default_wifi_price, default_parking_price, default_service_price, created_at, updated_at
		FROM houses
		WHERE id = $1 AND manager_id = $2
	`

	var h model.House
	err := r.db.QueryRowContext(ctx, query, id, managerID).Scan(
		&h.ID,
		&h.ManagerID,
		&h.Name,
		&h.Address,
		&h.DefaultElectricityPrice,
		&h.DefaultWaterPrice,
		&h.DefaultWifiPrice,
		&h.DefaultParkingPrice,
		&h.DefaultServicePrice,
		&h.CreatedAt,
		&h.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrHouseNotFound
		}

		return nil, fmt.Errorf("get house by id: %w", err)
	}

	return &h, nil
}
