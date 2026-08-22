package model

import (
	"context"
	"errors"
	"time"
)

var (
	ErrHouseNotFound   = errors.New("house not found")
	ErrRoomNotFound    = errors.New("room not found")
	ErrHouseCodeExists = errors.New("Mã nhà đã tồn tại")
)

type House struct {
	ID                      string  `json:"id"`
	ManagerID               string  `json:"manager_id"`
	Name                    string  `json:"name"`
	HouseCode               string  `json:"house_code"`
	Address                 string  `json:"address"`
	DefaultElectricityPrice float64 `json:"default_electricity_price"`
	DefaultWaterPrice       float64 `json:"default_water_price"`
	DefaultParkingPrice     float64 `json:"default_parking_price"`
	DefaultServicePrice     float64 `json:"default_service_price"`
	DefaultWifiPrice        float64 `json:"default_wifi_price"`
	ElectricityBillingType  string  `json:"electricity_billing_type"`
	WaterBillingType        string  `json:"water_billing_type"`
	ElectricityBillingUnit  string  `json:"electricity_billing_unit"`
	WaterBillingUnit        string  `json:"water_billing_unit"`
	ExtraPersonThreshold    int     `json:"extra_person_threshold"`
	ExtraPersonFee          float64 `json:"extra_person_fee"`
	ExtraVehicleThreshold   int     `json:"extra_vehicle_threshold"`
	ExtraVehicleFee         float64 `json:"extra_vehicle_fee"`

	// Thông tin thuê nguyên căn từ chủ nhà (manager là bên đi thuê rồi cho thuê lại từng phòng).
	OwnerName      string     `json:"owner_name"`
	OwnerPhone     string     `json:"owner_phone"`
	OwnerRentPrice float64    `json:"owner_rent_price"`
	OwnerDeposit   float64    `json:"owner_deposit"`
	RentStartDate  *time.Time `json:"rent_start_date"`
	RentEndDate    *time.Time `json:"rent_end_date"`
	// OwnerCCCDPath, OwnerContractPath: nhiều file ngăn cách bởi dấu phẩy, cập nhật qua endpoint multipart riêng.
	OwnerCCCDPath     string `json:"owner_cccd_path"`
	OwnerContractPath string `json:"owner_contract_path"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	OwnerName               string
	OwnerPhone              string
	OwnerRentPrice          float64
	OwnerDeposit            float64
	RentStartDate           *time.Time
	RentEndDate             *time.Time
}

type HouseRepository interface {
	CreateHouse(ctx context.Context, house *House) error
	GetByID(ctx context.Context, id, managerID string) (*House, error)
	GetHouseByCode(ctx context.Context, managerID, houseCode string) (*House, error)
	// IsHouseCodeTaken reports whether houseCode is already used by any house
	// system-wide (case-insensitive), ignoring the house identified by excludeHouseID.
	IsHouseCodeTaken(ctx context.Context, houseCode, excludeHouseID string) (bool, error)
	ListHouseByManagerID(ctx context.Context, managerID string, limit, offset int, search string) ([]House, error)
	UpdateHouse(ctx context.Context, id, managerID string, params UpdateHouseParams) (*House, error)
	// UpdateHouseDocuments ghi đè đường dẫn CCCD chủ nhà và hợp đồng thuê nguyên căn của nhà.
	UpdateHouseDocuments(ctx context.Context, id, managerID, cccdPath, contractPath string) (*House, error)
	// HasHouseWithFilePath reports whether a house owned by the manager references the upload path.
	HasHouseWithFilePath(ctx context.Context, managerID, filePath string) (bool, error)
	DeleteHouse(ctx context.Context, id, managerID string) error
	// IsHouseOwnedBy returns true when the house exists and belongs to managerID.
	IsHouseOwnedBy(ctx context.Context, houseID, managerID string) (bool, error)
}
