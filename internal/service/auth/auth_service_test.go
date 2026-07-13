package auth_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	sharedsvc "github.com/mihb123/quanly-phongtro/internal/service/shared"

	authsvc "github.com/mihb123/quanly-phongtro/internal/service/auth"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
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
	revokedToken       string
	revokedUserID      string
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
func (m *mockTokenProvider) UpdateSessionLocation(ctx context.Context, refreshToken, userID, location string, geocodingSource *string) error {
	return nil
}
func (m *mockTokenProvider) RevokeRefreshToken(ctx context.Context, token string, userID string) error {
	m.revokedToken = token
	m.revokedUserID = userID
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

type fakeGeocodingService struct {
	addr   string
	source string
	err    error
}

// ReverseGeocode returns configured geocoding results for auth location tests.
func (f fakeGeocodingService) ReverseGeocode(lat, lng float64) (string, string, error) {
	return f.addr, f.source, f.err
}

type fakeGeoIPService struct {
	location string
}

// LookupLocation returns a configured fallback location for auth tests.
func (f fakeGeoIPService) LookupLocation(ipAddress string) string {
	return f.location
}

// Close satisfies the GeoIPService contract for auth tests.
func (f fakeGeoIPService) Close() {}

func TestAuthService_Register(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)
	verifyRepo := mock_model.NewMockEmailVerificationRepository(ctrl)
	otpRepo := mock_model.NewMockOTPCheckRepository(ctrl)

	authSvc := authsvc.NewAuthService(
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
		input := authsvc.RegisterInput{
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
		input := authsvc.RegisterInput{Email: "invalid"}
		_, err := authSvc.Register(ctx, input, "", "", "")
		if err != sharedsvc.ErrInvalidInput {
			t.Errorf("expected shared.ErrInvalidInput, got %v", err)
		}
	})

	t.Run("Already exists", func(t *testing.T) {
		input := authsvc.RegisterInput{Email: "test@test.com"}
		userRepo.EXPECT().GetByEmail(ctx, "test@test.com").Return(&model.User{}, nil)
		_, err := authSvc.Register(ctx, input, "", "", "")
		if err != authsvc.ErrEmailAlreadyExists {
			t.Errorf("expected ErrEmailAlreadyExists, got %v", err)
		}
	})

	t.Run("GetByEmail returns unexpected error", func(t *testing.T) {
		input := authsvc.RegisterInput{Email: "test@test.com"}
		userRepo.EXPECT().GetByEmail(ctx, "test@test.com").Return(nil, errors.New("db error"))
		_, err := authSvc.Register(ctx, input, "", "", "")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("Hash password fails", func(t *testing.T) {
		input := authsvc.RegisterInput{Email: "test@test.com", Password: "pass"}
		userRepo.EXPECT().GetByEmail(ctx, "test@test.com").Return(nil, model.ErrNotFound)

		authSvcWithMockHash := authsvc.NewAuthService(userRepo, &mockPasswordHasher{hashErr: errors.New("hash error")}, &mockTokenProvider{}, verifyRepo, &mockEmailSender{}, 5*time.Minute, otpRepo, nil, nil)

		_, err := authSvcWithMockHash.Register(ctx, input, "", "", "")
		if err == nil || err.Error() != "hash error" {
			t.Errorf("expected hash error, got %v", err)
		}
	})

	t.Run("Create returns ErrAlreadyExists", func(t *testing.T) {
		input := authsvc.RegisterInput{Email: "test@test.com", Password: "pass"}
		userRepo.EXPECT().GetByEmail(ctx, "test@test.com").Return(nil, model.ErrNotFound)
		userRepo.EXPECT().Create(ctx, gomock.Any()).Return(model.ErrAlreadyExists)
		_, err := authSvc.Register(ctx, input, "", "", "")
		if err != authsvc.ErrEmailAlreadyExists {
			t.Errorf("expected ErrEmailAlreadyExists, got %v", err)
		}
	})

	t.Run("Create returns non-duplicate error", func(t *testing.T) {
		input := authsvc.RegisterInput{Email: "test@test.com", Password: "pass"}
		userRepo.EXPECT().GetByEmail(ctx, "test@test.com").Return(nil, model.ErrNotFound)
		userRepo.EXPECT().Create(ctx, gomock.Any()).Return(errors.New("db error"))
		_, err := authSvc.Register(ctx, input, "", "", "")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("GenerateAccessToken fails", func(t *testing.T) {
		input := authsvc.RegisterInput{Email: "test@test.com", Password: "pass"}
		userRepo.EXPECT().GetByEmail(ctx, "test@test.com").Return(nil, model.ErrNotFound)
		userRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		authSvcWithMockToken := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{accessErr: errors.New("token error")}, verifyRepo, &mockEmailSender{}, 5*time.Minute, otpRepo, nil, nil)
		_, err := authSvcWithMockToken.Register(ctx, input, "", "", "")
		if err == nil || err.Error() != "token error" {
			t.Errorf("expected token error, got %v", err)
		}
	})

	t.Run("Uses reverse geocoding for coordinates", func(t *testing.T) {
		lat := 21.0
		lng := 105.0
		input := authsvc.RegisterInput{Email: "geo@test.com", Password: "pass", Latitude: &lat, Longitude: &lng}
		userRepo.EXPECT().GetByEmail(ctx, "geo@test.com").Return(nil, model.ErrNotFound)
		userRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		authSvcWithGeo := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{}, verifyRepo, &mockEmailSender{}, 5*time.Minute, otpRepo, nil, fakeGeocodingService{addr: "Ha Noi", source: "test"})
		_, err := authSvcWithGeo.Register(ctx, input, "8.8.8.8", "", "")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})

	t.Run("Falls back to GeoIP when reverse geocoding fails", func(t *testing.T) {
		lat := 21.0
		lng := 105.0
		input := authsvc.RegisterInput{Email: "fallback@test.com", Password: "pass", Latitude: &lat, Longitude: &lng}
		userRepo.EXPECT().GetByEmail(ctx, "fallback@test.com").Return(nil, model.ErrNotFound)
		userRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		authSvcWithGeo := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{}, verifyRepo, &mockEmailSender{}, 5*time.Minute, otpRepo, fakeGeoIPService{location: "GeoIP"}, fakeGeocodingService{err: errors.New("geo error")})
		_, err := authSvcWithGeo.Register(ctx, input, "8.8.8.8", "", "")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})
}

func TestAuthService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)

	authSvc := authsvc.NewAuthService(
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
		input := authsvc.LoginInput{
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
		input := authsvc.LoginInput{Email: "test@test.com", Password: "pass"}
		userRepo.EXPECT().GetAuthUserByEmail(ctx, "test@test.com").Return(nil, model.ErrNotFound)
		_, err := authSvc.Login(ctx, input, "", "", "")
		if err != authsvc.ErrInvalidCredentials {
			t.Errorf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("GetAuthUserByEmail returns unexpected error", func(t *testing.T) {
		input := authsvc.LoginInput{Email: "test@test.com", Password: "pass"}
		userRepo.EXPECT().GetAuthUserByEmail(ctx, "test@test.com").Return(nil, errors.New("db error"))
		_, err := authSvc.Login(ctx, input, "", "", "")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("Wrong password", func(t *testing.T) {
		input := authsvc.LoginInput{Email: "test@test.com", Password: "wrong"}
		userRepo.EXPECT().GetAuthUserByEmail(ctx, "test@test.com").Return(&model.User{PasswordHash: "hashed-pass"}, nil)

		authSvcWithMockHash := authsvc.NewAuthService(userRepo, &mockPasswordHasher{compareErr: errors.New("compare failed")}, &mockTokenProvider{}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockHash.Login(ctx, input, "", "", "")
		if err != authsvc.ErrInvalidCredentials {
			t.Errorf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("GenerateAccessToken fails", func(t *testing.T) {
		input := authsvc.LoginInput{Email: "test@test.com", Password: "password"}
		userRepo.EXPECT().GetAuthUserByEmail(ctx, "test@test.com").Return(&model.User{PasswordHash: "hashed-password"}, nil)

		authSvcWithMockToken := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{accessErr: errors.New("token error")}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockToken.Login(ctx, input, "", "", "")
		if err == nil || err.Error() != "token error" {
			t.Errorf("expected token error, got %v", err)
		}
	})

	t.Run("GenerateRefreshToken fails", func(t *testing.T) {
		input := authsvc.LoginInput{Email: "test@test.com", Password: "password"}
		userRepo.EXPECT().GetAuthUserByEmail(ctx, "test@test.com").Return(&model.User{PasswordHash: "hashed-password"}, nil)

		authSvcWithMockToken := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{refreshErr: errors.New("refresh error")}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockToken.Login(ctx, input, "", "", "")
		if err == nil || err.Error() != "refresh error" {
			t.Errorf("expected refresh error, got %v", err)
		}
	})

	t.Run("Invalid input", func(t *testing.T) {
		_, err := authSvc.Login(ctx, authsvc.LoginInput{Email: "bad", Password: ""}, "", "", "")
		if err != sharedsvc.ErrInvalidInput {
			t.Errorf("expected shared.ErrInvalidInput, got %v", err)
		}
	})

	t.Run("Uses GeoIP without coordinates", func(t *testing.T) {
		input := authsvc.LoginInput{Email: "test@test.com", Password: "password"}
		userRepo.EXPECT().GetAuthUserByEmail(ctx, "test@test.com").Return(&model.User{ID: "u1", PasswordHash: "hashed-password"}, nil)

		authSvcWithGeo := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{}, nil, nil, 5*time.Minute, nil, fakeGeoIPService{location: "GeoIP"}, nil)
		_, err := authSvcWithGeo.Login(ctx, input, "8.8.8.8", "", "")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})

	t.Run("Uses reverse geocoding for coordinates", func(t *testing.T) {
		lat := 21.0
		lng := 105.0
		input := authsvc.LoginInput{Email: "test@test.com", Password: "password", Latitude: &lat, Longitude: &lng}
		userRepo.EXPECT().GetAuthUserByEmail(ctx, "test@test.com").Return(&model.User{ID: "u1", PasswordHash: "hashed-password"}, nil)

		authSvcWithGeo := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{}, nil, nil, 5*time.Minute, nil, nil, fakeGeocodingService{addr: "Ha Noi", source: "test"})
		_, err := authSvcWithGeo.Login(ctx, input, "8.8.8.8", "", "")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})
}

func TestAuthService_CreateOTP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	verifyRepo := mock_model.NewMockEmailVerificationRepository(ctrl)

	authSvc := authsvc.NewAuthService(
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

		authSvcWithSenderErr := authsvc.NewAuthService(nil, nil, nil, verifyRepo, &mockEmailSender{err: errors.New("send error")}, 5*time.Minute, nil, nil, nil)
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

	authSvc := authsvc.NewAuthService(
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

	t.Run("OTP not found", func(t *testing.T) {
		verifyRepo.EXPECT().GetOTP(ctx, gomock.Any()).Return(sql.ErrNoRows)

		_, ok, err := authSvc.VerifyEmail(ctx, "test@test.com", "123456", "")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
		if ok {
			t.Errorf("expected false")
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

	t.Run("GetByEmail fails", func(t *testing.T) {
		verifyRepo.EXPECT().GetOTP(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, v *model.EmailVerification) error {
			v.Expires = time.Now().Add(5 * time.Minute)
			v.IsUsed = false
			return nil
		})
		verifyRepo.EXPECT().UpdateUsedOTP(ctx, gomock.Any()).Return(nil)
		userRepo.EXPECT().ActivateUser(ctx, "test@test.com").Return(nil)
		userRepo.EXPECT().GetByEmail(ctx, "test@test.com").Return(nil, errors.New("db error"))

		_, _, err := authSvc.VerifyEmail(ctx, "test@test.com", "123456", "")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("GenerateAccessToken fails", func(t *testing.T) {
		verifyRepo.EXPECT().GetOTP(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, v *model.EmailVerification) error {
			v.Expires = time.Now().Add(5 * time.Minute)
			v.IsUsed = false
			return nil
		})
		verifyRepo.EXPECT().UpdateUsedOTP(ctx, gomock.Any()).Return(nil)
		userRepo.EXPECT().ActivateUser(ctx, "test@test.com").Return(nil)
		userRepo.EXPECT().GetByEmail(ctx, "test@test.com").Return(&model.User{ID: "1", Email: "test@test.com", Role: model.RoleManager}, nil)

		authSvcWithTokenErr := authsvc.NewAuthService(userRepo, nil, &mockTokenProvider{accessErr: errors.New("token error")}, verifyRepo, nil, 5*time.Minute, nil, nil, nil)
		_, _, err := authSvcWithTokenErr.VerifyEmail(ctx, "test@test.com", "123456", "")
		if err == nil || err.Error() != "token error" {
			t.Errorf("expected token error, got %v", err)
		}
	})
}

func TestAuthService_IncrementOTPCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	otpRepo := mock_model.NewMockOTPCheckRepository(ctrl)

	authSvc := authsvc.NewAuthService(
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

	authSvc := authsvc.NewAuthService(
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
			OTPFails:  5,
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
			OTPFails:  5,
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
			OTPFails:  5,
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
	tokenProvider := &mockTokenProvider{
		parseRes: &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "u1"}},
		findRes:  &model.AuthSession{Revoked: false, ExpiresAt: time.Now().Add(time.Hour)},
	}

	authSvc := authsvc.NewAuthService(
		userRepo,
		&mockPasswordHasher{},
		tokenProvider,
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
		if tokenProvider.revokedToken != "refresh" || tokenProvider.revokedUserID != "u1" {
			t.Errorf("expected old refresh token revoked, got token=%q user=%q", tokenProvider.revokedToken, tokenProvider.revokedUserID)
		}
	})

	t.Run("Parse fails", func(t *testing.T) {
		authSvcWithMockToken := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{parseErr: errors.New("parse error")}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockToken.RefreshToken(ctx, "refresh", "", "", "")
		if err == nil || err.Error() != "parse error" {
			t.Errorf("expected parse error, got %v", err)
		}
	})

	t.Run("FindByToken fails", func(t *testing.T) {
		authSvcWithMockToken := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{
			parseRes: &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "u1"}},
			findErr:  errors.New("db error"),
		}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockToken.RefreshToken(ctx, "refresh", "", "", "")
		if err == nil || err.Error() != "db error" {
			t.Errorf("expected db error, got %v", err)
		}
	})

	t.Run("Token is revoked", func(t *testing.T) {
		authSvcWithMockToken := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{
			parseRes: &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "u1"}},
			findRes:  &model.AuthSession{Revoked: true}, // revoked
		}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockToken.RefreshToken(ctx, "refresh", "", "", "")
		if err != authsvc.ErrInvalidCredentials {
			t.Errorf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("Token is expired", func(t *testing.T) {
		authSvcWithMockToken := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{
			parseRes: &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "u1"}},
			findRes:  &model.AuthSession{Revoked: false, ExpiresAt: time.Now().Add(-time.Minute)},
		}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockToken.RefreshToken(ctx, "refresh", "", "", "")
		if err != authsvc.ErrInvalidCredentials {
			t.Errorf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("JKT mismatch", func(t *testing.T) {
		authSvcWithMockToken := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{
			parseRes: &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "u1"}},
			findRes:  &model.AuthSession{Revoked: false, ExpiresAt: time.Now().Add(time.Hour), JKT: "expected"},
		}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockToken.RefreshToken(ctx, "refresh", "", "", "actual")
		if err != authsvc.ErrInvalidCredentials {
			t.Errorf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("GetByUserID fails", func(t *testing.T) {
		userRepo.EXPECT().GetByUserID(ctx, "u1").Return(nil, errors.New("db error"))
		_, err := authSvc.RefreshToken(ctx, "refresh", "", "", "")
		if err != authsvc.ErrInvalidCredentials {
			t.Errorf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("GenerateAccessToken fails after user lookup", func(t *testing.T) {
		userRepo.EXPECT().GetByUserID(ctx, "u1").Return(&model.User{ID: "u1"}, nil)
		authSvcWithMockToken := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{
			parseRes:  &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "u1"}},
			findRes:   &model.AuthSession{Revoked: false, ExpiresAt: time.Now().Add(time.Hour)},
			accessErr: errors.New("token error"),
		}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockToken.RefreshToken(ctx, "refresh", "", "", "")
		if err == nil || err.Error() != "token error" {
			t.Errorf("expected token error, got %v", err)
		}
	})

	t.Run("GenerateRefreshToken fails", func(t *testing.T) {
		userRepo.EXPECT().GetByUserID(ctx, "u1").Return(&model.User{ID: "u1"}, nil)
		authSvcWithMockToken := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{
			parseRes:   &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "u1"}},
			findRes:    &model.AuthSession{Revoked: false, ExpiresAt: time.Now().Add(time.Hour)},
			refreshErr: errors.New("refresh error"),
		}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockToken.RefreshToken(ctx, "refresh", "", "", "")
		if err == nil || err.Error() != "refresh error" {
			t.Errorf("expected refresh error, got %v", err)
		}
	})

	t.Run("Revoke consumed refresh token fails", func(t *testing.T) {
		userRepo.EXPECT().GetByUserID(ctx, "u1").Return(&model.User{ID: "u1"}, nil)
		authSvcWithMockToken := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, &mockTokenProvider{
			parseRes:  &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "u1"}},
			findRes:   &model.AuthSession{Revoked: false, ExpiresAt: time.Now().Add(time.Hour)},
			revokeErr: errors.New("revoke error"),
		}, nil, nil, 5*time.Minute, nil, nil, nil)
		_, err := authSvcWithMockToken.RefreshToken(ctx, "refresh", "", "", "")
		if err == nil || err.Error() != "revoke error" {
			t.Errorf("expected revoke error, got %v", err)
		}
	})
}

// TestAuthService_Logout covers refresh-token revocation and ignored invalid token paths.
func TestAuthService_Logout(t *testing.T) {
	ctx := context.Background()

	t.Run("Happy path", func(t *testing.T) {
		authSvc := authsvc.NewAuthService(
			nil,
			nil,
			&mockTokenProvider{
				parseRes: &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "u1"}},
			},
			nil,
			nil,
			5*time.Minute,
			nil,
			nil,
			nil,
		)

		err := authSvc.Logout(ctx, "refresh")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})

	t.Run("Invalid token is ignored", func(t *testing.T) {
		authSvc := authsvc.NewAuthService(
			nil,
			nil,
			&mockTokenProvider{parseErr: errors.New("parse error")},
			nil,
			nil,
			5*time.Minute,
			nil,
			nil,
			nil,
		)

		err := authSvc.Logout(ctx, "refresh")
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})

	t.Run("Revoke error is returned", func(t *testing.T) {
		authSvc := authsvc.NewAuthService(
			nil,
			nil,
			&mockTokenProvider{
				parseRes:  &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: "u1"}},
				revokeErr: errors.New("revoke error"),
			},
			nil,
			nil,
			5*time.Minute,
			nil,
			nil,
			nil,
		)

		err := authSvc.Logout(ctx, "refresh")
		if err == nil || err.Error() != "revoke error" {
			t.Errorf("expected revoke error, got %v", err)
		}
	})
}

// TestAuthService_UpdateProfile covers profile-only and password-change update paths.
func TestAuthService_UpdateProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("Profile fields only", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userRepo := mock_model.NewMockUserRepository(ctrl)
		authSvc := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, nil, nil, nil, 5*time.Minute, nil, nil, nil)

		fullName := "New Name"
		phone := "0912345678"
		userRepo.EXPECT().
			UpdateUser(ctx, "u1", gomock.Any()).
			Return(&model.User{ID: "u1", Email: "test@test.com", Role: model.RoleManager, FullName: fullName, Phone: phone, IsActivated: true}, nil)

		out, err := authSvc.UpdateProfile(ctx, "u1", authsvc.UpdateProfileInput{FullName: &fullName, Phone: &phone})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if out.FullName != fullName || out.Phone != phone {
			t.Errorf("unexpected profile output: %+v", out)
		}
	})

	t.Run("Password change", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userRepo := mock_model.NewMockUserRepository(ctrl)
		authSvc := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, nil, nil, nil, 5*time.Minute, nil, nil, nil)

		oldPassword := "old"
		newPassword := "new"
		userRepo.EXPECT().GetByUserID(ctx, "u1").Return(&model.User{ID: "u1", PasswordHash: "hashed-old"}, nil)
		userRepo.EXPECT().UpdateUser(ctx, "u1", gomock.Any()).DoAndReturn(func(_ context.Context, _ string, input model.UpdateUserInput) (*model.User, error) {
			if input.PasswordHash == nil || *input.PasswordHash != "hashed-new" {
				t.Errorf("expected password hash to be updated, got %+v", input.PasswordHash)
			}
			return &model.User{ID: "u1", Email: "test@test.com", Role: model.RoleManager}, nil
		})

		_, err := authSvc.UpdateProfile(ctx, "u1", authsvc.UpdateProfileInput{OldPassword: &oldPassword, Password: &newPassword})
		if err != nil {
			t.Errorf("unexpected err: %v", err)
		}
	})

	t.Run("Missing old password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userRepo := mock_model.NewMockUserRepository(ctrl)
		authSvc := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, nil, nil, nil, 5*time.Minute, nil, nil, nil)

		newPassword := "new"
		_, err := authSvc.UpdateProfile(ctx, "u1", authsvc.UpdateProfileInput{Password: &newPassword})
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("Invalid old password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userRepo := mock_model.NewMockUserRepository(ctrl)
		authSvc := authsvc.NewAuthService(userRepo, &mockPasswordHasher{compareErr: errors.New("compare failed")}, nil, nil, nil, 5*time.Minute, nil, nil, nil)

		oldPassword := "old"
		newPassword := "new"
		userRepo.EXPECT().GetByUserID(ctx, "u1").Return(&model.User{ID: "u1", PasswordHash: "hashed-old"}, nil)

		_, err := authSvc.UpdateProfile(ctx, "u1", authsvc.UpdateProfileInput{OldPassword: &oldPassword, Password: &newPassword})
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("User lookup fails during password change", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userRepo := mock_model.NewMockUserRepository(ctrl)
		authSvc := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, nil, nil, nil, 5*time.Minute, nil, nil, nil)

		oldPassword := "old"
		newPassword := "new"
		userRepo.EXPECT().GetByUserID(ctx, "u1").Return(nil, errors.New("db error"))

		_, err := authSvc.UpdateProfile(ctx, "u1", authsvc.UpdateProfileInput{OldPassword: &oldPassword, Password: &newPassword})
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("Hash new password fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userRepo := mock_model.NewMockUserRepository(ctrl)
		authSvc := authsvc.NewAuthService(userRepo, &mockPasswordHasher{hashErr: errors.New("hash error")}, nil, nil, nil, 5*time.Minute, nil, nil, nil)

		oldPassword := "old"
		newPassword := "new"
		userRepo.EXPECT().GetByUserID(ctx, "u1").Return(&model.User{ID: "u1", PasswordHash: "hashed-old"}, nil)

		_, err := authSvc.UpdateProfile(ctx, "u1", authsvc.UpdateProfileInput{OldPassword: &oldPassword, Password: &newPassword})
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("Update user fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		userRepo := mock_model.NewMockUserRepository(ctrl)
		authSvc := authsvc.NewAuthService(userRepo, &mockPasswordHasher{}, nil, nil, nil, 5*time.Minute, nil, nil, nil)

		fullName := "New Name"
		userRepo.EXPECT().UpdateUser(ctx, "u1", gomock.Any()).Return(nil, errors.New("db error"))

		_, err := authSvc.UpdateProfile(ctx, "u1", authsvc.UpdateProfileInput{FullName: &fullName})
		if err == nil {
			t.Errorf("expected error")
		}
	})
}

func TestAuthService_GetMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)

	authSvc := authsvc.NewAuthService(
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
		if err != authsvc.ErrInvalidCredentials {
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
