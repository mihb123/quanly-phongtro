package model

import (
	"context"
	"errors"
)

var ErrMaxTenans = errors.New("the room is full of capacity")
var ErrTenantNotFound = errors.New("tenant not found")
var ErrUnauthorized = errors.New("unauthorized: manager does not own this tenant")

type Tenant struct {
	ID        string
	TenantID  string
	RoomID    string
	ManagerID string
}

type FullInfoTenant struct {
	TenantID     string `json:"tenant_id"`
	RoomID       string `json:"room_id"`
	FullName     string `json:"full_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	ManagerID    string `json:"manager_id"`
	CCCDPath     string `json:"cccd_path"`
	IdentityCard string `json:"identity_card"`
	ContractPath string `json:"contract_path"`
}

type TenantRepository interface {
	AssignRoom(ctx context.Context, tenant Tenant) error
	GetCurrentNumTenantInRoom(ctx context.Context, roomID string) (int64, error)
	ListTenantByRoomID(ctx context.Context, managerID, roomID string) ([]FullInfoTenant, error)
	// VerifyTenantOwnership checks that the tenant exists and belongs to the manager.
	// Returns ErrTenantNotFound or ErrUnauthorized as appropriate.
	VerifyTenantOwnership(ctx context.Context, managerID, tenantID string) error
	// DeleteTenant removes the tenant row after verifying ownership.
	// Returns the room_id the tenant was in so the caller can decide whether to
	// update the room status.
	DeleteTenant(ctx context.Context, tenantID string) (roomID string, err error)
}
