DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'room_status') THEN
        CREATE TYPE room_status AS ENUM ('AVAILABLE', 'OCCUPIED', 'MAINTENANCE');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS rooms (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    house_id    UUID         NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    price       BIGINT       NOT NULL DEFAULT 0,
    max_tenants INT          NOT NULL DEFAULT 1,
    status      room_status  NOT NULL DEFAULT 'AVAILABLE',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rooms_house_id ON rooms(house_id);
CREATE INDEX IF NOT EXISTS idx_rooms_status   ON rooms(house_id, status);
