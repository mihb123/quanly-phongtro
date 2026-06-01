ALTER TABLE rooms
    DROP COLUMN IF EXISTS extra_person_threshold,
    DROP COLUMN IF EXISTS extra_person_fee,
    DROP COLUMN IF EXISTS extra_vehicle_threshold,
    DROP COLUMN IF EXISTS extra_vehicle_fee;
