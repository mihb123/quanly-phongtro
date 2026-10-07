CREATE TABLE background_jobs (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    kind TEXT NOT NULL,
    job_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error TEXT,
    UNIQUE (kind, job_key)
);
CREATE INDEX background_jobs_ready ON background_jobs (kind, available_at, id);

CREATE FUNCTION enqueue_revenue_job(p_house_id UUID, p_period TEXT) RETURNS VOID AS $$
BEGIN
    IF p_house_id IS NOT NULL THEN
        INSERT INTO background_jobs (kind, job_key, payload)
        VALUES ('revenue', p_house_id::text || ':' || p_period,
                jsonb_build_object('house_id', p_house_id, 'period', p_period))
        ON CONFLICT (kind, job_key) DO UPDATE
        SET available_at = NOW(), payload = EXCLUDED.payload;
    END IF;
END;
$$ LANGUAGE plpgsql;

CREATE FUNCTION queue_invoice_revenue() RETURNS TRIGGER AS $$
DECLARE v_house_id UUID;
BEGIN
    IF TG_OP <> 'INSERT' THEN
        SELECT house_id INTO v_house_id FROM rooms WHERE id = OLD.room_id;
        PERFORM enqueue_revenue_job(v_house_id, OLD.period);
    END IF;
    IF TG_OP <> 'DELETE' THEN
        SELECT house_id INTO v_house_id FROM rooms WHERE id = NEW.room_id;
        PERFORM enqueue_revenue_job(v_house_id, NEW.period);
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER invoice_revenue_job AFTER INSERT OR UPDATE OR DELETE ON invoices
FOR EACH ROW EXECUTE FUNCTION queue_invoice_revenue();

CREATE FUNCTION queue_cost_revenue() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP <> 'INSERT' THEN
        PERFORM enqueue_revenue_job(OLD.house_id, OLD.period);
    END IF;
    IF TG_OP <> 'DELETE' THEN
        PERFORM enqueue_revenue_job(NEW.house_id, NEW.period);
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER cost_revenue_job AFTER INSERT OR UPDATE OR DELETE ON house_costs
FOR EACH ROW EXECUTE FUNCTION queue_cost_revenue();

CREATE FUNCTION queue_deleted_room_revenue() RETURNS TRIGGER AS $$
DECLARE v_period TEXT;
BEGIN
    FOR v_period IN SELECT DISTINCT period FROM invoices WHERE room_id = OLD.id LOOP
        PERFORM enqueue_revenue_job(OLD.house_id, v_period);
    END LOOP;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER deleted_room_revenue_job BEFORE DELETE ON rooms
FOR EACH ROW EXECUTE FUNCTION queue_deleted_room_revenue();

INSERT INTO background_jobs (kind, job_key, payload)
SELECT 'revenue', house_id::text || ':' || period,
       jsonb_build_object('house_id', house_id, 'period', period)
FROM (
    SELECT r.house_id, i.period FROM invoices i JOIN rooms r ON r.id = i.room_id
    UNION SELECT house_id, period FROM house_costs
    UNION SELECT house_id, period FROM house_revenue_summaries
) AS periods;
