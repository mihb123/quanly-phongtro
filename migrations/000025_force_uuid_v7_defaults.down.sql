ALTER TABLE users ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE houses ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE rooms ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE tenants ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE invoices ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE auth_sessions ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE email_verifications ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE otp_checks ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE house_costs ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE house_revenue_summaries ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE pending_invoice_updates ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE payment_provider_credentials ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE invoice_payment_links ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE payment_events ALTER COLUMN id SET DEFAULT gen_random_uuid();

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema = 'public' AND table_name = 'payos_payment_events') THEN
        ALTER TABLE payos_payment_events ALTER COLUMN id SET DEFAULT gen_random_uuid();
    END IF;
END $$;
