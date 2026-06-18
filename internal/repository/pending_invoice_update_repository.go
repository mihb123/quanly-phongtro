package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/uptrace/bun"
)

type PendingInvoiceUpdateRepository struct {
	db *bun.DB
}

// NewPendingInvoiceUpdateRepository creates a PostgreSQL-backed pending update repository.
func NewPendingInvoiceUpdateRepository(db *bun.DB) *PendingInvoiceUpdateRepository {
	return &PendingInvoiceUpdateRepository{db: db}
}

// Create stores a pending invoice command state for a Zalo chat.
func (r *PendingInvoiceUpdateRepository) Create(ctx context.Context, pending *model.PendingInvoiceUpdate) error {
	_, err := r.db.NewInsert().
		Model(pending).
		Column("manager_id", "chat_id", "is_group_chat", "room_id", "action_type", "pending_data").
		Returning("id, created_at").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("create pending invoice update: %w", err)
	}
	return nil
}

// GetByChatID returns the latest pending state for a manager and Zalo chat.
func (r *PendingInvoiceUpdateRepository) GetByChatID(ctx context.Context, managerID, chatID string) (*model.PendingInvoiceUpdate, error) {
	var pending model.PendingInvoiceUpdate
	err := r.db.NewSelect().
		Model(&pending).
		Where("manager_id = ? AND chat_id = ?", managerID, chatID).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrPendingInvoiceUpdateNotFound
		}
		return nil, fmt.Errorf("get pending invoice update: %w", err)
	}
	return &pending, nil
}

// DeleteByChatID removes all pending command states for a manager and chat.
func (r *PendingInvoiceUpdateRepository) DeleteByChatID(ctx context.Context, managerID, chatID string) error {
	_, err := r.db.NewDelete().
		Model((*model.PendingInvoiceUpdate)(nil)).
		Where("manager_id = ? AND chat_id = ?", managerID, chatID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete pending invoice update by chat: %w", err)
	}
	return nil
}

// DeleteByID removes one pending command state by ID.
func (r *PendingInvoiceUpdateRepository) DeleteByID(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().
		Model((*model.PendingInvoiceUpdate)(nil)).
		Where("id = ?", id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete pending invoice update by id: %w", err)
	}
	return nil
}
