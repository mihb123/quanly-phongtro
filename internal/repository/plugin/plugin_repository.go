package plugin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	database "github.com/mihb123/quanly-phongtro/internal/db"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/uptrace/bun"
)

type PluginRepository struct {
	db *bun.DB
}

func NewPluginRepository(db *bun.DB) *PluginRepository {
	return &PluginRepository{db: db}
}

func (r *PluginRepository) ListPluginSettings(ctx context.Context, managerID string) ([]model.PluginSetting, error) {
	settings := []model.PluginSetting{}
	err := database.Executor(ctx, r.db).NewSelect().
		Model(&settings).
		Where("manager_id = ?", managerID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list plugin settings: %w", err)
	}
	return settings, nil
}

func (r *PluginRepository) GetPluginSetting(ctx context.Context, managerID, pluginID string) (*model.PluginSetting, error) {
	var setting model.PluginSetting
	err := database.Executor(ctx, r.db).NewSelect().
		Model(&setting).
		Where("manager_id = ?", managerID).
		Where("plugin_id = ?", pluginID).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get plugin setting: %w", err)
	}
	return &setting, nil
}

func (r *PluginRepository) SetPluginEnabled(ctx context.Context, managerID, pluginID string, enabled bool) error {
	setting := &model.PluginSetting{ManagerID: managerID, PluginID: pluginID, Enabled: &enabled}
	_, err := database.Executor(ctx, r.db).NewInsert().
		Model(setting).
		Column("manager_id", "plugin_id", "enabled").
		On("CONFLICT (manager_id, plugin_id) DO UPDATE").
		Set("enabled = EXCLUDED.enabled").
		Set("updated_at = CURRENT_TIMESTAMP").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("set plugin enabled: %w", err)
	}
	return nil
}

func (r *PluginRepository) SavePluginData(ctx context.Context, managerID, pluginID string, encryptedData *string) error {
	setting := &model.PluginSetting{ManagerID: managerID, PluginID: pluginID, EncryptedData: encryptedData}
	_, err := database.Executor(ctx, r.db).NewInsert().
		Model(setting).
		Column("manager_id", "plugin_id", "encrypted_data").
		On("CONFLICT (manager_id, plugin_id) DO UPDATE").
		Set("encrypted_data = EXCLUDED.encrypted_data").
		Set("updated_at = CURRENT_TIMESTAMP").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("save plugin data: %w", err)
	}
	return nil
}

func (r *PluginRepository) ListOutstandingInvoices(ctx context.Context, managerID string, periods []string) ([]model.OutstandingInvoice, error) {
	invoices, err := r.listInvoices(ctx, managerID, []string{model.InvoiceStatusUnpaid, model.InvoiceStatusPendingVerification}, periods)
	if err != nil {
		return nil, fmt.Errorf("list outstanding invoices: %w", err)
	}
	return invoices, nil
}

func (r *PluginRepository) ListPaidInvoices(ctx context.Context, managerID string, periods []string) ([]model.OutstandingInvoice, error) {
	invoices, err := r.listInvoices(ctx, managerID, []string{model.InvoiceStatusPaid}, periods)
	if err != nil {
		return nil, fmt.Errorf("list paid invoices: %w", err)
	}
	return invoices, nil
}

func (r *PluginRepository) listInvoices(ctx context.Context, managerID string, statuses, periods []string) ([]model.OutstandingInvoice, error) {
	invoices := []model.OutstandingInvoice{}
	q := database.Executor(ctx, r.db).NewSelect().
		TableExpr("invoices AS invoice").
		ColumnExpr("invoice.id, invoice.room_id, invoice.period, invoice.total_amount, invoice.status").
		ColumnExpr("r.name AS room_name, h.id AS house_id, h.name AS house_name").
		ColumnExpr("COALESCE((SELECT array_agg(u.full_name ORDER BY u.full_name) FROM tenants AS t JOIN users AS u ON u.id = t.user_id WHERE t.room_id = r.id AND t.status = ?), '{}') AS tenant_names", string(model.TenantStatusActive)).
		Join("JOIN rooms AS r ON invoice.room_id = r.id").
		Join("JOIN houses AS h ON r.house_id = h.id").
		Where("h.manager_id = ?", managerID).
		Where("invoice.status IN (?)", bun.In(statuses))
	if len(periods) > 0 {
		q = q.Where("invoice.period IN (?)", bun.In(periods))
	}
	err := q.Order("invoice.period ASC", "r.name ASC").Scan(ctx, &invoices)
	return invoices, err
}

