package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/uptrace/bun"
)

type InvoiceRepository struct {
	db *bun.DB
}

func NewInvoiceRepository(db *bun.DB) *InvoiceRepository {
	return &InvoiceRepository{db: db}
}

// CreateInvoice inserts a new invoice. Ownership must be verified by the caller before this.
func (r *InvoiceRepository) CreateInvoice(ctx context.Context, invoice *model.Invoice) error {
	_, err := r.db.NewInsert().
		Model(invoice).
		ExcludeColumn("created_at").
		Returning("id, created_at").
		Exec(ctx)

	if err != nil {
		if strings.Contains(err.Error(), "uq_invoices_room_period") || strings.Contains(err.Error(), "duplicate key value") {
			return model.ErrDuplicateInvoice
		}
		return fmt.Errorf("create invoice: %w", err)
	}
	return nil
}

// GetInvoiceByID fetches an invoice by its ID and ensures it belongs to the manager.
func (r *InvoiceRepository) GetInvoiceByID(ctx context.Context, managerID, id string) (*model.InvoiceWithRoom, error) {
	var invoice model.InvoiceWithRoom
	err := r.db.NewSelect().
		Model(&invoice).
		ModelTableExpr("invoices AS invoice").
		ColumnExpr("invoice.*").
		ColumnExpr("r.name AS room_name").
		ColumnExpr("r.house_id AS house_id").
		ColumnExpr("COALESCE(r.extra_person_threshold, h.extra_person_threshold) AS extra_person_threshold").
		ColumnExpr("COALESCE(r.extra_person_fee, h.extra_person_fee) AS extra_person_fee_unit").
		ColumnExpr("COALESCE(r.extra_vehicle_threshold, h.extra_vehicle_threshold) AS extra_vehicle_threshold").
		ColumnExpr("COALESCE(r.extra_vehicle_fee, h.extra_vehicle_fee) AS extra_vehicle_fee_unit").
		Join("JOIN rooms AS r ON invoice.room_id = r.id").
		Join("JOIN houses AS h ON r.house_id = h.id").
		Where("invoice.id = ?", id).
		Where("h.manager_id = ?", managerID).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("get invoice by id: %w", err)
	}
	return &invoice, nil
}

// ListInvoices lists invoices with filters and ensures they belong to the manager.
func (r *InvoiceRepository) ListInvoices(ctx context.Context, managerID string, filter model.InvoiceListFilter) ([]model.InvoiceWithRoom, error) {
	var invoices []model.InvoiceWithRoom

	q := r.db.NewSelect().
		Model(&invoices).
		ModelTableExpr("invoices AS invoice").
		ColumnExpr("invoice.*").
		ColumnExpr("r.name AS room_name").
		ColumnExpr("r.house_id AS house_id").
		ColumnExpr("COALESCE(r.extra_person_threshold, h.extra_person_threshold) AS extra_person_threshold").
		ColumnExpr("COALESCE(r.extra_person_fee, h.extra_person_fee) AS extra_person_fee_unit").
		ColumnExpr("COALESCE(r.extra_vehicle_threshold, h.extra_vehicle_threshold) AS extra_vehicle_threshold").
		ColumnExpr("COALESCE(r.extra_vehicle_fee, h.extra_vehicle_fee) AS extra_vehicle_fee_unit").
		Join("JOIN rooms AS r ON invoice.room_id = r.id").
		Join("JOIN houses AS h ON r.house_id = h.id").
		Where("h.manager_id = ?", managerID)

	if filter.RoomID != "" {
		q.Where("invoice.room_id = ?", filter.RoomID)
	}
	if filter.HouseID != "" {
		q.Where("r.house_id = ?", filter.HouseID)
	}
	if filter.Period != "" {
		q.Where("invoice.period = ?", filter.Period)
	}
	if filter.Status != "" {
		q.Where("invoice.status = ?", filter.Status)
	}

	err := q.Order("invoice.created_at DESC").
		Limit(filter.Limit).
		Offset((filter.Page - 1) * filter.Limit).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}

	return invoices, nil
}

// UpdateInvoiceStatusAndMethod updates the status and payment method of an invoice.
func (r *InvoiceRepository) UpdateInvoiceStatusAndMethod(ctx context.Context, managerID, id, status, method string) (*model.Invoice, error) {
	var invoice model.Invoice
	err := r.db.NewUpdate().
		Model(&invoice).
		Set("status = ?", status).
		Set("payment_method = ?", method).
		Where("id = ?", id).
		Where("room_id IN (SELECT r.id FROM rooms r JOIN houses h ON r.house_id = h.id WHERE h.manager_id = ?)", managerID).
		Returning("*").
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("update invoice status and method: %w", err)
	}

	return &invoice, nil
}

// SystemUpdateInvoiceStatusAndMethod updates the status and payment method of an invoice WITHOUT checking managerID.
// This is for system webhooks.
func (r *InvoiceRepository) SystemUpdateInvoiceStatusAndMethod(ctx context.Context, id, status, method string) (*model.InvoiceWithRoom, error) {
	var invoice model.Invoice
	err := r.db.NewUpdate().
		Model(&invoice).
		Set("status = ?", status).
		Set("payment_method = ?", method).
		Where("id = ?", id).
		Returning("*").
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("system update invoice status and method: %w", err)
	}

	// We also want to return InvoiceWithRoom so we can find the managerID to send Zalo notification
	var invoiceWithRoom model.InvoiceWithRoom
	err = r.db.NewSelect().
		Model(&invoiceWithRoom).
		ColumnExpr("invoice.*").
		ColumnExpr("r.name AS room_name, h.id AS house_id, h.manager_id AS manager_id").
		Join("JOIN rooms AS r ON r.id = invoice.room_id").
		Join("JOIN houses AS h ON h.id = r.house_id").
		Where("invoice.id = ?", id).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("get invoice with room after update: %w", err)
	}

	return &invoiceWithRoom, nil
}

