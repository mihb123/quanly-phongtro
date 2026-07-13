-- ============================================================================
-- migrate_uuid_v4_to_v7.sql
-- Đổi toàn bộ UUID v4 hiện có trong DB sang UUID v7.
--
-- Cách hoạt động:
--   - Sinh UUID v7 mới cho từng bản ghi, lấy timestamp từ created_at của
--     chính bản ghi đó → ID mới sắp xếp đúng theo thời gian tạo thật.
--   - Chỉ đổi các bản ghi đang là v4 (ký tự version = '4'); chạy lại lần 2
--     sẽ không làm gì (idempotent).
--   - 4 bảng cha bị FK tham chiếu (users, houses, rooms, invoices): dùng
--     bảng mapping old_id → new_id, tạm chuyển FK sang DEFERRED trong
--     transaction rồi cập nhật cả cha lẫn con.
--
-- KHÔNG đổi:
--   - houses.house_code: được sinh từ id cũ nhưng đã lưu cứng, đang dùng để
--     link nhóm Zalo → giữ nguyên, không ảnh hưởng.
--   - invoices.transaction_image_path: tên file trên đĩa chứa room_id cũ,
--     file không bị rename nên đường dẫn đã lưu vẫn đúng.
--
-- ⚠️ TRƯỚC KHI CHẠY:
--   1. BACKUP database (pg_dump).
--   2. Dừng ứng dụng (tránh ghi xen kẽ trong lúc migrate).
--   3. Chạy: psql "$DATABASE_URL" -f scripts/migrate_uuid_v4_to_v7.sql
--
-- Toàn bộ chạy trong 1 transaction: lỗi ở bất kỳ đâu → rollback sạch.
-- ============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- 0. Hàm sinh UUID v7 từ một timestamp cho trước (48 bit đầu = unix ms,
--    còn lại random). Tạo trong pg_temp → tự biến mất khi đóng session.
--    Cần pgcrypto (đã CREATE EXTENSION ở migration 000001).
-- ----------------------------------------------------------------------------
CREATE FUNCTION pg_temp.uuid_v7_at(ts TIMESTAMPTZ) RETURNS UUID AS $$
DECLARE
    unix_ms BIGINT := FLOOR(EXTRACT(EPOCH FROM COALESCE(ts, NOW())) * 1000)::BIGINT;
    buf BYTEA;
BEGIN
    -- 6 byte timestamp + 10 byte random
    buf := SUBSTRING(INT8SEND(unix_ms) FROM 3 FOR 6) || GEN_RANDOM_BYTES(10);
    -- set version = 7 (4 bit cao của byte 6)
    buf := SET_BYTE(buf, 6, (GET_BYTE(buf, 6) & 15) | 112);
    -- set variant = 10xx (2 bit cao của byte 8)
    buf := SET_BYTE(buf, 8, (GET_BYTE(buf, 8) & 63) | 128);
    RETURN ENCODE(buf, 'hex')::UUID;
END;
$$ LANGUAGE plpgsql VOLATILE;

-- Điều kiện nhận diện bản ghi v4: ký tự thứ 15 của chuỗi UUID là version.

-- ----------------------------------------------------------------------------
-- 1. Chuyển mọi FK trỏ tới 4 bảng cha sang DEFERRABLE, rồi DEFER trong
--    transaction này (FK gốc không có ON UPDATE CASCADE nên bắt buộc).
-- ----------------------------------------------------------------------------
DO $$
DECLARE r RECORD;
BEGIN
    FOR r IN
        SELECT conrelid::regclass AS tbl, conname
        FROM pg_constraint
        WHERE contype = 'f'
          AND confrelid IN ('users'::regclass, 'houses'::regclass,
                            'rooms'::regclass, 'invoices'::regclass)
    LOOP
        EXECUTE FORMAT('ALTER TABLE %s ALTER CONSTRAINT %I DEFERRABLE INITIALLY IMMEDIATE',
                       r.tbl, r.conname);
    END LOOP;
END $$;

SET CONSTRAINTS ALL DEFERRED;

-- ----------------------------------------------------------------------------
-- 2. Bảng mapping old_id → new_id cho 4 bảng cha
-- ----------------------------------------------------------------------------
CREATE TEMP TABLE map_users ON COMMIT DROP AS
SELECT id AS old_id, pg_temp.uuid_v7_at(created_at) AS new_id
FROM users WHERE SUBSTRING(id::TEXT, 15, 1) = '4';

