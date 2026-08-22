ALTER TABLE tenants ADD COLUMN IF NOT EXISTS contract_path TEXT NOT NULL DEFAULT '';

-- Trả hợp đồng của phòng về mọi người thuê đang ở phòng đó.
UPDATE tenants t
SET contract_path = r.contract_path
FROM rooms r
WHERE r.id = t.room_id AND r.contract_path <> '';

ALTER TABLE rooms DROP COLUMN IF EXISTS contract_path;
