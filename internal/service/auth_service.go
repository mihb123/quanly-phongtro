package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type TokenProvider interface {
	GenerateAccessToken(userID string) (string, error)
	GenerateRefreshToken(ctx context.Context, userID string) (string, error)
	RevokeRefreshToken(ctx context.Context, token string, userID string) error
	FindByToken(ctx context.Context, token string, userID string) (bool, error)
	GetAccessTokenTTL() time.Duration
	Parse(tokenString string, tokenType string) (*security.Claims, error)
}

type EmailSender interface {
	SendEmail(toEmail string, otpCode string, expiresIn time.Duration) error
}

type AuthServiceImpl struct {
	users             model.UserRepository
	hasher            PasswordHasher
	tokens            TokenProvider
	emailVerification model.EmailVerificationRepository
	emailSender       EmailSender
	OTPExpiresIn      time.Duration
}

type AuthService interface {
	Register(ctx context.Context, in RegisterInput) (*AuthOutput, error)
	Login(ctx context.Context, in LoginInput) (*LoginOutput, error)
	RefreshToken(ctx context.Context, refreshToken string) (*LoginOutput, error)
	CreateOTP(ctx context.Context, email string) error
	VerifyEmail(ctx context.Context, email, otp string) (bool, error)
}

type RegisterInput struct {
	Email    string
	Password string
	FullName string
	Phone    string
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type AuthOutput struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	FullName    string `json:"full_name,omitempty"`
	Phone       string `json:"phone,omitempty"`
	IsActivated bool   `json:"is_activated"`
	AccessToken string `json:"access_token"`
}

func NewAuthService(users model.UserRepository, hasher PasswordHasher, tokens TokenProvider, verifyEmailRepo model.EmailVerificationRepository, emailSender EmailSender, otpExpiresIn time.Duration) *AuthServiceImpl {
	return &AuthServiceImpl{
		users:             users,
		hasher:            hasher,
		tokens:            tokens,
		emailVerification: verifyEmailRepo,
		emailSender:       emailSender,
		OTPExpiresIn:      otpExpiresIn,
	}
}

func (s *AuthServiceImpl) Register(ctx context.Context, in RegisterInput) (*AuthOutput, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	password := strings.TrimSpace(in.Password)
	fullName := strings.TrimSpace(in.FullName)
	phone := strings.TrimSpace(in.Phone)

	if !isValidEmail(email) || len(password) < 6 {
		return nil, ErrInvalidInput
	}

	_, err := s.users.GetByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailAlreadyExists
	}

	if !errors.Is(err, model.ErrNotFound) {
		return nil, err
	}

	passwordHash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, err
	}

	newUser := &model.User{
		Email:        email,
		PasswordHash: passwordHash,
		Role:         model.RoleManager,
		FullName:     fullName,
		Phone:        phone,
		IsActivated:  false,
	}

	if err := s.users.Create(ctx, newUser); err != nil {
		if errors.Is(err, model.ErrAlreadyExists) {
			return nil, ErrEmailAlreadyExists
		}

		return nil, err
	}

	accessToken, err := s.tokens.GenerateAccessToken(newUser.ID)
	if err != nil {
		return nil, err
	}

	return &AuthOutput{
		UserID:      newUser.ID,
		Email:       newUser.Email,
		Role:        string(newUser.Role),
		FullName:    newUser.FullName,
		Phone:       newUser.Phone,
		IsActivated: newUser.IsActivated,
		AccessToken: accessToken,
	}, nil
}

func (s *AuthServiceImpl) Login(ctx context.Context, in LoginInput) (*LoginOutput, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	password := strings.TrimSpace(in.Password)

	if !isValidEmail(email) || password == "" {
		return nil, ErrInvalidInput
	}

	existingUser, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, err
	}

	if err := s.hasher.Compare(existingUser.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.tokens.GenerateAccessToken(existingUser.ID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokens.GenerateRefreshToken(ctx, existingUser.ID)
	if err != nil {
		return nil, err
	}

	ttl := s.tokens.GetAccessTokenTTL()

	return &LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(ttl.Seconds()),
	}, nil
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func generateSixDigitOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", n.Int64()), nil
}

func (s *AuthServiceImpl) RefreshToken(ctx context.Context, refreshToken string) (*LoginOutput, error) {
	claims, err := s.tokens.Parse(refreshToken, "refresh")
	if err != nil {
		return nil, err
	}

	isRevoked, err := s.tokens.FindByToken(ctx, refreshToken, claims.UserID)
	if err != nil {
		return nil, err
	}

	if isRevoked {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.tokens.GenerateAccessToken(claims.UserID)
	if err != nil {
		return nil, err
	}

	newRefreshToken, err := s.tokens.GenerateRefreshToken(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	// err = s.tokens.RevokeRefreshToken(ctx, refreshToken, claims.UserID)
	// if err != nil {
	// 	return nil, err
	// }

	return &LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.tokens.GetAccessTokenTTL().Seconds()),
	}, nil
}

func (s *AuthServiceImpl) CreateOTP(ctx context.Context, email string) (err error) {
	otp, err := generateSixDigitOTP()
	if err != nil {
		return err
	}

	emailVeri := &model.EmailVerification{
		Email:   email,
		Expires: time.Now().Add(s.OTPExpiresIn),
		OTP:     otp,
	}

	err = s.emailVerification.CreateOTP(ctx, emailVeri)
	if err != nil {
		return err
	}

	if s.emailSender != nil {
		err = s.emailSender.SendEmail(email, otp, s.OTPExpiresIn)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *AuthServiceImpl) VerifyEmail(ctx context.Context, email, otp string) (bool, error) {
	emailVeri := &model.EmailVerification{
		Email: email,
		OTP:   otp,
	}

	err := s.emailVerification.GetOTP(ctx, emailVeri)
	if err != nil {
		return false, err
	}

	if time.Since(emailVeri.Expires) > s.OTPExpiresIn {
		return false, errors.New("otp code is expired")
	}

	if emailVeri.IsUsed {
		return false, errors.New("otp code is already used")
	}

	err = s.emailVerification.UpdateUsedOTP(ctx, emailVeri)
	if err != nil {
		return false, errors.New("cannot update used otp")
	}
	err = s.users.ActivateUser(ctx, emailVeri.Email)
	if err != nil {
		return false, err
	}
	return true, nil

}
