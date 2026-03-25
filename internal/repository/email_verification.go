package repository

import (
	"context"
	"database/sql"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

type EmailVerificationRepository struct {
	db *sql.DB
}

func NewEmailVerificationRepository(db *sql.DB) *EmailVerificationRepository {
	return &EmailVerificationRepository{
		db: db,
	}
}

func (r *EmailVerificationRepository) CreateOTP(ctx context.Context, emailVeri *model.EmailVerification) error {
	query := `INSERT INTO email_verifications (email, otp, expires) VALUES ($1, $2, $3)`
	row := r.db.QueryRowContext(ctx, query, emailVeri.Email, emailVeri.OTP, emailVeri.Expires)
	return row.Err()
}

func (r *EmailVerificationRepository) GetOTP(ctx context.Context, emailVeri *model.EmailVerification) error {
	query := `SELECT is_used, expires FROM email_verifications where otp = $1 and email = $2`
	return r.db.QueryRowContext(ctx, query, emailVeri.OTP, emailVeri.Email).Scan(&emailVeri.IsUsed, &emailVeri.Expires)
}

func (r *EmailVerificationRepository) UpdateUsedOTP(ctx context.Context, emailVeri *model.EmailVerification) error {
	query := `UPDATE email_verifications SET is_used = true where otp = $1 and email = $2`
	return r.db.QueryRowContext(ctx, query, emailVeri.OTP, emailVeri.Email).Err()
}

type OTPCheckRepository struct {
	db *sql.DB
}

func NewOTPCheckRepository(db *sql.DB) *OTPCheckRepository {
	return &OTPCheckRepository{
		db: db,
	}
}

func (r *OTPCheckRepository) GetOTPCheck(ctx context.Context, email string) (model.OTPCheck, error) {
	otpCheck := model.OTPCheck{}
	query := `select email, otp_fails, block_time from otp_checks`
	err := r.db.QueryRowContext(ctx, query).Scan(&otpCheck.Email, &otpCheck.OTPFails, &otpCheck.BlockTime)
	return otpCheck, err
}

func (r *OTPCheckRepository) BlockOTP(ctx context.Context, email string) error {
	query := `UPDATE otp_checks SET block_time = NOW() where email = $1`
	row := r.db.QueryRowContext(ctx, query, email)

	return row.Err()
}

func (r *OTPCheckRepository) ResetOTP(ctx context.Context, email string) error {
	query := `UPDATE otp_checks SET otp_fails = 0 where email = $1`
	row := r.db.QueryRowContext(ctx, query, email)

	return row.Err()
}

func (r *OTPCheckRepository) CreateOTPCheck(ctx context.Context, email string) error {
	query := `INSERT INTO otp_checks (email, otp_fails, block_time) VALUES ($1, 1, NOW())`
	row := r.db.QueryRowContext(ctx, query, email)
	return row.Err()
}

func (r *OTPCheckRepository) IncrementOTPCheck(ctx context.Context, email string, num int64) error {
	query := ""
	if num == 5 {
		query = `UPDATE otp_checks SET otp_fails = $1, block_time = NOW() where email = $2`
	} else {
		query = `UPDATE otp_checks SET otp_fails = $1 where email = $2`
	}

	row := r.db.QueryRowContext(ctx, query, num, email)
	return row.Err()
}
