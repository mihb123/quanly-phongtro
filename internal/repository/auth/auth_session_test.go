package auth_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/repository/auth"
	"github.com/mihb123/quanly-phongtro/internal/repository/repotest"
)

func TestAuthSessionRepository_Create(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := auth.NewAuthSessionRepository(bunDB)
	ctx := context.Background()

	session := &model.AuthSession{
		UserID:       "user-1",
		RefreshToken: "refresh-token-123",
		IPAddress:    "127.0.0.1",
		UserAgent:    "Test-Agent",
		JKT:          "test-thumbprint",
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	rows := sqlmock.NewRows([]string{"id", "revoked", "created_at"}).
		AddRow("id-1", false, time.Now())

	mock.ExpectQuery(`INSERT INTO "auth_sessions"`).WillReturnRows(rows)

	err := repo.Create(ctx, session)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestAuthSessionRepository_FindByToken(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := auth.NewAuthSessionRepository(bunDB)
	ctx := context.Background()

	// Case: Found
	rows1 := sqlmock.NewRows([]string{"revoked", "created_at", "expires_at", "jkt"}).
		AddRow(false, time.Now(), time.Now().Add(24*time.Hour), "test-thumbprint")

	mock.ExpectQuery(`SELECT .* FROM "auth_sessions"`).WillReturnRows(rows1)

	session, err := repo.FindByToken(ctx, "refresh-token-123", "user-1")
	if err != nil || session == nil {
		t.Errorf("expected session to be found, got err: %v", err)
	}

	// Case: Not found -> false
	mock.ExpectQuery(`SELECT .* FROM "auth_sessions"`).WillReturnError(sql.ErrNoRows)

	session, err = repo.FindByToken(ctx, "not-found", "user-1")
	if err != nil || session != nil {
		t.Errorf("expected nil without error when not found, got %v, err: %v", session, err)
	}
}

func TestAuthSessionRepository_Revoke(t *testing.T) {
	bunDB, mock := repotest.SetupTestDB(t)
	defer bunDB.Close()

	repo := auth.NewAuthSessionRepository(bunDB)
	ctx := context.Background()

	mock.ExpectExec(`UPDATE "auth_sessions"`).WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Revoke(ctx, "refresh-token-123", "user-1")
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
}
