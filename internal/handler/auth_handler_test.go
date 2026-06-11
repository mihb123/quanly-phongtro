package handler_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mihb123/quanly-phongtro/internal/handler"
	"github.com/mihb123/quanly-phongtro/internal/mock/mock_service"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service"
	"go.uber.org/mock/gomock"
)

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockSetup  func(mockSvc *mock_service.MockAuthService)
		wantStatus int
	}{
		{
			name: "Happy path",
			body: `{"email": "test@test.local", "password": "password123"}`,
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&service.LoginOutput{}, nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "Invalid body format",
			body:       `invalid json`,
			mockSetup:  func(mockSvc *mock_service.MockAuthService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Validation error",
			body:       `{"email": "invalid-email", "password": "123"}`,
			mockSetup:  func(mockSvc *mock_service.MockAuthService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Service error - Email already exists",
			body: `{"email": "test@test.local", "password": "password123"}`,
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, service.ErrEmailAlreadyExists)
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "Service error - Invalid input",
			body: `{"email": "test@test.local", "password": "password123"}`,
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, service.ErrInvalidInput)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Service error - Unexpected error",
			body: `{"email": "test@test.local", "password": "password123"}`,
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("some error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSvc := mock_service.NewMockAuthService(ctrl)
			tt.mockSetup(mockSvc)

			h := handler.NewAuthHandler(mockSvc)

			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer([]byte(tt.body)))
			rec := httptest.NewRecorder()

			h.Register(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockSetup  func(mockSvc *mock_service.MockAuthService)
		wantStatus int
	}{
		{
			name: "Happy path",
			body: `{"email": "test@test.local", "password": "password123"}`,
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().Login(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&service.LoginOutput{
					AccessToken:  "access",
					RefreshToken: "refresh",
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "Invalid body format",
			body:       `invalid json`,
			mockSetup:  func(mockSvc *mock_service.MockAuthService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Validation error",
			body:       `{}`,
			mockSetup:  func(mockSvc *mock_service.MockAuthService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Service error - Invalid credentials",
			body: `{"email": "test@test.local", "password": "password123"}`,
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().Login(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, service.ErrInvalidCredentials)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "Service error - Invalid input",
			body: `{"email": "test@test.local", "password": "password123"}`,
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().Login(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, service.ErrInvalidInput)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Service error - Unexpected error",
			body: `{"email": "test@test.local", "password": "password123"}`,
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().Login(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("some error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSvc := mock_service.NewMockAuthService(ctrl)
			tt.mockSetup(mockSvc)

			h := handler.NewAuthHandler(mockSvc)

			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer([]byte(tt.body)))
			rec := httptest.NewRecorder()

			h.Login(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		mockSetup  func(mockSvc *mock_service.MockAuthService)
		wantStatus int
	}{
		{
			name: "Happy path",
			body: `{"refresh_token": "token123"}`,
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().RefreshToken(gomock.Any(), "token123", gomock.Any(), gomock.Any(), gomock.Any()).Return(&service.LoginOutput{
					AccessToken:  "access",
					RefreshToken: "refresh",
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "Invalid body format",
			body:       `invalid json`,
			mockSetup:  func(mockSvc *mock_service.MockAuthService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Validation error",
			body:       `{}`,
			mockSetup:  func(mockSvc *mock_service.MockAuthService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Service error - Invalid refresh token",
			body: `{"refresh_token": "token123"}`,
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().RefreshToken(gomock.Any(), "token123", gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, service.ErrInvalidCredentials)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "Service error - Invalid input",
			body: `{"refresh_token": "token123"}`,
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().RefreshToken(gomock.Any(), "token123", gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, service.ErrInvalidInput)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Service error - Unexpected error",
			body: `{"refresh_token": "token123"}`,
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().RefreshToken(gomock.Any(), "token123", gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("some error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSvc := mock_service.NewMockAuthService(ctrl)
			tt.mockSetup(mockSvc)

			h := handler.NewAuthHandler(mockSvc)

			req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
			if tt.body != "invalid json" && tt.body != "{}" && tt.body != "" {
				req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "token123"})
			}
			rec := httptest.NewRecorder()

			h.RefreshToken(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestAuthHandler_GetMe(t *testing.T) {
	tests := []struct {
		name       string
		setupCtx   func(req *http.Request) *http.Request
		mockSetup  func(mockSvc *mock_service.MockAuthService)
		wantStatus int
	}{
		{
			name: "Happy path",
			setupCtx: func(req *http.Request) *http.Request {
				claims := &security.Claims{
					RegisteredClaims: jwt.RegisteredClaims{
						Subject: "user-1",
					},
				}
				return req.WithContext(security.WithClaims(req.Context(), claims))
			},
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().GetMe(gomock.Any(), "user-1").Return(&service.AuthOutput{}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "Unauthorized - missing claims",
			setupCtx: func(req *http.Request) *http.Request {
				return req // no claims in context
			},
			mockSetup:  func(mockSvc *mock_service.MockAuthService) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "Service error - user not found",
			setupCtx: func(req *http.Request) *http.Request {
				claims := &security.Claims{
					RegisteredClaims: jwt.RegisteredClaims{
						Subject: "user-1",
					},
				}
				return req.WithContext(security.WithClaims(req.Context(), claims))
			},
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().GetMe(gomock.Any(), "user-1").Return(nil, errors.New("not found"))
			},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSvc := mock_service.NewMockAuthService(ctrl)
			tt.mockSetup(mockSvc)

			h := handler.NewAuthHandler(mockSvc)

			req := httptest.NewRequest(http.MethodGet, "/me", nil)
			req = tt.setupCtx(req)
			rec := httptest.NewRecorder()

			h.GetMe(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	t.Run("Happy path", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockSvc := mock_service.NewMockAuthService(ctrl)
		h := handler.NewAuthHandler(mockSvc)

		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		rec := httptest.NewRecorder()

		h.Logout(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected status %d, got %d", http.StatusNoContent, rec.Code)
		}
	})
}

func TestAuthHandler_CreateOTP(t *testing.T) {
	tests := []struct {
		name       string
		setupCtx   func(req *http.Request) *http.Request
		mockSetup  func(mockSvc *mock_service.MockAuthService)
		wantStatus int
	}{
		{
			name: "Happy path",
			setupCtx: func(req *http.Request) *http.Request {
				claims := &security.Claims{
					Email:       "test@test.local",
					IsActivated: false,
				}
				return req.WithContext(security.WithClaims(req.Context(), claims))
			},
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().CreateOTP(gomock.Any(), "test@test.local").Return(nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "Unauthorized - missing claims",
			setupCtx: func(req *http.Request) *http.Request {
				return req
			},
			mockSetup:  func(mockSvc *mock_service.MockAuthService) {},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Validation error - account already activated",
			setupCtx: func(req *http.Request) *http.Request {
				claims := &security.Claims{
					Email:       "test@test.local",
					IsActivated: true,
				}
				return req.WithContext(security.WithClaims(req.Context(), claims))
			},
			mockSetup:  func(mockSvc *mock_service.MockAuthService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Service error - CreateOTP fails",
			setupCtx: func(req *http.Request) *http.Request {
				claims := &security.Claims{
					Email:       "test@test.local",
					IsActivated: false,
				}
				return req.WithContext(security.WithClaims(req.Context(), claims))
			},
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().CreateOTP(gomock.Any(), "test@test.local").Return(errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSvc := mock_service.NewMockAuthService(ctrl)
			tt.mockSetup(mockSvc)

			h := handler.NewAuthHandler(mockSvc)

			req := httptest.NewRequest(http.MethodPost, "/create-otp", nil)
			req = tt.setupCtx(req)
			rec := httptest.NewRecorder()

			h.CreateOTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestAuthHandler_VerifyEmail(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setupCtx   func(req *http.Request) *http.Request
		mockSetup  func(mockSvc *mock_service.MockAuthService)
		wantStatus int
	}{
		{
			name: "Happy path",
			body: `{"otp": "123456"}`,
			setupCtx: func(req *http.Request) *http.Request {
				claims := &security.Claims{Email: "test@test.local", IsActivated: false}
				return req.WithContext(security.WithClaims(context.Background(), claims))
			},
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().IsBlockOTP(gomock.Any(), "test@test.local").Return(false, nil)
				mockSvc.EXPECT().VerifyEmail(gomock.Any(), "test@test.local", "123456", gomock.Any()).Return("access_token", true, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "Unauthorized - missing claims",
			body: `{"otp": "123456"}`,
			setupCtx: func(req *http.Request) *http.Request {
				return req
			},
			mockSetup:  func(mockSvc *mock_service.MockAuthService) {},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Validation error - account already activated",
			body: `{"otp": "123456"}`,
			setupCtx: func(req *http.Request) *http.Request {
				claims := &security.Claims{Email: "test@test.local", IsActivated: true}
				return req.WithContext(security.WithClaims(context.Background(), claims))
			},
			mockSetup:  func(mockSvc *mock_service.MockAuthService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid body format",
			body: `invalid json`,
			setupCtx: func(req *http.Request) *http.Request {
				claims := &security.Claims{Email: "test@test.local", IsActivated: false}
				return req.WithContext(security.WithClaims(context.Background(), claims))
			},
			mockSetup:  func(mockSvc *mock_service.MockAuthService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Validation error - empty body",
			body: `{}`,
			setupCtx: func(req *http.Request) *http.Request {
				claims := &security.Claims{Email: "test@test.local", IsActivated: false}
				return req.WithContext(security.WithClaims(context.Background(), claims))
			},
			mockSetup:  func(mockSvc *mock_service.MockAuthService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Service error - IsBlockOTP fails",
			body: `{"otp": "123456"}`,
			setupCtx: func(req *http.Request) *http.Request {
				claims := &security.Claims{Email: "test@test.local", IsActivated: false}
				return req.WithContext(security.WithClaims(context.Background(), claims))
			},
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().IsBlockOTP(gomock.Any(), "test@test.local").Return(false, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Service error - IsBlockOTP returns true (Blocked)",
			body: `{"otp": "123456"}`,
			setupCtx: func(req *http.Request) *http.Request {
				claims := &security.Claims{Email: "test@test.local", IsActivated: false}
				return req.WithContext(security.WithClaims(context.Background(), claims))
			},
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().IsBlockOTP(gomock.Any(), "test@test.local").Return(true, nil)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "Service error - VerifyEmail fails",
			body: `{"otp": "123456"}`,
			setupCtx: func(req *http.Request) *http.Request {
				claims := &security.Claims{Email: "test@test.local", IsActivated: false}
				return req.WithContext(security.WithClaims(context.Background(), claims))
			},
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().IsBlockOTP(gomock.Any(), "test@test.local").Return(false, nil)
				mockSvc.EXPECT().VerifyEmail(gomock.Any(), "test@test.local", "123456", gomock.Any()).Return("", false, errors.New("service error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "Unauthorized - Invalid OTP attempt",
			body: `{"otp": "123456"}`,
			setupCtx: func(req *http.Request) *http.Request {
				claims := &security.Claims{Email: "test@test.local", IsActivated: false}
				return req.WithContext(security.WithClaims(context.Background(), claims))
			},
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().IsBlockOTP(gomock.Any(), "test@test.local").Return(false, nil)
				mockSvc.EXPECT().VerifyEmail(gomock.Any(), "test@test.local", "123456", gomock.Any()).Return("", false, nil)
				mockSvc.EXPECT().IncrementOTPCheck(gomock.Any(), "test@test.local").Return(nil)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "Unauthorized - Invalid OTP attempt (Increment fails)",
			body: `{"otp": "123456"}`,
			setupCtx: func(req *http.Request) *http.Request {
				claims := &security.Claims{Email: "test@test.local", IsActivated: false}
				return req.WithContext(security.WithClaims(context.Background(), claims))
			},
			mockSetup: func(mockSvc *mock_service.MockAuthService) {
				mockSvc.EXPECT().IsBlockOTP(gomock.Any(), "test@test.local").Return(false, nil)
				mockSvc.EXPECT().VerifyEmail(gomock.Any(), "test@test.local", "123456", gomock.Any()).Return("", false, nil)
				mockSvc.EXPECT().IncrementOTPCheck(gomock.Any(), "test@test.local").Return(errors.New("db error"))
			},
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSvc := mock_service.NewMockAuthService(ctrl)
			tt.mockSetup(mockSvc)

			h := handler.NewAuthHandler(mockSvc)

			req := httptest.NewRequest(http.MethodPost, "/verify-email", bytes.NewBuffer([]byte(tt.body)))
			req = tt.setupCtx(req)
			rec := httptest.NewRecorder()

			h.VerifyEmail(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}
