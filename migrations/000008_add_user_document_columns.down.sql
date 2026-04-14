ALTER TABLE users
    DROP COLUMN IF EXISTS cccd_path,
    DROP COLUMN IF EXISTS identity_card,
    DROP COLUMN IF EXISTS contract_path;
