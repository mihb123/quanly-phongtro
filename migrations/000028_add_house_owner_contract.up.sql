-- Mô hình thuê lại: manager thuê nguyên căn của chủ nhà rồi cho thuê từng phòng,
-- nên mỗi nhà cần lưu thông tin chủ nhà + hợp đồng thuê nguyên căn (CCCD, hợp đồng).
-- Giá thuê/tiền cọc ở đây là điều khoản hợp đồng; số tiền thực chi mỗi tháng vẫn nằm ở house_costs.rent.
ALTER TABLE houses
    ADD COLUMN IF NOT EXISTS owner_name          TEXT          NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS owner_phone         TEXT          NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS owner_rent_price    DECIMAL(12,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS owner_deposit       DECIMAL(12,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS rent_start_date     DATE,
    ADD COLUMN IF NOT EXISTS rent_end_date       DATE,
    -- Nhiều file, ngăn cách bởi dấu phẩy, cùng định dạng với rooms.contract_path.
    ADD COLUMN IF NOT EXISTS owner_cccd_path     TEXT          NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS owner_contract_path TEXT          NOT NULL DEFAULT '';
