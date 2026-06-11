-- Di chuyển zalo_user_id sang bảng users
ALTER TABLE users ADD COLUMN zalo_user_id VARCHAR(255) NULL;
CREATE INDEX IF NOT EXISTS idx_users_zalo_user_id ON users(zalo_user_id);

-- Chuyển dữ liệu (từ tenants.zalo_user_id sang users.zalo_user_id)
UPDATE users 
SET zalo_user_id = t.zalo_user_id
FROM tenants t 
WHERE users.id = t.user_id AND t.zalo_user_id IS NOT NULL;

-- Xóa cột zalo_user_id khỏi tenants
DROP INDEX IF EXISTS idx_tenants_zalo_user_id;
ALTER TABLE tenants DROP COLUMN zalo_user_id;
