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
	filter := model.RoomListFilter{HouseID: "house-1", Search: "a_1", Status: "AVAILABLE", Limit: 10, Offset: 20}

	tests := []struct {
		name      string
		mock      func(mock sqlmock.Sqlmock)
		wantTotal int
		wantErr   bool
	}{
		{
			name: "Success",
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "house_id", "name", "tenant_count"}).
					AddRow("room-1", "house-1", "Room 1", 2)
				mock.ExpectQuery(`SELECT room\.\*, \(SELECT COUNT\(\*\) FROM tenants .* FROM "rooms" AS "room" WHERE \(room.house_id = 'house-1'\) AND \(room.name ILIKE '%a\\_1%'\) AND \(room.status = 'AVAILABLE'\) .* LIMIT 10 OFFSET 20`).
					WillReturnRows(rows)
				mock.ExpectQuery(`SELECT count\(\*\) FROM "rooms"`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(21))
			},
			wantTotal: 21,
		},
		{
			name: "DB Error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`LIMIT 10`).WillReturnError(sql.ErrConnDone)
				mock.ExpectQuery(`SELECT count\(\*\) FROM "rooms"`).WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bunDB, mock := repotest.SetupTestDB(t)
			defer bunDB.Close()
			mock.MatchExpectationsInOrder(false)
			repo := roomrepo.NewRoomRepository(bunDB)

			tt.mock(mock)
			rooms, total, err := repo.ListRoomsByHouseID(ctx, filter)

			if (err != nil) != tt.wantErr {
				t.Errorf("expected error presence %v, got %v", tt.wantErr, err)
			}
			if err == nil && (len(rooms) != 1 || rooms[0].TenantCount != 2 || total != tt.wantTotal) {
				t.Errorf("unexpected result: rooms=%+v total=%d", rooms, total)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRoomRepository_GetRoomStats(t *testing.T) {
	ctx := context.Background()

	t.Run("Success scoped to house", func(t *testing.T) {
		bunDB, mock := repotest.SetupTestDB(t)
		defer bunDB.Close()
		repo := roomrepo.NewRoomRepository(bunDB)

		rows := sqlmock.NewRows([]string{"total", "occupied", "available", "maintenance", "capacity", "tenants"}).
			AddRow(5, 3, 1, 1, 12, 7)
		mock.ExpectQuery(`SELECT COUNT\(\*\) AS total, .* FROM rooms AS room JOIN houses AS h ON h.id = room.house_id WHERE \(h.manager_id = 'manager-1'\) AND \(room.house_id = 'house-1'\)`).
			WillReturnRows(rows)

		stats, err := repo.GetRoomStats(ctx, "manager-1", "house-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := model.RoomStats{Total: 5, Occupied: 3, Available: 1, Maintenance: 1, Capacity: 12, Tenants: 7}
		if *stats != want {
			t.Errorf("expected %+v, got %+v", want, *stats)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

	t.Run("DB Error", func(t *testing.T) {
		bunDB, mock := repotest.SetupTestDB(t)
		defer bunDB.Close()
		repo := roomrepo.NewRoomRepository(bunDB)

		mock.ExpectQuery(`WHERE \(h.manager_id = 'manager-1'\)$`).WillReturnError(sql.ErrConnDone)
		if _, err := repo.GetRoomStats(ctx, "manager-1", ""); err == nil {
			t.Error("expected error, got nil")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
}

func TestRoomRepository_ListAvailableRooms(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		bunDB, mock := repotest.SetupTestDB(t)
		defer bunDB.Close()
		mock.MatchExpectationsInOrder(false)
		repo := roomrepo.NewRoomRepository(bunDB)

		rows := sqlmock.NewRows([]string{"id", "house_id", "name", "house_name"}).
			AddRow("room-1", "house-1", "Room 1", "House A")
		mock.ExpectQuery(`SELECT room\.\*, h.name AS house_name FROM "rooms" AS "room" JOIN houses AS h .* WHERE \(h.manager_id = 'manager-1'\) AND \(room.status = 'AVAILABLE'\) .* LIMIT 50`).
			WillReturnRows(rows)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "rooms"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		rooms, total, err := repo.ListAvailableRooms(ctx, "manager-1", 50)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(rooms) != 1 || rooms[0].HouseName != "House A" || total != 3 {
			t.Errorf("unexpected result: rooms=%+v total=%d", rooms, total)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})
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
					"group_chat_id", "contract_path", "created_at", "updated_at",
				}).AddRow(
					"room-1", "house-1", "Updated Room", int64(0), 0, "",
					0.0, 0.0, 0.0, 0.0, 0.0,
					0, 0.0, 0, 0.0,
					"", "", time.Now(), time.Now(),
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

func TestRoomRepository_UpdateRoomContract(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		mock    func(mock sqlmock.Sqlmock)
		wantErr error
	}{
		{
			name: "Success",
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "house_id", "contract_path"}).
					AddRow("room-1", "house-1", "/api/v1/tenant/files/a.pdf")
				mock.ExpectQuery(`UPDATE "rooms"`).WillReturnRows(rows)
			},
		},
		{
			name: "Not Found",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE "rooms"`).WillReturnError(sql.ErrNoRows)
			},
			wantErr: model.ErrRoomNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bunDB, mock := repotest.SetupTestDB(t)
			repo := roomrepo.NewRoomRepository(bunDB)
			tt.mock(mock)

			room, err := repo.UpdateRoomContract(ctx, "room-1", "house-1", "/api/v1/tenant/files/a.pdf")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if room.ContractPath != "/api/v1/tenant/files/a.pdf" {
				t.Errorf("expected contract path stored, got %q", room.ContractPath)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestRoomRepository_HasRoomWithFilePath(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		mock    func(mock sqlmock.Sqlmock)
		want    bool
		wantErr bool
	}{
		{
			name: "Owned",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS`).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
			},
			want: true,
		},
		{
			name: "Not owned",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS`).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
			},
			want: false,
		},
		{
			name: "DB Error",
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS`).WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bunDB, mock := repotest.SetupTestDB(t)
			repo := roomrepo.NewRoomRepository(bunDB)
			tt.mock(mock)

			got, err := repo.HasRoomWithFilePath(ctx, "manager-1", "/api/v1/tenant/files/a.pdf")
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
