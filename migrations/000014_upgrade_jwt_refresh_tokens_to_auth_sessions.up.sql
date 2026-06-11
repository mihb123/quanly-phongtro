ALTER TABLE jwt_refresh_tokens RENAME TO auth_sessions;

ALTER TABLE auth_sessions
    ADD COLUMN ip_address VARCHAR(45) NOT NULL DEFAULT '',
    ADD COLUMN user_agent TEXT NOT NULL DEFAULT '',
    ADD COLUMN jkt VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN expires_at TIMESTAMP WITH TIME ZONE;

-- Update expires_at to 30 days after created_at for existing rows
UPDATE auth_sessions SET expires_at = created_at + INTERVAL '30 days';

-- Make expires_at NOT NULL after data population
ALTER TABLE auth_sessions ALTER COLUMN expires_at SET NOT NULL;
