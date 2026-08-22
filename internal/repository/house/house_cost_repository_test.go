package house_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/repository/house"
	"github.com/mihb123/quanly-phongtro/internal/repository/repotest"
)

func TestHouseCostRepository_Create(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := house.NewHouseCostRepository(bunDB)
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

	mock.ExpectQuery(`INSERT INTO "house_costs"`).WillReturnError(errors.New("duplicate key value violates unique constraint"))
	err = repo.Create(ctx, cost)
	if err == nil || err.Error() != "house cost record already exists for this period" {
		t.Errorf("expected duplicate error, got %v", err)
	}
}

func TestHouseCostRepository_GetByHouseAndPeriod(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := house.NewHouseCostRepository(bunDB)
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

	mock.ExpectQuery(`SELECT .* FROM "house_costs"`).WillReturnError(errors.New("db error"))
	_, err = repo.GetByHouseAndPeriod(ctx, "house-1", "2023-12")
	if err == nil {
		t.Errorf("expected db error")
	}
}

func TestHouseCostRepository_GetLatestByHouseID(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := house.NewHouseCostRepository(bunDB)
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

	mock.ExpectQuery(`SELECT .* FROM "house_costs"`).WillReturnError(errors.New("db error"))
	_, err = repo.GetLatestByHouseID(ctx, "house-3")
	if err == nil {
		t.Errorf("expected db error")
	}
}

func TestHouseCostRepository_ListByHouseIDs(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := house.NewHouseCostRepository(bunDB)
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

	mock.ExpectQuery(`SELECT .* FROM "house_costs"`).WillReturnError(errors.New("db error"))
	_, err = repo.ListByHouseIDs(ctx, []string{"house-1"}, "2023-10")
	if err == nil {
		t.Errorf("expected db error")
	}
}

func TestHouseCostRepository_Update(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := house.NewHouseCostRepository(bunDB)
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

	mock.ExpectExec(`UPDATE "house_costs"`).WillReturnError(errors.New("db error"))
	err = repo.Update(ctx, cost)
	if err == nil {
		t.Errorf("expected db error")
	}
}

// TestHouseCostRepository_Delete covers successful deletion and not-found handling.
func TestHouseCostRepository_Delete(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := house.NewHouseCostRepository(bunDB)
	ctx := context.Background()

	mock.ExpectExec(`DELETE FROM "house_costs"`).WillReturnResult(sqlmock.NewResult(0, 1))
	err := repo.Delete(ctx, "cost-1", "house-1")
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	mock.ExpectExec(`DELETE FROM "house_costs"`).WillReturnResult(sqlmock.NewResult(0, 0))
	err = repo.Delete(ctx, "cost-1", "house-1")
	if err == nil || err.Error() != "house cost not found" {
		t.Errorf("expected house cost not found, got %v", err)
	}

	mock.ExpectExec(`DELETE FROM "house_costs"`).WillReturnError(errors.New("db error"))
	err = repo.Delete(ctx, "cost-1", "house-1")
	if err == nil {
		t.Errorf("expected db error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestHouseCostRepository_ListByPeriod(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := house.NewHouseCostRepository(bunDB)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"id", "house_id", "period", "rent"}).
		AddRow("cost-1", "house-1", "2023-10", 2000.0).
		AddRow("cost-2", "house-2", "2023-10", 1500.0)

	mock.ExpectQuery(`SELECT .* FROM "house_costs"`).WillReturnRows(rows)
	costs, err := repo.ListByPeriod(ctx, "2023-10")
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
	if len(costs) != 2 {
		t.Errorf("expected 2 costs, got %d", len(costs))
	}

	mock.ExpectQuery(`SELECT .* FROM "house_costs"`).WillReturnError(errors.New("db error"))
	if _, err := repo.ListByPeriod(ctx, "2023-10"); err == nil {
		t.Errorf("expected db error")
	}
}
