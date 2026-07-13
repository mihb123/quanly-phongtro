package house_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/repository/house"
	"github.com/mihb123/quanly-phongtro/internal/repository/repotest"
)

func TestHouseRepository_CreateHouse(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := house.NewHouseRepository(bunDB)
	ctx := context.Background()

	h := &model.House{
		ManagerID: "manager-1",
		Name:      "House 1",
	}

	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
		AddRow("house-1", time.Now(), time.Now())

	mock.ExpectQuery(`INSERT INTO "houses"`).WillReturnRows(rows)

	err := repo.CreateHouse(ctx, h)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestHouseRepository_GetByID(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := house.NewHouseRepository(bunDB)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "manager_id", "name"}).
		AddRow("house-1", "manager-1", "House 1")

	// Happy path
	mock.ExpectQuery(`SELECT .* FROM "houses"`).WillReturnRows(rows)
	h, err := repo.GetByID(ctx, "house-1", "manager-1")
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
	if h.ID != "house-1" {
		t.Errorf("expected house-1")
	}

	// Not found path
	mock.ExpectQuery(`SELECT .* FROM "houses"`).WillReturnError(sql.ErrNoRows)
	_, err = repo.GetByID(ctx, "house-1", "manager-1")
	if err != model.ErrHouseNotFound {
		t.Errorf("expected ErrHouseNotFound, got %v", err)
	}
}

func TestHouseRepository_ListHouseByManagerID(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := house.NewHouseRepository(bunDB)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "manager_id", "name"}).
		AddRow("house-1", "manager-1", "House 1").
		AddRow("house-2", "manager-1", "House 2")

	mock.ExpectQuery(`SELECT .* FROM "houses"`).WillReturnRows(rows)

	houses, err := repo.ListHouseByManagerID(ctx, "manager-1", 10, 0, "search-term")
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
	if len(houses) != 2 {
		t.Errorf("expected 2 houses, got %d", len(houses))
	}
}

func TestHouseRepository_UpdateHouse(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := house.NewHouseRepository(bunDB)
	ctx := context.Background()

	params := model.UpdateHouseParams{
		Name: "Updated House",
	}

	rows := sqlmock.NewRows([]string{
		"id", "manager_id", "name", "house_code", "address",
		"default_electricity_price", "default_water_price",
		"default_wifi_price", "default_parking_price",
		"default_service_price", "electricity_billing_type",
		"water_billing_type", "electricity_billing_unit",
		"water_billing_unit", "extra_person_threshold",
		"extra_person_fee", "extra_vehicle_threshold",
		"extra_vehicle_fee", "created_at", "updated_at",
	}).AddRow(
		"house-1", "manager-1", "Updated House", "h1", "",
		0.0, 0.0, 0.0, 0.0, 0.0, "", "", "", "", 0, 0.0, 0, 0.0, time.Now(), time.Now(),
	)

	mock.ExpectQuery(`UPDATE "houses"`).WillReturnRows(rows)

	h, err := repo.UpdateHouse(ctx, "house-1", "manager-1", params)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
	if h != nil && h.Name != "Updated House" {
		t.Errorf("unexpected name: %s", h.Name)
	}
}

func TestHouseRepository_DeleteHouse(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := house.NewHouseRepository(bunDB)
	ctx := context.Background()

	mock.ExpectExec(`DELETE FROM "houses"`).WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteHouse(ctx, "house-1", "manager-1")
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	// Not found path
	mock.ExpectExec(`DELETE FROM "houses"`).WillReturnResult(sqlmock.NewResult(0, 0))
	err = repo.DeleteHouse(ctx, "house-not-found", "manager-1")
	if err != model.ErrHouseNotFound {
		t.Errorf("expected ErrHouseNotFound, got %v", err)
	}
}

func TestHouseRepository_IsHouseOwnedBy(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := house.NewHouseRepository(bunDB)
	ctx := context.Background()

	// Owned
	mock.ExpectQuery(`SELECT 1 FROM "houses"`).WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
	owned, err := repo.IsHouseOwnedBy(ctx, "house-1", "manager-1")
	if err != nil || !owned {
		t.Errorf("expected true, got %v, err: %v", owned, err)
	}

	// Not owned
	mock.ExpectQuery(`SELECT 1 FROM "houses"`).WillReturnError(sql.ErrNoRows)
	owned, err = repo.IsHouseOwnedBy(ctx, "house-2", "manager-1")
	if err != nil || owned {
		t.Errorf("expected false without error, got %v, err: %v", owned, err)
	}
}
