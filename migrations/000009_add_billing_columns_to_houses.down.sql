ALTER TABLE houses
    DROP COLUMN IF EXISTS electricity_billing_type,
    DROP COLUMN IF EXISTS water_billing_type,
    DROP COLUMN IF EXISTS electricity_billing_unit,
    DROP COLUMN IF EXISTS water_billing_unit,
    DROP COLUMN IF EXISTS extra_person_threshold,
    DROP COLUMN IF EXISTS extra_person_fee,
    DROP COLUMN IF EXISTS extra_vehicle_threshold,
    DROP COLUMN IF EXISTS extra_vehicle_fee;
