package model

import (
	"context"
	"errors"
	"time"
)

var (
	ErrHouseNotFound = errors.New("house not found")
	ErrRoomNotFound  = errors.New("room not found")
)

type House struct {
	ID                      string    `json:"id"`
	ManagerID               string    `json:"manager_id"`
	Name                    string    `json:"name"`
	Address                 string    `json:"address"`
	DefaultElectricityPrice float64   `json:"default_electricity_price" bun:"default_electricity_price"`
	DefaultWaterPrice       float64   `json:"default_water_price" bun:"default_water_price"`
	DefaultParkingPrice     float64   `json:"default_parking_price" bun:"default_parking_price"`
	DefaultServicePrice     float64   `json:"default_service_price" bun:"default_service_price"`
	DefaultWifiPrice        float64   `json:"default_wifi_price" bun:"default_wifi_price"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type Room struct {
	ID         string    `json:"id"`
	HouseID    string    `json:"house_id"`
	Name       string    `json:"name"`
	Price      int64     `json:"price"`
	Maxtenants int       `json:"max_tenants" bun:"max_tenants"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
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
	// IsHouseOwnedBy returns true when the house exists and belongs to managerID.
	IsHouseOwnedBy(ctx context.Context, houseID, managerID string) (bool, error)
}

// UpdateRoomParams holds the fields that may be partially updated.
// Only non-nil fields are written to the UPDATE query.
type UpdateRoomParams struct {
	Name       *string
	Price      *int64
	Maxtenants *int
	Status     *string
}

type RoomRepository interface {
	CreateRoom(ctx context.Context, room *Room) error
	GetRoomByID(ctx context.Context, id, houseID string) (*Room, error)
	ListRoomsByHouseID(ctx context.Context, houseID string, limit, offset int) ([]Room, error)
	UpdateRoom(ctx context.Context, id, houseID string, params UpdateRoomParams) (*Room, error)
	DeleteRoom(ctx context.Context, id, houseID string) error
	GetMaxTenants(ctx context.Context, roomID string) (int64, error)
	UpdateRoomStatus(ctx context.Context, roomID, status string) error
}
