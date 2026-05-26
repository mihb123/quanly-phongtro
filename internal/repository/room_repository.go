package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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
		Column("house_id", "name", "price", "max_tenants", "status").
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

// ListRoomsByHouseID lists rooms for a house with pagination.
// Ownership must be verified by the caller before this.
func (r *RoomRepository) ListRoomsByHouseID(ctx context.Context, houseID string, limit, offset int) ([]model.Room, error) {
	var rooms []model.Room
	err := r.db.NewSelect().
		Model(&rooms).
		Where("house_id = ?", houseID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}
	return rooms, nil
}

// UpdateRoom partially updates a room within a specific house.
// Ownership must be verified by the caller before this.
func (r *RoomRepository) UpdateRoom(ctx context.Context, id, houseID string, params model.UpdateRoomParams) (*model.Room, error) {
	q := r.db.NewUpdate().
		Model((*model.Room)(nil)).
		Where("id = ? AND house_id = ?", id, houseID).
		Returning("id, house_id, name, price, max_tenants, status, created_at, updated_at")

	updated := false
	if params.Name != nil {
		q.Set("name = ?", *params.Name)
		updated = true
	}
	if params.Price != nil {
		q.Set("price = ?", *params.Price)
		updated = true
	}
	if params.MaxTenants != nil {
		q.Set("max_tenants = ?", *params.MaxTenants)
		updated = true
	}
	if params.Status != nil {
		q.Set("status = ?", *params.Status)
		updated = true
	}
	if !updated {
		return nil, fmt.Errorf("update room: no fields to update")
	}
	q.Set("updated_at = NOW()")

	var room model.Room
	err := q.Scan(ctx, &room.ID, &room.HouseID, &room.Name, &room.Price, &room.MaxTenants, &room.Status, &room.CreatedAt, &room.UpdatedAt)
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
