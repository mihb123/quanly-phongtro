ALTER TABLE auth_sessions
    DROP COLUMN expires_at,
    DROP COLUMN jkt,
    DROP COLUMN user_agent,
    DROP COLUMN ip_address;

ALTER TABLE auth_sessions RENAME TO jwt_refresh_tokens;
