package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

type TenantRepository struct {
	db *sql.DB
}

func NewTenantRepository(db *sql.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

// CreateTenant inserts a new tenant into the database.
func (r *TenantRepository) CreateTenant(ctx context.Context, tenant *model.Tenant) error {
	const query = `
		INSERT INTO tenants (room_id, created_by, full_name, phone, identity_card, email, start_date, status, cccd_path, contract_path)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`
	
	err := r.db.QueryRowContext(ctx, query,
		tenant.RoomID, tenant.CreatedBy, tenant.FullName, tenant.Phone,
		tenant.IdentityCard, tenant.Email, tenant.StartDate, tenant.Status,
		tenant.CCCDPath, tenant.ContractPath,
	).Scan(&tenant.ID, &tenant.CreatedAt, &tenant.UpdatedAt)
	
	if err != nil {
		return fmt.Errorf("create tenant: %w", err)
	}
	return nil
}

// GetTenantByID fetches a tenant by its ID.
func (r *TenantRepository) GetTenantByID(ctx context.Context, id string) (*model.Tenant, error) {
	const query = `
		SELECT id, room_id, created_by, full_name, phone, identity_card, email, start_date, end_date, status, cccd_path, contract_path, created_at, updated_at
		FROM tenants
		WHERE id = $1`
	
	var tenant model.Tenant
	var email, cccd, contract sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tenant.ID, &tenant.RoomID, &tenant.CreatedBy, &tenant.FullName,
		&tenant.Phone, &tenant.IdentityCard, &email, &tenant.StartDate, &tenant.EndDate,
		&tenant.Status, &cccd, &contract, &tenant.CreatedAt, &tenant.UpdatedAt,
	)
	if err == nil {
		tenant.Email = email.String
		tenant.CCCDPath = cccd.String
		tenant.ContractPath = contract.String
	}
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrTenantNotFound
		}
		return nil, fmt.Errorf("get tenant by id: %w", err)
	}
	return &tenant, nil
}

// GetTenantByRoomID fetches the ACTIVE tenant for a specific room.
func (r *TenantRepository) GetTenantByRoomID(ctx context.Context, roomID string) (*model.Tenant, error) {
	const query = `
		SELECT id, room_id, created_by, full_name, phone, identity_card, email, start_date, end_date, status, cccd_path, contract_path, created_at, updated_at
		FROM tenants
		WHERE room_id = $1 AND status = 'ACTIVE'
		ORDER BY created_at DESC LIMIT 1`
	
	var tenant model.Tenant
	var email, cccd, contract sql.NullString
	err := r.db.QueryRowContext(ctx, query, roomID).Scan(
		&tenant.ID, &tenant.RoomID, &tenant.CreatedBy, &tenant.FullName,
		&tenant.Phone, &tenant.IdentityCard, &email, &tenant.StartDate, &tenant.EndDate,
		&tenant.Status, &cccd, &contract, &tenant.CreatedAt, &tenant.UpdatedAt,
	)
	if err == nil {
		tenant.Email = email.String
		tenant.CCCDPath = cccd.String
		tenant.ContractPath = contract.String
	}
	
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrTenantNotFound
		}
		return nil, fmt.Errorf("get tenant by room id: %w", err)
	}
	return &tenant, nil
}
// ListActiveTenantsByRoomID fetches all ACTIVE tenants for a specific room.
func (r *TenantRepository) ListActiveTenantsByRoomID(ctx context.Context, roomID string) ([]*model.Tenant, error) {
	const query = `
		SELECT id, room_id, created_by, full_name, phone, identity_card, email, start_date, end_date, status, cccd_path, contract_path, created_at, updated_at
		FROM tenants
		WHERE room_id = $1 AND status = 'ACTIVE'
		ORDER BY created_at ASC`
	
	rows, err := r.db.QueryContext(ctx, query, roomID)
	if err != nil {
		return nil, fmt.Errorf("list active tenants by room id: %w", err)
	}
	defer rows.Close()
	
	var tenants []*model.Tenant
	for rows.Next() {
		var tenant model.Tenant
		var email, cccd, contract sql.NullString
		err := rows.Scan(
			&tenant.ID, &tenant.RoomID, &tenant.CreatedBy, &tenant.FullName,
			&tenant.Phone, &tenant.IdentityCard, &email, &tenant.StartDate, &tenant.EndDate,
			&tenant.Status, &cccd, &contract, &tenant.CreatedAt, &tenant.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan active tenant: %w", err)
		}
		tenant.Email = email.String
		tenant.CCCDPath = cccd.String
		tenant.ContractPath = contract.String
		tenants = append(tenants, &tenant)
	}
	return tenants, nil
}

