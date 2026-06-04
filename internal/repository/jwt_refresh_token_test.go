package repository_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/repository"
)

func TestJWTRefreshTokenRepository_Create(t *testing.T) {
	bunDB, mock := setupTestDB(t)
	defer bunDB.Close()

	repo := repository.NewJWTRefreshTokenRepository(bunDB)
	ctx := context.Background()

	token := &model.JWTRefreshToken{
		UserID:       "user-1",
		RefreshToken: "refresh-token-123",
	}

	rows := sqlmock.NewRows([]string{"id", "revoked", "created_at"}).
		AddRow("id-1", false, time.Now())

	mock.ExpectQuery(`INSERT INTO "jwt_refresh_tokens"`).WillReturnRows(rows)

	err := repo.Create(ctx, token)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestJWTRefreshTokenRepository_FindByToken(t *testing.T) {
	bunDB, mock := setupTestDB(t)
	defer bunDB.Close()

	repo := repository.NewJWTRefreshTokenRepository(bunDB)
	ctx := context.Background()

	// Case: Not revoked, not expired -> false (not revoked/expired)
	rows1 := sqlmock.NewRows([]string{"revoked", "created_at"}).
		AddRow(false, time.Now())

	mock.ExpectQuery(`SELECT .* FROM "jwt_refresh_tokens"`).WillReturnRows(rows1)

	revoked, err := repo.FindByToken(ctx, "refresh-token-123", "user-1")
	if err != nil || revoked {
		t.Errorf("expected token to be valid (false), got %v, err: %v", revoked, err)
	}

	// Case: Revoked -> true
	rows2 := sqlmock.NewRows([]string{"revoked", "created_at"}).
		AddRow(true, time.Now())
	
	mock.ExpectQuery(`SELECT .* FROM "jwt_refresh_tokens"`).WillReturnRows(rows2)
	
	revoked, err = repo.FindByToken(ctx, "refresh-token-123", "user-1")
	if err != nil || !revoked {
		t.Errorf("expected token to be revoked (true), got %v, err: %v", revoked, err)
	}

	// Case: Expired -> true
	rows3 := sqlmock.NewRows([]string{"revoked", "created_at"}).
		AddRow(false, time.Now().Add(-31*24*time.Hour)) // 31 days ago

	mock.ExpectQuery(`SELECT .* FROM "jwt_refresh_tokens"`).WillReturnRows(rows3)

	revoked, err = repo.FindByToken(ctx, "refresh-token-123", "user-1")
	if err != nil || !revoked {
		t.Errorf("expected token to be expired (true), got %v, err: %v", revoked, err)
	}

	// Case: Not found -> false
	mock.ExpectQuery(`SELECT .* FROM "jwt_refresh_tokens"`).WillReturnError(sql.ErrNoRows)

	revoked, err = repo.FindByToken(ctx, "not-found", "user-1")
	if err != nil || revoked {
		t.Errorf("expected false without error when not found, got %v, err: %v", revoked, err)
	}
}

func TestJWTRefreshTokenRepository_Revoke(t *testing.T) {
	bunDB, mock := setupTestDB(t)
	defer bunDB.Close()

	repo := repository.NewJWTRefreshTokenRepository(bunDB)
	ctx := context.Background()

	mock.ExpectExec(`UPDATE "jwt_refresh_tokens"`).WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Revoke(ctx, "refresh-token-123", "user-1")
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
}
