package room

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/uptrace/bun"
)

type RoomRepository struct {
	db *bun.DB
}

func NewRoomRepository(db *bun.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

// CreateRoom inserts a new room. Ownership must be verified by the caller before this.
func (r *RoomRepository) CreateRoom(ctx context.Context, room *model.Room) error {
	_, err := r.db.NewInsert().
		Model(room).
		Column("house_id", "name", "price", "max_tenants", "status", "electricity_price", "water_price", "wifi_price", "parking_price", "service_price", "extra_person_threshold", "extra_person_fee", "extra_vehicle_threshold", "extra_vehicle_fee", "group_chat_id").
		Returning("id, created_at, updated_at").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("create room: %w", err)
	}
	return nil
}

// GetRoomByID fetches a room by its id within a specific house.
// GetRoomByID fetches a room by its id. If houseID is provided, it must match.
// Ownership must be verified by the caller before this.
func (r *RoomRepository) GetRoomByID(ctx context.Context, id, houseID string) (*model.Room, error) {
	var room model.Room
	err := r.db.NewSelect().
		Model(&room).
		Where("id = ? AND house_id = ?", id, houseID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrRoomNotFound
		}
		return nil, fmt.Errorf("get room by id: %w", err)
	}
	return &room, nil
}

// ListRoomsByHouseID lists rooms for a house with pagination, search and active tenant counts.
// Ownership must be verified by the caller before this.
func (r *RoomRepository) ListRoomsByHouseID(ctx context.Context, filter model.RoomListFilter) ([]model.Room, int, error) {
	var rooms []model.Room
	q := r.db.NewSelect().
		Model(&rooms).
		ColumnExpr("room.*").
		ColumnExpr("(SELECT COUNT(*) FROM tenants AS t WHERE t.room_id = room.id AND t.status = ?) AS tenant_count", string(model.TenantStatusActive)).
		Where("room.house_id = ?", filter.HouseID)

	if search := strings.TrimSpace(filter.Search); search != "" {
		q.Where("room.name ILIKE ?", "%"+escapeLike(search)+"%")
	}
	if filter.Status != "" {
		q.Where("room.status = ?", filter.Status)
	}

	total, err := q.
		OrderExpr("length(room.name) ASC, room.name ASC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		ScanAndCount(ctx)

	if err != nil {
		return nil, 0, fmt.Errorf("list rooms: %w", err)
	}
	return rooms, total, nil
}

// GetRoomStats counts rooms by status and active tenants in one aggregate query.
func (r *RoomRepository) GetRoomStats(ctx context.Context, managerID, houseID string) (*model.RoomStats, error) {
	var stats model.RoomStats
	q := r.db.NewSelect().
		TableExpr("rooms AS room").
		Join("JOIN houses AS h ON h.id = room.house_id").
		ColumnExpr("COUNT(*) AS total").
		ColumnExpr("COUNT(*) FILTER (WHERE room.status = 'OCCUPIED') AS occupied").
		ColumnExpr("COUNT(*) FILTER (WHERE room.status = 'AVAILABLE') AS available").
		ColumnExpr("COUNT(*) FILTER (WHERE room.status = 'MAINTENANCE') AS maintenance").
		ColumnExpr("COALESCE(SUM(room.max_tenants), 0) AS capacity").
		ColumnExpr("COALESCE(SUM((SELECT COUNT(*) FROM tenants AS t WHERE t.room_id = room.id AND t.status = ?)), 0) AS tenants", string(model.TenantStatusActive)).
		Where("h.manager_id = ?", managerID)
	if houseID != "" {
		q.Where("room.house_id = ?", houseID)
	}
	if err := q.Scan(ctx, &stats); err != nil {
		return nil, fmt.Errorf("get room stats: %w", err)
	}
	return &stats, nil
}

// ListAvailableRooms lists available rooms across every house of the manager.
func (r *RoomRepository) ListAvailableRooms(ctx context.Context, managerID string, limit int) ([]model.Room, int, error) {
	var rooms []model.Room
	total, err := r.db.NewSelect().
		Model(&rooms).
		ColumnExpr("room.*").
		ColumnExpr("h.name AS house_name").
		Join("JOIN houses AS h ON h.id = room.house_id").
		Where("h.manager_id = ?", managerID).
		Where("room.status = ?", "AVAILABLE").
		OrderExpr("h.name ASC, length(room.name) ASC, room.name ASC").
		Limit(limit).
		ScanAndCount(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list available rooms: %w", err)
	}
	return rooms, total, nil
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(s)
}

// ListAllRoomsByHouseID lists all rooms for a house without pagination.
// Ownership must be verified by the caller before this.
func (r *RoomRepository) ListAllRoomsByHouseID(ctx context.Context, houseID string) ([]model.Room, error) {
	var rooms []model.Room
	err := r.db.NewSelect().
		Model(&rooms).
		Where("house_id = ?", houseID).
		Order("name ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("list all rooms: %w", err)
	}
	return rooms, nil
}

// UpdateRoom partially updates a room within a specific house.
// Ownership must be verified by the caller before this.
func (r *RoomRepository) UpdateRoom(ctx context.Context, id, houseID string, params model.UpdateRoomParams) (*model.Room, error) {
	q := r.db.NewUpdate().
		Model((*model.Room)(nil)).
		Where("id = ? AND house_id = ?", id, houseID).
		Returning("id, house_id, name, price, max_tenants, status, electricity_price, water_price, wifi_price, parking_price, service_price, extra_person_threshold, extra_person_fee, extra_vehicle_threshold, extra_vehicle_fee, group_chat_id, contract_path, created_at, updated_at")

	q.Set("name = ?", params.Name)
	q.Set("price = ?", params.Price)
	q.Set("max_tenants = ?", params.MaxTenants)
	q.Set("status = ?", params.Status)
	q.Set("electricity_price = ?", params.ElectricityPrice)
	q.Set("water_price = ?", params.WaterPrice)
	q.Set("wifi_price = ?", params.WifiPrice)
	q.Set("parking_price = ?", params.ParkingPrice)
	q.Set("service_price = ?", params.ServicePrice)
	q.Set("extra_person_threshold = ?", params.ExtraPersonThreshold)
	q.Set("extra_person_fee = ?", params.ExtraPersonFee)
	q.Set("extra_vehicle_threshold = ?", params.ExtraVehicleThreshold)
	q.Set("extra_vehicle_fee = ?", params.ExtraVehicleFee)
	q.Set("group_chat_id = ?", params.GroupChatID)
	q.Set("updated_at = NOW()")

	var room model.Room
	err := q.Scan(ctx, &room.ID, &room.HouseID, &room.Name, &room.Price, &room.MaxTenants, &room.Status, &room.ElectricityPrice, &room.WaterPrice, &room.WifiPrice, &room.ParkingPrice, &room.ServicePrice, &room.ExtraPersonThreshold, &room.ExtraPersonFee, &room.ExtraVehicleThreshold, &room.ExtraVehicleFee, &room.GroupChatID, &room.ContractPath, &room.CreatedAt, &room.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrRoomNotFound
		}
		return nil, fmt.Errorf("update room: %w", err)
	}
	return &room, nil
}

