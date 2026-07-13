package room_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/repository/repotest"
	roomrepo "github.com/mihb123/quanly-phongtro/internal/repository/room"
)

func TestRoomRepository_CreateRoom(t *testing.T) {
	ctx := context.Background()
	room := &model.Room{
		HouseID: "house-1",
		Name:    "Room 1",
	}

	tests := []struct {
		name    string
		mock    func(mock sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name: "Success",
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
					AddRow("room-1", time.Now(), time.Now())
				mock.ExpectQuery(`INSERT INTO "rooms"`).WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "DB Error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO "rooms"`).WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := roomrepo.NewRoomRepository(bunDB)

			tt.mock(mock)
			err := repo.CreateRoom(ctx, room)
			if (err != nil) != tt.wantErr {
				t.Errorf("expected error presence: %v, got: %v", tt.wantErr, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRoomRepository_GetRoomByID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		mock    func(mock sqlmock.Sqlmock)
		wantErr error
	}{
		{
			name: "Success",
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "house_id", "name"}).
					AddRow("room-1", "house-1", "Room 1")
				mock.ExpectQuery(`SELECT .* FROM "rooms"`).WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name: "Not Found",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "rooms"`).WillReturnError(sql.ErrNoRows)
			},
			wantErr: model.ErrRoomNotFound,
		},
		{
			name: "DB Error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "rooms"`).WillReturnError(sql.ErrConnDone)
			},
			wantErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := roomrepo.NewRoomRepository(bunDB)

			tt.mock(mock)
			room, err := repo.GetRoomByID(ctx, "room-1", "house-1")

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				if room != nil {
					t.Errorf("expected nil room, got %v", room)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if room == nil || room.ID != "room-1" {
					t.Errorf("expected room-1, got %v", room)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRoomRepository_ListRoomsByHouseID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		mock    func(mock sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name: "Success",
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "house_id", "name"}).
					AddRow("room-1", "house-1", "Room 1")
				mock.ExpectQuery(`SELECT .* FROM "rooms"`).WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "DB Error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "rooms"`).WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := roomrepo.NewRoomRepository(bunDB)

			tt.mock(mock)
			rooms, err := repo.ListRoomsByHouseID(ctx, "house-1", 10, 0)

			if (err != nil) != tt.wantErr {
				t.Errorf("expected error presence %v, got %v", tt.wantErr, err)
			}
			if err == nil && len(rooms) != 1 {
				t.Errorf("expected 1 room, got %d", len(rooms))
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRoomRepository_ListAllRoomsByHouseID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		mock    func(mock sqlmock.Sqlmock)
		wantErr bool
	}{
		{
			name: "Success",
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "house_id", "name"}).
					AddRow("room-1", "house-1", "Room 1")
				mock.ExpectQuery(`SELECT .* FROM "rooms"`).WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "DB Error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "rooms"`).WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := roomrepo.NewRoomRepository(bunDB)

			tt.mock(mock)
			rooms, err := repo.ListAllRoomsByHouseID(ctx, "house-1")

			if (err != nil) != tt.wantErr {
				t.Errorf("expected error presence %v, got %v", tt.wantErr, err)
			}
			if err == nil && len(rooms) != 1 {
				t.Errorf("expected 1 room, got %d", len(rooms))
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRoomRepository_UpdateRoom(t *testing.T) {
	ctx := context.Background()
	params := model.UpdateRoomParams{
		Name: "Updated Room",
	}

	tests := []struct {
		name    string
		mock    func(mock sqlmock.Sqlmock)
		wantErr error
	}{
		{
			name: "Success",
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "house_id", "name", "price", "max_tenants", "status",
					"electricity_price", "water_price", "wifi_price", "parking_price", "service_price",
					"extra_person_threshold", "extra_person_fee", "extra_vehicle_threshold", "extra_vehicle_fee",
					"group_chat_id", "created_at", "updated_at",
				}).AddRow(
					"room-1", "house-1", "Updated Room", int64(0), 0, "",
					0.0, 0.0, 0.0, 0.0, 0.0,
					0, 0.0, 0, 0.0,
					"", time.Now(), time.Now(),
				)
				mock.ExpectQuery(`UPDATE "rooms"`).WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name: "Not Found",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE "rooms"`).WillReturnError(sql.ErrNoRows)
			},
			wantErr: model.ErrRoomNotFound,
		},
		{
			name: "DB Error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE "rooms"`).WillReturnError(sql.ErrConnDone)
			},
			wantErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := roomrepo.NewRoomRepository(bunDB)

			tt.mock(mock)
			room, err := repo.UpdateRoom(ctx, "room-1", "house-1", params)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				if room != nil {
					t.Errorf("expected nil room, got %v", room)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if room == nil || room.Name != "Updated Room" {
					t.Errorf("expected updated room, got %v", room)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRoomRepository_DeleteRoom(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		mock    func(mock sqlmock.Sqlmock)
		wantErr error
	}{
		{
			name: "Success",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM "rooms"`).WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: nil,
		},
		{
			name: "Not Found",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM "rooms"`).WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: model.ErrRoomNotFound,
		},
		{
			name: "DB Error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM "rooms"`).WillReturnError(sql.ErrConnDone)
			},
			wantErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := roomrepo.NewRoomRepository(bunDB)

			tt.mock(mock)
			err := repo.DeleteRoom(ctx, "room-1", "house-1")

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRoomRepository_GetMaxTenants(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		mock    func(mock sqlmock.Sqlmock)
		wantErr error
	}{
		{
			name: "Success",
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"max_tenants"}).AddRow(4)
				mock.ExpectQuery(`SELECT .* FROM "rooms"`).WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name: "Not Found",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "rooms"`).WillReturnError(sql.ErrNoRows)
			},
			wantErr: model.ErrNotFound,
		},
		{
			name: "DB Error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "rooms"`).WillReturnError(sql.ErrConnDone)
			},
			wantErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := roomrepo.NewRoomRepository(bunDB)

			tt.mock(mock)
			max, err := repo.GetMaxTenants(ctx, "room-1")

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if max != 0 {
					t.Errorf("expected 0 max tenants, got %v", max)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if max != 4 {
					t.Errorf("expected 4, got %v", max)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRoomRepository_UpdateRoomStatus(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		mock    func(mock sqlmock.Sqlmock)
		wantErr error
	}{
		{
			name: "Success",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE "rooms"`).WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: nil,
		},
		{
			name: "Not Found",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE "rooms"`).WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: model.ErrNotFound,
		},
		{
			name: "DB Error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE "rooms"`).WillReturnError(sql.ErrConnDone)
			},
			wantErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := roomrepo.NewRoomRepository(bunDB)

			tt.mock(mock)
			err := repo.UpdateRoomStatus(ctx, "room-1", "OCCUPIED")

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRoomRepository_GetRoomByIDOnly(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		mock    func(mock sqlmock.Sqlmock)
		wantErr error
	}{
		{
			name: "Success",
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name"}).AddRow("room-1", "Room 1")
				mock.ExpectQuery(`SELECT .* FROM "rooms"`).WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name: "Not Found",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "rooms"`).WillReturnError(sql.ErrNoRows)
			},
			wantErr: model.ErrRoomNotFound,
		},
		{
			name: "DB Error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "rooms"`).WillReturnError(sql.ErrConnDone)
			},
			wantErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := roomrepo.NewRoomRepository(bunDB)

			tt.mock(mock)
			room, err := repo.GetRoomByIDOnly(ctx, "room-1")

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				if room != nil {
					t.Errorf("expected nil room, got %v", room)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if room == nil || room.ID != "room-1" {
					t.Errorf("expected room-1, got %v", room)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRoomRepository_GetRoomByGroupChatID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		mock    func(mock sqlmock.Sqlmock)
		wantErr error
	}{
		{
			name: "Success",
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "group_chat_id"}).AddRow("room-1", "group-1")
				mock.ExpectQuery(`SELECT .* FROM "rooms"`).WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name: "Not Found",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "rooms"`).WillReturnError(sql.ErrNoRows)
			},
			wantErr: model.ErrRoomNotFound,
		},
		{
			name: "DB Error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "rooms"`).WillReturnError(sql.ErrConnDone)
			},
			wantErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			repo := roomrepo.NewRoomRepository(bunDB)

			tt.mock(mock)
			room, err := repo.GetRoomByGroupChatID(ctx, "group-1")

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				if room != nil {
					t.Errorf("expected nil room, got %v", room)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if room == nil || room.ID != "room-1" {
					t.Errorf("expected room-1, got %v", room)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