func (r *PluginRepository) RecordedPayment(ctx context.Context, source, accountRef, transactionRef string) (string, bool, error) {
	var invoiceIDs []*string
	err := database.Executor(ctx, r.db).NewSelect().
		Model((*model.PaymentEvent)(nil)).
		Column("invoice_id").
		Where("provider = ?", source).
		Where("provider_order_ref = ?", accountRef).
		Where("transaction_reference = ?", transactionRef).
		Limit(1).
		Scan(ctx, &invoiceIDs)
	if err != nil {
		return "", false, fmt.Errorf("find plugin payment: %w", err)
	}
	if len(invoiceIDs) == 0 {
		return "", false, nil
	}
	if invoiceIDs[0] == nil {
		return "", true, nil
	}
	return *invoiceIDs[0], true, nil
}

func (r *PluginRepository) PaymentRecorded(ctx context.Context, source, accountRef, transactionRef string) (bool, error) {
	exists, err := database.Executor(ctx, r.db).NewSelect().
		Model((*model.PaymentEvent)(nil)).
		Where("provider = ?", source).
		Where("provider_order_ref = ?", accountRef).
		Where("transaction_reference = ?", transactionRef).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check plugin payment recorded: %w", err)
	}
	return exists, nil
}

func (r *PluginRepository) SettleInvoice(ctx context.Context, managerID string, payment model.PluginPayment) error {
	return database.WithinTransaction(ctx, r.db, func(ctx context.Context) error {
		db := database.Executor(ctx, r.db)
		if _, err := db.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", payment.Source+":"+payment.AccountRef+":"+payment.TransactionReference); err != nil {
			return fmt.Errorf("lock plugin payment: %w", err)
		}
		recorded, err := r.PaymentRecorded(ctx, payment.Source, payment.AccountRef, payment.TransactionReference)
		if err != nil {
			return err
		}
		if recorded {
			return model.ErrPluginPaymentRecorded
		}

		res, err := db.NewUpdate().
			Model((*model.Invoice)(nil)).
			Set("status = ?", model.InvoiceStatusPaid).
			Set("payment_method = ?", payment.PaymentMethod).
			Where("id = ?", payment.InvoiceID).
			Where("status IN (?)", bun.In([]string{model.InvoiceStatusUnpaid, model.InvoiceStatusPendingVerification})).
			Where("room_id IN (SELECT r.id FROM rooms r JOIN houses h ON r.house_id = h.id WHERE h.manager_id = ?)", managerID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("settle invoice: %w", err)
		}
		if rows, err := res.RowsAffected(); err != nil || rows == 0 {
			return model.ErrPluginInvoiceNotPayable
		}

		matchingMethod := payment.MatchingMethod
		event := &model.PaymentEvent{
			Provider:             payment.Source,
			ManagerID:            &managerID,
			InvoiceID:            &payment.InvoiceID,
			ProviderOrderRef:     payment.AccountRef,
			Amount:               payment.Amount,
			TransactionReference: payment.TransactionReference,
			RawPayload:           payment.RawPayload,
			SignatureResult:      "STATEMENT",
			MatchingMethod:       &matchingMethod,
			Status:               model.PaymentEventStatusProcessed,
		}
		if _, err := db.NewInsert().Model(event).ExcludeColumn("id", "created_at").Returning("id, created_at").Exec(ctx); err != nil {
			return fmt.Errorf("record plugin payment: %w", err)
		}
		return nil
	})
}
