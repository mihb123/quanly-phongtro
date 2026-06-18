package model

import (
	"context"
	"errors"
	"time"
)

var ErrPendingInvoiceUpdateNotFound = errors.New("pending invoice update not found")

const (
	PendingActionConfirmOverwrite = "CONFIRM_OVERWRITE"
	PendingActionAwaitPeriod      = "AWAIT_PERIOD"
	PendingActionAwaitUtility     = "AWAIT_UTILITY"
)

type PendingInvoiceUpdate struct {
	ID          string         `json:"id"`
	ManagerID   string         `json:"manager_id"`
	ChatID      string         `json:"chat_id"`
	IsGroupChat bool           `json:"is_group_chat"`
	RoomID      *string        `json:"room_id,omitempty"`
	ActionType  string         `json:"action_type"`
	PendingData map[string]any `json:"pending_data" bun:"type:jsonb"`
	CreatedAt   time.Time      `json:"created_at"`
}

type PendingInvoiceUpdateRepository interface {
	Create(ctx context.Context, pending *PendingInvoiceUpdate) error
	GetByChatID(ctx context.Context, managerID, chatID string) (*PendingInvoiceUpdate, error)
	DeleteByChatID(ctx context.Context, managerID, chatID string) error
	DeleteByID(ctx context.Context, id string) error
}
