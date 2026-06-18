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
	HouseCode               string    `json:"house_code"`
	Address                 string    `json:"address"`
	DefaultElectricityPrice float64   `json:"default_electricity_price"`
	DefaultWaterPrice       float64   `json:"default_water_price"`
	DefaultParkingPrice     float64   `json:"default_parking_price"`
	DefaultServicePrice     float64   `json:"default_service_price"`
	DefaultWifiPrice        float64   `json:"default_wifi_price"`
	ElectricityBillingType  string    `json:"electricity_billing_type"`
	WaterBillingType        string    `json:"water_billing_type"`
	ElectricityBillingUnit  string    `json:"electricity_billing_unit"`
	WaterBillingUnit        string    `json:"water_billing_unit"`
	ExtraPersonThreshold    int       `json:"extra_person_threshold"`
	ExtraPersonFee          float64   `json:"extra_person_fee"`
	ExtraVehicleThreshold   int       `json:"extra_vehicle_threshold"`
	ExtraVehicleFee         float64   `json:"extra_vehicle_fee"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// UpdateHouseParams holds the full fields for a house update.
type UpdateHouseParams struct {
	Name                    string
	HouseCode               string
	Address                 string
	DefaultElectricityPrice float64
	DefaultWaterPrice       float64
	DefaultWifiPrice        float64
	DefaultParkingPrice     float64
	DefaultServicePrice     float64
	ElectricityBillingType  string
	WaterBillingType        string
	ElectricityBillingUnit  string
	WaterBillingUnit        string
	ExtraPersonThreshold    int
	ExtraPersonFee          float64
	ExtraVehicleThreshold   int
	ExtraVehicleFee         float64
}

type HouseRepository interface {
	CreateHouse(ctx context.Context, house *House) error
	GetByID(ctx context.Context, id, managerID string) (*House, error)
	GetHouseByCode(ctx context.Context, managerID, houseCode string) (*House, error)
	ListHouseByManagerID(ctx context.Context, managerID string, limit, offset int, search string) ([]House, error)
	UpdateHouse(ctx context.Context, id, managerID string, params UpdateHouseParams) (*House, error)
	DeleteHouse(ctx context.Context, id, managerID string) error
	// IsHouseOwnedBy returns true when the house exists and belongs to managerID.
	IsHouseOwnedBy(ctx context.Context, houseID, managerID string) (bool, error)
}
