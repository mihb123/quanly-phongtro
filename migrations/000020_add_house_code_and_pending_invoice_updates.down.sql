DROP TABLE IF EXISTS pending_invoice_updates;

DROP INDEX IF EXISTS idx_pending_updates_chat;
DROP INDEX IF EXISTS uq_houses_manager_code;

ALTER TABLE houses DROP CONSTRAINT IF EXISTS chk_houses_house_code_format;
ALTER TABLE houses DROP COLUMN IF EXISTS house_code;
