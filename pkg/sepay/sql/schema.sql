-- PostgreSQL. Đổi tên bảng/khóa ngoại cho khớp dự án đích:
--   owner_id  -> chủ tài khoản nhận tiền (manager, shop, tenant...)
--   order_id  -> đơn cần thanh toán (invoice, order...)
-- uuidv7() cần PostgreSQL 18; bản cũ hơn dùng gen_random_uuid().

CREATE TABLE IF NOT EXISTS payment_provider_credentials (
    id                    uuid PRIMARY KEY DEFAULT uuidv7(),
    owner_id              uuid NOT NULL,
    provider              varchar(50) NOT NULL,
    encrypted_credentials text NOT NULL,
    is_active             boolean NOT NULL DEFAULT true,
    created_at            timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (owner_id, provider)
);

CREATE TABLE IF NOT EXISTS payment_links (
    id                 uuid PRIMARY KEY DEFAULT uuidv7(),
    order_id           uuid NOT NULL,
    provider           varchar(50) NOT NULL,
    provider_order_ref varchar(255) NOT NULL,
    payment_link_id    varchar(255) NOT NULL,
    checkout_url       varchar(1024) NOT NULL,
    qr_code            text NOT NULL,
    amount             integer NOT NULL,
    status             varchar(50) NOT NULL DEFAULT 'ACTIVE',
    created_at         timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_payment_links_order_id ON payment_links (order_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_links_provider_order_ref ON payment_links (provider, provider_order_ref);

CREATE TABLE IF NOT EXISTS payment_events (
    id                    uuid PRIMARY KEY DEFAULT uuidv7(),
    provider              varchar(50) NOT NULL,
    owner_id              uuid,
    order_id              uuid,
    provider_order_ref    varchar(255) NOT NULL,
    amount                integer NOT NULL,
    transaction_reference varchar(255) NOT NULL,
    counter_account       varchar(255),
    raw_payload           text NOT NULL,
    signature_result      varchar(50) NOT NULL,
    matching_method       varchar(50),
    status                varchar(50) NOT NULL DEFAULT 'PENDING',
    created_at            timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (provider, provider_order_ref, transaction_reference)
);
CREATE INDEX IF NOT EXISTS idx_payment_events_provider_order_ref ON payment_events (provider, provider_order_ref);
