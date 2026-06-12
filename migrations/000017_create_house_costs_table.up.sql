CREATE TABLE IF NOT EXISTS house_costs (
    id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    house_id        UUID            NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
    period          VARCHAR(7)      NOT NULL,  -- yyyy-mm

    -- 6 loại chi phí chính (cột cố định)
    rent            DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- Tiền thuê nguyên căn
    electricity     DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- Tiền điện (biến đổi)
    water           DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- Tiền nước (biến đổi)
    wifi            DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- Tiền wifi
    cleaning        DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- Tiền vệ sinh/rác
    maintenance     DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- Tiền bảo trì (biến đổi)

    -- Chi phí tùy chỉnh (user tự thêm)
    extra_costs     JSONB           NOT NULL DEFAULT '[]',
    -- Format: [{"name": "Tiền thấm", "amount": 500000}, ...]

    note            TEXT            DEFAULT '',
    total_cost      DECIMAL(12,2)   NOT NULL DEFAULT 0,  -- Tổng cộng (auto-calculated)
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_house_costs_house_period UNIQUE (house_id, period)
);

CREATE INDEX IF NOT EXISTS idx_house_costs_house_id ON house_costs(house_id);
CREATE INDEX IF NOT EXISTS idx_house_costs_period ON house_costs(house_id, period);
