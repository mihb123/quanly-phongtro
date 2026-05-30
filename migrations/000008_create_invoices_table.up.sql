DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'invoice_status') THEN
        CREATE TYPE invoice_status AS ENUM ('UNPAID', 'PARTIALLY_PAID', 'PAID');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS invoices (
    id                      UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id                 UUID            NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    period                  VARCHAR(7)      NOT NULL,  -- yyyy-mm format
    room_fee                DECIMAL(12,2)   NOT NULL DEFAULT 0,
    old_electricity_index   INT             NOT NULL DEFAULT 0,
    new_electricity_index   INT             NOT NULL DEFAULT 0,
    electricity_fee         DECIMAL(12,2)   NOT NULL DEFAULT 0,
    old_water_index         INT             NOT NULL DEFAULT 0,
    new_water_index         INT             NOT NULL DEFAULT 0,
    water_fee               DECIMAL(12,2)   NOT NULL DEFAULT 0,
    wifi_fee                DECIMAL(12,2)   NOT NULL DEFAULT 0,
    parking_fee             DECIMAL(12,2)   NOT NULL DEFAULT 0,
    service_fee             DECIMAL(12,2)   NOT NULL DEFAULT 0,
    other_fee               DECIMAL(12,2)   NOT NULL DEFAULT 0,
    tenant_count            INT             NOT NULL DEFAULT 0,
    vehicle_count           INT             NOT NULL DEFAULT 0,
    extra_person_fee        DECIMAL(12,2)   NOT NULL DEFAULT 0,
    extra_vehicle_fee       DECIMAL(12,2)   NOT NULL DEFAULT 0,
    discount                DECIMAL(12,2)   NOT NULL DEFAULT 0,
    total_amount            DECIMAL(12,2)   NOT NULL DEFAULT 0,
    status                  invoice_status  NOT NULL DEFAULT 'UNPAID',
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_invoices_room_period UNIQUE (room_id, period)
);

CREATE INDEX IF NOT EXISTS idx_invoices_room_id ON invoices(room_id);
CREATE INDEX IF NOT EXISTS idx_invoices_period  ON invoices(room_id, period);
CREATE INDEX IF NOT EXISTS idx_invoices_status  ON invoices(status);
