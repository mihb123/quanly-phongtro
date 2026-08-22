ALTER TABLE houses
    DROP COLUMN IF EXISTS owner_name,
    DROP COLUMN IF EXISTS owner_phone,
    DROP COLUMN IF EXISTS owner_rent_price,
    DROP COLUMN IF EXISTS owner_deposit,
    DROP COLUMN IF EXISTS rent_start_date,
    DROP COLUMN IF EXISTS rent_end_date,
    DROP COLUMN IF EXISTS owner_cccd_path,
    DROP COLUMN IF EXISTS owner_contract_path;