CREATE TEMP TABLE map_houses ON COMMIT DROP AS
SELECT id AS old_id, pg_temp.uuid_v7_at(created_at) AS new_id
FROM houses WHERE SUBSTRING(id::TEXT, 15, 1) = '4';

CREATE TEMP TABLE map_rooms ON COMMIT DROP AS
SELECT id AS old_id, pg_temp.uuid_v7_at(created_at) AS new_id
FROM rooms WHERE SUBSTRING(id::TEXT, 15, 1) = '4';

CREATE TEMP TABLE map_invoices ON COMMIT DROP AS
SELECT id AS old_id, pg_temp.uuid_v7_at(created_at) AS new_id
FROM invoices WHERE SUBSTRING(id::TEXT, 15, 1) = '4';

-- ----------------------------------------------------------------------------
-- 3. users: cập nhật các cột FK tham chiếu rồi đổi PK
-- ----------------------------------------------------------------------------
UPDATE houses c SET manager_id = m.new_id FROM map_users m WHERE c.manager_id = m.old_id;
UPDATE auth_sessions c SET user_id = m.new_id FROM map_users m WHERE c.user_id = m.old_id;
UPDATE tenants c SET user_id = m.new_id FROM map_users m WHERE c.user_id = m.old_id;
UPDATE tenants c SET manager_id = m.new_id FROM map_users m WHERE c.manager_id = m.old_id;
UPDATE pending_invoice_updates c SET manager_id = m.new_id FROM map_users m WHERE c.manager_id = m.old_id;
UPDATE payment_provider_credentials c SET manager_id = m.new_id FROM map_users m WHERE c.manager_id = m.old_id;
UPDATE payment_events c SET manager_id = m.new_id FROM map_users m WHERE c.manager_id = m.old_id;

UPDATE users t SET id = m.new_id FROM map_users m WHERE t.id = m.old_id;

-- ----------------------------------------------------------------------------
-- 4. houses (giữ nguyên house_code)
-- ----------------------------------------------------------------------------
UPDATE rooms c SET house_id = m.new_id FROM map_houses m WHERE c.house_id = m.old_id;
UPDATE house_costs c SET house_id = m.new_id FROM map_houses m WHERE c.house_id = m.old_id;
UPDATE house_revenue_summaries c SET house_id = m.new_id FROM map_houses m WHERE c.house_id = m.old_id;

UPDATE houses t SET id = m.new_id FROM map_houses m WHERE t.id = m.old_id;

-- ----------------------------------------------------------------------------
-- 5. rooms
-- ----------------------------------------------------------------------------
UPDATE tenants c SET room_id = m.new_id FROM map_rooms m WHERE c.room_id = m.old_id;
UPDATE invoices c SET room_id = m.new_id FROM map_rooms m WHERE c.room_id = m.old_id;
UPDATE pending_invoice_updates c SET room_id = m.new_id FROM map_rooms m WHERE c.room_id = m.old_id;

UPDATE rooms t SET id = m.new_id FROM map_rooms m WHERE t.id = m.old_id;

-- ----------------------------------------------------------------------------
-- 6. invoices
-- ----------------------------------------------------------------------------
UPDATE invoice_payment_links c SET invoice_id = m.new_id FROM map_invoices m WHERE c.invoice_id = m.old_id;
UPDATE payment_events c SET invoice_id = m.new_id FROM map_invoices m WHERE c.invoice_id = m.old_id;

UPDATE invoices t SET id = m.new_id FROM map_invoices m WHERE t.id = m.old_id;

-- ----------------------------------------------------------------------------
-- 7. Các bảng chỉ có PK, không bị bảng nào tham chiếu → đổi trực tiếp.
--    (otp_checks không có created_at → dùng NOW();
--     house_revenue_summaries không có created_at → dùng updated_at)
-- ----------------------------------------------------------------------------
UPDATE auth_sessions SET id = pg_temp.uuid_v7_at(created_at) WHERE SUBSTRING(id::TEXT, 15, 1) = '4';
UPDATE email_verifications SET id = pg_temp.uuid_v7_at(created_at) WHERE SUBSTRING(id::TEXT, 15, 1) = '4';
UPDATE otp_checks SET id = pg_temp.uuid_v7_at(NULL) WHERE SUBSTRING(id::TEXT, 15, 1) = '4';
UPDATE tenants SET id = pg_temp.uuid_v7_at(created_at) WHERE SUBSTRING(id::TEXT, 15, 1) = '4';
UPDATE house_costs SET id = pg_temp.uuid_v7_at(created_at) WHERE SUBSTRING(id::TEXT, 15, 1) = '4';
UPDATE house_revenue_summaries SET id = pg_temp.uuid_v7_at(updated_at) WHERE SUBSTRING(id::TEXT, 15, 1) = '4';
UPDATE pending_invoice_updates SET id = pg_temp.uuid_v7_at(created_at) WHERE SUBSTRING(id::TEXT, 15, 1) = '4';
UPDATE payment_provider_credentials SET id = pg_temp.uuid_v7_at(created_at) WHERE SUBSTRING(id::TEXT, 15, 1) = '4';
UPDATE invoice_payment_links SET id = pg_temp.uuid_v7_at(created_at) WHERE SUBSTRING(id::TEXT, 15, 1) = '4';
UPDATE payment_events SET id = pg_temp.uuid_v7_at(created_at) WHERE SUBSTRING(id::TEXT, 15, 1) = '4';
UPDATE payos_payment_events SET id = pg_temp.uuid_v7_at(created_at) WHERE SUBSTRING(id::TEXT, 15, 1) = '4';

