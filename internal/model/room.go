package model

import (
	"context"
	"time"
)

type Room struct {
	ID               string    `json:"id"`
	HouseID          string    `json:"house_id"`
	Name             string    `json:"name"`
	Price            int64     `json:"price"`
	MaxTenants       int       `json:"max_tenants"`
	Status           string    `json:"status"`
	ElectricityPrice *float64  `json:"electricity_price,omitempty"`
	WaterPrice       *float64  `json:"water_price,omitempty"`
	WifiPrice        *float64  `json:"wifi_price,omitempty"`
	ParkingPrice     *float64  `json:"parking_price,omitempty"`
	ServicePrice     *float64  `json:"service_price,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// UpdateRoomParams holds the fields for a full room update (prices are pointers for NULL).
type UpdateRoomParams struct {
	Name             string
	Price            int64
	MaxTenants       int
	Status           string
	ElectricityPrice *float64
	WaterPrice       *float64
	WifiPrice        *float64
	ParkingPrice     *float64
	ServicePrice     *float64
}

type RoomRepository interface {
	CreateRoom(ctx context.Context, room *Room) error
	GetRoomByID(ctx context.Context, id, houseID string) (*Room, error)
	ListRoomsByHouseID(ctx context.Context, houseID string, limit, offset int) ([]Room, error)
	ListAllRoomsByHouseID(ctx context.Context, houseID string) ([]Room, error)
	UpdateRoom(ctx context.Context, id, houseID string, params UpdateRoomParams) (*Room, error)
	DeleteRoom(ctx context.Context, id, houseID string) error
	GetMaxTenants(ctx context.Context, roomID string) (int64, error)
	UpdateRoomStatus(ctx context.Context, roomID, status string) error
	GetRoomByIDOnly(ctx context.Context, id string) (*Room, error)
}
