package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service"
	"go.uber.org/mock/gomock"
)

type mockPasswordHasher struct {
	hashErr    error
	compareErr error
}

func (m *mockPasswordHasher) Hash(password string) (string, error) {
	if m.hashErr != nil {
		return "", m.hashErr
	}
	return "hashed-" + password, nil
}

func (m *mockPasswordHasher) Compare(hash, password string) error {
	return m.compareErr
}

type mockTokenProvider struct {
	accessErr          error
	refreshErr         error
	generateRefreshErr error
	revokeErr          error
	findRes            *model.AuthSession
	findErr            error
	parseRes           *security.Claims
	parseErr           error
}

func (m *mockTokenProvider) GenerateAccessToken(role, email, userID string, isActivated bool, jkt string) (string, error) {
	return "access-token", m.accessErr
}
func (m *mockTokenProvider) GenerateRefreshToken(ctx context.Context, userID, ipAddress, userAgent, location, jkt string, latitude, longitude *float64, geocodingSource *string) (string, error) {
	if m.generateRefreshErr != nil {
		return "", m.generateRefreshErr
	}
	return "refresh-token", m.refreshErr
}
func (m *mockTokenProvider) RevokeRefreshToken(ctx context.Context, token string, userID string) error {
	return m.revokeErr
}
func (m *mockTokenProvider) FindByToken(ctx context.Context, token string, userID string) (*model.AuthSession, error) {
	return m.findRes, m.findErr
}
func (m *mockTokenProvider) GetAccessTokenTTL() time.Duration {
	return time.Hour
}
func (m *mockTokenProvider) Parse(tokenString string, tokenType string) (*security.Claims, error) {
	return m.parseRes, m.parseErr
}

type mockEmailSender struct {
	err error
}

func (m *mockEmailSender) SendEmail(toEmail string, otpCode string, expiresIn time.Duration) error {
	return m.err
}

