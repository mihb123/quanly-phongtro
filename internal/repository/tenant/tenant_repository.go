package tenant

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/uptrace/bun"
)

type TenantRepository struct {
	db *bun.DB
}

func NewTenantRepository(db *bun.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

// CreateTenantWithAccount creates the login account, tenant profile, and room status in one transaction.
func (r *TenantRepository) CreateTenantWithAccount(ctx context.Context, user *model.User, tenant *model.Tenant) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tenant registration transaction: %w", err)
	}

	_, err = tx.NewInsert().
		Model(user).
		Column("email", "password_hash", "role", "full_name", "phone", "is_activated").
		Returning("id, created_at, updated_at").
		Exec(ctx)

	if err != nil {
		return rollbackTenantTx(tx, translateCreateUserError(err), "create tenant account")
	}

	tenant.UserID = user.ID

	_, err = tx.NewInsert().
		Model(tenant).
		Column("user_id", "room_id", "manager_id", "identity_card", "cccd_path", "contract_path", "start_date", "status").
		Returning("id, created_at, updated_at").
		Exec(ctx)

	if err != nil {
		return rollbackTenantTx(tx, err, "create tenant profile")
	}

	res, err := tx.NewUpdate().
		Model((*model.Room)(nil)).
		Set("status = ?", "OCCUPIED").
		Where("id = ?", tenant.RoomID).
		Exec(ctx)

	if err != nil {
		return rollbackTenantTx(tx, err, "update room status")
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return rollbackTenantTx(tx, err, "update room status rows affected")
	}
	if rowsAffected == 0 {
		return rollbackTenantTx(tx, model.ErrNotFound, "update room status")
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tenant registration transaction: %w", err)
	}
	return nil
}

// AssignRoom inserts a tenant profile linked to a tenant login account.
func (r *TenantRepository) AssignRoom(ctx context.Context, tenant *model.Tenant) error {
	_, err := r.db.NewInsert().
		Model(tenant).
		Column("user_id", "room_id", "manager_id", "identity_card", "cccd_path", "contract_path", "start_date", "status").
		Returning("id, created_at, updated_at").
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("assign room: %w", err)
	}
	return nil
}

// GetCurrentNumTenantInRoom returns the number of active tenants in a room.
func (r *TenantRepository) GetCurrentNumTenantInRoom(ctx context.Context, roomID string) (int64, error) {
	count, err := r.db.NewSelect().
		Model((*model.Tenant)(nil)).
		Where("room_id = ? AND status = ?", roomID, string(model.TenantStatusActive)).
		Count(ctx)

	if err != nil {
		return 0, fmt.Errorf("get current num tenant in room: %w", err)
	}
	return int64(count), nil
}

// ListTenantByRoomID retrieves tenant profile and account fields for a room.
func (r *TenantRepository) ListTenantByRoomID(ctx context.Context, managerID, roomID string) ([]model.FullInfoTenant, error) {
	var tenants []model.FullInfoTenant

	err := r.db.NewSelect().
		TableExpr("tenants AS t").
		ColumnExpr("t.id AS tenant_id, t.user_id, t.room_id, t.manager_id").
		ColumnExpr("u.full_name, u.email, u.phone").
		ColumnExpr("COALESCE(t.cccd_path, '') AS cccd_path").
		ColumnExpr("COALESCE(t.identity_card, '') AS identity_card").
		ColumnExpr("COALESCE(t.contract_path, '') AS contract_path").
		ColumnExpr("t.start_date, t.end_date, t.status, u.zalo_user_id").
		Join("JOIN users AS u ON u.id = t.user_id").
		Where("t.room_id = ?", roomID).
		Where("t.manager_id = ?", managerID).
		Where("t.status = ?", string(model.TenantStatusActive)).
		Scan(ctx, &tenants)

	if err != nil {
		return nil, fmt.Errorf("list tenant by room id: %w", err)
	}

	for i := range tenants {
		if tenants[i].StartDate != "" {
			tenants[i].StartDate = tenants[i].StartDate[:10]
		}
		if tenants[i].EndDate != "" {
			tenants[i].EndDate = tenants[i].EndDate[:10]
		}
	}

	return tenants, nil
}

