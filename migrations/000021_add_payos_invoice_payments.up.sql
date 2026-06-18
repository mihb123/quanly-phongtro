ALTER TABLE invoices ADD COLUMN payment_method VARCHAR(50);

CREATE TABLE payment_provider_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    manager_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    encrypted_credentials TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(manager_id, provider)
);

CREATE TABLE invoice_payment_links (
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

CREATE INDEX idx_invoice_payment_links_invoice_id ON invoice_payment_links(invoice_id);
CREATE UNIQUE INDEX idx_invoice_payment_links_provider_order_ref ON invoice_payment_links(provider, provider_order_ref);
CREATE INDEX idx_invoice_payment_links_order_code ON invoice_payment_links(order_code) WHERE order_code IS NOT NULL;

CREATE TABLE payment_events (
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

CREATE INDEX idx_payment_events_provider_order_ref ON payment_events(provider, provider_order_ref);
CREATE INDEX idx_payment_events_order_code ON payment_events(order_code) WHERE order_code IS NOT NULL;
