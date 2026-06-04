-- Thêm lại cột zalo_user_id vào tenants
ALTER TABLE tenants ADD COLUMN zalo_user_id VARCHAR(255) NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_zalo_user_id ON tenants(zalo_user_id);

-- Chuyển lại dữ liệu
UPDATE tenants 
SET zalo_user_id = u.zalo_user_id
FROM users u 
WHERE tenants.user_id = u.id AND u.zalo_user_id IS NOT NULL;

-- Xóa cột khỏi users
DROP INDEX IF EXISTS idx_users_zalo_user_id;
ALTER TABLE users DROP COLUMN zalo_user_id;
