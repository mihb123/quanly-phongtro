ALTER TABLE houses ADD COLUMN IF NOT EXISTS house_code VARCHAR(12);

UPDATE houses
SET house_code = 'h' || REPLACE(LEFT(id::text, 10), '-', '')
WHERE house_code IS NULL OR house_code = '';

ALTER TABLE houses ALTER COLUMN house_code SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_houses_house_code_format'
    ) THEN
        ALTER TABLE houses
            ADD CONSTRAINT chk_houses_house_code_format
            CHECK (house_code ~ '^[a-zA-Z0-9_-]{1,12}$');
    END IF;
END$$;

CREATE UNIQUE INDEX IF NOT EXISTS uq_houses_manager_code
    ON houses (manager_id, LOWER(house_code));

CREATE TABLE IF NOT EXISTS pending_invoice_updates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    manager_id UUID NOT NULL REFERENCES users(id),
    chat_id VARCHAR(255) NOT NULL,
    is_group_chat BOOLEAN NOT NULL DEFAULT false,
    room_id UUID REFERENCES rooms(id),
    action_type VARCHAR(50) NOT NULL,
    pending_data JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pending_updates_chat
    ON pending_invoice_updates (manager_id, chat_id);
