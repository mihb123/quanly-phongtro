package model

import (
	"context"
	"time"
)

type EmailVerification struct {
	ID        string
	Email     string
	OTP       string
	IsUsed    bool
	Expires   time.Time
	CreatedAt time.Time
}

type EmailVerificationRepository interface {
	CreateOTP(ctx context.Context, emailVeri *EmailVerification) error
	GetOTP(ctx context.Context, emailVeri *EmailVerification) error
	UpdateUsedOTP(ctx context.Context, emailVeri *EmailVerification) error
}
