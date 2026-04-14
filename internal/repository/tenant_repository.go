package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

type TenantRepository struct {
	db *sql.DB
}

func NewTenantRepository(db *sql.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

// AssignRoom inserts a new row into the tenants table linking a user (tenant)
// to a room under a specific manager.
func (r *TenantRepository) AssignRoom(ctx context.Context, tenant model.Tenant) error {
	const query = `
		INSERT INTO tenants (tenant_id, room_id, manager_id)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.ExecContext(ctx, query, tenant.TenantID, tenant.RoomID, tenant.ManagerID)
	if err != nil {
		return fmt.Errorf("assign room: %w", err)
	}
	return nil
}

// GetCurrentNumTenantInRoom returns the number of active tenants in a room.
func (r *TenantRepository) GetCurrentNumTenantInRoom(ctx context.Context, roomID string) (int64, error) {
	const query = `SELECT COUNT(*) FROM tenants WHERE room_id = $1`
	var count int64
	err := r.db.QueryRowContext(ctx, query, roomID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get current num tenant in room: %w", err)
	}
	return count, nil
}

// ListTenantByRoomID retrieves full tenant information for all tenants in a
// given room, verifying that the room belongs to the requesting manager.
func (r *TenantRepository) ListTenantByRoomID(ctx context.Context, managerID, roomID string) ([]model.FullInfoTenant, error) {
	const query = `
		SELECT
			t.tenant_id,
			t.room_id,
			u.full_name,
			u.email,
			u.phone,
			t.manager_id,
			COALESCE(u.cccd_path, ''),
			COALESCE(u.identity_card, ''),
			COALESCE(u.contract_path, '')
		FROM tenants t
		JOIN users u ON u.id = t.tenant_id
		WHERE t.room_id = $1
		  AND t.manager_id = $2
	`
	rows, err := r.db.QueryContext(ctx, query, roomID, managerID)
	if err != nil {
		return nil, fmt.Errorf("list tenant by room id: %w", err)
	}
	defer rows.Close()

	var tenants []model.FullInfoTenant
	for rows.Next() {
		var ft model.FullInfoTenant
		if err := rows.Scan(
			&ft.TenantID,
			&ft.RoomID,
			&ft.FullName,
			&ft.Email,
			&ft.Phone,
			&ft.ManagerID,
			&ft.CCCDPath,
			&ft.IdentityCard,
			&ft.ContractPath,
		); err != nil {
			return nil, fmt.Errorf("list tenant by room id scan: %w", err)
		}
		tenants = append(tenants, ft)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tenant by room id rows: %w", err)
	}
	return tenants, nil
}

// VerifyTenantOwnership checks that a tenant exists in the tenants table and
// that it is managed by the given managerID.
// Returns model.ErrTenantNotFound if the tenant_id does not exist,
// or model.ErrUnauthorized if the manager does not own the tenant.
func (r *TenantRepository) VerifyTenantOwnership(ctx context.Context, managerID, tenantID string) error {
	const query = `
		SELECT manager_id
		FROM tenants
		WHERE tenant_id = $1
	`
	var storedManagerID string
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&storedManagerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.ErrTenantNotFound
		}
		return fmt.Errorf("verify tenant ownership: %w", err)
	}
	if storedManagerID != managerID {
		return model.ErrUnauthorized
	}
	return nil
}

// DeleteTenant removes the tenant row from the tenants table after verifying
// that the requesting manager owns the tenant.
// It returns the room_id the tenant was in so the caller can decide whether
// to update the room status.
func (r *TenantRepository) DeleteTenant(ctx context.Context, tenantID string) (string, error) {

	const query = `
		DELETE FROM tenants
		WHERE tenant_id = $1
		RETURNING room_id
	`
	var roomID string
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&roomID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", model.ErrTenantNotFound
		}
		return "", fmt.Errorf("delete tenant: %w", err)
	}
	return roomID, nil
}
