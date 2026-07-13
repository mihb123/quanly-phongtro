-- Ép mọi cột id UUID dùng uuidv7() (native từ PostgreSQL 18) thay cho
-- gen_random_uuid() (v4) — ID mới sắp theo thời gian, giảm phân mảnh B-tree index.
-- App tự sinh ID ở tầng Go (đã chuyển sang v7); default này là lưới an toàn
-- cho các insert trực tiếp vào DB.
ALTER TABLE users ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE houses ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE rooms ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE tenants ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE invoices ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE auth_sessions ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE email_verifications ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE otp_checks ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE house_costs ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE house_revenue_summaries ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE pending_invoice_updates ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE payment_provider_credentials ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE invoice_payment_links ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE payment_events ALTER COLUMN id SET DEFAULT uuidv7();

-- Bảng chỉ tồn tại ở DB cũ (không có trong migrations) — đổi nếu có
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema = 'public' AND table_name = 'payos_payment_events') THEN
        ALTER TABLE payos_payment_events ALTER COLUMN id SET DEFAULT uuidv7();
    END IF;
END $$;
