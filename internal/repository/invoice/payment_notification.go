package invoice

import (
	"context"
	"encoding/json"

	database "github.com/mihb123/quanly-phongtro/internal/db"
)

func (r *InvoicePaymentRepository) QueueNotification(ctx context.Context, invoiceID, managerID, chatID, message string) error {
	payload, err := json.Marshal(struct {
		ManagerID string `json:"manager_id"`
		ChatID    string `json:"chat_id"`
		Message   string `json:"message"`
	}{managerID, chatID, message})
	if err != nil {
		return err
	}
	_, err = database.Executor(ctx, r.db).ExecContext(ctx,
		"INSERT INTO background_jobs (kind, job_key, payload) VALUES ('payment_notification', ?, ?::jsonb) ON CONFLICT (kind, job_key) DO NOTHING",
		invoiceID+":"+chatID, string(payload))
	return err
}
