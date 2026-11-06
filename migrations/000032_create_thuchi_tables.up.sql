CREATE TABLE IF NOT EXISTS thuchi_sync_state (
    owner_id UUID PRIMARY KEY,
    current_seq BIGINT NOT NULL DEFAULT 0,
    purged_seq BIGINT NOT NULL DEFAULT 0,
    seeded_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS thuchi_accounts (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    kind TEXT NOT NULL DEFAULT 'bank',
    bank_code TEXT NOT NULL DEFAULT '',
    account_no_last4 TEXT NOT NULL DEFAULT '',
    account_no_hash TEXT NOT NULL DEFAULT '',
    holder_name TEXT NOT NULL DEFAULT '',
    connector TEXT NOT NULL DEFAULT 'manual',
    balance BIGINT NOT NULL DEFAULT 0,
    balance_at TIMESTAMPTZ,
    include_in_stats BOOLEAN NOT NULL DEFAULT TRUE,
    sort INTEGER NOT NULL DEFAULT 0,
    field_ts JSONB NOT NULL DEFAULT '{}'::jsonb,
    version INTEGER NOT NULL DEFAULT 1,
    change_seq BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_thuchi_accounts_owner_seq ON thuchi_accounts(owner_id, change_seq);
CREATE INDEX IF NOT EXISTS idx_thuchi_accounts_owner_hash ON thuchi_accounts(owner_id, account_no_hash) WHERE account_no_hash <> '';

CREATE TABLE IF NOT EXISTS thuchi_categories (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL,
    key TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL DEFAULT '',
    icon TEXT NOT NULL DEFAULT '',
    color TEXT NOT NULL DEFAULT '',
    kind TEXT NOT NULL DEFAULT 'expense',
    parent_id UUID,
    sort INTEGER NOT NULL DEFAULT 0,
    field_ts JSONB NOT NULL DEFAULT '{}'::jsonb,
    version INTEGER NOT NULL DEFAULT 1,
    change_seq BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_thuchi_categories_owner_seq ON thuchi_categories(owner_id, change_seq);

CREATE TABLE IF NOT EXISTS thuchi_transactions (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL,
    account_id UUID,
    direction TEXT NOT NULL,
    amount BIGINT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    raw_content TEXT NOT NULL DEFAULT '',
    merchant TEXT NOT NULL DEFAULT '',
    counterparty_name TEXT NOT NULL DEFAULT '',
    counterparty_account TEXT NOT NULL DEFAULT '',
    balance_after BIGINT,
    category_id UUID,
    note TEXT NOT NULL DEFAULT '',
    lat DOUBLE PRECISION,
    lng DOUBLE PRECISION,
    place_name TEXT NOT NULL DEFAULT '',
    kind TEXT NOT NULL DEFAULT 'normal',
    pair_id UUID,
    pair_status TEXT NOT NULL DEFAULT '',
    refund_of_category_id UUID,
    excluded BOOLEAN NOT NULL DEFAULT FALSE,
    fingerprint TEXT NOT NULL DEFAULT '',
    sources JSONB NOT NULL DEFAULT '[]'::jsonb,
    reconciled BOOLEAN NOT NULL DEFAULT FALSE,
    needs_review BOOLEAN NOT NULL DEFAULT FALSE,
    party TEXT NOT NULL DEFAULT '',
    recurrence_id UUID,
    field_ts JSONB NOT NULL DEFAULT '{}'::jsonb,
    version INTEGER NOT NULL DEFAULT 1,
    change_seq BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_thuchi_tx_owner_seq ON thuchi_transactions(owner_id, change_seq);
CREATE INDEX IF NOT EXISTS idx_thuchi_tx_owner_time ON thuchi_transactions(owner_id, occurred_at) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS thuchi_tx_sources (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    owner_id UUID NOT NULL,
    transaction_id UUID NOT NULL REFERENCES thuchi_transactions(id) ON DELETE CASCADE,
    source TEXT NOT NULL,
    source_ref TEXT NOT NULL,
    raw_event_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (owner_id, source, source_ref)
);
CREATE INDEX IF NOT EXISTS idx_thuchi_tx_sources_tx ON thuchi_tx_sources(transaction_id);

CREATE TABLE IF NOT EXISTS thuchi_rules (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL,
    pattern TEXT NOT NULL DEFAULT '',
    match TEXT NOT NULL DEFAULT 'contains',
    category_id UUID,
    merchant_name TEXT NOT NULL DEFAULT '',
    origin TEXT NOT NULL DEFAULT 'manual',
    hits INTEGER NOT NULL DEFAULT 0,
    field_ts JSONB NOT NULL DEFAULT '{}'::jsonb,
    version INTEGER NOT NULL DEFAULT 1,
    change_seq BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_thuchi_rules_owner_seq ON thuchi_rules(owner_id, change_seq);

CREATE TABLE IF NOT EXISTS thuchi_budgets (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL,
    category_id UUID,
    month TEXT NOT NULL DEFAULT '',
    limit_amount BIGINT NOT NULL DEFAULT 0,
    field_ts JSONB NOT NULL DEFAULT '{}'::jsonb,
    version INTEGER NOT NULL DEFAULT 1,
    change_seq BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_thuchi_budgets_owner_seq ON thuchi_budgets(owner_id, change_seq);

CREATE TABLE IF NOT EXISTS thuchi_recurrences (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL,
    account_id UUID,
    direction TEXT NOT NULL DEFAULT 'out',
    amount BIGINT NOT NULL,
    category_id UUID,
    merchant TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    freq TEXT NOT NULL DEFAULT 'monthly',
    freq_interval INTEGER NOT NULL DEFAULT 1,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ,
    next_at TIMESTAMPTZ NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    field_ts JSONB NOT NULL DEFAULT '{}'::jsonb,
    version INTEGER NOT NULL DEFAULT 1,
    change_seq BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_thuchi_recurrences_owner_seq ON thuchi_recurrences(owner_id, change_seq);
CREATE INDEX IF NOT EXISTS idx_thuchi_recurrences_due ON thuchi_recurrences(owner_id, next_at) WHERE active AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS thuchi_raw_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    owner_id UUID NOT NULL,
    source TEXT NOT NULL,
    source_key TEXT NOT NULL DEFAULT '',
    payload TEXT NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT NOT NULL DEFAULT 'pending',
    error TEXT NOT NULL DEFAULT '',
    transaction_id UUID,
    attempts INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    change_seq BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_thuchi_raw_owner_seq ON thuchi_raw_events(owner_id, change_seq);
CREATE INDEX IF NOT EXISTS idx_thuchi_raw_pending ON thuchi_raw_events(received_at) WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS thuchi_ingest_tokens (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    owner_id UUID NOT NULL,
    kind TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    secret_enc TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_thuchi_ingest_tokens_owner ON thuchi_ingest_tokens(owner_id, kind) WHERE revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS thuchi_telegram_links (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    owner_id UUID NOT NULL,
    code TEXT UNIQUE,
    code_expires_at TIMESTAMPTZ,
    telegram_user_id BIGINT UNIQUE,
    telegram_username TEXT NOT NULL DEFAULT '',
    telegram_name TEXT NOT NULL DEFAULT '',
    default_account_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    linked_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_thuchi_telegram_links_owner ON thuchi_telegram_links(owner_id);

CREATE TABLE IF NOT EXISTS thuchi_telegram_messages (
    chat_id BIGINT NOT NULL,
    message_id BIGINT NOT NULL,
    owner_id UUID NOT NULL,
    reply_message_id BIGINT,
    transaction_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chat_id, message_id)
);
CREATE INDEX IF NOT EXISTS idx_thuchi_telegram_messages_reply ON thuchi_telegram_messages(chat_id, reply_message_id);