// ListTenantByHouseID retrieves tenant profile and account fields for a house.
func (r *TenantRepository) ListTenantByHouseID(ctx context.Context, managerID, houseID string) ([]model.FullInfoTenant, error) {
	var tenants []model.FullInfoTenant

	err := r.db.NewSelect().
		TableExpr("tenants AS t").
		ColumnExpr("t.id AS tenant_id, t.user_id, t.room_id, t.manager_id").
		ColumnExpr("u.full_name, u.email, u.phone").
		ColumnExpr("COALESCE(t.cccd_path, '') AS cccd_path").
		ColumnExpr("COALESCE(t.identity_card, '') AS identity_card").
		ColumnExpr("COALESCE(t.contract_path, '') AS contract_path").
		ColumnExpr("t.start_date, t.end_date, t.status, u.zalo_user_id").
		ColumnExpr("rm.name AS room_name").
		Join("JOIN users AS u ON u.id = t.user_id").
		Join("JOIN rooms AS rm ON rm.id = t.room_id").
		Where("rm.house_id = ?", houseID).
		Where("t.manager_id = ?", managerID).
		Where("t.status = ?", string(model.TenantStatusActive)).
		Scan(ctx, &tenants)

	if err != nil {
		return nil, fmt.Errorf("list tenant by house id: %w", err)
	}

	for i := range tenants {
		if tenants[i].StartDate != "" {
			tenants[i].StartDate = tenants[i].StartDate[:10]
		}
		if tenants[i].EndDate != "" {
			tenants[i].EndDate = tenants[i].EndDate[:10]
		}
	}

	return tenants, nil
}

// GetTenantByID retrieves one tenant profile and its account fields.
func (r *TenantRepository) GetTenantByID(ctx context.Context, managerID, tenantID string) (*model.FullInfoTenant, error) {
	var ft model.FullInfoTenant
	err := r.db.NewSelect().
		TableExpr("tenants AS t").
		ColumnExpr("t.id AS tenant_id, t.user_id, t.room_id, t.manager_id").
		ColumnExpr("u.full_name, u.email, u.phone").
		ColumnExpr("COALESCE(t.cccd_path, '') AS cccd_path").
		ColumnExpr("COALESCE(t.identity_card, '') AS identity_card").
		ColumnExpr("COALESCE(t.contract_path, '') AS contract_path").
		ColumnExpr("t.start_date, t.end_date, t.status, u.zalo_user_id").
		Join("JOIN users AS u ON u.id = t.user_id").
		Where("t.id = ?", tenantID).
		Scan(ctx, &ft)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrTenantNotFound
		}
		return nil, fmt.Errorf("get tenant by id: %w", err)
	}
	if ft.ManagerID != managerID {
		return nil, model.ErrUnauthorized
	}

	if ft.StartDate != "" {
		ft.StartDate = ft.StartDate[:10]
	}
	if ft.EndDate != "" {
		ft.EndDate = ft.EndDate[:10]
	}

	return &ft, nil
}

// UpdateTenant partially updates tenant profile fields only.
func (r *TenantRepository) UpdateTenant(ctx context.Context, tenantID string, input model.UpdateTenantInput) (*model.Tenant, error) {
	q := r.db.NewUpdate().
		Model((*model.Tenant)(nil)).
		Where("id = ?", tenantID).
		Returning("id, user_id, room_id, manager_id").
		Returning("identity_card, cccd_path, contract_path").
		Returning("start_date, end_date, status").
		Returning("created_at, updated_at")

	updated := false
	if input.IdentityCard != nil {
		q.Set("identity_card = ?", *input.IdentityCard)
		updated = true
	}
	if input.CCCDPath != nil {
		q.Set("cccd_path = ?", *input.CCCDPath)
		updated = true
	}
	if input.ContractPath != nil {
		q.Set("contract_path = ?", *input.ContractPath)
		updated = true
	}
	if !updated {
		return r.getTenantRecordByID(ctx, tenantID)
	}

	q.Set("updated_at = NOW()")

	var tenant model.Tenant
	err := q.Scan(ctx, &tenant)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrTenantNotFound
		}
		return nil, fmt.Errorf("update tenant: %w", err)
	}
	return &tenant, nil
}

// VerifyTenantOwnership checks that a tenant profile belongs to the manager.
func (r *TenantRepository) VerifyTenantOwnership(ctx context.Context, managerID, tenantID string) error {
	var storedManagerID string
	err := r.db.NewSelect().
		Model((*model.Tenant)(nil)).
		Column("manager_id").
		Where("id = ?", tenantID).
		Scan(ctx, &storedManagerID)

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

// DeleteTenant marks the tenant profile inactive and returns its room id.
func (r *TenantRepository) DeleteTenant(ctx context.Context, tenantID string) (string, error) {
	var roomID string
	err := r.db.NewUpdate().
		Model((*model.Tenant)(nil)).
		Set("status = ?", string(model.TenantStatusInactive)).
		Set("end_date = COALESCE(end_date, CURRENT_DATE)").
		Set("updated_at = NOW()").
		Where("id = ?", tenantID).
		Returning("room_id").
		Scan(ctx, &roomID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", model.ErrTenantNotFound
		}
		return "", fmt.Errorf("delete tenant: %w", err)
	}
	return roomID, nil
}

