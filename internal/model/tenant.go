package model

import (
	"context"
	"errors"
	"time"
)

var (
	ErrTenantNotFound = errors.New("tenant not found")
)

type TenantStatus string

const (
	TenantStatusActive   TenantStatus = "ACTIVE"
	TenantStatusInactive TenantStatus = "INACTIVE"
)

type Tenant struct {
	ID           string       `json:"id"`
	RoomID       string       `json:"room_id"`
	CreatedBy    string       `json:"created_by"`
	FullName     string       `json:"full_name"`
	Phone        string       `json:"phone"`
	IdentityCard string       `json:"identity_card"`
	Email        string       `json:"email,omitempty"`
	StartDate    time.Time    `json:"start_date"`
	EndDate      *time.Time   `json:"end_date,omitempty"`
	Status       TenantStatus `json:"status"`
	CCCDPath     string       `json:"cccd_path,omitempty"`
	ContractPath string       `json:"contract_path,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

type TenantRepository interface {
	CreateTenant(ctx context.Context, tenant *Tenant) error
	GetTenantByID(ctx context.Context, id string) (*Tenant, error)
	GetTenantByRoomID(ctx context.Context, roomID string) (*Tenant, error)
	ListActiveTenantsByRoomID(ctx context.Context, roomID string) ([]*Tenant, error)
	CountActiveTenantsByRoomID(ctx context.Context, roomID string) (int, error)
	UpdateTenantStatus(ctx context.Context, id string, status TenantStatus, endDate *time.Time) error
	UpdateTenantInfo(ctx context.Context, id string, params UpdateTenantParams) (*Tenant, error)
	DeleteTenant(ctx context.Context, id string) error
}

// UpdateTenantParams holds optional fields to update.
type UpdateTenantParams struct {
	FullName     *string
	Phone        *string
	Email        *string
	IdentityCard *string
	StartDate    *time.Time
	CCCDPath     *string
	ContractPath *string
}
