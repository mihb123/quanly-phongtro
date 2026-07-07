package invoice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/uptrace/bun"
)

type InvoicePaymentRepository struct {
	db *bun.DB
}

// NewInvoicePaymentRepository creates the repository for payment links, credentials, and events.
func NewInvoicePaymentRepository(db *bun.DB) *InvoicePaymentRepository {
	return &InvoicePaymentRepository{db: db}
}

// CreatePaymentLink inserts a new payment link.
func (r *InvoicePaymentRepository) CreatePaymentLink(ctx context.Context, link *model.InvoicePaymentLink) error {
	_, err := r.db.NewInsert().
		Model(link).
		ExcludeColumn("created_at", "updated_at").
		Returning("id, created_at, updated_at").
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("create payment link: %w", err)
	}
	return nil
}

// GetActivePaymentLink returns the currently active PayOS payment link for an invoice.
func (r *InvoicePaymentRepository) GetActivePaymentLink(ctx context.Context, invoiceID string) (*model.InvoicePaymentLink, error) {
	return r.GetActivePaymentLinkByProvider(ctx, invoiceID, model.PaymentProviderPayOS)
}

// GetActivePaymentLinkByProvider returns the latest active payment link for an invoice/provider pair.
func (r *InvoicePaymentRepository) GetActivePaymentLinkByProvider(ctx context.Context, invoiceID, provider string) (*model.InvoicePaymentLink, error) {
	var link model.InvoicePaymentLink
	err := r.db.NewSelect().
		Model(&link).
		Where("invoice_id = ?", invoiceID).
		Where("provider = ?", provider).
		Where("status = ?", model.PaymentLinkStatusActive).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get active payment link: %w", err)
	}
	return &link, nil
}

// UpdatePaymentLinkStatus updates the status of a payment link (e.g. to CANCELLED or PAID).
func (r *InvoicePaymentRepository) UpdatePaymentLinkStatus(ctx context.Context, id, status string) error {
	_, err := r.db.NewUpdate().
		Model((*model.InvoicePaymentLink)(nil)).
		Set("status = ?", status).
		Where("id = ?", id).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("update payment link status: %w", err)
	}
	return nil
}

// GetPaymentLinkByOrderCode retrieves a PayOS payment link by its unique order code.
func (r *InvoicePaymentRepository) GetPaymentLinkByOrderCode(ctx context.Context, orderCode int64) (*model.InvoicePaymentLink, error) {
	var link model.InvoicePaymentLink
	err := r.db.NewSelect().
		Model(&link).
		Where("provider = ?", model.PaymentProviderPayOS).
		Where("order_code = ?", orderCode).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get payment link by order code: %w", err)
	}
	return &link, nil
}

// GetPaymentLinkByProviderOrderRef retrieves a payment link by provider and provider order reference.
func (r *InvoicePaymentRepository) GetPaymentLinkByProviderOrderRef(ctx context.Context, provider, providerOrderRef string) (*model.InvoicePaymentLink, error) {
	var link model.InvoicePaymentLink
	err := r.db.NewSelect().
		Model(&link).
		Where("provider = ?", provider).
		Where("provider_order_ref = ?", providerOrderRef).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get payment link by provider order ref: %w", err)
	}
	return &link, nil
}

// GetPaymentLinkByProviderOrderRefForManager retrieves a payment link only when its invoice belongs to the manager.
func (r *InvoicePaymentRepository) GetPaymentLinkByProviderOrderRefForManager(ctx context.Context, managerID, provider, providerOrderRef string) (*model.InvoicePaymentLink, error) {
	var link model.InvoicePaymentLink
	err := r.db.NewSelect().
		Model(&link).
		Join("JOIN invoices AS i ON i.id = invoice_payment_link.invoice_id").
		Join("JOIN rooms AS r ON r.id = i.room_id").
		Join("JOIN houses AS h ON h.id = r.house_id").
		Where("invoice_payment_link.provider = ?", provider).
		Where("invoice_payment_link.provider_order_ref = ?", providerOrderRef).
		Where("h.manager_id = ?", managerID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get manager payment link by provider order ref: %w", err)
	}
	return &link, nil
}