// getTenantRecordByID retrieves the raw tenant record without account fields.
func (r *TenantRepository) getTenantRecordByID(ctx context.Context, tenantID string) (*model.Tenant, error) {
	var tenant model.Tenant
	err := r.db.NewSelect().
		Model(&tenant).
		Where("id = ?", tenantID).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrTenantNotFound
		}
		return nil, fmt.Errorf("get tenant record by id: %w", err)
	}
	return &tenant, nil
}

// GetTenantByPhoneAndManager retrieves an active tenant by their user phone number and manager ID.
func (r *TenantRepository) GetTenantByPhoneAndManager(ctx context.Context, managerID, phone string) (*model.Tenant, error) {
	var tenant model.Tenant
	err := r.db.NewSelect().
		TableExpr("tenants AS t").
		ColumnExpr("t.*").
		Join("JOIN users AS u ON u.id = t.user_id").
		Where("u.phone = ?", phone).
		Where("t.manager_id = ?", managerID).
		Where("t.status = ?", string(model.TenantStatusActive)).
		Scan(ctx, &tenant)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrTenantNotFound
		}
		return nil, fmt.Errorf("get tenant by phone: %w", err)
	}
	return &tenant, nil
}

// GetFirstTenantByUserID retrieves the first active tenant profile for a user.
func (r *TenantRepository) GetFirstTenantByUserID(ctx context.Context, managerID, userID string) (*model.FullInfoTenant, error) {
	var ft model.FullInfoTenant
	err := r.db.NewSelect().
		TableExpr("tenants AS t").
		ColumnExpr("t.id AS tenant_id, t.user_id, t.room_id, t.manager_id").
		ColumnExpr("u.full_name, u.email, u.phone").
		ColumnExpr("COALESCE(t.cccd_path, '') AS cccd_path").
		ColumnExpr("COALESCE(t.identity_card, '') AS identity_card").
		ColumnExpr("COALESCE(t.contract_path, '') AS contract_path").
		ColumnExpr("t.start_date, t.end_date, t.status, u.zalo_user_id").
		ColumnExpr("rm.name AS room_name").
		Join("JOIN users AS u ON u.id = t.user_id").
		Join("JOIN rooms AS rm ON rm.id = t.room_id").
		Where("t.user_id = ?", userID).
		Where("t.manager_id = ?", managerID).
		Where("t.status = ?", string(model.TenantStatusActive)).
		Limit(1).
		Scan(ctx, &ft)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrTenantNotFound
		}
		return nil, fmt.Errorf("get first tenant by user id: %w", err)
	}

	if ft.StartDate != "" {
		ft.StartDate = ft.StartDate[:10]
	}
	if ft.EndDate != "" {
		ft.EndDate = ft.EndDate[:10]
	}

	return &ft, nil
}

// GetTenantByFilePath retrieves the active tenant that owns an uploaded file path.
func (r *TenantRepository) GetTenantByFilePath(ctx context.Context, managerID, filePath string) (*model.FullInfoTenant, error) {
	var ft model.FullInfoTenant
	err := r.db.NewSelect().
		TableExpr("tenants AS t").
		ColumnExpr("t.id AS tenant_id, t.user_id, t.room_id, t.manager_id").
		ColumnExpr("u.full_name, u.email, u.phone").
		ColumnExpr("COALESCE(t.cccd_path, '') AS cccd_path").
		ColumnExpr("COALESCE(t.identity_card, '') AS identity_card").
		ColumnExpr("COALESCE(t.contract_path, '') AS contract_path").
		ColumnExpr("t.start_date, t.end_date, t.status, u.zalo_user_id").
		Join("JOIN users AS u ON u.id = t.user_id").
		Where("t.manager_id = ?", managerID).
		Where("t.status = ?", string(model.TenantStatusActive)).
		Where(`(
			EXISTS (
				SELECT 1 FROM unnest(string_to_array(COALESCE(t.cccd_path, ''), ',')) AS path
				WHERE trim(path) = ?
			)
			OR EXISTS (
				SELECT 1 FROM unnest(string_to_array(COALESCE(t.contract_path, ''), ',')) AS path
				WHERE trim(path) = ?
			)
		)`, filePath, filePath).
		Limit(1).
		Scan(ctx, &ft)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrTenantNotFound
		}
		return nil, fmt.Errorf("get tenant by file path: %w", err)
	}

	return &ft, nil
}

// rollbackTenantTx rolls back a tenant transaction and preserves the original cause.
func rollbackTenantTx(tx bun.Tx, cause error, operation string) error {
	if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
		return fmt.Errorf("%s: %w; rollback tenant transaction: %v", operation, cause, err)
	}
	return fmt.Errorf("%s: %w", operation, cause)
}

// translateCreateUserError maps database user insert errors to domain errors.
func translateCreateUserError(err error) error {
	if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" && strings.Contains(pqErr.Constraint, "users_email") {
		return model.ErrAlreadyExists
	}
	return err
}
