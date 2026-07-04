package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/service/geo"
	"github.com/mihb123/quanly-phongtro/internal/service/shared"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type TokenProvider interface {
	GenerateAccessToken(role, email, userID string, isActivated bool, jkt string) (string, error)
	GenerateRefreshToken(ctx context.Context, userID, ipAddress, userAgent, location, jkt string, latitude, longitude *float64, geocodingSource *string) (string, error)
	UpdateSessionLocation(ctx context.Context, refreshToken, userID, location string, geocodingSource *string) error
	RevokeRefreshToken(ctx context.Context, token string, userID string) error
	FindByToken(ctx context.Context, token string, userID string) (*model.AuthSession, error)
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
	otpCheck          model.OTPCheckRepository
	geoip             geo.GeoIPService
	geocoding         geo.GeocodingService
}

type AuthService interface {
	Register(ctx context.Context, in RegisterInput, ipAddress, userAgent, jkt string) (*LoginOutput, error)
	Login(ctx context.Context, in LoginInput, ipAddress, userAgent, jkt string) (*LoginOutput, error)
	RefreshToken(ctx context.Context, refreshToken, ipAddress, userAgent, jkt string) (*LoginOutput, error)
	Logout(ctx context.Context, refreshToken string) error
	GetMe(ctx context.Context, userID string) (*AuthOutput, error)
	CreateOTP(ctx context.Context, email string) error
	VerifyEmail(ctx context.Context, email, otp, jkt string) (string, bool, error)
	IncrementOTPCheck(ctx context.Context, email string) error
	IsBlockOTP(ctx context.Context, email string) (bool, error)
	UpdateProfile(ctx context.Context, userID string, in UpdateProfileInput) (*AuthOutput, error)
}

type UpdateProfileInput struct {
	FullName    *string `json:"full_name"`
	Phone       *string `json:"phone"`
	Theme       *string `json:"theme"`
	ColorMode   *string `json:"color_mode"`
	OldPassword *string `json:"old_password"`
	Password    *string `json:"password"`
}

type RegisterInput struct {
	Email     string
	Password  string
	FullName  string
	Phone     string
	Latitude  *float64
	Longitude *float64
}

type LoginInput struct {
	Email     string
	Password  string
	Latitude  *float64
	Longitude *float64
}

type LoginOutput struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresIn    int64       `json:"expires_in"`
	User         *AuthOutput `json:"user,omitempty"`
}

type AuthOutput struct {
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	FullName    string `json:"full_name"`
	Phone       string `json:"phone"`
	IsActivated bool   `json:"is_activated"`
	Theme       string `json:"theme"`
	ColorMode   string `json:"color_mode"`
	AccessToken string `json:"access_token,omitempty"`
}

func NewAuthService(users model.UserRepository, hasher PasswordHasher, tokens TokenProvider, verifyEmailRepo model.EmailVerificationRepository, emailSender EmailSender, otpExpiresIn time.Duration, otpCheck model.OTPCheckRepository, geoip geo.GeoIPService, geocoding geo.GeocodingService) *AuthServiceImpl {
	return &AuthServiceImpl{
		users:             users,
		hasher:            hasher,
		tokens:            tokens,
		emailVerification: verifyEmailRepo,
		emailSender:       emailSender,
		OTPExpiresIn:      otpExpiresIn,
		otpCheck:          otpCheck,
		geoip:             geoip,
		geocoding:         geocoding,
	}
}

// newAuthOutput maps a user model to the public profile returned to clients.
// It intentionally omits secret fields (password hash, zalo secrets).
func newAuthOutput(user *model.User) *AuthOutput {
	return &AuthOutput{
		UserID:      user.ID,
		Email:       user.Email,
		Role:        string(user.Role),
		FullName:    user.FullName,
		Phone:       user.Phone,
		IsActivated: user.IsActivated,
		Theme:       user.Theme,
		ColorMode:   user.ColorMode,
	}
}

// resolveInitialLocation returns a best-effort location using only the local
// GeoIP database. It performs no network I/O, so it is safe on the login path.
func (s *AuthServiceImpl) resolveInitialLocation(ipAddress string) string {
	if s.geoip != nil {
		return s.geoip.LookupLocation(ipAddress)
	}
	return "Unknown"
}

// refineSessionLocation reverse-geocodes GPS coordinates and updates the stored
// session location. It runs in a background goroutine off the login critical
// path, so a failure simply leaves the faster GeoIP location in place. It uses
// its own context because the originating request context is already done.
func (s *AuthServiceImpl) refineSessionLocation(refreshToken, userID string, lat, lng float64) {
	if s.geocoding == nil {
		return
	}
	addr, source, err := s.geocoding.ReverseGeocode(lat, lng)
	if err != nil {
		logger.Warn(nil, 0, "background reverse geocode failed", err)
		return
	}
	if err := s.tokens.UpdateSessionLocation(context.Background(), refreshToken, userID, addr, &source); err != nil {
		logger.Error(nil, 0, "failed to update session location", err)
	}
}

