CREATE TABLE IF NOT EXISTS houses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    manager_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL,
    default_electricity_price DECIMAL(12,2) NOT NULL DEFAULT 0,
    default_water_price DECIMAL(12,2) NOT NULL DEFAULT 0,
    default_wifi_price DECIMAL(12,2) NOT NULL DEFAULT 0,
    default_parking_price DECIMAL(12,2) NOT NULL DEFAULT 0,
    default_service_price DECIMAL(12,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_houses_manager_id ON houses(manager_id);