// DeleteRoom deletes a room within a specific house.
// Ownership must be verified by the caller before this.
func (r *RoomRepository) DeleteRoom(ctx context.Context, id, houseID string) error {
	res, err := r.db.NewDelete().
		Model((*model.Room)(nil)).
		Where("id = ? AND house_id = ?", id, houseID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete room: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete room rows affected: %w", err)
	}
	if n == 0 {
		return model.ErrRoomNotFound
	}
	return nil
}

func (r *RoomRepository) GetMaxTenants(ctx context.Context, roomID string) (int64, error) {
	var maxTenants int64
	err := r.db.NewSelect().
		Model((*model.Room)(nil)).
		Column("max_tenants").
		Where("id = ?", roomID).
		Scan(ctx, &maxTenants)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, model.ErrNotFound
		}
		return 0, fmt.Errorf("error get max tenant by id: %v", err)
	}
	return maxTenants, nil
}

func (r *RoomRepository) UpdateRoomStatus(ctx context.Context, roomID, status string) error {
	res, err := r.db.NewUpdate().
		Model((*model.Room)(nil)).
		Set("status = ?", status).
		Where("id = ?", roomID).
		Exec(ctx)
	if err != nil {
		return err
	}
	row, err := res.RowsAffected()
	if err != nil || row == 0 {
		return model.ErrNotFound
	}
	return nil
}

// GetRoomByIDOnly fetches a room by its ID only.
func (r *RoomRepository) GetRoomByIDOnly(ctx context.Context, id string) (*model.Room, error) {
	var room model.Room
	err := r.db.NewSelect().
		Model(&room).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrRoomNotFound
		}
		return nil, fmt.Errorf("get room by id only: %w", err)
	}
	return &room, nil
}

// GetRoomByIDForManager fetches a room only when its house belongs to the manager.
func (r *RoomRepository) GetRoomByIDForManager(ctx context.Context, managerID, roomID string) (*model.Room, error) {
	var room model.Room
	err := r.db.NewSelect().
		Model(&room).
		ModelTableExpr("rooms AS room").
		ColumnExpr("room.*").
		Join("JOIN houses AS h ON h.id = room.house_id").
		Where("room.id = ?", roomID).
		Where("h.manager_id = ?", managerID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrRoomNotFound
		}
		return nil, fmt.Errorf("get room by id for manager: %w", err)
	}
	return &room, nil
}

// GetRoomByGroupChatID fetches a room by its Zalo group chat ID.
func (r *RoomRepository) GetRoomByGroupChatID(ctx context.Context, groupChatID string) (*model.Room, error) {
	var room model.Room
	err := r.db.NewSelect().
		Model(&room).
		Where("group_chat_id = ?", groupChatID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrRoomNotFound
		}
		return nil, fmt.Errorf("get room by group chat id: %w", err)
	}
	return &room, nil
}

// UpdateRoomContract replaces the contract file paths stored on a room.
// Ownership must be verified by the caller before this.
func (r *RoomRepository) UpdateRoomContract(ctx context.Context, id, houseID, contractPath string) (*model.Room, error) {
	var room model.Room
	err := r.db.NewUpdate().
		Model(&room).
		Set("contract_path = ?", contractPath).
		Set("updated_at = NOW()").
		Where("id = ? AND house_id = ?", id, houseID).
		Returning("*").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrRoomNotFound
		}
		return nil, fmt.Errorf("update room contract: %w", err)
	}
	return &room, nil
}

// HasRoomWithFilePath reports whether any room of the manager references the given upload path.
func (r *RoomRepository) HasRoomWithFilePath(ctx context.Context, managerID, filePath string) (bool, error) {
	exists, err := r.db.NewSelect().
		Model((*model.Room)(nil)).
		ModelTableExpr("rooms AS room").
		Join("JOIN houses AS h ON h.id = room.house_id").
		Where("h.manager_id = ?", managerID).
		Where(`EXISTS (
			SELECT 1 FROM unnest(string_to_array(COALESCE(room.contract_path, ''), ',')) AS path
			WHERE trim(path) = ?
		)`, filePath).
		Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("has room with file path: %w", err)
	}
	return exists, nil
}
