package expense

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	database "github.com/mihb123/quanly-phongtro/internal/db"
	"github.com/mihb123/quanly-phongtro/internal/model"

	"github.com/lib/pq"
	"github.com/uptrace/bun"
)

const managerHouses = "house_id IN (SELECT id FROM houses WHERE manager_id = ?)"

type ExpenseRepository struct {
	db *bun.DB
}

func NewExpenseRepository(db *bun.DB) *ExpenseRepository {
	return &ExpenseRepository{db: db}
}

func (r *ExpenseRepository) FindByID(ctx context.Context, managerID, id string) (*model.Expense, error) {
	var expense model.Expense
	err := database.Executor(ctx, r.db).NewSelect().Model(&expense).
		Where("id = ?", id).
		Where(managerHouses, managerID).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrExpenseNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select expense %s: %w", id, err)
	}
	return &expense, nil
}

func (r *ExpenseRepository) List(ctx context.Context, managerID string, filter model.ExpenseFilter) ([]model.Expense, int, error) {
	items := make([]model.Expense, 0, filter.Limit)
	query := database.Executor(ctx, r.db).NewSelect().Model(&items).Where(managerHouses, managerID)
	if filter.HouseID != "" {
		query = query.Where("house_id = ?", filter.HouseID)
	}
	if !filter.From.IsZero() {
		query = query.Where("spent_at >= ?", filter.From)
	}
	if !filter.To.IsZero() {
		query = query.Where("spent_at < ?", filter.To)
	}
	total, err := query.Order("spent_at DESC", "id DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		ScanAndCount(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list expenses: %w", err)
	}
	return items, total, nil
}

func (r *ExpenseRepository) Insert(ctx context.Context, expense *model.Expense) error {
	_, err := database.Executor(ctx, r.db).NewInsert().Model(expense).
		ExcludeColumn("id", "created_at").
		Returning("id, created_at").
		Exec(ctx)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return model.ErrExpenseDuplicate
	}
	if err != nil {
		return fmt.Errorf("insert expense: %w", err)
	}
	return nil
}

func (r *ExpenseRepository) Delete(ctx context.Context, managerID, id string) error {
	res, err := database.Executor(ctx, r.db).NewDelete().Model((*model.Expense)(nil)).
		Where("id = ?", id).
		Where(managerHouses, managerID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete expense %s: %w", id, err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return model.ErrExpenseNotFound
	}
	return nil
}
