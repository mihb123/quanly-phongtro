ALTER TABLE rooms
    ADD COLUMN extra_person_threshold INT DEFAULT NULL,
    ADD COLUMN extra_person_fee DECIMAL(12,2) DEFAULT NULL,
    ADD COLUMN extra_vehicle_threshold INT DEFAULT NULL,
    ADD COLUMN extra_vehicle_fee DECIMAL(12,2) DEFAULT NULL;
