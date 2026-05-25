package model

import (
	"context"
	"time"
)

type EmailVerification struct {
	ID        string
	Email     string
	OTP       string
	IsUsed    bool      `bun:"is_used"`
	Expires   time.Time
	CreatedAt time.Time
}

type OTPCheck struct {
	ID        string
	Email     string
	OTPFails  int64     `bun:"otp_fails"`
	BlockTime time.Time `bun:"block_time"`
}

type EmailVerificationRepository interface {
	CreateOTP(ctx context.Context, emailVeri *EmailVerification) error
	GetOTP(ctx context.Context, emailVeri *EmailVerification) error
	UpdateUsedOTP(ctx context.Context, emailVeri *EmailVerification) error
}

type OTPCheckRepository interface {
	GetOTPCheck(ctx context.Context, email string) (OTPCheck, error)
	BlockOTP(ctx context.Context, email string) error
	ResetOTP(ctx context.Context, email string) error
	CreateOTPCheck(ctx context.Context, email string) error
	IncrementOTPCheck(ctx context.Context, email string, num int64) error
}
