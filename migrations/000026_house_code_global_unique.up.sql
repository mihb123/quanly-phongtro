-- Chuyển quy tắc duy nhất của house_code từ phạm vi từng manager sang toàn hệ thống (case-insensitive).

-- 1) Khử trùng dữ liệu hiện có: giữ nguyên mã của nhà tạo sớm nhất trong mỗi nhóm trùng,
--    các nhà còn lại được thêm hậu tố "-<n>" (cắt phần gốc để không vượt 12 ký tự).
WITH ranked AS (
    SELECT id,
           house_code,
           row_number() OVER (PARTITION BY LOWER(house_code) ORDER BY created_at, id) AS rn
    FROM houses
)
UPDATE houses h
SET house_code = LEFT(h.house_code, 12 - LENGTH('-' || r.rn)) || '-' || r.rn
FROM ranked r
WHERE h.id = r.id
  AND r.rn > 1;

-- 2) Thay unique index theo manager bằng unique index toàn hệ thống.
DROP INDEX IF EXISTS uq_houses_manager_code;

CREATE UNIQUE INDEX IF NOT EXISTS uq_houses_code
    ON houses (LOWER(house_code));
