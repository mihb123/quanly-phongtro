DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'house_costs'
          AND column_name = 'maintenance'
    ) THEN
        UPDATE house_costs
        SET extra_costs = COALESCE(extra_costs, '[]'::jsonb) || jsonb_build_array(
            jsonb_build_object('name', 'Tiền bảo trì, sửa chữa', 'amount', maintenance, 'note', '')
        )
        WHERE maintenance > 0;

        ALTER TABLE house_costs DROP COLUMN maintenance;
    END IF;
END $$;
