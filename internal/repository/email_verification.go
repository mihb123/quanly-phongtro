package repository

import (
	"context"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/uptrace/bun"
)

type EmailVerificationRepository struct {
	db *bun.DB
}

func NewEmailVerificationRepository(db *bun.DB) *EmailVerificationRepository {
	return &EmailVerificationRepository{
		db: db,
	}
}

func (r *EmailVerificationRepository) CreateOTP(ctx context.Context, emailVeri *model.EmailVerification) error {
	_, err := r.db.NewInsert().
		Model(emailVeri).
		Column("email", "otp", "expires").
		Exec(ctx)
	return err
}

func (r *EmailVerificationRepository) GetOTP(ctx context.Context, emailVeri *model.EmailVerification) error {
	return r.db.NewSelect().
		Model(emailVeri).
		Column("is_used", "expires").
		Where("otp = ? and email = ?", emailVeri.OTP, emailVeri.Email).
		Scan(ctx)
}

func (r *EmailVerificationRepository) UpdateUsedOTP(ctx context.Context, emailVeri *model.EmailVerification) error {
	_, err := r.db.NewUpdate().
		Model(emailVeri).
		Set("is_used = ?", true).
		Where("otp = ? and email = ?", emailVeri.OTP, emailVeri.Email).
		Exec(ctx)
	return err
}

type OTPCheckRepository struct {
	db *bun.DB
}

func NewOTPCheckRepository(db *bun.DB) *OTPCheckRepository {
	return &OTPCheckRepository{
		db: db,
	}
}

func (r *OTPCheckRepository) GetOTPCheck(ctx context.Context, email string) (model.OTPCheck, error) {
	otpCheck := model.OTPCheck{}
	err := r.db.NewSelect().
		Model(&otpCheck).
		Column("email", "otp_fails", "block_time").
		Where("email = ?", email). // Note: original code missed WHERE clause which would return random row
		Scan(ctx)
	return otpCheck, err
}

func (r *OTPCheckRepository) BlockOTP(ctx context.Context, email string) error {
	_, err := r.db.NewUpdate().
		Model((*model.OTPCheck)(nil)).
		Set("block_time = NOW()").
		Where("email = ?", email).
		Exec(ctx)
	return err
}

func (r *OTPCheckRepository) ResetOTP(ctx context.Context, email string) error {
	_, err := r.db.NewUpdate().
		Model((*model.OTPCheck)(nil)).
		Set("otp_fails = 0").
		Where("email = ?", email).
		Exec(ctx)
	return err
}

func (r *OTPCheckRepository) CreateOTPCheck(ctx context.Context, email string) error {
	_, err := r.db.NewInsert().
		Model((*model.OTPCheck)(nil)).
		Value("email", "?", email).
		Value("otp_fails", "?", 1).
		Value("block_time", "NOW()").
		Exec(ctx)
	return err
}

func (r *OTPCheckRepository) IncrementOTPCheck(ctx context.Context, email string, num int64) error {
	q := r.db.NewUpdate().
		Model((*model.OTPCheck)(nil)).
		Set("otp_fails = ?", num).
		Where("email = ?", email)

	if num == 5 {
		q = q.Set("block_time = NOW()")
	}
	_, err := q.Exec(ctx)
	return err
}
