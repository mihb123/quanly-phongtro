package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/repository"
)

var userColumns = []string{
	"id", "email", "role", "full_name", "phone", "is_activated",
	"zalo_bot_token", "is_zalo_bot_active", "zalo_user_id", "created_at", "updated_at",
}

func newUserRow(id string) *sqlmock.Rows {
	return sqlmock.NewRows(userColumns).AddRow(
		id, "test@test.com", "manager", "Test User", "123456", true,
		"", false, "", time.Now(), time.Now(),
	)
}

func ptr[T any](v T) *T {
	return &v
}

func TestUserRepository_GetByEmail(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		mockSetup func(mock sqlmock.Sqlmock)
		check     func(t *testing.T, user *model.User, err error)
	}{
		{
			name:  "Happy path",
			email: "test@test.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(newUserRow("user-1"))
			},
			check: func(t *testing.T, user *model.User, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if user.ID != "user-1" {
					t.Errorf("expected user-1, got %s", user.ID)
				}
			},
		},
		{
			name:  "Not found",
			email: "notfound@test.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnError(sql.ErrNoRows)
			},
			check: func(t *testing.T, user *model.User, err error) {
				if !errors.Is(err, model.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:  "Query error",
			email: "error@test.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnError(errors.New("db error"))
			},
			check: func(t *testing.T, user *model.User, err error) {
				if err == nil || errors.Is(err, model.ErrNotFound) {
					t.Errorf("expected db error, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			defer db.Close()
			tt.mockSetup(mock)
			repo := repository.NewUserRepository(db)
			user, err := repo.GetByEmail(context.Background(), tt.email)
			tt.check(t, user, err)
		})
	}
}

func TestUserRepository_GetAuthUserByEmail(t *testing.T) {
	authCols := []string{
		"id", "email", "password_hash", "role", "full_name", "phone", "is_activated",
		"zalo_bot_token", "is_zalo_bot_active", "zalo_user_id", "created_at", "updated_at",
	}

	tests := []struct {
		name      string
		email     string
		mockSetup func(mock sqlmock.Sqlmock)
		check     func(t *testing.T, user *model.User, err error)
	}{
		{
			name:  "Happy path",
			email: "test@test.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(authCols).AddRow(
					"user-1", "test@test.com", "hashed", "manager", "Test User", "123456", true,
					"", false, "", time.Now(), time.Now(),
				)
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(rows)
			},
			check: func(t *testing.T, user *model.User, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if user.PasswordHash != "hashed" {
					t.Errorf("expected hashed, got %s", user.PasswordHash)
				}
			},
		},
		{
			name:  "Not found",
			email: "notfound@test.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnError(sql.ErrNoRows)
			},
			check: func(t *testing.T, user *model.User, err error) {
				if !errors.Is(err, model.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:  "Query error",
			email: "error@test.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnError(errors.New("db error"))
			},
			check: func(t *testing.T, user *model.User, err error) {
				if err == nil || errors.Is(err, model.ErrNotFound) {
					t.Errorf("expected db error, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			defer db.Close()
			tt.mockSetup(mock)
			repo := repository.NewUserRepository(db)
			user, err := repo.GetAuthUserByEmail(context.Background(), tt.email)
			tt.check(t, user, err)
		})
	}
}

func TestUserRepository_GetByUserID(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		mockSetup func(mock sqlmock.Sqlmock)
		check     func(t *testing.T, user *model.User, err error)
	}{
		{
			name:   "Happy path",
			userID: "user-1",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(newUserRow("user-1"))
			},
			check: func(t *testing.T, user *model.User, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if user.ID != "user-1" {
					t.Errorf("expected user-1, got %s", user.ID)
				}
			},
		},
		{
			name:   "Not found",
			userID: "notfound",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnError(sql.ErrNoRows)
			},
			check: func(t *testing.T, user *model.User, err error) {
				if !errors.Is(err, model.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:   "Query error",
			userID: "error",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnError(errors.New("db error"))
			},
			check: func(t *testing.T, user *model.User, err error) {
				if err == nil || errors.Is(err, model.ErrNotFound) {
					t.Errorf("expected db error, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			defer db.Close()
			tt.mockSetup(mock)
			repo := repository.NewUserRepository(db)
			user, err := repo.GetByUserID(context.Background(), tt.userID)
			tt.check(t, user, err)
		})
	}
}

