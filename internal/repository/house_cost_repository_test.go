package repository_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/repository"
)

func TestHouseCostRepository_Create(t *testing.T) {
	bunDB, mock := setupTestDB(t)
	defer bunDB.Close()

	repo := repository.NewHouseCostRepository(bunDB)
	ctx := context.Background()

	cost := &model.HouseCost{
		ID:      "cost-1",
		HouseID: "house-1",
		Period:  "2023-10",
		Rent:    1000,
	}

	mock.ExpectQuery(`INSERT INTO "house_costs"`).WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("cost-1", nil, nil))

	err := repo.Create(ctx, cost)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestHouseCostRepository_GetByHouseAndPeriod(t *testing.T) {
	bunDB, mock := setupTestDB(t)
	defer bunDB.Close()

	repo := repository.NewHouseCostRepository(bunDB)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "house_id", "period"}).
		AddRow("cost-1", "house-1", "2023-10")

	// Found
	mock.ExpectQuery(`SELECT .* FROM "house_costs"`).WillReturnRows(rows)
	cost, err := repo.GetByHouseAndPeriod(ctx, "house-1", "2023-10")
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
	if cost.ID != "cost-1" {
		t.Errorf("expected cost-1")
	}

	// Not found
	mock.ExpectQuery(`SELECT .* FROM "house_costs"`).WillReturnError(sql.ErrNoRows)
	_, err = repo.GetByHouseAndPeriod(ctx, "house-1", "2023-11")
	if err == nil || err.Error() != "house cost not found" {
		t.Errorf("expected house cost not found, got %v", err)
	}
}

func TestHouseCostRepository_GetLatestByHouseID(t *testing.T) {
	bunDB, mock := setupTestDB(t)
	defer bunDB.Close()

	repo := repository.NewHouseCostRepository(bunDB)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "house_id", "period"}).
		AddRow("cost-2", "house-1", "2023-11")

	// Found
	mock.ExpectQuery(`SELECT .* FROM "house_costs"`).WillReturnRows(rows)
	cost, err := repo.GetLatestByHouseID(ctx, "house-1")
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
	if cost.ID != "cost-2" {
		t.Errorf("expected cost-2")
	}

	// Not found
	mock.ExpectQuery(`SELECT .* FROM "house_costs"`).WillReturnError(sql.ErrNoRows)
	_, err = repo.GetLatestByHouseID(ctx, "house-2")
	if err == nil || err.Error() != "house cost not found" {
		t.Errorf("expected house cost not found, got %v", err)
	}
}

func TestHouseCostRepository_ListByHouseIDs(t *testing.T) {
	bunDB, mock := setupTestDB(t)
	defer bunDB.Close()

	repo := repository.NewHouseCostRepository(bunDB)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "house_id", "period"}).
		AddRow("cost-1", "house-1", "2023-10").
		AddRow("cost-2", "house-2", "2023-10")

	// Empty ids
	costs, err := repo.ListByHouseIDs(ctx, []string{}, "2023-10")
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
	if len(costs) != 0 {
		t.Errorf("expected 0 costs")
	}

	// With ids
	mock.ExpectQuery(`SELECT .* FROM "house_costs"`).WillReturnRows(rows)
	costs, err = repo.ListByHouseIDs(ctx, []string{"house-1", "house-2"}, "2023-10")
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
	if len(costs) != 2 {
		t.Errorf("expected 2 costs")
	}
}

func TestHouseCostRepository_Update(t *testing.T) {
	bunDB, mock := setupTestDB(t)
	defer bunDB.Close()

	repo := repository.NewHouseCostRepository(bunDB)
	ctx := context.Background()

	cost := &model.HouseCost{
		ID:      "cost-1",
		HouseID: "house-1",
		Period:  "2023-10",
		Rent:    1500,
	}

	// Success
	mock.ExpectExec(`UPDATE "house_costs"`).WillReturnResult(sqlmock.NewResult(1, 1))
	err := repo.Update(ctx, cost)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	// Not found
	mock.ExpectExec(`UPDATE "house_costs"`).WillReturnResult(sqlmock.NewResult(0, 0))
	err = repo.Update(ctx, cost)
	if err == nil || err.Error() != "house cost not found" {
		t.Errorf("expected house cost not found, got %v", err)
	}
}