// MarkActivePaymentLinksStaleByManager marks active provider links stale for invoices owned by the manager.
func (r *InvoicePaymentRepository) MarkActivePaymentLinksStaleByManager(ctx context.Context, managerID, provider string) error {
	_, err := r.db.NewUpdate().
		Model((*model.InvoicePaymentLink)(nil)).
		Set("status = ?", model.PaymentLinkStatusStale).
		Set("updated_at = CURRENT_TIMESTAMP").
		Where("provider = ?", provider).
		Where("status = ?", model.PaymentLinkStatusActive).
		Where("invoice_id IN (?)",
			r.db.NewSelect().
				TableExpr("invoices AS i").
				ColumnExpr("i.id").
				Join("JOIN rooms AS r ON r.id = i.room_id").
				Join("JOIN houses AS h ON h.id = r.house_id").
				Where("h.manager_id = ?", managerID),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("mark active payment links stale by manager: %w", err)
	}
	return nil
}

// CreatePaymentEvent records a new webhook event.
func (r *InvoicePaymentRepository) CreatePaymentEvent(ctx context.Context, event *model.PaymentEvent) error {
	_, err := r.db.NewInsert().
		Model(event).
		ExcludeColumn("created_at").
		Returning("id, created_at").
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("create payment event: %w", err)
	}
	return nil
}

// CheckEventExistsByOrderCode checks PayOS event idempotency by order code and transaction reference.
func (r *InvoicePaymentRepository) CheckEventExists(ctx context.Context, orderCode int64, transactionRef string) (bool, error) {
	return r.CheckProviderEventExists(ctx, model.PaymentProviderPayOS, fmt.Sprint(orderCode), transactionRef)
}

// CheckProviderEventExists checks if a provider event has already been processed.
func (r *InvoicePaymentRepository) CheckProviderEventExists(ctx context.Context, provider, providerOrderRef, transactionRef string) (bool, error) {
	exists, err := r.db.NewSelect().
		Model((*model.PaymentEvent)(nil)).
		Where("provider = ?", provider).
		Where("provider_order_ref = ?", providerOrderRef).
		Where("transaction_reference = ?", transactionRef).
		Exists(ctx)

	if err != nil {
		return false, fmt.Errorf("check event exists: %w", err)
	}
	return exists, nil
}

// GetActivePaymentProviderCredential returns an active manager credential for a provider.
func (r *InvoicePaymentRepository) GetActivePaymentProviderCredential(ctx context.Context, managerID, provider string) (*model.PaymentProviderCredential, error) {
	var credential model.PaymentProviderCredential
	err := r.db.NewSelect().
		Model(&credential).
		Where("manager_id = ?", managerID).
		Where("provider = ?", provider).
		Where("is_active = ?", true).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get payment provider credential: %w", err)
	}
	return &credential, nil
}

// UpsertPaymentProviderCredential stores encrypted provider credentials for a manager.
func (r *InvoicePaymentRepository) UpsertPaymentProviderCredential(ctx context.Context, credential *model.PaymentProviderCredential) error {
	_, err := r.db.NewInsert().
		Model(credential).
		ExcludeColumn("id", "created_at", "updated_at").
		On("CONFLICT (manager_id, provider) DO UPDATE").
		Set("encrypted_credentials = EXCLUDED.encrypted_credentials").
		Set("is_active = EXCLUDED.is_active").
		Set("updated_at = CURRENT_TIMESTAMP").
		Returning("id, created_at, updated_at").
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("upsert payment provider credential: %w", err)
	}
	return nil
}

// DeletePaymentProviderCredential deactivates a manager's provider credentials.
func (r *InvoicePaymentRepository) DeletePaymentProviderCredential(ctx context.Context, managerID, provider string) error {
	_, err := r.db.NewUpdate().
		Model((*model.PaymentProviderCredential)(nil)).
		Set("is_active = ?", false).
		Set("updated_at = CURRENT_TIMESTAMP").
		Where("manager_id = ?", managerID).
		Where("provider = ?", provider).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("delete payment provider credential: %w", err)
	}
	return nil
}
