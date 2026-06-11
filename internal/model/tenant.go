package model

import (
	"context"
	"errors"
	"time"
)

var (
	ErrMaxTenans      = errors.New("the room is full of capacity")
	ErrTenantNotFound = errors.New("tenant not found")
	ErrUnauthorized   = errors.New("unauthorized: manager does not own this tenant")
)

type TenantStatus string

const (
	TenantStatusActive   TenantStatus = "ACTIVE"
	TenantStatusInactive TenantStatus = "INACTIVE"
)

type Tenant struct {
	ID           string       `json:"id"`
	UserID       string       `json:"user_id"`
	RoomID       string       `json:"room_id"`
	ManagerID    string       `json:"manager_id"`
	IdentityCard string       `json:"identity_card"`
	CCCDPath     string       `json:"cccd_path"`
	ContractPath string       `json:"contract_path"`
	StartDate    time.Time    `json:"start_date"`
	EndDate      *time.Time   `json:"end_date"`
	Status       TenantStatus `json:"status"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

type FullInfoTenant struct {
	TenantID     string `json:"tenant_id"`
	UserID       string `json:"user_id"`
	RoomID       string `json:"room_id"`
	RoomName     string `json:"room_name,omitempty"`
	FullName     string `json:"full_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	ManagerID    string `json:"manager_id"`
	CCCDPath     string `json:"cccd_path"`
	IdentityCard string `json:"identity_card"`
	ContractPath string `json:"contract_path"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date,omitempty"`
	Status       string `json:"status"`
	ZaloUserID   string `json:"zalo_user_id,omitempty"`
}

// UpdateTenantInput holds optional fields for tenant profile updates.
// Only non-nil pointer fields will be written to the DB.
type UpdateTenantInput struct {
	IdentityCard *string
	CCCDPath     *string
	ContractPath *string
}

type TenantRepository interface {
	CreateTenantWithAccount(ctx context.Context, user *User, tenant *Tenant) error
	AssignRoom(ctx context.Context, tenant *Tenant) error
	GetCurrentNumTenantInRoom(ctx context.Context, roomID string) (int64, error)
	ListTenantByRoomID(ctx context.Context, managerID, roomID string) ([]FullInfoTenant, error)
	ListTenantByHouseID(ctx context.Context, managerID, houseID string) ([]FullInfoTenant, error)
	GetTenantByID(ctx context.Context, managerID, tenantID string) (*FullInfoTenant, error)
	UpdateTenant(ctx context.Context, tenantID string, input UpdateTenantInput) (*Tenant, error)
	// VerifyTenantOwnership checks that the tenant exists and belongs to the manager.
	// Returns ErrTenantNotFound or ErrUnauthorized as appropriate.
	VerifyTenantOwnership(ctx context.Context, managerID, tenantID string) error
	// DeleteTenant marks the tenant profile inactive after verifying ownership.
	// Returns the room_id the tenant was in so the caller can decide whether to
	// update the room status.
	DeleteTenant(ctx context.Context, tenantID string) (roomID string, err error)
	GetTenantByPhoneAndManager(ctx context.Context, managerID, phone string) (*Tenant, error)
	GetFirstTenantByUserID(ctx context.Context, managerID, userID string) (*FullInfoTenant, error)
}
