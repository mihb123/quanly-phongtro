CREATE TABLE IF NOT EXISTS house_revenue_summaries (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    house_id        UUID            NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
    period          VARCHAR(7)      NOT NULL,

    total_revenue   DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- SUM(invoices.total_amount) WHERE PAID
    total_cost      DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- FROM house_costs.total_cost
    profit          DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- revenue - cost
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_revenue_summary_house_period UNIQUE (house_id, period)
);

CREATE INDEX IF NOT EXISTS idx_revenue_summary_house_period ON house_revenue_summaries(house_id, period);
