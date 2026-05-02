CREATE TABLE IF NOT EXISTS otp_checks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL REFERENCES users(email) ON DELETE CASCADE,
    otp_fails INTEGER NOT NULL DEFAULT 0,
    block_time TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_otp_checks_email ON otp_checks(email);
