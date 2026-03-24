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
