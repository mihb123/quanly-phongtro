package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

type HouseRepository struct {
	db *sql.DB
}

func NewHouseRepository(db *sql.DB) *HouseRepository {
	return &HouseRepository{db: db}
}

func (r *HouseRepository) CreateHouse(ctx context.Context, h *model.House) error {
	query := `INSERT INTO houses (manager_id, name, address, default_electricity_price, default_water_price, default_wifi_price, default_parking_price, default_service_price)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			  RETURNING id, created_at, updated_at`
	err := r.db.QueryRowContext(ctx, query, h.ManagerID, h.Name, h.Address, h.DefaultElectricityPrice, h.DefaultWaterPrice, h.DefaultWifiPrice, h.DefaultParkingPrice, h.DefaultServicePrice).Scan(
		&h.ID,
		&h.CreatedAt,
		&h.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create house: %w", err)
	}
	return nil
}

func (r *HouseRepository) GetByID(ctx context.Context, id, managerID string) (*model.House, error) {
	const query = `
		SELECT id, manager_id, name, address, default_electricity_price, default_water_price,
		       default_wifi_price, default_parking_price, default_service_price, created_at, updated_at
		FROM houses
		WHERE id = $1 AND manager_id = $2
	`

	var h model.House
	err := r.db.QueryRowContext(ctx, query, id, managerID).Scan(
		&h.ID,
		&h.ManagerID,
		&h.Name,
		&h.Address,
		&h.DefaultElectricityPrice,
		&h.DefaultWaterPrice,
		&h.DefaultWifiPrice,
		&h.DefaultParkingPrice,
		&h.DefaultServicePrice,
		&h.CreatedAt,
		&h.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrHouseNotFound
		}

		return nil, fmt.Errorf("get house by id: %w", err)
	}

	return &h, nil
}

func (r *HouseRepository) ListHouseByManagerID(ctx context.Context, managerID string, limit, offset int, search string) ([]model.House, error) {
	args := []any{managerID}
	argIdx := 2 // $1 is already managerID

	var whereClauses []string
	whereClauses = append(whereClauses, "manager_id = $1")

	if search != "" {
		whereClauses = append(whereClauses,
			fmt.Sprintf("(name ILIKE $%d OR address ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}

	limitClause := fmt.Sprintf("LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	query := fmt.Sprintf(`
		SELECT id, manager_id, name, address,
		       default_electricity_price, default_water_price,
		       default_wifi_price, default_parking_price, default_service_price,
		       created_at, updated_at
		FROM houses
		WHERE %s
		ORDER BY created_at DESC, name ASC
		%s`,
		strings.Join(whereClauses, " AND "),
		limitClause,
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list houses: %w", err)
	}
	defer rows.Close()

	var houses []model.House
	for rows.Next() {
		var h model.House
		if err := rows.Scan(
			&h.ID,
			&h.ManagerID,
			&h.Name,
			&h.Address,
			&h.DefaultElectricityPrice,
			&h.DefaultWaterPrice,
			&h.DefaultWifiPrice,
			&h.DefaultParkingPrice,
			&h.DefaultServicePrice,
			&h.CreatedAt,
			&h.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("list houses scan: %w", err)
		}
		houses = append(houses, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list houses rows: %w", err)
	}

	return houses, nil
}

func (r *HouseRepository) UpdateHouse(ctx context.Context, id, managerID string, params model.UpdateHouseParams) (*model.House, error) {
	var setClauses []string
	args := []any{}
	argIdx := 1

	if params.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *params.Name)
		argIdx++
	}
	if params.Address != nil {
		setClauses = append(setClauses, fmt.Sprintf("address = $%d", argIdx))
		args = append(args, *params.Address)
		argIdx++
	}
	if params.DefaultElectricityPrice != nil {
		setClauses = append(setClauses, fmt.Sprintf("default_electricity_price = $%d", argIdx))
		args = append(args, *params.DefaultElectricityPrice)
		argIdx++
	}
	if params.DefaultWaterPrice != nil {
		setClauses = append(setClauses, fmt.Sprintf("default_water_price = $%d", argIdx))
		args = append(args, *params.DefaultWaterPrice)
		argIdx++
	}
	if params.DefaultWifiPrice != nil {
		setClauses = append(setClauses, fmt.Sprintf("default_wifi_price = $%d", argIdx))
		args = append(args, *params.DefaultWifiPrice)
		argIdx++
	}
	if params.DefaultParkingPrice != nil {
		setClauses = append(setClauses, fmt.Sprintf("default_parking_price = $%d", argIdx))
		args = append(args, *params.DefaultParkingPrice)
		argIdx++
	}
	if params.DefaultServicePrice != nil {
		setClauses = append(setClauses, fmt.Sprintf("default_service_price = $%d", argIdx))
		args = append(args, *params.DefaultServicePrice)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil, fmt.Errorf("update house: no fields to update")
	}

	// Always refresh updated_at on every write.
	setClauses = append(setClauses, "updated_at = NOW()")

	// Append WHERE args: id and manager_id.
	args = append(args, id, managerID)

	query := fmt.Sprintf(`
		UPDATE houses
		SET %s
		WHERE id = $%d AND manager_id = $%d
		RETURNING id, manager_id, name, address,
		          default_electricity_price, default_water_price,
		          default_wifi_price, default_parking_price, default_service_price,
		          created_at, updated_at`,
		strings.Join(setClauses, ", "),
		argIdx,
		argIdx+1,
	)

	var h model.House
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&h.ID,
		&h.ManagerID,
		&h.Name,
		&h.Address,
		&h.DefaultElectricityPrice,
		&h.DefaultWaterPrice,
		&h.DefaultWifiPrice,
		&h.DefaultParkingPrice,
		&h.DefaultServicePrice,
		&h.CreatedAt,
		&h.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrHouseNotFound
		}
		return nil, fmt.Errorf("update house: %w", err)
	}

	return &h, nil
}

func (r *HouseRepository) DeleteHouse(ctx context.Context, id, managerID string) error {
	const query = `DELETE FROM houses WHERE id = $1 AND manager_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, managerID)
	if err != nil {
		return fmt.Errorf("delete house: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete house rows affected: %w", err)
	}
	if rows == 0 {
		return model.ErrHouseNotFound
	}
	return nil
}

// IsHouseOwnedBy returns true when a house with the given id exists and its
// manager_id matches managerID. Uses SELECT 1 to avoid fetching the full row.
func (r *HouseRepository) IsHouseOwnedBy(ctx context.Context, houseID, managerID string) (bool, error) {
	const query = `SELECT 1 FROM houses WHERE id = $1 AND manager_id = $2`
	var exists int
	err := r.db.QueryRowContext(ctx, query, houseID, managerID).Scan(&exists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("is house owned by: %w", err)
	}
	return true, nil
}