func TestUserRepository_GetByPhone(t *testing.T) {
	tests := []struct {
		name      string
		phone     string
		mockSetup func(mock sqlmock.Sqlmock)
		check     func(t *testing.T, user *model.User, err error)
	}{
		{
			name:  "Happy path",
			phone: "123456",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(newUserRow("user-1"))
			},
			check: func(t *testing.T, user *model.User, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if user.Phone != "123456" {
					t.Errorf("expected 123456, got %s", user.Phone)
				}
			},
		},
		{
			name:  "Not found",
			phone: "0000",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnError(sql.ErrNoRows)
			},
			check: func(t *testing.T, user *model.User, err error) {
				if !errors.Is(err, model.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:  "Query error",
			phone: "error",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnError(errors.New("db error"))
			},
			check: func(t *testing.T, user *model.User, err error) {
				if err == nil || errors.Is(err, model.ErrNotFound) {
					t.Errorf("expected db error, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			defer db.Close()
			tt.mockSetup(mock)
			repo := repository.NewUserRepository(db)
			user, err := repo.GetByPhone(context.Background(), tt.phone)
			tt.check(t, user, err)
		})
	}
}

func TestUserRepository_GetByZaloUserID(t *testing.T) {
	tests := []struct {
		name       string
		zaloUserID string
		mockSetup  func(mock sqlmock.Sqlmock)
		check      func(t *testing.T, user *model.User, err error)
	}{
		{
			name:       "Happy path",
			zaloUserID: "zalo-1",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(userColumns).AddRow(
					"user-1", "test@test.com", "manager", "Test User", "123456", true,
					"", false, "zalo-1", time.Now(), time.Now(),
				)
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(rows)
			},
			check: func(t *testing.T, user *model.User, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if user.ZaloUserID == nil || *user.ZaloUserID != "zalo-1" {
					t.Errorf("expected zalo-1")
				}
			},
		},
		{
			name:       "Not found",
			zaloUserID: "notfound",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnError(sql.ErrNoRows)
			},
			check: func(t *testing.T, user *model.User, err error) {
				if !errors.Is(err, model.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:       "Query error",
			zaloUserID: "error",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnError(errors.New("db error"))
			},
			check: func(t *testing.T, user *model.User, err error) {
				if err == nil || errors.Is(err, model.ErrNotFound) {
					t.Errorf("expected db error, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			defer db.Close()
			tt.mockSetup(mock)
			repo := repository.NewUserRepository(db)
			user, err := repo.GetByZaloUserID(context.Background(), tt.zaloUserID)
			tt.check(t, user, err)
		})
	}
}

func TestUserRepository_Create(t *testing.T) {
	tests := []struct {
		name      string
		user      *model.User
		mockSetup func(mock sqlmock.Sqlmock)
		check     func(t *testing.T, err error)
	}{
		{
			name: "Happy path",
			user: &model.User{Email: "test@test.com", FullName: "Test"},
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
					AddRow("user-1", time.Now(), time.Now())
				mock.ExpectQuery(`INSERT INTO "users"`).WillReturnRows(rows)
			},
			check: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			},
		},
		{
			name: "Already exists",
			user: &model.User{Email: "exists@test.com", FullName: "Test"},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO "users"`).WillReturnError(&pq.Error{
					Code:       "23505",
					Constraint: "users_email",
				})
			},
			check: func(t *testing.T, err error) {
				if !errors.Is(err, model.ErrAlreadyExists) {
					t.Errorf("expected ErrAlreadyExists, got %v", err)
				}
			},
		},
		{
			name: "Query error",
			user: &model.User{Email: "error@test.com", FullName: "Test"},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO "users"`).WillReturnError(errors.New("db error"))
			},
			check: func(t *testing.T, err error) {
				if err == nil || errors.Is(err, model.ErrAlreadyExists) {
					t.Errorf("expected query error, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			defer db.Close()
			tt.mockSetup(mock)
			repo := repository.NewUserRepository(db)
			err := repo.Create(context.Background(), tt.user)
			tt.check(t, err)
		})
	}
}

func TestUserRepository_UpdateUser(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		input     model.UpdateUserInput
		mockSetup func(mock sqlmock.Sqlmock)
		check     func(t *testing.T, user *model.User, err error)
	}{
		{
			name:   "Happy path - update full name",
			userID: "user-1",
			input: model.UpdateUserInput{
				FullName: ptr("Updated Name"),
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(userColumns).AddRow(
					"user-1", "test@test.com", "manager", "Updated Name", "123456", true,
					"", false, "", time.Now(), time.Now(),
				)
				mock.ExpectQuery(`UPDATE "users"`).WillReturnRows(rows)
			},
			check: func(t *testing.T, user *model.User, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if user.FullName != "Updated Name" {
					t.Errorf("expected Updated Name, got %s", user.FullName)
				}
			},
		},
		{
			name:   "No fields to update - returns existing",
			userID: "user-1",
			input:  model.UpdateUserInput{},
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(userColumns).AddRow(
					"user-1", "test@test.com", "manager", "Old Name", "123456", true,
					"", false, "", time.Now(), time.Now(),
				)
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(rows)
			},
			check: func(t *testing.T, user *model.User, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if user.FullName != "Old Name" {
					t.Errorf("expected Old Name, got %s", user.FullName)
				}
			},
		},
		{
			name:   "Update all fields",
			userID: "user-1",
			input: model.UpdateUserInput{
				FullName:        ptr("Name"),
				Phone:           ptr("123"),
				Email:           ptr("a@a.com"),
				ZaloBotToken:    ptr("token"),
				IsZaloBotActive: ptr(true),
				ZaloUserID:      ptr("zalo"),
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(userColumns).AddRow(
					"user-1", "a@a.com", "manager", "Name", "123", true,
					"token", true, "zalo", time.Now(), time.Now(),
				)
				mock.ExpectQuery(`UPDATE "users"`).WillReturnRows(rows)
			},
			check: func(t *testing.T, user *model.User, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if user.FullName != "Name" {
					t.Errorf("expected Name, got %s", user.FullName)
				}
			},
		},
		{
			name:   "Not found during update",
			userID: "user-1",
			input: model.UpdateUserInput{
				FullName: ptr("Name"),
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE "users"`).WillReturnError(sql.ErrNoRows)
			},
			check: func(t *testing.T, user *model.User, err error) {
				if !errors.Is(err, model.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:   "Query error during update",
			userID: "user-1",
			input: model.UpdateUserInput{
				FullName: ptr("Name"),
			},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE "users"`).WillReturnError(errors.New("db error"))
			},
			check: func(t *testing.T, user *model.User, err error) {
				if err == nil || errors.Is(err, model.ErrNotFound) {
					t.Errorf("expected db error, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			defer db.Close()
			tt.mockSetup(mock)
			repo := repository.NewUserRepository(db)
			user, err := repo.UpdateUser(context.Background(), tt.userID, tt.input)
			tt.check(t, user, err)
		})
	}
}

