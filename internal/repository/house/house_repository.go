package house

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/uptrace/bun"
)

type HouseRepository struct {
	db *bun.DB
}

func NewHouseRepository(db *bun.DB) *HouseRepository {
	return &HouseRepository{db: db}
}

func (r *HouseRepository) CreateHouse(ctx context.Context, h *model.House) error {
	_, err := r.db.NewInsert().
		Model(h).
		Column("manager_id", "name", "house_code", "address", "default_electricity_price", "default_water_price", "default_wifi_price", "default_parking_price", "default_service_price", "electricity_billing_type", "water_billing_type", "electricity_billing_unit", "water_billing_unit", "extra_person_threshold", "extra_person_fee", "extra_vehicle_threshold", "extra_vehicle_fee", "owner_name", "owner_phone", "owner_rent_price", "owner_deposit", "rent_start_date", "rent_end_date").
		Returning("id, created_at, updated_at").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("create house: %w", err)
	}
	return nil
}

func (r *HouseRepository) GetByID(ctx context.Context, id, managerID string) (*model.House, error) {
	var h model.House
	err := r.db.NewSelect().
		Model(&h).
		Where("id = ? AND manager_id = ?", id, managerID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrHouseNotFound
		}
		return nil, fmt.Errorf("get house by id: %w", err)
	}
	return &h, nil
}

// GetHouseByCode finds a house by its short code within a manager's houses.
func (r *HouseRepository) GetHouseByCode(ctx context.Context, managerID, houseCode string) (*model.House, error) {
	var h model.House
	err := r.db.NewSelect().
		Model(&h).
		Where("manager_id = ? AND LOWER(house_code) = LOWER(?)", managerID, houseCode).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrHouseNotFound
		}
		return nil, fmt.Errorf("get house by code: %w", err)
	}
	return &h, nil
}

// IsHouseCodeTaken reports whether house_code is used by any house system-wide
// (case-insensitive), ignoring excludeHouseID (empty means check all houses).
func (r *HouseRepository) IsHouseCodeTaken(ctx context.Context, houseCode, excludeHouseID string) (bool, error) {
	q := r.db.NewSelect().
		Model((*model.House)(nil)).
		Where("LOWER(house_code) = LOWER(?)", houseCode)
	if excludeHouseID != "" {
		q = q.Where("id <> ?", excludeHouseID)
	}
	exists, err := q.Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("is house code taken: %w", err)
	}
	return exists, nil
}

func (r *HouseRepository) ListHouseByManagerID(ctx context.Context, managerID string, limit, offset int, search string) ([]model.House, error) {
	var houses []model.House
	q := r.db.NewSelect().
		Model(&houses).
		Where("manager_id = ?", managerID).
		OrderExpr("created_at DESC, name ASC").
		Limit(limit).
		Offset(offset)

	if search != "" {
		q.Where("(name ILIKE ? OR address ILIKE ?)", "%"+search+"%", "%"+search+"%")
	}

	err := q.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list houses: %w", err)
	}
	return houses, nil
}

func (r *HouseRepository) UpdateHouse(ctx context.Context, id, managerID string, params model.UpdateHouseParams) (*model.House, error) {
	var h model.House
	q := r.db.NewUpdate().
		Model(&h).
		Where("id = ? AND manager_id = ?", id, managerID).
		Returning("*")

	q.Set("name = ?", params.Name)
	q.Set("house_code = ?", params.HouseCode)
	q.Set("address = ?", params.Address)
	q.Set("default_electricity_price = ?", params.DefaultElectricityPrice)
	q.Set("default_water_price = ?", params.DefaultWaterPrice)
	q.Set("default_wifi_price = ?", params.DefaultWifiPrice)
	q.Set("default_parking_price = ?", params.DefaultParkingPrice)
	q.Set("default_service_price = ?", params.DefaultServicePrice)
	q.Set("electricity_billing_type = ?", params.ElectricityBillingType)
	q.Set("water_billing_type = ?", params.WaterBillingType)
	q.Set("electricity_billing_unit = ?", params.ElectricityBillingUnit)
	q.Set("water_billing_unit = ?", params.WaterBillingUnit)
	q.Set("extra_person_threshold = ?", params.ExtraPersonThreshold)
	q.Set("extra_person_fee = ?", params.ExtraPersonFee)
	q.Set("extra_vehicle_threshold = ?", params.ExtraVehicleThreshold)
	q.Set("extra_vehicle_fee = ?", params.ExtraVehicleFee)
	q.Set("owner_name = ?", params.OwnerName)
	q.Set("owner_phone = ?", params.OwnerPhone)
	q.Set("owner_rent_price = ?", params.OwnerRentPrice)
	q.Set("owner_deposit = ?", params.OwnerDeposit)
	q.Set("rent_start_date = ?", params.RentStartDate)
	q.Set("rent_end_date = ?", params.RentEndDate)
	q.Set("updated_at = NOW()")

	if err := q.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrHouseNotFound
		}
		return nil, fmt.Errorf("update house: %w", err)
	}
	return &h, nil
}

// UpdateHouseDocuments replaces the landlord CCCD and head-lease contract paths of a house.
// Ownership is enforced by the manager_id filter.
func (r *HouseRepository) UpdateHouseDocuments(ctx context.Context, id, managerID, cccdPath, contractPath string) (*model.House, error) {
	var h model.House
	err := r.db.NewUpdate().
		Model(&h).
		Set("owner_cccd_path = ?", cccdPath).
		Set("owner_contract_path = ?", contractPath).
		Set("updated_at = NOW()").
		Where("id = ? AND manager_id = ?", id, managerID).
		Returning("*").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrHouseNotFound
		}
		return nil, fmt.Errorf("update house documents: %w", err)
	}
	return &h, nil
}

// HasHouseWithFilePath reports whether any house of the manager references the given upload path
// in its landlord CCCD or head-lease contract.
func (r *HouseRepository) HasHouseWithFilePath(ctx context.Context, managerID, filePath string) (bool, error) {
	exists, err := r.db.NewSelect().
		Model((*model.House)(nil)).
		ModelTableExpr("houses AS house").
		Where("house.manager_id = ?", managerID).
		Where(`EXISTS (
			SELECT 1 FROM unnest(string_to_array(COALESCE(house.owner_cccd_path, '') || ',' || COALESCE(house.owner_contract_path, ''), ',')) AS path
			WHERE trim(path) <> '' AND trim(path) = ?
		)`, filePath).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("has house with file path: %w", err)
	}
	return exists, nil
}

func (r *HouseRepository) DeleteHouse(ctx context.Context, id, managerID string) error {
	res, err := r.db.NewDelete().
		Model((*model.House)(nil)).
		Where("id = ? AND manager_id = ?", id, managerID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete house: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete house rows affected: %w", err)
	}
	if rows == 0 {
		return model.ErrHouseNotFound
	}
	return nil
}

func (r *HouseRepository) IsHouseOwnedBy(ctx context.Context, houseID, managerID string) (bool, error) {
	var exists int
	err := r.db.NewSelect().
		ColumnExpr("1").
		Model((*model.House)(nil)).
		Where("id = ? AND manager_id = ?", houseID, managerID).
		Scan(ctx, &exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("is house owned by: %w", err)
	}
	return true, nil
}
