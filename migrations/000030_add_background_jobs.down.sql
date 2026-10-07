DROP TRIGGER deleted_room_revenue_job ON rooms;
DROP FUNCTION queue_deleted_room_revenue();
DROP TRIGGER cost_revenue_job ON house_costs;
DROP TRIGGER invoice_revenue_job ON invoices;
DROP FUNCTION queue_cost_revenue();
DROP FUNCTION queue_invoice_revenue();
DROP FUNCTION enqueue_revenue_job(UUID, TEXT);
DROP TABLE background_jobs;