func TestUserRepository_ActivateUser(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		mockSetup func(mock sqlmock.Sqlmock)
		check     func(t *testing.T, err error)
	}{
		{
			name:  "Happy path",
			email: "test@test.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE "users"`).WillReturnResult(sqlmock.NewResult(0, 1))
			},
			check: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			},
		},
		{
			name:  "Query error",
			email: "error@test.com",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE "users"`).WillReturnError(errors.New("db error"))
			},
			check: func(t *testing.T, err error) {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			defer db.Close()
			tt.mockSetup(mock)
			repo := repository.NewUserRepository(db)
			err := repo.ActivateUser(context.Background(), tt.email)
			tt.check(t, err)
		})
	}
}

func TestUserRepository_DeactivateUser(t *testing.T) {
	tests := []struct {
		name      string
		userID    string
		mockSetup func(mock sqlmock.Sqlmock)
		check     func(t *testing.T, err error)
	}{
		{
			name:   "Happy path",
			userID: "user-1",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE "users"`).WillReturnResult(sqlmock.NewResult(0, 1))
			},
			check: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			},
		},
		{
			name:   "Not found",
			userID: "user-1",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE "users"`).WillReturnResult(sqlmock.NewResult(0, 0))
			},
			check: func(t *testing.T, err error) {
				if !errors.Is(err, model.ErrNotFound) {
					t.Errorf("expected ErrNotFound, got %v", err)
				}
			},
		},
		{
			name:   "Query error",
			userID: "user-1",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE "users"`).WillReturnError(errors.New("db error"))
			},
			check: func(t *testing.T, err error) {
				if err == nil || errors.Is(err, model.ErrNotFound) {
					t.Errorf("expected db error, got %v", err)
				}
			},
		},
		{
			name:   "Rows affected error",
			userID: "user-1",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE "users"`).WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			check: func(t *testing.T, err error) {
				if err == nil || errors.Is(err, model.ErrNotFound) {
					t.Errorf("expected error, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			defer db.Close()
			tt.mockSetup(mock)
			repo := repository.NewUserRepository(db)
			err := repo.DeactivateUser(context.Background(), tt.userID)
			tt.check(t, err)
		})
	}
}

func TestUserRepository_GetAllUsersWithZaloToken(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(mock sqlmock.Sqlmock)
		check     func(t *testing.T, users []model.User, err error)
	}{
		{
			name: "Happy path",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "zalo_bot_token", "is_zalo_bot_active"}).
					AddRow("user-1", "test@test.com", "token-1", true).
					AddRow("user-2", "test2@test.com", "token-2", false)
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(rows)
			},
			check: func(t *testing.T, users []model.User, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(users) != 2 {
					t.Errorf("expected 2 users, got %d", len(users))
				}
			},
		},
		{
			name: "Query error",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnError(errors.New("db error"))
			},
			check: func(t *testing.T, users []model.User, err error) {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			},
		},
		{
			name: "Row scan error",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email", "zalo_bot_token", "is_zalo_bot_active"}).
					AddRow("user-1", "test@test.com", "token-1", true).
					RowError(0, errors.New("row error"))
				mock.ExpectQuery(`SELECT .* FROM "users"`).WillReturnRows(rows)
			},
			check: func(t *testing.T, users []model.User, err error) {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupTestDB(t)
			defer db.Close()
			tt.mockSetup(mock)
			repo := repository.NewUserRepository(db)
			users, err := repo.GetAllUsersWithZaloToken(context.Background())
			tt.check(t, users, err)
		})
	}
}
