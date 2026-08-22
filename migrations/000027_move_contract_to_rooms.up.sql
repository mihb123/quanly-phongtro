ALTER TABLE rooms ADD COLUMN IF NOT EXISTS contract_path TEXT NOT NULL DEFAULT '';

-- Hợp đồng trước đây gắn theo người thuê; gộp về đúng phòng của họ.
-- Nhiều người cùng phòng có thể giữ cùng một hợp đồng nên phải loại trùng.
UPDATE rooms r
SET contract_path = agg.paths
FROM (
    SELECT room_id, string_agg(DISTINCT path, ',' ORDER BY path) AS paths
    FROM (
        SELECT t.room_id, btrim(p) AS path
        FROM tenants t,
             unnest(string_to_array(COALESCE(t.contract_path, ''), ',')) AS p
        WHERE btrim(p) <> ''
    ) AS flat
    GROUP BY room_id
) agg
WHERE r.id = agg.room_id AND r.contract_path = '';

ALTER TABLE tenants DROP COLUMN IF EXISTS contract_path;
