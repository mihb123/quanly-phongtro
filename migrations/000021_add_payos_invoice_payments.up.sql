ALTER TABLE invoices ADD COLUMN IF NOT EXISTS payment_method VARCHAR(50);

CREATE TABLE IF NOT EXISTS payment_provider_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    manager_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    encrypted_credentials TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(manager_id, provider)
);

DO $$
DECLARE
    manager_id_att SMALLINT;
    provider_att SMALLINT;
BEGIN
    SELECT attnum INTO manager_id_att
    FROM pg_attribute
    WHERE attrelid = 'payment_provider_credentials'::regclass
      AND attname = 'manager_id';

    SELECT attnum INTO provider_att
    FROM pg_attribute
    WHERE attrelid = 'payment_provider_credentials'::regclass
      AND attname = 'provider';

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'payment_provider_credentials'::regclass
          AND contype = 'u'
          AND conkey @> ARRAY[manager_id_att, provider_att]::SMALLINT[]
          AND conkey <@ ARRAY[manager_id_att, provider_att]::SMALLINT[]
    ) THEN
        ALTER TABLE payment_provider_credentials
        ADD CONSTRAINT payment_provider_credentials_manager_provider_key UNIQUE (manager_id, provider);
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS invoice_payment_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL DEFAULT 'payos',
    provider_order_ref VARCHAR(255),
    order_code BIGINT,
    payment_link_id VARCHAR(255) NOT NULL,
    checkout_url VARCHAR(1024) NOT NULL,
    qr_code TEXT NOT NULL,
    amount INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE invoice_payment_links ADD COLUMN IF NOT EXISTS provider VARCHAR(50) NOT NULL DEFAULT 'payos';
ALTER TABLE invoice_payment_links ADD COLUMN IF NOT EXISTS provider_order_ref VARCHAR(255);
ALTER TABLE invoice_payment_links ALTER COLUMN order_code DROP NOT NULL;
ALTER TABLE invoice_payment_links ALTER COLUMN provider SET DEFAULT 'payos';
UPDATE invoice_payment_links
SET provider_order_ref = order_code::TEXT
WHERE provider_order_ref IS NULL AND order_code IS NOT NULL;
ALTER TABLE invoice_payment_links DROP CONSTRAINT IF EXISTS invoice_payment_links_order_code_key;
DROP INDEX IF EXISTS invoice_payment_links_order_code_key;
CREATE INDEX IF NOT EXISTS idx_invoice_payment_links_invoice_id ON invoice_payment_links(invoice_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_invoice_payment_links_provider_order_ref ON invoice_payment_links(provider, provider_order_ref);
CREATE INDEX IF NOT EXISTS idx_invoice_payment_links_order_code ON invoice_payment_links(order_code) WHERE order_code IS NOT NULL;

CREATE TABLE IF NOT EXISTS payment_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) NOT NULL DEFAULT 'payos',
    manager_id UUID REFERENCES users(id) ON DELETE SET NULL,
    invoice_id UUID REFERENCES invoices(id) ON DELETE SET NULL,
    provider_order_ref VARCHAR(255) NOT NULL,
    order_code BIGINT,
    amount INTEGER NOT NULL,
    transaction_reference VARCHAR(255) NOT NULL,
    payer_account VARCHAR(255),
    counter_account VARCHAR(255),
    raw_payload TEXT NOT NULL,
    signature_result VARCHAR(50) NOT NULL,
    matching_method VARCHAR(50),
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, provider_order_ref, transaction_reference)
);

ALTER TABLE payment_events ADD COLUMN IF NOT EXISTS provider VARCHAR(50) NOT NULL DEFAULT 'payos';
ALTER TABLE payment_events ADD COLUMN IF NOT EXISTS provider_order_ref VARCHAR(255);
ALTER TABLE payment_events ADD COLUMN IF NOT EXISTS order_code BIGINT;
UPDATE payment_events
SET provider_order_ref = COALESCE(order_code::TEXT, '')
WHERE provider_order_ref IS NULL;
ALTER TABLE payment_events ALTER COLUMN provider_order_ref SET NOT NULL;
DO $$
DECLARE
    provider_att SMALLINT;
    provider_order_ref_att SMALLINT;
    transaction_reference_att SMALLINT;
BEGIN
    SELECT attnum INTO provider_att
    FROM pg_attribute
    WHERE attrelid = 'payment_events'::regclass
      AND attname = 'provider';

    SELECT attnum INTO provider_order_ref_att
    FROM pg_attribute
    WHERE attrelid = 'payment_events'::regclass
      AND attname = 'provider_order_ref';

    SELECT attnum INTO transaction_reference_att
    FROM pg_attribute
    WHERE attrelid = 'payment_events'::regclass
      AND attname = 'transaction_reference';

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'payment_events'::regclass
          AND contype = 'u'
          AND conkey @> ARRAY[provider_att, provider_order_ref_att, transaction_reference_att]::SMALLINT[]
          AND conkey <@ ARRAY[provider_att, provider_order_ref_att, transaction_reference_att]::SMALLINT[]
    ) THEN
        ALTER TABLE payment_events
        ADD CONSTRAINT payment_events_provider_order_ref_transaction_reference_key UNIQUE (provider, provider_order_ref, transaction_reference);
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_payment_events_provider_order_ref ON payment_events(provider, provider_order_ref);
CREATE INDEX IF NOT EXISTS idx_payment_events_order_code ON payment_events(order_code) WHERE order_code IS NOT NULL;
