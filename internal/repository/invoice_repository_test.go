package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/repository"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func TestInvoiceRepository_CreateInvoice(t *testing.T) {
	bunDB, mock := setupInvoiceTestDB(t)
	defer bunDB.Close()

	repo := repository.NewInvoiceRepository(bunDB)
	ctx := context.Background()
	invoice := &model.Invoice{
		RoomID: "room-1",
		Period: "2023-10",
	}

	tests := []struct {
		name        string
		mock        func()
		wantErr     bool
		wantErrType error
	}{
		{
			name: "Happy Path",
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "created_at"}).
					AddRow("inv-1", time.Now())
				mock.ExpectQuery(`.*`).WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "Duplicate Error",
			mock: func() {
				mock.ExpectQuery(`.*`).WillReturnError(errors.New("duplicate key value violates unique constraint uq_invoices_room_period"))
			},
			wantErr:     true,
			wantErrType: model.ErrDuplicateInvoice,
		},
		{
			name: "DB Error",
			mock: func() {
				mock.ExpectQuery(`.*`).WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			err := repo.CreateInvoice(ctx, invoice)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if tt.wantErrType != nil && !errors.Is(err, tt.wantErrType) {
				t.Errorf("expected error type: %v, got: %v", tt.wantErrType, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestInvoiceRepository_GetInvoiceByID(t *testing.T) {
	bunDB, mock := setupInvoiceTestDB(t)
	defer bunDB.Close()

	repo := repository.NewInvoiceRepository(bunDB)
	ctx := context.Background()

	columns := []string{"id", "room_id", "period", "room_name", "house_id", "extra_person_threshold", "extra_person_fee_unit", "extra_vehicle_threshold", "extra_vehicle_fee_unit"}

	tests := []struct {
		name        string
		mock        func()
		wantInvoice *model.InvoiceWithRoom
		wantErr     bool
		wantErrType error
	}{
		{
			name: "Happy Path",
			mock: func() {
				rows := sqlmock.NewRows(columns).
					AddRow("inv-1", "room-1", "2023-10", "Room 1", "house-1", 0, 0.0, 0, 0.0)
				mock.ExpectQuery(`.*`).WillReturnRows(rows)
			},
			wantInvoice: &model.InvoiceWithRoom{
				Invoice: model.Invoice{
					ID:     "inv-1",
					RoomID: "room-1",
					Period: "2023-10",
				},
				RoomName: "Room 1",
				HouseID:  "house-1",
			},
			wantErr: false,
		},
		{
			name: "Not Found",
			mock: func() {
				mock.ExpectQuery(`.*`).WillReturnError(sql.ErrNoRows)
			},
			wantErr:     true,
			wantErrType: model.ErrInvoiceNotFound,
		},
		{
			name: "DB Error",
			mock: func() {
				mock.ExpectQuery(`.*`).WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			inv, err := repo.GetInvoiceByID(ctx, "manager-1", "inv-1")
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if tt.wantErrType != nil && !errors.Is(err, tt.wantErrType) {
				t.Errorf("expected error type: %v, got: %v", tt.wantErrType, err)
			}
			if !tt.wantErr && inv != nil && tt.wantInvoice != nil && inv.ID != tt.wantInvoice.ID {
				t.Errorf("expected invoice ID: %s, got: %s", tt.wantInvoice.ID, inv.ID)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestInvoiceRepository_ListInvoices(t *testing.T) {
	bunDB, mock := setupInvoiceTestDB(t)
	defer bunDB.Close()

	repo := repository.NewInvoiceRepository(bunDB)
	ctx := context.Background()

	columns := []string{"id", "room_id", "period", "status", "room_name", "house_id", "extra_person_threshold", "extra_person_fee_unit", "extra_vehicle_threshold", "extra_vehicle_fee_unit"}
	filter := model.InvoiceListFilter{
		RoomID:  "room-1",
		HouseID: "house-1",
		Period:  "2023-10",
		Status:  "UNPAID",
		Limit:   10,
		Page:    1,
	}

	tests := []struct {
		name      string
		mock      func()
		wantCount int
		wantErr   bool
	}{
		{
			name: "Happy Path - With Filters",
			mock: func() {
				rows := sqlmock.NewRows(columns).
					AddRow("inv-1", "room-1", "2023-10", "UNPAID", "Room 1", "house-1", 0, 0.0, 0, 0.0)
				mock.ExpectQuery(`.*`).WillReturnRows(rows)
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "DB Error",
			mock: func() {
				mock.ExpectQuery(`.*`).WillReturnError(errors.New("db error"))
			},
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			invs, err := repo.ListInvoices(ctx, "manager-1", filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if len(invs) != tt.wantCount {
				t.Errorf("expected %d invoices, got: %d", tt.wantCount, len(invs))
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestInvoiceRepository_UpdateInvoiceStatus(t *testing.T) {
	bunDB, mock := setupInvoiceTestDB(t)
	defer bunDB.Close()

	repo := repository.NewInvoiceRepository(bunDB)
	ctx := context.Background()

	tests := []struct {
		name        string
		mock        func()
		wantErr     bool
		wantErrType error
	}{
		{
			name: "Happy Path",
			mock: func() {
				rows := sqlmock.NewRows([]string{"id", "status"}).AddRow("inv-1", "PAID")
				mock.ExpectQuery(`.*`).WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "Not Found",
			mock: func() {
				mock.ExpectQuery(`.*`).WillReturnError(sql.ErrNoRows)
			},
			wantErr:     true,
			wantErrType: model.ErrInvoiceNotFound,
		},
		{
			name: "DB Error",
			mock: func() {
				mock.ExpectQuery(`.*`).WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			inv, err := repo.UpdateInvoiceStatus(ctx, "manager-1", "inv-1", "PAID")
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if tt.wantErrType != nil && !errors.Is(err, tt.wantErrType) {
				t.Errorf("expected error type: %v, got: %v", tt.wantErrType, err)
			}
			if !tt.wantErr && inv != nil && inv.Status != "PAID" {
				t.Errorf("expected status PAID, got: %s", inv.Status)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestInvoiceRepository_GetLatestInvoiceByRoomID(t *testing.T) {
	bunDB, mock := setupInvoiceTestDB(t)
	defer bunDB.Close()

	repo := repository.NewInvoiceRepository(bunDB)
	ctx := context.Background()

	columns := []string{"id", "room_id", "period"}

	tests := []struct {
		name        string
		mock        func()
		wantID      string
		wantErr     bool
		wantErrType error
	}{
		{
			name: "Happy Path",
			mock: func() {
				rows := sqlmock.NewRows(columns).AddRow("inv-1", "room-1", "2023-10")
				mock.ExpectQuery(`.*`).WillReturnRows(rows)
			},
			wantID:  "inv-1",
			wantErr: false,
		},
		{
			name: "Not Found",
			mock: func() {
				mock.ExpectQuery(`.*`).WillReturnError(sql.ErrNoRows)
			},
			wantErr:     true,
			wantErrType: model.ErrInvoiceNotFound,
		},
		{
			name: "DB Error",
			mock: func() {
				mock.ExpectQuery(`.*`).WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			inv, err := repo.GetLatestInvoiceByRoomID(ctx, "room-1")
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if tt.wantErrType != nil && !errors.Is(err, tt.wantErrType) {
				t.Errorf("expected error type: %v, got: %v", tt.wantErrType, err)
			}
			if !tt.wantErr && inv != nil && inv.ID != tt.wantID {
				t.Errorf("expected id %s, got: %s", tt.wantID, inv.ID)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestInvoiceRepository_GetInvoiceByRoomAndPeriod(t *testing.T) {
	bunDB, mock := setupInvoiceTestDB(t)
	defer bunDB.Close()

	repo := repository.NewInvoiceRepository(bunDB)
	ctx := context.Background()

	columns := []string{"id", "room_id", "period"}

	tests := []struct {
		name        string
		mock        func()
		wantID      string
		wantErr     bool
		wantErrType error
	}{
		{
			name: "Happy Path",
			mock: func() {
				rows := sqlmock.NewRows(columns).AddRow("inv-1", "room-1", "2023-10")
				mock.ExpectQuery(`.*`).WillReturnRows(rows)
			},
			wantID:  "inv-1",
			wantErr: false,
		},
		{
			name: "Not Found",
			mock: func() {
				mock.ExpectQuery(`.*`).WillReturnError(sql.ErrNoRows)
			},
			wantErr:     true,
			wantErrType: model.ErrInvoiceNotFound,
		},
		{
			name: "DB Error",
			mock: func() {
				mock.ExpectQuery(`.*`).WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			inv, err := repo.GetInvoiceByRoomAndPeriod(ctx, "room-1", "2023-10")
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if tt.wantErrType != nil && !errors.Is(err, tt.wantErrType) {
				t.Errorf("expected error type: %v, got: %v", tt.wantErrType, err)
			}
			if !tt.wantErr && inv != nil && inv.ID != tt.wantID {
				t.Errorf("expected id %s, got: %s", tt.wantID, inv.ID)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestInvoiceRepository_GetPreviousInvoice(t *testing.T) {
	bunDB, mock := setupInvoiceTestDB(t)
	defer bunDB.Close()

	repo := repository.NewInvoiceRepository(bunDB)
	ctx := context.Background()

	columns := []string{"id", "room_id", "period"}

	tests := []struct {
		name        string
		mock        func()
		wantID      string
		wantErr     bool
		wantErrType error
	}{
		{
			name: "Happy Path",
			mock: func() {
				rows := sqlmock.NewRows(columns).AddRow("inv-1", "room-1", "2023-09")
				mock.ExpectQuery(`.*`).WillReturnRows(rows)
			},
			wantID:  "inv-1",
			wantErr: false,
		},
		{
			name: "Not Found",
			mock: func() {
				mock.ExpectQuery(`.*`).WillReturnError(sql.ErrNoRows)
			},
			wantErr:     true,
			wantErrType: model.ErrInvoiceNotFound,
		},
		{
			name: "DB Error",
			mock: func() {
				mock.ExpectQuery(`.*`).WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			inv, err := repo.GetPreviousInvoice(ctx, "room-1", "2023-10")
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if tt.wantErrType != nil && !errors.Is(err, tt.wantErrType) {
				t.Errorf("expected error type: %v, got: %v", tt.wantErrType, err)
			}
			if !tt.wantErr && inv != nil && inv.ID != tt.wantID {
				t.Errorf("expected id %s, got: %s", tt.wantID, inv.ID)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestInvoiceRepository_GetUnpaidInvoicesByRoomID(t *testing.T) {
	bunDB, mock := setupInvoiceTestDB(t)
	defer bunDB.Close()

	repo := repository.NewInvoiceRepository(bunDB)
	ctx := context.Background()

	columns := []string{"id", "room_id", "status"}

	tests := []struct {
		name      string
		mock      func()
		wantCount int
		wantErr   bool
	}{
		{
			name: "Happy Path",
			mock: func() {
				rows := sqlmock.NewRows(columns).AddRow("inv-1", "room-1", "UNPAID")
				mock.ExpectQuery(`.*`).WillReturnRows(rows)
			},
			wantCount: 1,
			wantErr:   false,
		},
		{
			name: "DB Error",
			mock: func() {
				mock.ExpectQuery(`.*`).WillReturnError(errors.New("db error"))
			},
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			invs, err := repo.GetUnpaidInvoicesByRoomID(ctx, "room-1")
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if len(invs) != tt.wantCount {
				t.Errorf("expected %d invoices, got: %d", tt.wantCount, len(invs))
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestInvoiceRepository_GetLatestUnpaidInvoiceByRoomID(t *testing.T) {
	bunDB, mock := setupInvoiceTestDB(t)
	defer bunDB.Close()

	repo := repository.NewInvoiceRepository(bunDB)
	ctx := context.Background()

	columns := []string{"id", "room_id", "status"}

	tests := []struct {
		name        string
		mock        func()
		wantID      string
		wantErr     bool
		wantErrType error
	}{
		{
			name: "Happy Path",
			mock: func() {
				rows := sqlmock.NewRows(columns).AddRow("inv-1", "room-1", "UNPAID")
				mock.ExpectQuery(`.*`).WillReturnRows(rows)
			},
			wantID:  "inv-1",
			wantErr: false,
		},
		{
			name: "Not Found",
			mock: func() {
				mock.ExpectQuery(`.*`).WillReturnError(sql.ErrNoRows)
			},
			wantErr:     true,
			wantErrType: model.ErrInvoiceNotFound,
		},
		{
			name: "DB Error",
			mock: func() {
				mock.ExpectQuery(`.*`).WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			inv, err := repo.GetLatestUnpaidInvoiceByRoomID(ctx, "room-1")
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if tt.wantErrType != nil && !errors.Is(err, tt.wantErrType) {
				t.Errorf("expected error type: %v, got: %v", tt.wantErrType, err)
			}
			if !tt.wantErr && inv != nil && inv.ID != tt.wantID {
				t.Errorf("expected id %s, got: %s", tt.wantID, inv.ID)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestInvoiceRepository_UpdateInvoice(t *testing.T) {
	bunDB, mock := setupInvoiceTestDB(t)
	defer bunDB.Close()

	repo := repository.NewInvoiceRepository(bunDB)
	ctx := context.Background()
	inv := &model.Invoice{
		ID:     "inv-1",
		RoomID: "room-1",
	}

	tests := []struct {
		name        string
		mock        func()
		wantErr     bool
		wantErrType error
	}{
		{
			name: "Happy Path",
			mock: func() {
				mock.ExpectExec(`UPDATE "invoices"`).WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name: "Not Found (0 rows affected)",
			mock: func() {
				mock.ExpectExec(`UPDATE "invoices"`).WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr:     true,
			wantErrType: model.ErrInvoiceNotFound,
		},
		{
			name: "DB Error",
			mock: func() {
				mock.ExpectExec(`UPDATE "invoices"`).WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			err := repo.UpdateInvoice(ctx, "manager-1", inv)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if tt.wantErrType != nil && !errors.Is(err, tt.wantErrType) {
				t.Errorf("expected error type: %v, got: %v", tt.wantErrType, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestInvoiceRepository_DeleteInvoice(t *testing.T) {
	bunDB, mock := setupInvoiceTestDB(t)
	defer bunDB.Close()

	repo := repository.NewInvoiceRepository(bunDB)
	ctx := context.Background()

	tests := []struct {
		name        string
		mock        func()
		wantErr     bool
		wantErrType error
	}{
		{
			name: "Happy Path",
			mock: func() {
				mock.ExpectExec(`DELETE FROM "invoices"`).WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name: "Not Found (0 rows affected)",
			mock: func() {
				mock.ExpectExec(`DELETE FROM "invoices"`).WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr:     true,
			wantErrType: model.ErrInvoiceNotFound,
		},
		{
			name: "DB Error",
			mock: func() {
				mock.ExpectExec(`DELETE FROM "invoices"`).WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			err := repo.DeleteInvoice(ctx, "manager-1", "inv-1")
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error: %v, got: %v", tt.wantErr, err)
			}
			if tt.wantErrType != nil && !errors.Is(err, tt.wantErrType) {
				t.Errorf("expected error type: %v, got: %v", tt.wantErrType, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %s", err)
			}
		})
	}
}

func setupInvoiceTestDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	bunDB := bun.NewDB(db, pgdialect.New())
	return bunDB, mock
}

func TestInvoiceRepository_UpdateInvoiceStatusAndMethod(t *testing.T) {
	bunDB, mock := setupInvoiceTestDB(t)
	defer bunDB.Close()

	repo := repository.NewInvoiceRepository(bunDB)
	ctx := context.Background()

	tests := []struct {
		name    string
		mock    func()
		wantErr bool
	}{
		{
			name: "success",
			mock: func() {
				mock.ExpectQuery(`.*`).
					WillReturnRows(sqlmock.NewRows([]string{"id", "status", "payment_method"}).
						AddRow("inv-1", "PAID", "PAYOS"))
			},
			wantErr: false,
		},
		{
			name: "not found",
			mock: func() {
				mock.ExpectQuery(`.*`).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name: "db error",
			mock: func() {
				mock.ExpectQuery(`.*`).
					WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			_, err := repo.UpdateInvoiceStatusAndMethod(ctx, "mgr-1", "inv-1", "PAID", "PAYOS")
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateInvoiceStatusAndMethod() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestInvoiceRepository_SystemUpdateInvoiceStatusAndMethod(t *testing.T) {
	bunDB, mock := setupInvoiceTestDB(t)
	defer bunDB.Close()

	repo := repository.NewInvoiceRepository(bunDB)
	ctx := context.Background()

	tests := []struct {
		name    string
		mock    func()
		wantErr bool
	}{
		{
			name: "success",
			mock: func() {
				mock.ExpectQuery(`.*`).
					WillReturnRows(sqlmock.NewRows([]string{"id", "status", "payment_method"}).
						AddRow("inv-1", "PAID", "PAYOS"))

				mock.ExpectQuery(`.*`).
					WillReturnRows(sqlmock.NewRows([]string{"id", "room_name", "house_id", "manager_id"}).
						AddRow("inv-1", "Room 1", "house-1", "mgr-1"))
			},
			wantErr: false,
		},
		{
			name: "update not found",
			mock: func() {
				mock.ExpectQuery(`.*`).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name: "update db error",
			mock: func() {
				mock.ExpectQuery(`.*`).
					WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name: "select error",
			mock: func() {
				mock.ExpectQuery(`.*`).
					WillReturnRows(sqlmock.NewRows([]string{"id", "status", "payment_method"}).
						AddRow("inv-1", "PAID", "PAYOS"))

				mock.ExpectQuery(`.*`).
					WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			_, err := repo.SystemUpdateInvoiceStatusAndMethod(ctx, "inv-1", "PAID", "PAYOS")
			if (err != nil) != tt.wantErr {
				t.Errorf("SystemUpdateInvoiceStatusAndMethod() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
