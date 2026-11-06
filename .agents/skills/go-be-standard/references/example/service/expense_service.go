package expense

import (
	"context"
	"fmt"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/apperr"
	"github.com/mihb123/quanly-phongtro/internal/model"
)

var ErrHouseNotOwned = apperr.NotFound("Không tìm thấy nhà")

type CreateInput struct {
	HouseID   string
	ReceiptNo string
	Amount    int64
	Note      string
	SpentAt   time.Time
}

type ListResult struct {
	Items []model.Expense
	Total int
}

type ExpenseService interface {
	Create(ctx context.Context, managerID string, in CreateInput) (*model.Expense, error)
	List(ctx context.Context, managerID string, filter model.ExpenseFilter) (ListResult, error)
	Delete(ctx context.Context, managerID, id string) error
}

type expenseServiceImpl struct {
	expenses model.ExpenseRepository
	houses   model.HouseRepository
}

func NewExpenseService(expenses model.ExpenseRepository, houses model.HouseRepository) ExpenseService {
	return &expenseServiceImpl{expenses: expenses, houses: houses}
}

func (s *expenseServiceImpl) Create(ctx context.Context, managerID string, in CreateInput) (*model.Expense, error) {
	owned, err := s.houses.IsHouseOwnedBy(ctx, in.HouseID, managerID)
	if err != nil {
		return nil, fmt.Errorf("verify house ownership: %w", err)
	}
	if !owned {
		return nil, ErrHouseNotOwned
	}
	expense := &model.Expense{
		HouseID:   in.HouseID,
		ReceiptNo: in.ReceiptNo,
		Amount:    in.Amount,
		Note:      in.Note,
		SpentAt:   in.SpentAt,
	}
	if err := s.expenses.Insert(ctx, expense); err != nil {
		return nil, err
	}
	return expense, nil
}

func (s *expenseServiceImpl) List(ctx context.Context, managerID string, filter model.ExpenseFilter) (ListResult, error) {
	items, total, err := s.expenses.List(ctx, managerID, filter)
	if err != nil {
		return ListResult{}, err
	}
	return ListResult{Items: items, Total: total}, nil
}

func (s *expenseServiceImpl) Delete(ctx context.Context, managerID, id string) error {
	expense, err := s.expenses.FindByID(ctx, managerID, id)
	if err != nil {
		return err
	}
	if expense.Reconciled {
		return model.ErrExpenseLocked
	}
	return s.expenses.Delete(ctx, managerID, id)
}
