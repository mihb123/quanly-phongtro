-- Thêm thông tin Zalo Bot (encrypted) cho users (manager)
ALTER TABLE users ADD COLUMN zalo_bot_token TEXT NULL;
ALTER TABLE users ADD COLUMN is_zalo_bot_active BOOLEAN NOT NULL DEFAULT false;

-- Thêm cột group_chat_id cho rooms
ALTER TABLE rooms ADD COLUMN group_chat_id VARCHAR(255) NULL;
ALTER TABLE invoices ADD COLUMN transaction_image_path VARCHAR(255) NULL;

-- Thêm Zalo User ID cho tenant để nhắn tin riêng
ALTER TABLE tenants ADD COLUMN zalo_user_id VARCHAR(255) NULL;
CREATE INDEX IF NOT EXISTS idx_tenants_zalo_user_id ON tenants(zalo_user_id);
