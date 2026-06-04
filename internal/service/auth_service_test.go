package service_test

import (
	"context"
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
	accessErr  error
	refreshErr error
	revokeErr  error
	findRes    bool
	findErr    error
	parseRes   *security.Claims
	parseErr   error
}

func (m *mockTokenProvider) GenerateAccessToken(role, email, userID string, isActivated bool) (string, error) {
	return "access-token", m.accessErr
}
func (m *mockTokenProvider) GenerateRefreshToken(ctx context.Context, userID string) (string, error) {
	return "refresh-token", m.refreshErr
}
func (m *mockTokenProvider) RevokeRefreshToken(ctx context.Context, token string, userID string) error {
	return m.revokeErr
}
func (m *mockTokenProvider) FindByToken(ctx context.Context, token string, userID string) (bool, error) {
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

		out, err := authSvc.Register(ctx, input)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if out.Email != "test@test.com" {
			t.Errorf("unexpected email")
		}
	})

	t.Run("Invalid email", func(t *testing.T) {
		input := service.RegisterInput{Email: "invalid"}
		_, err := authSvc.Register(ctx, input)
		if err != service.ErrInvalidInput {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("Already exists", func(t *testing.T) {
		input := service.RegisterInput{Email: "test@test.com"}
		userRepo.EXPECT().GetByEmail(ctx, "test@test.com").Return(&model.User{}, nil)
		_, err := authSvc.Register(ctx, input)
		if err != service.ErrEmailAlreadyExists {
			t.Errorf("expected ErrEmailAlreadyExists, got %v", err)
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
	)
	ctx := context.Background()

	t.Run("Happy path", func(t *testing.T) {
		input := service.LoginInput{
			Email:    "test@test.com",
			Password: "password",
		}
		
		userRepo.EXPECT().GetAuthUserByEmail(ctx, "test@test.com").Return(&model.User{ID: "user-1", Role: model.RoleManager, Email: "test@test.com"}, nil)

		out, err := authSvc.Login(ctx, input)
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
		_, err := authSvc.Login(ctx, input)
		if err != service.ErrInvalidCredentials {
			t.Errorf("expected ErrInvalidCredentials, got %v", err)
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
	)
	ctx := context.Background()

	t.Run("Happy path", func(t *testing.T) {
		verifyRepo.EXPECT().CreateOTP(ctx, gomock.Any()).Return(nil)
		
		err := authSvc.CreateOTP(ctx, "test@test.com")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
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
		nil,
		verifyRepo,
		nil,
		5*time.Minute,
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
		
		ok, err := authSvc.VerifyEmail(ctx, "test@test.com", "123456")
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
		
		ok, err := authSvc.VerifyEmail(ctx, "test@test.com", "123456")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if ok {
			t.Errorf("expected false")
		}
	})
}

func TestAuthService_IncrementOTPCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	otpRepo := mock_model.NewMockOTPCheckRepository(ctrl)
	
	authSvc := service.NewAuthService(
		nil, nil, nil, nil, nil, 5*time.Minute, otpRepo,
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
}

func TestAuthService_IsBlockOTP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	otpRepo := mock_model.NewMockOTPCheckRepository(ctrl)
	
	authSvc := service.NewAuthService(
		nil, nil, nil, nil, nil, 5*time.Minute, otpRepo,
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
		},
		nil,
		nil,
		5*time.Minute,
		nil,
	)
	ctx := context.Background()

	t.Run("Happy path", func(t *testing.T) {
		userRepo.EXPECT().GetByUserID(ctx, "u1").Return(&model.User{ID: "u1", Role: model.RoleManager}, nil)

		res, err := authSvc.RefreshToken(ctx, "refresh")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if res.AccessToken != "access-token" {
			t.Errorf("expected access-token, got %s", res.AccessToken)
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
}
