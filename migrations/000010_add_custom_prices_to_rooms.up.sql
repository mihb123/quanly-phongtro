ALTER TABLE rooms
    ADD COLUMN electricity_price NUMERIC(15, 2),
    ADD COLUMN water_price NUMERIC(15, 2),
    ADD COLUMN wifi_price NUMERIC(15, 2),
    ADD COLUMN parking_price NUMERIC(15, 2),
    ADD COLUMN service_price NUMERIC(15, 2);
