package revenue_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/repository/repotest"
	"github.com/mihb123/quanly-phongtro/internal/repository/revenue"
)

func TestRevenueSummaryRepository_Upsert(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := revenue.NewRevenueSummaryRepository(bunDB)
	ctx := context.Background()

	summary := &model.HouseRevenueSummary{
		HouseID:      "house-1",
		Period:       "2023-10",
		TotalRevenue: 5000,
		TotalCost:    2000,
		Profit:       3000,
	}

	mock.ExpectQuery(`INSERT INTO "house_revenue_summaries"`).WillReturnRows(sqlmock.NewRows([]string{"id", "updated_at"}).AddRow("some-id", nil))

	err := repo.Upsert(ctx, summary)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}

	mock.ExpectQuery(`INSERT INTO "house_revenue_summaries"`).WillReturnError(errors.New("db error"))
	err = repo.Upsert(ctx, summary)
	if err == nil {
		t.Errorf("expected db error")
	}
}

// TestRevenueSummaryRepository_GetByHouseAndPeriod covers fetch and missing-summary behavior.
func TestRevenueSummaryRepository_GetByHouseAndPeriod(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := revenue.NewRevenueSummaryRepository(bunDB)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"house_id", "period", "total_revenue", "total_cost", "profit"}).
		AddRow("house-1", "2023-10", 5000.0, 2000.0, 3000.0)

	mock.ExpectQuery(`SELECT .* FROM "house_revenue_summaries"`).WillReturnRows(rows)
	summary, err := repo.GetByHouseAndPeriod(ctx, "house-1", "2023-10")
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
	if summary.Profit != 3000 {
		t.Errorf("expected profit 3000, got %v", summary.Profit)
	}

	mock.ExpectQuery(`SELECT .* FROM "house_revenue_summaries"`).WillReturnError(sql.ErrNoRows)
	_, err = repo.GetByHouseAndPeriod(ctx, "house-1", "2023-11")
	if err == nil || err.Error() != "revenue summary not found" {
		t.Errorf("expected revenue summary not found, got %v", err)
	}

	mock.ExpectQuery(`SELECT .* FROM "house_revenue_summaries"`).WillReturnError(errors.New("db error"))
	_, err = repo.GetByHouseAndPeriod(ctx, "house-1", "2023-12")
	if err == nil {
		t.Errorf("expected db error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}

func TestRevenueSummaryRepository_ListByHouseIDs(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := revenue.NewRevenueSummaryRepository(bunDB)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{"house_id", "period", "total_revenue", "total_cost", "profit"}).
		AddRow("house-1", "2023-10", 5000.0, 2000.0, 3000.0).
		AddRow("house-2", "2023-10", 6000.0, 1000.0, 5000.0)

	// Empty ids
	summaries, err := repo.ListByHouseIDs(ctx, []string{}, "2023-10")
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
	if len(summaries) != 0 {
		t.Errorf("expected 0 summaries")
	}

	// With ids
	mock.ExpectQuery(`SELECT .* FROM "house_revenue_summaries"`).WillReturnRows(rows)
	summaries, err = repo.ListByHouseIDs(ctx, []string{"house-1", "house-2"}, "2023-10")
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
	if len(summaries) != 2 {
		t.Errorf("expected 2 summaries")
	}

	mock.ExpectQuery(`SELECT .* FROM "house_revenue_summaries"`).WillReturnError(errors.New("db error"))
	_, err = repo.ListByHouseIDs(ctx, []string{"house-1"}, "2023-10")
	if err == nil {
		t.Errorf("expected db error")
	}
}

func TestRevenueSummaryRepository_CalculateRevenue(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := revenue.NewRevenueSummaryRepository(bunDB)
	ctx := context.Background()

	// Revenue found
	rows := sqlmock.NewRows([]string{"sum"}).AddRow(15000.5)
	mock.ExpectQuery(`SELECT SUM\(i.total_amount\) FROM invoices`).WillReturnRows(rows)

	revenue, err := repo.CalculateRevenue(ctx, "house-1", "2023-10")
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
	if revenue != 15000.5 {
		t.Errorf("expected 15000.5, got %v", revenue)
	}

	rows = sqlmock.NewRows([]string{"sum"}).AddRow(nil)
	mock.ExpectQuery(`SELECT SUM\(i.total_amount\) FROM invoices`).WillReturnRows(rows)

	revenue, err = repo.CalculateRevenue(ctx, "house-1", "2023-11")
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
	if revenue != 0 {
		t.Errorf("expected 0, got %v", revenue)
	}

	mock.ExpectQuery(`SELECT SUM\(i.total_amount\) FROM invoices`).WillReturnError(errors.New("db error"))
	_, err = repo.CalculateRevenue(ctx, "house-1", "2023-12")
	if err == nil {
		t.Errorf("expected db error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %s", err)
	}
}