-- ----------------------------------------------------------------------------
-- 8. Trả FK về NOT DEFERRABLE như ban đầu (dữ liệu đã nhất quán nên các
--    check dồn lại sẽ pass tại đây).
-- ----------------------------------------------------------------------------
SET CONSTRAINTS ALL IMMEDIATE;

DO $$
DECLARE r RECORD;
BEGIN
    FOR r IN
        SELECT conrelid::regclass AS tbl, conname
        FROM pg_constraint
        WHERE contype = 'f'
          AND confrelid IN ('users'::regclass, 'houses'::regclass,
                            'rooms'::regclass, 'invoices'::regclass)
    LOOP
        EXECUTE FORMAT('ALTER TABLE %s ALTER CONSTRAINT %I NOT DEFERRABLE',
                       r.tbl, r.conname);
    END LOOP;
END $$;

-- ----------------------------------------------------------------------------
-- 9. Kiểm tra nhanh: còn bản ghi v4 nào không (mong đợi tất cả = 0)
-- ----------------------------------------------------------------------------
SELECT 'users' AS tbl, COUNT(*) AS remaining_v4 FROM users WHERE SUBSTRING(id::TEXT, 15, 1) = '4'
UNION ALL SELECT 'houses', COUNT(*) FROM houses WHERE SUBSTRING(id::TEXT, 15, 1) = '4'
UNION ALL SELECT 'rooms', COUNT(*) FROM rooms WHERE SUBSTRING(id::TEXT, 15, 1) = '4'
UNION ALL SELECT 'invoices', COUNT(*) FROM invoices WHERE SUBSTRING(id::TEXT, 15, 1) = '4'
UNION ALL SELECT 'tenants', COUNT(*) FROM tenants WHERE SUBSTRING(id::TEXT, 15, 1) = '4'
UNION ALL SELECT 'auth_sessions', COUNT(*) FROM auth_sessions WHERE SUBSTRING(id::TEXT, 15, 1) = '4'
UNION ALL SELECT 'email_verifications', COUNT(*) FROM email_verifications WHERE SUBSTRING(id::TEXT, 15, 1) = '4'
UNION ALL SELECT 'otp_checks', COUNT(*) FROM otp_checks WHERE SUBSTRING(id::TEXT, 15, 1) = '4'
UNION ALL SELECT 'house_costs', COUNT(*) FROM house_costs WHERE SUBSTRING(id::TEXT, 15, 1) = '4'
UNION ALL SELECT 'house_revenue_summaries', COUNT(*) FROM house_revenue_summaries WHERE SUBSTRING(id::TEXT, 15, 1) = '4'
UNION ALL SELECT 'pending_invoice_updates', COUNT(*) FROM pending_invoice_updates WHERE SUBSTRING(id::TEXT, 15, 1) = '4'
UNION ALL SELECT 'payment_provider_credentials', COUNT(*) FROM payment_provider_credentials WHERE SUBSTRING(id::TEXT, 15, 1) = '4'
UNION ALL SELECT 'invoice_payment_links', COUNT(*) FROM invoice_payment_links WHERE SUBSTRING(id::TEXT, 15, 1) = '4'
UNION ALL SELECT 'payment_events', COUNT(*) FROM payment_events WHERE SUBSTRING(id::TEXT, 15, 1) = '4'
UNION ALL SELECT 'payos_payment_events', COUNT(*) FROM payos_payment_events WHERE SUBSTRING(id::TEXT, 15, 1) = '4';

COMMIT;