// UpdateInvoiceStatus updates the status of an invoice.
func (r *InvoiceRepository) UpdateInvoiceStatus(ctx context.Context, managerID, id, status string) (*model.Invoice, error) {
	var invoice model.Invoice
	err := r.db.NewUpdate().
		Model(&invoice).
		Set("status = ?", status).
		Where("id = ?", id).
		Where("room_id IN (SELECT r.id FROM rooms r JOIN houses h ON r.house_id = h.id WHERE h.manager_id = ?)", managerID).
		Returning("*").
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("update invoice status: %w", err)
	}

	return &invoice, nil
}

// GetLatestInvoiceByRoomID fetches the latest invoice for a room based on period.
func (r *InvoiceRepository) GetLatestInvoiceByRoomID(ctx context.Context, roomID string) (*model.Invoice, error) {
	var invoice model.Invoice
	err := r.db.NewSelect().
		Model(&invoice).
		Where("room_id = ?", roomID).
		Order("period DESC").
		Limit(1).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("get latest invoice: %w", err)
	}
	return &invoice, nil
}

// GetInvoiceByRoomAndPeriod fetches an invoice by room ID and period.
func (r *InvoiceRepository) GetInvoiceByRoomAndPeriod(ctx context.Context, roomID, period string) (*model.Invoice, error) {
	var invoice model.Invoice
	err := r.db.NewSelect().
		Model(&invoice).
		Where("room_id = ?", roomID).
		Where("period = ?", period).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("get invoice by room and period: %w", err)
	}
	return &invoice, nil
}

// GetPreviousInvoice fetches the latest invoice for a room before a given period.
func (r *InvoiceRepository) GetPreviousInvoice(ctx context.Context, roomID, period string) (*model.Invoice, error) {
	var invoice model.Invoice
	err := r.db.NewSelect().
		Model(&invoice).
		Where("room_id = ?", roomID).
		Where("period < ?", period).
		Order("period DESC").
		Limit(1).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("get previous invoice: %w", err)
	}
	return &invoice, nil
}

// GetUnpaidInvoicesByRoomID fetches all unpaid invoices for a specific room.
func (r *InvoiceRepository) GetUnpaidInvoicesByRoomID(ctx context.Context, roomID string) ([]model.Invoice, error) {
	var invoices []model.Invoice
	err := r.db.NewSelect().
		Model(&invoices).
		Where("room_id = ?", roomID).
		Where("status = ?", "UNPAID").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("get unpaid invoices by room: %w", err)
	}
	return invoices, nil
}

// GetLatestUnpaidInvoiceByRoomID fetches the most recent unpaid invoice for a specific room.
func (r *InvoiceRepository) GetLatestUnpaidInvoiceByRoomID(ctx context.Context, roomID string) (*model.Invoice, error) {
	var invoice model.Invoice
	err := r.db.NewSelect().
		Model(&invoice).
		Where("room_id = ?", roomID).
		Where("status = ?", model.InvoiceStatusUnpaid).
		Order("period DESC").
		Limit(1).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("get latest unpaid invoice by room: %w", err)
	}
	return &invoice, nil
}

// UpdateInvoice completely updates an existing invoice.
func (r *InvoiceRepository) UpdateInvoice(ctx context.Context, managerID string, invoice *model.Invoice) error {
	res, err := r.db.NewUpdate().
		Model(invoice).
		Where("id = ?", invoice.ID).
		Where("room_id IN (SELECT r.id FROM rooms r JOIN houses h ON r.house_id = h.id WHERE h.manager_id = ?)", managerID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("update invoice: %w", err)
	}
	rows, err := res.RowsAffected()
	if err == nil && rows == 0 {
		return model.ErrInvoiceNotFound
	}
	return nil
}

// DeleteInvoice completely removes an invoice from the database.
func (r *InvoiceRepository) DeleteInvoice(ctx context.Context, managerID, id string) error {
	res, err := r.db.NewDelete().
		Model((*model.Invoice)(nil)).
		Where("id = ?", id).
		Where("room_id IN (SELECT r.id FROM rooms r JOIN houses h ON r.house_id = h.id WHERE h.manager_id = ?)", managerID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("delete invoice: %w", err)
	}

	rows, err := res.RowsAffected()
	if err == nil && rows == 0 {
		return model.ErrInvoiceNotFound
	}
	return nil
}

// GetInvoiceByTransactionImagePath fetches an invoice image record owned by a manager.
func (r *InvoiceRepository) GetInvoiceByTransactionImagePath(ctx context.Context, managerID, imagePath string) (*model.InvoiceWithRoom, error) {
	var invoice model.InvoiceWithRoom
	err := r.db.NewSelect().
		Model(&invoice).
		ModelTableExpr("invoices AS invoice").
		ColumnExpr("invoice.*").
		ColumnExpr("r.name AS room_name").
		ColumnExpr("r.house_id AS house_id").
		ColumnExpr("h.manager_id AS manager_id").
		Join("JOIN rooms AS r ON invoice.room_id = r.id").
		Join("JOIN houses AS h ON r.house_id = h.id").
		Where("invoice.transaction_image_path = ?", imagePath).
		Where("h.manager_id = ?", managerID).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("get invoice by transaction image path: %w", err)
	}
	return &invoice, nil
}
