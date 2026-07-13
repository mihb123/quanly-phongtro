package tenant_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/repository/repotest"
	"github.com/mihb123/quanly-phongtro/internal/repository/tenant"
)

var errDB = errors.New("db error")

func checkError(t *testing.T, err, expectedErr error, wantErr bool) {
	t.Helper()
	if wantErr {
		if err == nil {
			t.Errorf("expected error %v, got nil", expectedErr)
		} else if expectedErr != nil && !errors.Is(err, expectedErr) {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	} else if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// CreateTenantWithAccount tests
func TestTenantRepository_CreateTenantWithAccount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		user    *model.User
		tenant  *model.Tenant
		mockFn  func(mock sqlmock.Sqlmock)
		wantErr bool
		err     error
	}{
		{
			name:   "happy path",
			user:   &model.User{Email: "test@example.com"},
			tenant: &model.Tenant{RoomID: "room-1"},
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO "users"`).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("u1", time.Now(), time.Now()))
				mock.ExpectQuery(`INSERT INTO "tenants"`).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("t1", time.Now(), time.Now()))
				mock.ExpectExec(`UPDATE "rooms"`).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name:   "begin tx error",
			user:   &model.User{Email: "test@example.com"},
			tenant: &model.Tenant{RoomID: "room-1"},
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(errDB)
			},
			wantErr: true,
			err:     errDB,
		},
		{
			name:   "insert user duplicate error",
			user:   &model.User{Email: "test@example.com"},
			tenant: &model.Tenant{RoomID: "room-1"},
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO "users"`).
					WillReturnError(&pq.Error{Code: "23505", Constraint: "users_email"})
				mock.ExpectRollback()
			},
			wantErr: true,
			err:     model.ErrAlreadyExists,
		},
		{
			name:   "insert tenant error",
			user:   &model.User{Email: "test@example.com"},
			tenant: &model.Tenant{RoomID: "room-1"},
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO "users"`).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("u1", time.Now(), time.Now()))
				mock.ExpectQuery(`INSERT INTO "tenants"`).WillReturnError(errDB)
				mock.ExpectRollback()
			},
			wantErr: true,
			err:     errDB,
		},
		{
			name:   "update room error",
			user:   &model.User{Email: "test@example.com"},
			tenant: &model.Tenant{RoomID: "room-1"},
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO "users"`).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("u1", time.Now(), time.Now()))
				mock.ExpectQuery(`INSERT INTO "tenants"`).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("t1", time.Now(), time.Now()))
				mock.ExpectExec(`UPDATE "rooms"`).WillReturnError(errDB)
				mock.ExpectRollback()
			},
			wantErr: true,
			err:     errDB,
		},
		{
			name:   "update room not found",
			user:   &model.User{Email: "test@example.com"},
			tenant: &model.Tenant{RoomID: "room-1"},
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO "users"`).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("u1", time.Now(), time.Now()))
				mock.ExpectQuery(`INSERT INTO "tenants"`).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("t1", time.Now(), time.Now()))
				mock.ExpectExec(`UPDATE "rooms"`).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectRollback()
			},
			wantErr: true,
			err:     model.ErrNotFound,
		},
		{
			name:   "commit error",
			user:   &model.User{Email: "test@example.com"},
			tenant: &model.Tenant{RoomID: "room-1"},
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(`INSERT INTO "users"`).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("u1", time.Now(), time.Now()))
				mock.ExpectQuery(`INSERT INTO "tenants"`).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("t1", time.Now(), time.Now()))
				mock.ExpectExec(`UPDATE "rooms"`).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit().WillReturnError(errDB)
			},
			wantErr: true,
			err:     errDB,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()

			repo := tenant.NewTenantRepository(bunDB)
			tt.mockFn(mock)

			err := repo.CreateTenantWithAccount(context.Background(), tt.user, tt.tenant)
			checkError(t, err, tt.err, tt.wantErr)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// DeleteTenant tests
func TestTenantRepository_DeleteTenant(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mockFn  func(mock sqlmock.Sqlmock)
		wantID  string
		wantErr bool
		err     error
	}{
		{
			name: "happy path",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE "tenants"`).WillReturnRows(sqlmock.NewRows([]string{"room_id"}).AddRow("room-1"))
			},
			wantID:  "room-1",
			wantErr: false,
		},
		{
			name: "not found",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE "tenants"`).WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			err:     model.ErrTenantNotFound,
		},
		{
			name: "db error",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE "tenants"`).WillReturnError(errDB)
			},
			wantErr: true,
			err:     errDB,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := tenant.NewTenantRepository(bunDB)
			tt.mockFn(mock)

			roomID, err := repo.DeleteTenant(context.Background(), "tenant-1")
			checkError(t, err, tt.err, tt.wantErr)
			if !tt.wantErr && roomID != tt.wantID {
				t.Errorf("expected roomID %v, got %v", tt.wantID, roomID)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// GetTenantByPhoneAndManager tests
func TestTenantRepository_GetTenantByPhoneAndManager(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mockFn  func(mock sqlmock.Sqlmock)
		wantErr bool
		err     error
	}{
		{
			name: "happy path",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM tenants`).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("tenant-1"))
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM tenants`).WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			err:     model.ErrTenantNotFound,
		},
		{
			name: "db error",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM tenants`).WillReturnError(errDB)
			},
			wantErr: true,
			err:     errDB,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := tenant.NewTenantRepository(bunDB)
			tt.mockFn(mock)

			tenant, err := repo.GetTenantByPhoneAndManager(context.Background(), "manager-1", "123456")
			checkError(t, err, tt.err, tt.wantErr)
			if !tt.wantErr && tenant == nil {
				t.Errorf("expected tenant, got nil")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// GetTenantByID tests
func TestTenantRepository_GetTenantByID(t *testing.T) {
	t.Parallel()
	cols := []string{
		"tenant_id", "user_id", "room_id", "manager_id",
		"full_name", "email", "phone",
		"cccd_path", "identity_card", "contract_path",
		"start_date", "end_date", "status", "zalo_user_id",
	}
	tests := []struct {
		name    string
		mockFn  func(mock sqlmock.Sqlmock)
		wantErr bool
		err     error
	}{
		{
			name: "happy path",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM tenants`).WillReturnRows(
					sqlmock.NewRows(cols).AddRow("t1", "u1", "r1", "manager-1", "Name", "e", "p", "", "", "", "2023-01-01T00:00:00Z", "", "ACTIVE", ""))
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM tenants`).WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			err:     model.ErrTenantNotFound,
		},
		{
			name: "unauthorized",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM tenants`).WillReturnRows(
					sqlmock.NewRows(cols).AddRow("t1", "u1", "r1", "manager-2", "Name", "e", "p", "", "", "", "2023-01-01T00:00:00Z", "", "ACTIVE", ""))
			},
			wantErr: true,
			err:     model.ErrUnauthorized,
		},
		{
			name: "db error",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM tenants`).WillReturnError(errDB)
			},
			wantErr: true,
			err:     errDB,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := tenant.NewTenantRepository(bunDB)
			tt.mockFn(mock)

			tenant, err := repo.GetTenantByID(context.Background(), "manager-1", "t1")
			checkError(t, err, tt.err, tt.wantErr)
			if !tt.wantErr && tenant == nil {
				t.Errorf("expected tenant, got nil")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// UpdateTenant tests
func TestTenantRepository_UpdateTenant(t *testing.T) {
	t.Parallel()
	ic := "123"
	now := time.Now()
	cols := []string{
		"id", "user_id", "room_id", "manager_id",
		"identity_card", "cccd_path", "contract_path",
		"start_date", "end_date", "status",
		"created_at", "updated_at",
	}
	tests := []struct {
		name    string
		input   model.UpdateTenantInput
		mockFn  func(mock sqlmock.Sqlmock)
		wantErr bool
		err     error
	}{
		{
			name:  "happy path update",
			input: model.UpdateTenantInput{IdentityCard: &ic},
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE "tenants"`).WillReturnRows(
					sqlmock.NewRows(cols).AddRow("t1", "u1", "r1", "m1", "123", "", "", now, now, "ACTIVE", now, now))
			},
			wantErr: false,
		},
		{
			name:  "update no rows",
			input: model.UpdateTenantInput{IdentityCard: &ic},
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE "tenants"`).WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			err:     model.ErrTenantNotFound,
		},
		{
			name:  "db error",
			input: model.UpdateTenantInput{IdentityCard: &ic},
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE "tenants"`).WillReturnError(errDB)
			},
			wantErr: true,
			err:     errDB,
		},
		{
			name:  "no updates needed",
			input: model.UpdateTenantInput{},
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "tenants"`).WillReturnRows(
					sqlmock.NewRows(cols).AddRow("t1", "u1", "r1", "m1", "123", "", "", now, now, "ACTIVE", now, now))
			},
			wantErr: false,
		},
		{
			name:  "no updates needed not found",
			input: model.UpdateTenantInput{},
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "tenants"`).WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			err:     model.ErrTenantNotFound,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := tenant.NewTenantRepository(bunDB)
			tt.mockFn(mock)

			tenant, err := repo.UpdateTenant(context.Background(), "t1", tt.input)
			checkError(t, err, tt.err, tt.wantErr)
			if !tt.wantErr && tenant == nil {
				t.Errorf("expected tenant, got nil")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// GetFirstTenantByUserID tests
func TestTenantRepository_GetFirstTenantByUserID(t *testing.T) {
	t.Parallel()
	cols := []string{
		"tenant_id", "user_id", "room_id", "manager_id",
		"full_name", "email", "phone",
		"cccd_path", "identity_card", "contract_path",
		"start_date", "end_date", "status", "zalo_user_id", "room_name",
	}
	tests := []struct {
		name    string
		mockFn  func(mock sqlmock.Sqlmock)
		wantErr bool
		err     error
	}{
		{
			name: "happy path",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM tenants`).WillReturnRows(
					sqlmock.NewRows(cols).AddRow("t1", "u1", "r1", "m1", "Name", "e", "p", "", "", "", "2023-01-01T00:00:00Z", "", "ACTIVE", "", "Room 1"))
			},
			wantErr: false,
		},
		{
			name: "not found",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM tenants`).WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			err:     model.ErrTenantNotFound,
		},
		{
			name: "db error",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM tenants`).WillReturnError(errDB)
			},
			wantErr: true,
			err:     errDB,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := tenant.NewTenantRepository(bunDB)
			tt.mockFn(mock)

			tenant, err := repo.GetFirstTenantByUserID(context.Background(), "m1", "u1")
			checkError(t, err, tt.err, tt.wantErr)
			if !tt.wantErr && tenant == nil {
				t.Errorf("expected tenant, got nil")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// VerifyTenantOwnership tests
func TestTenantRepository_VerifyTenantOwnership(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		manager string
		mockFn  func(mock sqlmock.Sqlmock)
		wantErr bool
		err     error
	}{
		{
			name:    "happy path",
			manager: "m1",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "tenants"`).WillReturnRows(sqlmock.NewRows([]string{"manager_id"}).AddRow("m1"))
			},
			wantErr: false,
		},
		{
			name:    "unauthorized",
			manager: "m2",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "tenants"`).WillReturnRows(sqlmock.NewRows([]string{"manager_id"}).AddRow("m1"))
			},
			wantErr: true,
			err:     model.ErrUnauthorized,
		},
		{
			name:    "not found",
			manager: "m1",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "tenants"`).WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			err:     model.ErrTenantNotFound,
		},
		{
			name:    "db error",
			manager: "m1",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "tenants"`).WillReturnError(errDB)
			},
			wantErr: true,
			err:     errDB,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := tenant.NewTenantRepository(bunDB)
			tt.mockFn(mock)

			err := repo.VerifyTenantOwnership(context.Background(), tt.manager, "t1")
			checkError(t, err, tt.err, tt.wantErr)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// AssignRoom tests
func TestTenantRepository_AssignRoom(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		tenant  *model.Tenant
		mockFn  func(mock sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name:   "happy path",
			tenant: &model.Tenant{UserID: "u1", RoomID: "r1"},
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO "tenants"`).WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("t1", time.Now(), time.Now()))
			},
			wantErr: false,
		},
		{
			name:   "db error",
			tenant: &model.Tenant{UserID: "u1", RoomID: "r1"},
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO "tenants"`).WillReturnError(errDB)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := tenant.NewTenantRepository(bunDB)
			tt.mockFn(mock)

			err := repo.AssignRoom(context.Background(), tt.tenant)
			checkError(t, err, nil, tt.wantErr)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// GetCurrentNumTenantInRoom tests
func TestTenantRepository_GetCurrentNumTenantInRoom(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mockFn  func(mock sqlmock.Sqlmock)
		wantNum int64
		wantErr bool
	}{
		{
			name: "happy path",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT count`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
			},
			wantNum: 2,
			wantErr: false,
		},
		{
			name: "db error",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT count`).WillReturnError(errDB)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := tenant.NewTenantRepository(bunDB)
			tt.mockFn(mock)

			num, err := repo.GetCurrentNumTenantInRoom(context.Background(), "r1")
			checkError(t, err, nil, tt.wantErr)
			if !tt.wantErr && num != tt.wantNum {
				t.Errorf("expected %v, got %v", tt.wantNum, num)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// ListTenantByRoomID tests
func TestTenantRepository_ListTenantByRoomID(t *testing.T) {
	t.Parallel()
	cols := []string{
		"tenant_id", "user_id", "room_id", "manager_id",
		"full_name", "email", "phone",
		"cccd_path", "identity_card", "contract_path",
		"start_date", "end_date", "status", "zalo_user_id",
	}
	tests := []struct {
		name    string
		mockFn  func(mock sqlmock.Sqlmock)
		wantLen int
		wantErr bool
	}{
		{
			name: "happy path",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM tenants`).WillReturnRows(
					sqlmock.NewRows(cols).AddRow("t1", "u1", "r1", "m1", "Name", "e", "p", "", "", "", "2023-01-01T00:00:00Z", "", "ACTIVE", ""))
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "db error",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM tenants`).WillReturnError(errDB)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := tenant.NewTenantRepository(bunDB)
			tt.mockFn(mock)

			tenants, err := repo.ListTenantByRoomID(context.Background(), "m1", "r1")
			checkError(t, err, nil, tt.wantErr)
			if !tt.wantErr && len(tenants) != tt.wantLen {
				t.Errorf("expected %v tenants, got %v", tt.wantLen, len(tenants))
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// ListTenantByHouseID tests
func TestTenantRepository_ListTenantByHouseID(t *testing.T) {
	t.Parallel()
	cols := []string{
		"tenant_id", "user_id", "room_id", "manager_id",
		"full_name", "email", "phone",
		"cccd_path", "identity_card", "contract_path",
		"start_date", "end_date", "status", "zalo_user_id", "room_name",
	}
	tests := []struct {
		name    string
		mockFn  func(mock sqlmock.Sqlmock)
		wantLen int
		wantErr bool
	}{
		{
			name: "happy path",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM tenants`).WillReturnRows(
					sqlmock.NewRows(cols).AddRow("t1", "u1", "r1", "m1", "Name", "e", "p", "", "", "", "2023-01-01T00:00:00Z", "", "ACTIVE", "", "Room 1"))
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "db error",
			mockFn: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM tenants`).WillReturnError(errDB)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := tenant.NewTenantRepository(bunDB)
			tt.mockFn(mock)

			tenants, err := repo.ListTenantByHouseID(context.Background(), "m1", "h1")
			checkError(t, err, nil, tt.wantErr)
			if !tt.wantErr && len(tenants) != tt.wantLen {
				t.Errorf("expected %v tenants, got %v", tt.wantLen, len(tenants))
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}