func TestAuthService_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)
	verifyRepo := mock_model.NewMockEmailVerificationRepository(ctrl)
	otpRepo := mock_model.NewMockOTPCheckRepository(ctrl)

	authSvc := service.NewAuthService(
		userRepo,
		&mockPasswordHasher{},
		&mockTokenProvider{},
		verifyRepo,
		&mockEmailSender{},
		5*time.Minute,
		otpRepo,
		nil,
		nil,
	)
	ctx := context.Background()

	t.Run("Happy path", func(t *testing.T) {
		input := service.RegisterInput{
			Email:    "test@test.com",
			Password: "password",
			FullName: "Test User",
			Phone:    "123",
		}
		
		userRepo.EXPECT().GetByEmail(ctx, "test@test.com").Return(nil, model.ErrNotFound)
		userRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		out, err := authSvc.Register(ctx, input, "", "", "")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if out.AccessToken == "" || out.RefreshToken == "" {
			t.Errorf("expected tokens")
		}
	})

	t.Run("Invalid email", func(t *testing.T) {
		input := service.RegisterInput{Email: "invalid"}
		_, err := authSvc.Register(ctx, input, "", "", "")
		if err != service.ErrInvalidInput {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("Already exists", func(t *testing.T) {
		input := service.RegisterInput{Email: "test@test.com"}
		userRepo.EXPECT().GetByEmail(ctx, "test@test.com").Return(&model.User{}, nil)
		_, err := authSvc.Register(ctx, input, "", "", "")
		if err != service.ErrEmailAlreadyExists {
			t.Errorf("expected ErrEmailAlreadyExists, got %v", err)
		}
	})

	t.Run("GetByEmail returns unexpected error", func(t *testing.T) {
		input := service.RegisterInput{Email: "test@test.com"}
		userRepo.EXPECT().GetByEmail(ctx, "test@test.com").Return(nil, errors.New("db error"))
		_, err := authSvc.Register(ctx, input, "", "", "")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("Hash password fails", func(t *testing.T) {
		input := service.RegisterInput{Email: "test@test.com", Password: "pass"}
		userRepo.EXPECT().GetByEmail(ctx, "test@test.com").Return(nil, model.ErrNotFound)
		
		authSvcWithMockHash := service.NewAuthService(userRepo, &mockPasswordHasher{hashErr: errors.New("hash error")}, &mockTokenProvider{}, verifyRepo, &mockEmailSender{}, 5*time.Minute, otpRepo, nil, nil)
		
		_, err := authSvcWithMockHash.Register(ctx, input, "", "", "")
		if err == nil || err.Error() != "hash error" {
			t.Errorf("expected hash error, got %v", err)
		}
	})

	t.Run("Create returns ErrAlreadyExists", func(t *testing.T) {
		input := service.RegisterInput{Email: "test@test.com", Password: "pass"}
		userRepo.EXPECT().GetByEmail(ctx, "test@test.com").Return(nil, model.ErrNotFound)
		userRepo.EXPECT().Create(ctx, gomock.Any()).Return(model.ErrAlreadyExists)
		_, err := authSvc.Register(ctx, input, "", "", "")
		if err != service.ErrEmailAlreadyExists {
			t.Errorf("expected ErrEmailAlreadyExists, got %v", err)
		}
	})

	t.Run("Create returns non-duplicate error", func(t *testing.T) {
		input := service.RegisterInput{Email: "test@test.com", Password: "pass"}
		userRepo.EXPECT().GetByEmail(ctx, "test@test.com").Return(nil, model.ErrNotFound)
		userRepo.EXPECT().Create(ctx, gomock.Any()).Return(errors.New("db error"))
		_, err := authSvc.Register(ctx, input, "", "", "")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("GenerateAccessToken fails", func(t *testing.T) {
		input := service.RegisterInput{Email: "test@test.com", Password: "pass"}
		userRepo.EXPECT().GetByEmail(ctx, "test@test.com").Return(nil, model.ErrNotFound)
		userRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)
		
		authSvcWithMockToken := service.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{accessErr: errors.New("token error")}, verifyRepo, &mockEmailSender{}, 5*time.Minute, otpRepo, nil, nil)
		_, err := authSvcWithMockToken.Register(ctx, input, "", "", "")
		if err == nil || err.Error() != "token error" {
			t.Errorf("expected token error, got %v", err)
		}
	})
}

func TestAuthService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)

	authSvc := service.NewAuthService(
		userRepo,
		&mockPasswordHasher{},
		&mockTokenProvider{},
		nil,
		nil,
		5*time.Minute,
		nil,
		nil,
		nil,
	)
	ctx := context.Background()

	t.Run("Happy path", func(t *testing.T) {
		input := service.LoginInput{
			Email:    "test@test.com",
			Password: "password",
		}
		
		userRepo.EXPECT().GetAuthUserByEmail(ctx, "test@test.com").Return(&model.User{ID: "user-1", Role: model.RoleManager, Email: "test@test.com"}, nil)

		out, err := authSvc.Login(ctx, input, "", "", "")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if out.AccessToken != "access-token" {
			t.Errorf("unexpected access token")
		}
	})

	t.Run("User not found", func(t *testing.T) {
		input := service.LoginInput{Email: "test@test.com", Password: "pass"}
		userRepo.EXPECT().GetAuthUserByEmail(ctx, "test@test.com").Return(nil, model.ErrNotFound)
		_, err := authSvc.Login(ctx, input, "", "", "")
		if err != service.ErrInvalidCredentials {
			t.Errorf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("GetAuthUserByEmail returns unexpected error", func(t *testing.T) {
		input := service.LoginInput{Email: "test@test.com", Password: "pass"}
		userRepo.EXPECT().GetAuthUserByEmail(ctx, "test@test.com").Return(nil, errors.New("db error"))
		_, err := authSvc.Login(ctx, input, "", "", "")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("Wrong password", func(t *testing.T) {
		input := service.LoginInput{Email: "test@test.com", Password: "wrong"}
		userRepo.EXPECT().GetAuthUserByEmail(ctx, "test@test.com").Return(&model.User{PasswordHash: "hashed-pass"}, nil)
		
		authSvcWithMockHash := service.NewAuthService(userRepo, &mockPasswordHasher{compareErr: errors.New("compare failed")}, &mockTokenProvider{}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockHash.Login(ctx, input, "", "", "")
		if err != service.ErrInvalidCredentials {
			t.Errorf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("GenerateAccessToken fails", func(t *testing.T) {
		input := service.LoginInput{Email: "test@test.com", Password: "password"}
		userRepo.EXPECT().GetAuthUserByEmail(ctx, "test@test.com").Return(&model.User{PasswordHash: "hashed-password"}, nil)
		
		authSvcWithMockToken := service.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{accessErr: errors.New("token error")}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockToken.Login(ctx, input, "", "", "")
		if err == nil || err.Error() != "token error" {
			t.Errorf("expected token error, got %v", err)
		}
	})

	t.Run("GenerateRefreshToken fails", func(t *testing.T) {
		input := service.LoginInput{Email: "test@test.com", Password: "password"}
		userRepo.EXPECT().GetAuthUserByEmail(ctx, "test@test.com").Return(&model.User{PasswordHash: "hashed-password"}, nil)
		
		authSvcWithMockToken := service.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{refreshErr: errors.New("refresh error")}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockToken.Login(ctx, input, "", "", "")
		if err == nil || err.Error() != "refresh error" {
			t.Errorf("expected refresh error, got %v", err)
		}
	})
}

func TestAuthService_CreateOTP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	verifyRepo := mock_model.NewMockEmailVerificationRepository(ctrl)
	
	authSvc := service.NewAuthService(
		nil,
		nil,
		nil,
		verifyRepo,
		&mockEmailSender{},
		5*time.Minute,
		nil,
		nil,
		nil,
	)
	ctx := context.Background()

	t.Run("Happy path", func(t *testing.T) {
		verifyRepo.EXPECT().CreateOTP(ctx, gomock.Any()).Return(nil)
		
		err := authSvc.CreateOTP(ctx, "test@test.com")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})

	t.Run("CreateOTP repo fails", func(t *testing.T) {
		verifyRepo.EXPECT().CreateOTP(ctx, gomock.Any()).Return(errors.New("db error"))
		
		err := authSvc.CreateOTP(ctx, "test@test.com")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("EmailSender fails", func(t *testing.T) {
		verifyRepo.EXPECT().CreateOTP(ctx, gomock.Any()).Return(nil)
		
		authSvcWithSenderErr := service.NewAuthService(nil, nil, nil, verifyRepo, &mockEmailSender{err: errors.New("send error")}, 5*time.Minute, nil, nil, nil)
		err := authSvcWithSenderErr.CreateOTP(ctx, "test@test.com")
		if err == nil || err.Error() != "send error" {
			t.Errorf("expected send error, got %v", err)
		}
	})
}

func TestAuthService_VerifyEmail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)
	verifyRepo := mock_model.NewMockEmailVerificationRepository(ctrl)
	
	authSvc := service.NewAuthService(
		userRepo,
		nil,
		&mockTokenProvider{},
		verifyRepo,
		nil,
		5*time.Minute,
		nil,
		nil,
		nil,
	)
	ctx := context.Background()

	t.Run("Happy path", func(t *testing.T) {
		verifyRepo.EXPECT().GetOTP(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, v *model.EmailVerification) error {
			v.Expires = time.Now().Add(5 * time.Minute)
			v.IsUsed = false
			return nil
		})
		verifyRepo.EXPECT().UpdateUsedOTP(ctx, gomock.Any()).Return(nil)
		userRepo.EXPECT().ActivateUser(ctx, "test@test.com").Return(nil)
		userRepo.EXPECT().GetByEmail(ctx, "test@test.com").Return(&model.User{ID: "1", Email: "test@test.com", Role: "MANAGER"}, nil)
		
		_, ok, err := authSvc.VerifyEmail(ctx, "test@test.com", "123456", "")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if !ok {
			t.Errorf("expected true")
		}
	})

	t.Run("Expired", func(t *testing.T) {
		verifyRepo.EXPECT().GetOTP(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, v *model.EmailVerification) error {
			v.Expires = time.Now().Add(-5 * time.Minute) // Expired
			return nil
		})
		
		_, ok, err := authSvc.VerifyEmail(ctx, "test@test.com", "123456", "")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if ok {
			t.Errorf("expected false")
		}
	})

	t.Run("GetOTP returns non-ErrNoRows error", func(t *testing.T) {
		verifyRepo.EXPECT().GetOTP(ctx, gomock.Any()).Return(errors.New("db error"))
		
		_, _, err := authSvc.VerifyEmail(ctx, "test@test.com", "123456", "")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("OTP already used", func(t *testing.T) {
		verifyRepo.EXPECT().GetOTP(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, v *model.EmailVerification) error {
			v.Expires = time.Now().Add(5 * time.Minute)
			v.IsUsed = true // Already used
			return nil
		})
		
		_, ok, err := authSvc.VerifyEmail(ctx, "test@test.com", "123456", "")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if ok {
			t.Errorf("expected false")
		}
	})

	t.Run("UpdateUsedOTP fails", func(t *testing.T) {
		verifyRepo.EXPECT().GetOTP(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, v *model.EmailVerification) error {
			v.Expires = time.Now().Add(5 * time.Minute)
			v.IsUsed = false
			return nil
		})
		verifyRepo.EXPECT().UpdateUsedOTP(ctx, gomock.Any()).Return(errors.New("db error"))
		
		_, _, err := authSvc.VerifyEmail(ctx, "test@test.com", "123456", "")
		if err == nil || err.Error() != "cannot update used otp" {
			t.Errorf("expected cannot update used otp, got %v", err)
		}
	})

	t.Run("ActivateUser fails", func(t *testing.T) {
		verifyRepo.EXPECT().GetOTP(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, v *model.EmailVerification) error {
			v.Expires = time.Now().Add(5 * time.Minute)
			v.IsUsed = false
			return nil
		})
		verifyRepo.EXPECT().UpdateUsedOTP(ctx, gomock.Any()).Return(nil)
		userRepo.EXPECT().ActivateUser(ctx, "test@test.com").Return(errors.New("activate error"))
		
		_, _, err := authSvc.VerifyEmail(ctx, "test@test.com", "123456", "")
		if err == nil || err.Error() != "activate error" {
			t.Errorf("expected activate error, got %v", err)
		}
	})
}

func TestAuthService_IncrementOTPCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	otpRepo := mock_model.NewMockOTPCheckRepository(ctrl)
	
	authSvc := service.NewAuthService(
		nil, nil, nil, nil, nil, 5*time.Minute, otpRepo, nil, nil,
	)
	ctx := context.Background()

	t.Run("Happy path", func(t *testing.T) {
		otpRepo.EXPECT().GetOTPCheck(ctx, "test@test.com").Return(model.OTPCheck{OTPFails: 1}, nil)
		otpRepo.EXPECT().IncrementOTPCheck(ctx, "test@test.com", int64(2)).Return(nil)

		err := authSvc.IncrementOTPCheck(ctx, "test@test.com")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})

	t.Run("GetOTPCheck returns ErrNoRows, CreateOTPCheck fails", func(t *testing.T) {
		otpRepo.EXPECT().GetOTPCheck(ctx, "test@test.com").Return(model.OTPCheck{}, sql.ErrNoRows)
		otpRepo.EXPECT().CreateOTPCheck(ctx, "test@test.com").Return(errors.New("db error"))

		err := authSvc.IncrementOTPCheck(ctx, "test@test.com")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("GetOTPCheck returns unexpected error", func(t *testing.T) {
		otpRepo.EXPECT().GetOTPCheck(ctx, "test@test.com").Return(model.OTPCheck{}, errors.New("db error"))

		err := authSvc.IncrementOTPCheck(ctx, "test@test.com")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})
}

func TestAuthService_IsBlockOTP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	otpRepo := mock_model.NewMockOTPCheckRepository(ctrl)
	
	authSvc := service.NewAuthService(
		nil, nil, nil, nil, nil, 5*time.Minute, otpRepo, nil, nil,
	)
	ctx := context.Background()

	t.Run("Not blocked", func(t *testing.T) {
		otpRepo.EXPECT().GetOTPCheck(ctx, "test@test.com").Return(model.OTPCheck{OTPFails: 1}, nil)
		blocked, err := authSvc.IsBlockOTP(ctx, "test@test.com")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if blocked {
			t.Errorf("expected false")
		}
	})

	t.Run("Blocked but expired", func(t *testing.T) {
		otpRepo.EXPECT().GetOTPCheck(ctx, "test@test.com").Return(model.OTPCheck{
			OTPFails: 5,
			BlockTime: time.Now().Add(-20 * time.Minute),
		}, nil)
		otpRepo.EXPECT().ResetOTP(ctx, "test@test.com").Return(nil)
		
		blocked, err := authSvc.IsBlockOTP(ctx, "test@test.com")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if blocked {
			t.Errorf("expected false after reset")
		}
	})
	
	t.Run("Blocked", func(t *testing.T) {
		otpRepo.EXPECT().GetOTPCheck(ctx, "test@test.com").Return(model.OTPCheck{
			OTPFails: 5,
			BlockTime: time.Now().Add(-5 * time.Minute),
		}, nil)
		
		blocked, err := authSvc.IsBlockOTP(ctx, "test@test.com")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if !blocked {
			t.Errorf("expected true")
		}
	})

	t.Run("GetOTPCheck returns unexpected error", func(t *testing.T) {
		otpRepo.EXPECT().GetOTPCheck(ctx, "test@test.com").Return(model.OTPCheck{}, errors.New("db error"))
		
		blocked, err := authSvc.IsBlockOTP(ctx, "test@test.com")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
		if !blocked {
			t.Errorf("expected true on error")
		}
	})

	t.Run("ResetOTP fails", func(t *testing.T) {
		otpRepo.EXPECT().GetOTPCheck(ctx, "test@test.com").Return(model.OTPCheck{
			OTPFails: 5,
			BlockTime: time.Now().Add(-20 * time.Minute),
		}, nil)
		otpRepo.EXPECT().ResetOTP(ctx, "test@test.com").Return(errors.New("db error"))
		
		blocked, err := authSvc.IsBlockOTP(ctx, "test@test.com")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
		if !blocked {
			t.Errorf("expected true on error")
		}
	})
}

func TestAuthService_RefreshToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)

	authSvc := service.NewAuthService(
		userRepo,
		&mockPasswordHasher{},
		&mockTokenProvider{
			parseRes: &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "u1"}},
			findRes: &model.AuthSession{Revoked: false, ExpiresAt: time.Now().Add(time.Hour)},
		},
		nil,
		nil,
		5*time.Minute,
		nil,
		nil,
		nil,
	)
	ctx := context.Background()

	t.Run("Happy path", func(t *testing.T) {
		userRepo.EXPECT().GetByUserID(ctx, "u1").Return(&model.User{ID: "u1", Role: model.RoleManager}, nil)

		res, err := authSvc.RefreshToken(ctx, "refresh", "", "", "")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if res.AccessToken != "access-token" {
			t.Errorf("expected access-token, got %s", res.AccessToken)
		}
	})

	t.Run("Parse fails", func(t *testing.T) {
		authSvcWithMockToken := service.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{parseErr: errors.New("parse error")}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockToken.RefreshToken(ctx, "refresh", "", "", "")
		if err == nil || err.Error() != "parse error" {
			t.Errorf("expected parse error, got %v", err)
		}
	})

	t.Run("FindByToken fails", func(t *testing.T) {
		authSvcWithMockToken := service.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{
			parseRes: &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "u1"}},
			findErr: errors.New("db error"),
		}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockToken.RefreshToken(ctx, "refresh", "", "", "")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("Token is revoked", func(t *testing.T) {
		authSvcWithMockToken := service.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{
			parseRes: &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "u1"}},
			findRes: &model.AuthSession{Revoked: true}, // revoked
		}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockToken.RefreshToken(ctx, "refresh", "", "", "")
		if err != service.ErrInvalidCredentials {
			t.Errorf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("GetByUserID fails", func(t *testing.T) {
		userRepo.EXPECT().GetByUserID(ctx, "u1").Return(nil, errors.New("db error"))
		_, err := authSvc.RefreshToken(ctx, "refresh", "", "", "")
		if err != service.ErrInvalidCredentials {
			t.Errorf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("GenerateAccessToken fails after user lookup", func(t *testing.T) {
		userRepo.EXPECT().GetByUserID(ctx, "u1").Return(&model.User{ID: "u1"}, nil)
		authSvcWithMockToken := service.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{
			parseRes: &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "u1"}},
			findRes: &model.AuthSession{Revoked: false, ExpiresAt: time.Now().Add(time.Hour)},
			accessErr: errors.New("token error"),
		}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockToken.RefreshToken(ctx, "refresh", "", "", "")
		if err == nil || err.Error() != "token error" {
			t.Errorf("expected token error, got %v", err)
		}
	})

	t.Run("GenerateRefreshToken fails", func(t *testing.T) {
		userRepo.EXPECT().GetByUserID(ctx, "u1").Return(&model.User{ID: "u1"}, nil)
		authSvcWithMockToken := service.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{
			parseRes: &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "u1"}},
			findRes: &model.AuthSession{Revoked: false, ExpiresAt: time.Now().Add(time.Hour)},
			refreshErr: errors.New("refresh error"),
		}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockToken.RefreshToken(ctx, "refresh", "", "", "")
		if err == nil || err.Error() != "refresh error" {
			t.Errorf("expected refresh error, got %v", err)
		}
	})
}

func TestAuthService_GetMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)

	authSvc := service.NewAuthService(
		userRepo,
		&mockPasswordHasher{},
		&mockTokenProvider{},
		nil,
		nil,
		5*time.Minute,
		nil,
		nil,
		nil,
	)
	ctx := context.Background()

	t.Run("Happy path", func(t *testing.T) {
		userRepo.EXPECT().GetByUserID(ctx, "u1").Return(&model.User{ID: "u1"}, nil)

		user, err := authSvc.GetMe(ctx, "u1")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if user.UserID != "u1" {
			t.Errorf("expected user u1, got %v", user.UserID)
		}
	})

	t.Run("GetByUserID returns ErrNotFound", func(t *testing.T) {
		userRepo.EXPECT().GetByUserID(ctx, "u1").Return(nil, model.ErrNotFound)

		_, err := authSvc.GetMe(ctx, "u1")
		if err != service.ErrInvalidCredentials {
			t.Errorf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("GetByUserID returns unexpected error", func(t *testing.T) {
		userRepo.EXPECT().GetByUserID(ctx, "u1").Return(nil, errors.New("db error"))

		_, err := authSvc.GetMe(ctx, "u1")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})
}
