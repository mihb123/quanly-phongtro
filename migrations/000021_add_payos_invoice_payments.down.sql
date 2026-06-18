DROP TABLE IF EXISTS payment_events;
DROP TABLE IF EXISTS invoice_payment_links;
DROP TABLE IF EXISTS payment_provider_credentials;

ALTER TABLE invoices DROP COLUMN IF EXISTS payment_method;
