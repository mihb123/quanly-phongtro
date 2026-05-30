package model

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvoiceNotFound         = errors.New("invoice not found")
	ErrDuplicateInvoice        = errors.New("duplicate invoice for this room and period")
	ErrInvalidElectricityIndex = errors.New("new electricity index must be greater than or equal to old index")
	ErrInvalidWaterIndex       = errors.New("new water index must be greater than or equal to old index")
)

// Invoice represents the invoices table
type Invoice struct {
	ID                  string    `json:"id"`
	RoomID              string    `json:"room_id"`
	Period              string    `json:"period"` // format: yyyy-mm
	RoomFee             float64   `json:"room_fee"`
	OldElectricityIndex int       `json:"old_electricity_index"`
	NewElectricityIndex int       `json:"new_electricity_index"`
	ElectricityFee      float64   `json:"electricity_fee"`
	OldWaterIndex       int       `json:"old_water_index"`
	NewWaterIndex       int       `json:"new_water_index"`
	WaterFee            float64   `json:"water_fee"`
	WifiFee             float64   `json:"wifi_fee"`
	ParkingFee          float64   `json:"parking_fee"`
	ServiceFee          float64   `json:"service_fee"`
	OtherFee            float64   `json:"other_fee"`
	Discount            float64   `json:"discount"`
	TenantCount         int       `json:"tenant_count"`
	VehicleCount        int       `json:"vehicle_count"`
	ExtraPersonFee      float64   `json:"extra_person_fee"`
	ExtraVehicleFee     float64   `json:"extra_vehicle_fee"`
	TotalAmount         float64   `json:"total_amount"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
}

// InvoiceWithRoom includes the room name and house properties from JOIN
type InvoiceWithRoom struct {
	Invoice
	RoomName              string  `json:"room_name"`
	HouseID               string  `json:"house_id"`
	ExtraPersonThreshold  int     `json:"extra_person_threshold"`
	ExtraPersonFeeUnit    float64 `json:"extra_person_fee_unit"`
	ExtraVehicleThreshold int     `json:"extra_vehicle_threshold"`
	ExtraVehicleFeeUnit   float64 `json:"extra_vehicle_fee_unit"`
}

// InvoiceListFilter contains filters for listing invoices
type InvoiceListFilter struct {
	HouseID string
	RoomID  string
	Period  string
	Status  string
	Page    int
	Limit   int
}

// InvoiceRepository defines the contract for invoice database operations
type InvoiceRepository interface {
	CreateInvoice(ctx context.Context, invoice *Invoice) error
	GetInvoiceByID(ctx context.Context, managerID, id string) (*InvoiceWithRoom, error)
	ListInvoices(ctx context.Context, managerID string, filter InvoiceListFilter) ([]InvoiceWithRoom, error)
	UpdateInvoiceStatus(ctx context.Context, managerID, id, status string) (*Invoice, error)
	GetLatestInvoiceByRoomID(ctx context.Context, roomID string) (*Invoice, error)
	GetInvoiceByRoomAndPeriod(ctx context.Context, roomID, period string) (*Invoice, error)
	GetPreviousInvoice(ctx context.Context, roomID, period string) (*Invoice, error)
	GetUnpaidInvoicesByRoomID(ctx context.Context, roomID string) ([]Invoice, error)
	UpdateInvoice(ctx context.Context, managerID string, invoice *Invoice) error
}
