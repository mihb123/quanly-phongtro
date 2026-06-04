package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/repository"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func setupTestDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	bunDB := bun.NewDB(db, pgdialect.New())
	return bunDB, mock
}

// Tests for EmailVerificationRepository

func TestEmailVerificationRepository_CreateOTP(t *testing.T) {
	bunDB, mock := setupTestDB(t)
	defer bunDB.Close()

	repo := repository.NewEmailVerificationRepository(bunDB)
	ctx := context.Background()
	
	emailVeri := &model.EmailVerification{
		Email:   "test@example.com",
		OTP:     "123456",
		Expires: time.Now().Add(5 * time.Minute),
	}

	mock.ExpectExec(`INSERT INTO "email_verifications"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateOTP(ctx, emailVeri)
	if err != nil {
		t.Errorf("error was not expected while inserting otp: %s", err)
	}
	
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestEmailVerificationRepository_GetOTP(t *testing.T) {
	bunDB, mock := setupTestDB(t)
	defer bunDB.Close()

	repo := repository.NewEmailVerificationRepository(bunDB)
	ctx := context.Background()

	emailVeri := &model.EmailVerification{
		Email: "test@example.com",
		OTP:   "123456",
	}

	rows := sqlmock.NewRows([]string{"is_used", "expires"}).
		AddRow(false, time.Now().Add(5*time.Minute))

	mock.ExpectQuery(`SELECT .* FROM "email_verifications"`).
		WillReturnRows(rows)

	err := repo.GetOTP(ctx, emailVeri)
	if err != nil {
		t.Errorf("error was not expected while getting otp: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestEmailVerificationRepository_UpdateUsedOTP(t *testing.T) {
	bunDB, mock := setupTestDB(t)
	defer bunDB.Close()

	repo := repository.NewEmailVerificationRepository(bunDB)
	ctx := context.Background()

	emailVeri := &model.EmailVerification{
		Email: "test@example.com",
		OTP:   "123456",
	}

	mock.ExpectExec(`UPDATE "email_verifications"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdateUsedOTP(ctx, emailVeri)
	if err != nil {
		t.Errorf("error was not expected while updating otp: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

// Tests for OTPCheckRepository

func TestOTPCheckRepository_GetOTPCheck(t *testing.T) {
	bunDB, mock := setupTestDB(t)
	defer bunDB.Close()

	repo := repository.NewOTPCheckRepository(bunDB)
	ctx := context.Background()

	email := "test@example.com"
	rows := sqlmock.NewRows([]string{"email", "otp_fails", "block_time"}).
		AddRow(email, 2, time.Now())

	mock.ExpectQuery(`SELECT .* FROM "otp_checks"`).
		WillReturnRows(rows)

	check, err := repo.GetOTPCheck(ctx, email)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}
	if check.Email != email || check.OTPFails != 2 {
		t.Errorf("unexpected scan result")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestOTPCheckRepository_BlockOTP(t *testing.T) {
	bunDB, mock := setupTestDB(t)
	defer bunDB.Close()

	repo := repository.NewOTPCheckRepository(bunDB)
	ctx := context.Background()

	email := "test@example.com"

	mock.ExpectExec(`UPDATE "otp_checks"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.BlockOTP(ctx, email)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestOTPCheckRepository_ResetOTP(t *testing.T) {
	bunDB, mock := setupTestDB(t)
	defer bunDB.Close()

	repo := repository.NewOTPCheckRepository(bunDB)
	ctx := context.Background()

	email := "test@example.com"

	mock.ExpectExec(`UPDATE "otp_checks"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.ResetOTP(ctx, email)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestOTPCheckRepository_CreateOTPCheck(t *testing.T) {
	bunDB, mock := setupTestDB(t)
	defer bunDB.Close()

	repo := repository.NewOTPCheckRepository(bunDB)
	ctx := context.Background()

	email := "test@example.com"

	rows := sqlmock.NewRows([]string{"email", "otp_fails", "block_time"}).
		AddRow(email, 1, time.Now())

	mock.ExpectQuery(`INSERT INTO "otp_checks"`).
		WillReturnRows(rows)

	err := repo.CreateOTPCheck(ctx, email)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestOTPCheckRepository_IncrementOTPCheck(t *testing.T) {
	bunDB, mock := setupTestDB(t)
	defer bunDB.Close()

	repo := repository.NewOTPCheckRepository(bunDB)
	ctx := context.Background()

	email := "test@example.com"

	// Case 1: num < 5
	mock.ExpectExec(`UPDATE "otp_checks"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.IncrementOTPCheck(ctx, email, 3)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	// Case 2: num == 5 (blocks)
	mock.ExpectExec(`UPDATE "otp_checks"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.IncrementOTPCheck(ctx, email, 5)
	if err != nil {
		t.Errorf("error was not expected: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