// CountActiveTenantsByRoomID counts the current active tenants in a room.
func (r *TenantRepository) CountActiveTenantsByRoomID(ctx context.Context, roomID string) (int, error) {
	const query = `
		SELECT COUNT(*)
		FROM tenants
		WHERE room_id = $1 AND status = 'ACTIVE'`
	
	var count int
	err := r.db.QueryRowContext(ctx, query, roomID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active tenants: %w", err)
	}
	return count, nil
}

// UpdateTenantStatus updates the status and end date of a tenant manually.
func (r *TenantRepository) UpdateTenantStatus(ctx context.Context, id string, status model.TenantStatus, endDate *time.Time) error {
	const query = `
		UPDATE tenants
		SET status = $2, end_date = $3, updated_at = NOW()
		WHERE id = $1`
		
	result, err := r.db.ExecContext(ctx, query, id, status, endDate)
	if err != nil {
		return fmt.Errorf("update tenant status: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update tenant status rows affected: %w", err)
	}
	if rows == 0 {
		return model.ErrTenantNotFound
	}
	
	return nil
}

// UpdateTenantInfo partially updates tenant identifying information.
func (r *TenantRepository) UpdateTenantInfo(ctx context.Context, id string, params model.UpdateTenantParams) (*model.Tenant, error) {
	var setClauses []string
	args := []any{}
	argIdx := 1

	if params.FullName != nil {
		setClauses = append(setClauses, fmt.Sprintf("full_name = $%d", argIdx))
		args = append(args, *params.FullName)
		argIdx++
	}
	if params.Phone != nil {
		setClauses = append(setClauses, fmt.Sprintf("phone = $%d", argIdx))
		args = append(args, *params.Phone)
		argIdx++
	}
	if params.Email != nil {
		setClauses = append(setClauses, fmt.Sprintf("email = $%d", argIdx))
		args = append(args, *params.Email)
		argIdx++
	}
	if params.IdentityCard != nil {
		setClauses = append(setClauses, fmt.Sprintf("identity_card = $%d", argIdx))
		args = append(args, *params.IdentityCard)
		argIdx++
	}
	if params.StartDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("start_date = $%d", argIdx))
		args = append(args, *params.StartDate)
		argIdx++
	}
	if params.CCCDPath != nil {
		setClauses = append(setClauses, fmt.Sprintf("cccd_path = $%d", argIdx))
		args = append(args, *params.CCCDPath)
		argIdx++
	}
	if params.ContractPath != nil {
		setClauses = append(setClauses, fmt.Sprintf("contract_path = $%d", argIdx))
		args = append(args, *params.ContractPath)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil, fmt.Errorf("update tenant info: no fields to update")
	}
	setClauses = append(setClauses, "updated_at = NOW()")

	// Append WHERE args: id.
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE tenants
		SET    %s
		WHERE  id = $%d
		RETURNING id, room_id, created_by, full_name, phone, identity_card, email, start_date, end_date, status, cccd_path, contract_path, created_at, updated_at`,
		strings.Join(setClauses, ", "),
		argIdx,
	)

	var tenant model.Tenant
	var email, cccd, contract sql.NullString
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&tenant.ID, &tenant.RoomID, &tenant.CreatedBy, &tenant.FullName,
		&tenant.Phone, &tenant.IdentityCard, &email, &tenant.StartDate, &tenant.EndDate,
		&tenant.Status, &cccd, &contract, &tenant.CreatedAt, &tenant.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrTenantNotFound
		}
		return nil, fmt.Errorf("update tenant info: %w", err)
	}
	tenant.Email = email.String
	tenant.CCCDPath = cccd.String
	tenant.ContractPath = contract.String
	return &tenant, nil
}

// DeleteTenant deletes a tenant.
func (r *TenantRepository) DeleteTenant(ctx context.Context, id string) error {
	const query = `DELETE FROM tenants WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete tenant rows affected: %w", err)
	}
	if rows == 0 {
		return model.ErrTenantNotFound
	}
	return nil
}
