package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

type RoomRepository struct {
	db *sql.DB
}

func NewRoomRepository(db *sql.DB) *RoomRepository {
	return &RoomRepository{db: db}
}

// CreateRoom inserts a new room. Ownership must be verified by the caller before this.
func (r *RoomRepository) CreateRoom(ctx context.Context, room *model.Room) error {
	const query = `
		INSERT INTO rooms (house_id, name, price, max_tennants, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	err := r.db.QueryRowContext(ctx, query,
		room.HouseID, room.Name, room.Price, room.MaxTennants, room.Status,
	).Scan(&room.ID, &room.CreatedAt, &room.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create room: %w", err)
	}
	return nil
}

// GetRoomByID fetches a room by its id within a specific house.
// Ownership must be verified by the caller before this.
func (r *RoomRepository) GetRoomByID(ctx context.Context, id, houseID string) (*model.Room, error) {
	const query = `
		SELECT id, house_id, name, price, max_tennants, status, created_at, updated_at
		FROM   rooms
		WHERE  id = $1 AND house_id = $2`
	var room model.Room
	err := r.db.QueryRowContext(ctx, query, id, houseID).Scan(
		&room.ID, &room.HouseID, &room.Name, &room.Price,
		&room.MaxTennants, &room.Status, &room.CreatedAt, &room.UpdatedAt,
	)
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
	const query = `
		SELECT id, house_id, name, price, max_tennants, status, created_at, updated_at
		FROM   rooms
		WHERE  house_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryContext(ctx, query, houseID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list rooms: %w", err)
	}
	defer rows.Close()

	var rooms []model.Room
	for rows.Next() {
		var room model.Room
		if err := rows.Scan(
			&room.ID, &room.HouseID, &room.Name, &room.Price,
			&room.MaxTennants, &room.Status, &room.CreatedAt, &room.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("list rooms scan: %w", err)
		}
		rooms = append(rooms, room)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list rooms rows: %w", err)
	}
	return rooms, nil
}

// UpdateRoom partially updates a room within a specific house.
// Ownership must be verified by the caller before this.
func (r *RoomRepository) UpdateRoom(ctx context.Context, id, houseID string, params model.UpdateRoomParams) (*model.Room, error) {
	var setClauses []string
	args := []any{}
	argIdx := 1

	if params.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *params.Name)
		argIdx++
	}
	if params.Price != nil {
		setClauses = append(setClauses, fmt.Sprintf("price = $%d", argIdx))
		args = append(args, *params.Price)
		argIdx++
	}
	if params.MaxTennants != nil {
		setClauses = append(setClauses, fmt.Sprintf("max_tennants = $%d", argIdx))
		args = append(args, *params.MaxTennants)
		argIdx++
	}
	if params.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *params.Status)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil, fmt.Errorf("update room: no fields to update")
	}
	setClauses = append(setClauses, "updated_at = NOW()")

	// Append WHERE args: id and houseID.
	args = append(args, id, houseID)

	query := fmt.Sprintf(`
		UPDATE rooms
		SET    %s
		WHERE  id = $%d AND house_id = $%d
		RETURNING id, house_id, name, price, max_tennants, status, created_at, updated_at`,
		strings.Join(setClauses, ", "),
		argIdx,
		argIdx+1,
	)

	var room model.Room
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&room.ID, &room.HouseID, &room.Name, &room.Price,
		&room.MaxTennants, &room.Status, &room.CreatedAt, &room.UpdatedAt,
	)
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
	const query = `DELETE FROM rooms WHERE id = $1 AND house_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, houseID)
	if err != nil {
		return fmt.Errorf("delete room: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete room rows affected: %w", err)
	}
	if n == 0 {
		return model.ErrRoomNotFound
	}
	return nil
}