func (s *AuthServiceImpl) Register(ctx context.Context, in RegisterInput, ipAddress, userAgent, jkt string) (*LoginOutput, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))

	if !IsValidEmail(email) {
		return nil, shared.ErrInvalidInput
	}

	_, err := s.users.GetByEmail(ctx, email)
	if err == nil {
		return nil, ErrEmailAlreadyExists
	}

	if !errors.Is(err, model.ErrNotFound) {
		return nil, err
	}

	passwordHash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	newUser := &model.User{
		Email:        in.Email,
		PasswordHash: passwordHash,
		Role:         model.RoleManager,
		FullName:     in.FullName,
		Phone:        in.Phone,
		IsActivated:  false,
	}

	if err := s.users.Create(ctx, newUser); err != nil {
		if errors.Is(err, model.ErrAlreadyExists) {
			return nil, ErrEmailAlreadyExists
		}

		return nil, err
	}

	accessToken, err := s.tokens.GenerateAccessToken(string(newUser.Role), newUser.Email, newUser.ID, newUser.IsActivated, jkt)
	if err != nil {
		return nil, err
	}

	location := s.resolveInitialLocation(ipAddress)

	refreshToken, err := s.tokens.GenerateRefreshToken(ctx, newUser.ID, ipAddress, userAgent, location, jkt, in.Latitude, in.Longitude, nil)
	if err != nil {
		return nil, err
	}

	// Reverse-geocode precise GPS coordinates off the critical path.
	if in.Latitude != nil && in.Longitude != nil {
		go s.refineSessionLocation(refreshToken, newUser.ID, *in.Latitude, *in.Longitude)
	}

	ttl := s.tokens.GetAccessTokenTTL()

	return &LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(ttl.Seconds()),
		User:         newAuthOutput(newUser),
	}, nil
}

func (s *AuthServiceImpl) Login(ctx context.Context, in LoginInput, ipAddress, userAgent, jkt string) (*LoginOutput, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	password := strings.TrimSpace(in.Password)

	if !IsValidEmail(email) || password == "" {
		return nil, shared.ErrInvalidInput
	}

	existingUser, err := s.users.GetAuthUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, err
	}

	if err := s.hasher.Compare(existingUser.PasswordHash, in.Password); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.tokens.GenerateAccessToken(string(existingUser.Role), existingUser.Email, existingUser.ID, existingUser.IsActivated, jkt)
	if err != nil {
		return nil, err
	}

	location := s.resolveInitialLocation(ipAddress)

	refreshToken, err := s.tokens.GenerateRefreshToken(ctx, existingUser.ID, ipAddress, userAgent, location, jkt, in.Latitude, in.Longitude, nil)
	if err != nil {
		return nil, err
	}

	// Reverse-geocode precise GPS coordinates off the critical path.
	if in.Latitude != nil && in.Longitude != nil {
		go s.refineSessionLocation(refreshToken, existingUser.ID, *in.Latitude, *in.Longitude)
	}

	ttl := s.tokens.GetAccessTokenTTL()

	return &LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(ttl.Seconds()),
		User:         newAuthOutput(existingUser),
	}, nil
}

func IsValidEmail(email string) bool {
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

func (s *AuthServiceImpl) RefreshToken(ctx context.Context, refreshToken, ipAddress, userAgent, jkt string) (*LoginOutput, error) {
	claims, err := s.tokens.Parse(refreshToken, "refresh")
	if err != nil {
		return nil, err
	}

	userID, err := claims.GetSubject()
	if err != nil {
		return nil, err
	}

	session, err := s.tokens.FindByToken(ctx, refreshToken, userID)
	if err != nil {
		return nil, err
	}

	if session == nil || session.Revoked || time.Now().After(session.ExpiresAt) {
		return nil, ErrInvalidCredentials
	}

	if session.JKT != jkt && session.JKT != "" {
		return nil, ErrInvalidCredentials
	}
	user, err := s.users.GetByUserID(ctx, userID)

	if err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.tokens.GenerateAccessToken(string(user.Role), user.Email, user.ID, user.IsActivated, jkt)
	if err != nil {
		return nil, err
	}

	location := "Unknown"
	if s.geoip != nil {
		location = s.geoip.LookupLocation(ipAddress)
	}

	newRefreshToken, err := s.tokens.GenerateRefreshToken(ctx, user.ID, ipAddress, userAgent, location, jkt, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	if err := s.tokens.RevokeRefreshToken(ctx, refreshToken, user.ID); err != nil {
		return nil, err
	}

	return &LoginOutput{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(s.tokens.GetAccessTokenTTL().Seconds()),
	}, nil
}

func (s *AuthServiceImpl) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.tokens.Parse(refreshToken, "refresh")
	if err != nil {
		// If it's already invalid/expired, we don't care
		return nil
	}

	userID, err := claims.GetSubject()
	if err != nil {
		return nil
	}

	return s.tokens.RevokeRefreshToken(ctx, refreshToken, userID)
}

func (s *AuthServiceImpl) GetMe(ctx context.Context, userID string) (*AuthOutput, error) {
	user, err := s.users.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	return newAuthOutput(user), nil
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

func (s *AuthServiceImpl) VerifyEmail(ctx context.Context, email, otp, jkt string) (string, bool, error) {
	emailVeri := &model.EmailVerification{
		Email: email,
		OTP:   otp,
	}

	err := s.emailVerification.GetOTP(ctx, emailVeri)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false, nil
		}
		return "", false, err
	}
	if time.Since(emailVeri.Expires) > s.OTPExpiresIn || emailVeri.IsUsed {
		return "", false, nil
	}

	err = s.emailVerification.UpdateUsedOTP(ctx, emailVeri)
	if err != nil {
		return "", false, errors.New("cannot update used otp")
	}
	err = s.users.ActivateUser(ctx, emailVeri.Email)
	if err != nil {
		return "", false, err
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return "", false, err
	}

	accessToken, err := s.tokens.GenerateAccessToken(string(user.Role), user.Email, user.ID, true, jkt)
	if err != nil {
		return "", false, err
	}

	return accessToken, true, nil
}

