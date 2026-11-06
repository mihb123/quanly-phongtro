package model

import (
	"context"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/apperr"
	"github.com/uptrace/bun"
)

var (
	ErrExpenseNotFound  = apperr.NotFound("Không tìm thấy khoản chi")
	ErrExpenseLocked    = apperr.Invalid("Khoản chi đã đối soát, không xoá được")
	ErrExpenseDuplicate = apperr.Conflict("Khoản chi với mã chứng từ này đã tồn tại")
)

type Expense struct {
	bun.BaseModel `bun:"table:expenses"`

	ID         string    `bun:"id,pk"`
	HouseID    string    `bun:"house_id"`
	ReceiptNo  string    `bun:"receipt_no"`
	Amount     int64     `bun:"amount"`
	Note       string    `bun:"note"`
	Reconciled bool      `bun:"reconciled"`
	SpentAt    time.Time `bun:"spent_at"`
	CreatedAt  time.Time `bun:"created_at"`
}

type ExpenseFilter struct {
	HouseID string
	From    time.Time
	To      time.Time
	Limit   int
	Offset  int
}

type ExpenseRepository interface {
	FindByID(ctx context.Context, managerID, id string) (*Expense, error)
	List(ctx context.Context, managerID string, filter ExpenseFilter) ([]Expense, int, error)
	Insert(ctx context.Context, expense *Expense) error
	Delete(ctx context.Context, managerID, id string) error
}
