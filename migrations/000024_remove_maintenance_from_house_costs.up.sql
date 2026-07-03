UPDATE house_costs
SET extra_costs = extra_costs || jsonb_build_object('name', 'Tiền bảo trì, sửa chữa', 'amount', maintenance, 'note', '')
WHERE maintenance > 0;

ALTER TABLE house_costs DROP COLUMN maintenance;
