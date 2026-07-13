-- Khôi phục unique index theo từng manager (không hoàn tác được phần đổi tên mã đã khử trùng).
DROP INDEX IF EXISTS uq_houses_code;

CREATE UNIQUE INDEX IF NOT EXISTS uq_houses_manager_code
    ON houses (manager_id, LOWER(house_code));
