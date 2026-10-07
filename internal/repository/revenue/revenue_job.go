package revenue

import (
	"context"
	"encoding/json"
	"fmt"

	database "github.com/mihb123/quanly-phongtro/internal/db"
)

func (r *RevenueSummaryRepository) RecalculateJob(ctx context.Context, payload json.RawMessage) error {
	var key struct {
		HouseID string `json:"house_id"`
		Period  string `json:"period"`
	}
	if err := json.Unmarshal(payload, &key); err != nil {
		return err
	}
	if key.HouseID == "" || key.Period == "" {
		return fmt.Errorf("invalid revenue job")
	}
	_, err := database.Executor(ctx, r.db).ExecContext(ctx, `
 INSERT INTO house_revenue_summaries (house_id, period, total_revenue, total_cost, profit)
 SELECT h.id, ?, revenue.total, cost.total, revenue.total - cost.total
 FROM houses h
 CROSS JOIN LATERAL (
  SELECT COALESCE(SUM(i.total_amount), 0) AS total FROM invoices i
  JOIN rooms r ON r.id = i.room_id WHERE r.house_id = h.id AND i.period = ? AND i.status = 'PAID'
 ) revenue
 CROSS JOIN LATERAL (
  SELECT COALESCE(SUM(c.total_cost), 0) AS total FROM house_costs c WHERE c.house_id = h.id AND c.period = ?
 ) cost
 WHERE h.id = ?
 ON CONFLICT (house_id, period) DO UPDATE SET total_revenue = EXCLUDED.total_revenue,
 total_cost = EXCLUDED.total_cost, profit = EXCLUDED.profit, updated_at = NOW()`, key.Period, key.Period, key.Period, key.HouseID)
	return err
}