func (s *AuthServiceImpl) IncrementOTPCheck(ctx context.Context, email string) error {
	otpCheck, err := s.otpCheck.GetOTPCheck(ctx, email)
	if err == sql.ErrNoRows {
		err := s.otpCheck.CreateOTPCheck(ctx, email)
		if err != nil {
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	err = s.otpCheck.IncrementOTPCheck(ctx, email, otpCheck.OTPFails+1)
	return err
}

func (s *AuthServiceImpl) IsBlockOTP(ctx context.Context, email string) (bool, error) {
	otpCheck, err := s.otpCheck.GetOTPCheck(ctx, email)
	if err != nil && err != sql.ErrNoRows {
		return true, err
	}
	if otpCheck.OTPFails == 5 {
		if time.Since(otpCheck.BlockTime) > time.Duration(15)*time.Minute {
			err = s.otpCheck.ResetOTP(ctx, email)
			if err != nil {
				return true, err
			}
			return false, nil
		} else {
			return true, nil
		}
	}
	return false, nil
}

// validThemes lists the color themes the UI supports. Kept in sync with the
// frontend theme registry (see frontend/src/lib/theme.ts).
var validThemes = map[string]bool{"lime": true, "sky": true}

// isValidTheme reports whether the given id is a supported UI color theme.
func isValidTheme(theme string) bool {
	return validThemes[theme]
}

// validColorModes lists the light/dark appearance modes the UI supports. Kept
// in sync with the frontend registry (see frontend/src/lib/colorMode.ts).
var validColorModes = map[string]bool{"system": true, "light": true, "dark": true}

// isValidColorMode reports whether the given id is a supported appearance mode.
func isValidColorMode(mode string) bool {
	return validColorModes[mode]
}

func (s *AuthServiceImpl) UpdateProfile(ctx context.Context, userID string, in UpdateProfileInput) (*AuthOutput, error) {
	updateInput := model.UpdateUserInput{
		FullName: in.FullName,
		Phone:    in.Phone,
	}

	// Persist the chosen UI theme so it stays consistent across the user's devices.
	if in.Theme != nil {
		theme := strings.TrimSpace(*in.Theme)
		if !isValidTheme(theme) {
			return nil, fmt.Errorf("invalid theme %q: %w", theme, shared.ErrInvalidInput)
		}
		updateInput.Theme = &theme
	}

	// Persist the chosen appearance mode so light/dark stays consistent per user.
	if in.ColorMode != nil {
		mode := strings.TrimSpace(*in.ColorMode)
		if !isValidColorMode(mode) {
			return nil, fmt.Errorf("invalid color mode %q: %w", mode, shared.ErrInvalidInput)
		}
		updateInput.ColorMode = &mode
	}

	if in.Password != nil && *in.Password != "" {
		if in.OldPassword == nil || *in.OldPassword == "" {
			return nil, fmt.Errorf("old password is required to set a new password: %w", shared.ErrInvalidInput)
		}

		user, err := s.users.GetByUserID(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("user not found: %w", err)
		}

		if err := s.hasher.Compare(user.PasswordHash, *in.OldPassword); err != nil {
			return nil, fmt.Errorf("invalid old password: %w", ErrInvalidCredentials)
		}

		hash, err := s.hasher.Hash(*in.Password)
		if err != nil {
			return nil, fmt.Errorf("hash password failed: %w", err)
		}
		updateInput.PasswordHash = &hash
	}

	user, err := s.users.UpdateUser(ctx, userID, updateInput)
	if err != nil {
		return nil, fmt.Errorf("update user failed: %w", err)
	}

	return newAuthOutput(user), nil
}
