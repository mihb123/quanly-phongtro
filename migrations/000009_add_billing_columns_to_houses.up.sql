ALTER TABLE houses
    ADD COLUMN electricity_billing_type VARCHAR(10) NOT NULL DEFAULT 'USAGE',
    ADD COLUMN water_billing_type VARCHAR(10) NOT NULL DEFAULT 'USAGE',
    ADD COLUMN electricity_billing_unit VARCHAR(10) NOT NULL DEFAULT 'ROOM',
    ADD COLUMN water_billing_unit VARCHAR(10) NOT NULL DEFAULT 'ROOM',
    ADD COLUMN extra_person_threshold INT NOT NULL DEFAULT 0,
    ADD COLUMN extra_person_fee DECIMAL(12,2) NOT NULL DEFAULT 0,
    ADD COLUMN extra_vehicle_threshold INT NOT NULL DEFAULT 0,
    ADD COLUMN extra_vehicle_fee DECIMAL(12,2) NOT NULL DEFAULT 0;
