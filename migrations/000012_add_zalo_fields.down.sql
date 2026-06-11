ALTER TABLE users DROP COLUMN IF EXISTS zalo_bot_token;
ALTER TABLE users DROP COLUMN IF EXISTS is_zalo_bot_active;
ALTER TABLE rooms DROP COLUMN IF EXISTS group_chat_id;
ALTER TABLE invoices DROP COLUMN IF EXISTS transaction_image_path;
ALTER TABLE tenants DROP COLUMN IF EXISTS zalo_user_id;
