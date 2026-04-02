package model

import (
	"context"
	"errors"
	"time"
)

var (
	ErrHouseNotFound = errors.New("house not found")
)

type House struct {
	ID                      string    `json:"id"`
	ManagerID               string    `json:"manager_id"`
	Name                    string    `json:"name"`
	Address                 string    `json:"address"`
	DefaultElectricityPrice float64   `json:"default_electricity_price"`
	DefaultWaterPrice       float64   `json:"default_water_price"`
	DefaultParkingPrice     float64   `json:"default_parking_price"`
	DefaultServicePrice     float64   `json:"default_service_price"`
	DefaultWifiPrice        float64   `json:"default_wifi_price"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// UpdateHouseParams holds the fields that may be partially updated.
// Only non-nil fields are included in the UPDATE query.
type UpdateHouseParams struct {
	Name                    *string
	Address                 *string
	DefaultElectricityPrice *float64
	DefaultWaterPrice       *float64
	DefaultWifiPrice        *float64
	DefaultParkingPrice     *float64
	DefaultServicePrice     *float64
}

type HouseRepository interface {
	CreateHouse(ctx context.Context, house *House) error
	GetByID(ctx context.Context, id, managerID string) (*House, error)
	ListHouseByManagerID(ctx context.Context, managerID string, limit, offset int, search string) ([]House, error)
	UpdateHouse(ctx context.Context, id, managerID string, params UpdateHouseParams) (*House, error)
	DeleteHouse(ctx context.Context, id, managerID string) error
}
