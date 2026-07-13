package zalo_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mihb123/quanly-phongtro/internal/handler/zalo"
	"github.com/mihb123/quanly-phongtro/internal/mock/mock_service"
	"github.com/mihb123/quanly-phongtro/internal/security"
	zalosvc "github.com/mihb123/quanly-phongtro/internal/service/zalo"
	"go.uber.org/mock/gomock"
)

type errReader struct{}

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func TestZaloHandler_GetPublicKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	zaloSvc := mock_service.NewMockZaloService(ctrl)
	h := zalo.NewZaloHandler(zaloSvc, "test.local")

	tests := []struct {
		name         string
		expectedCode int
	}{
		{
			name:         "Success",
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/zalo/public-key", nil)
			rr := httptest.NewRecorder()
			h.GetPublicKey(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("expected code %d, got %d", tt.expectedCode, rr.Code)
			}

			var res struct {
				PublicKey string `json:"public_key"`
			}
			if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if res.PublicKey == "" {
				t.Errorf("expected non-empty public key")
			}
		})
	}
}

func TestZaloHandler_SaveConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	zaloSvc := mock_service.NewMockZaloService(ctrl)
	h := zalo.NewZaloHandler(zaloSvc, "test.local")

	// Helper to get public key from handler and encrypt token
	reqPK := httptest.NewRequest(http.MethodGet, "/api/v1/zalo/public-key", nil)
	rrPK := httptest.NewRecorder()
	h.GetPublicKey(rrPK, reqPK)
	var resPK struct {
		PublicKey string `json:"public_key"`
	}
	json.NewDecoder(rrPK.Body).Decode(&resPK)
	pubASN1, _ := base64.StdEncoding.DecodeString(resPK.PublicKey)
	pubKey, _ := x509.ParsePKIXPublicKey(pubASN1)
	rsaPubKey := pubKey.(*rsa.PublicKey)

	encryptToken := func(text string) string {
		cipher, _ := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPubKey, []byte(text), nil)
		return base64.StdEncoding.EncodeToString(cipher)
	}

	tests := []struct {
		name         string
		managerID    string
		hasClaims    bool
		payload      interface{}
		setupMock    func()
		expectedCode int
	}{
		{
			name:      "Success",
			managerID: "manager1",
			hasClaims: true,
			payload: map[string]string{
				"bot_token": encryptToken("valid_token"),
			},
			setupMock: func() {
				zaloSvc.EXPECT().SaveZaloConfig(gomock.Any(), "manager1", "valid_token", "test.local/api/v1/zalo/webhooks/manager1", gomock.Any()).Return(nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "Unauthorized - Missing Claims",
			hasClaims:    false,
			payload:      nil,
			setupMock:    func() {},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Invalid request body",
			managerID:    "manager1",
			hasClaims:    true,
			payload:      "invalid json",
			setupMock:    func() {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:      "Invalid bot_token encoding",
			managerID: "manager1",
			hasClaims: true,
			payload: map[string]string{
				"bot_token": "not-base64!!!",
			},
			setupMock:    func() {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:      "Failed to decrypt bot_token",
			managerID: "manager1",
			hasClaims: true,
			payload: map[string]string{
				"bot_token": base64.StdEncoding.EncodeToString([]byte("invalid-cipher-text")),
			},
			setupMock:    func() {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:      "Service Error",
			managerID: "manager1",
			hasClaims: true,
			payload: map[string]string{
				"bot_token": encryptToken("valid_token"),
			},
			setupMock: func() {
				zaloSvc.EXPECT().SaveZaloConfig(gomock.Any(), "manager1", "valid_token", gomock.Any(), gomock.Any()).Return(errors.New("db error"))
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			var bodyBytes []byte
			if str, ok := tt.payload.(string); ok {
				bodyBytes = []byte(str)
			} else if tt.payload != nil {
				bodyBytes, _ = json.Marshal(tt.payload)
			}

			var req *http.Request
			if bodyBytes == nil && tt.payload != nil {
				// Should not happen with current test cases
				req = httptest.NewRequest(http.MethodPost, "/api/v1/zalo/config", nil)
			} else {
				req = httptest.NewRequest(http.MethodPost, "/api/v1/zalo/config", bytes.NewBuffer(bodyBytes))
			}

			if tt.hasClaims {
				claims := &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: tt.managerID}}
				req = req.WithContext(security.WithClaims(req.Context(), claims))
			}
			rr := httptest.NewRecorder()

			h.SaveConfig(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("expected code %d, got %d. Body: %s", tt.expectedCode, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestZaloHandler_GetConfigStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	zaloSvc := mock_service.NewMockZaloService(ctrl)
	h := zalo.NewZaloHandler(zaloSvc, "test.local")

	tests := []struct {
		name         string
		hasClaims    bool
		managerID    string
		setupMock    func()
		expectedCode int
	}{
		{
			name:      "Success",
			hasClaims: true,
			managerID: "manager1",
			setupMock: func() {
				zaloSvc.EXPECT().GetZaloConfigStatus(gomock.Any(), "manager1").Return(zalosvc.ZaloBotStatus{
					HasConfig: true,
					IsActive:  true,
					IsLinked:  true,
					BotID:     "bot123",
					ManagerID: "manager1",
				}, nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "Unauthorized - Missing Claims",
			hasClaims:    false,
			setupMock:    func() {},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:      "Service Error",
			hasClaims: true,
			managerID: "manager1",
			setupMock: func() {
				zaloSvc.EXPECT().GetZaloConfigStatus(gomock.Any(), "manager1").Return(zalosvc.ZaloBotStatus{}, errors.New("db error"))
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			req := httptest.NewRequest(http.MethodGet, "/api/v1/zalo/config", nil)
			if tt.hasClaims {
				claims := &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: tt.managerID}}
				req = req.WithContext(security.WithClaims(req.Context(), claims))
			}
			rr := httptest.NewRecorder()

			h.GetConfigStatus(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("expected code %d, got %d. Body: %s", tt.expectedCode, rr.Code, rr.Body.String())
			}
		})
	}
}

func TestZaloHandler_Webhook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	zaloSvc := mock_service.NewMockZaloService(ctrl)
	h := zalo.NewZaloHandler(zaloSvc, "test.local")

	tests := []struct {
		name         string
		managerID    string
		secret       string
		body         interface{}
		setupMock    func()
		expectedCode int
	}{
		{
			name:      "Success",
			managerID: "manager1",
			secret:    "secret123",
			body:      `{"event":"message"}`,
			setupMock: func() {
				zaloSvc.EXPECT().HandleWebhook(gomock.Any(), "manager1", []byte(`{"event":"message"}`), "secret123").Return(nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name:      "Service Error (still returns 200)",
			managerID: "manager1",
			secret:    "secret123",
			body:      `{"event":"message"}`,
			setupMock: func() {
				zaloSvc.EXPECT().HandleWebhook(gomock.Any(), "manager1", []byte(`{"event":"message"}`), "secret123").Return(errors.New("webhook processing failed"))
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "Body Read Error",
			managerID:    "manager1",
			body:         errReader{},
			setupMock:    func() {},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			var req *http.Request
			if r, ok := tt.body.(errReader); ok {
				req = httptest.NewRequest(http.MethodPost, "/webhook", r)
			} else {
				req = httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer([]byte(tt.body.(string))))
			}
			req.Header.Set("X-Bot-Api-Secret-Token", tt.secret)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("managerID", tt.managerID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()
			h.Webhook(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("expected code %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestZaloHandler_SendMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	zaloSvc := mock_service.NewMockZaloService(ctrl)
	h := zalo.NewZaloHandler(zaloSvc, "test.local")

	tests := []struct {
		name         string
		hasClaims    bool
		managerID    string
		payload      interface{}
		setupMock    func()
		expectedCode int
	}{
		{
			name:      "Success",
			hasClaims: true,
			managerID: "manager1",
			payload: map[string]string{
				"chat_id": "chat123",
				"text":    "hello world",
			},
			setupMock: func() {
				zaloSvc.EXPECT().SendTextMessage(gomock.Any(), "manager1", "chat123", "hello world").Return(nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "Unauthorized - Missing Claims",
			hasClaims:    false,
			setupMock:    func() {},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Invalid request body",
			hasClaims:    true,
			managerID:    "manager1",
			payload:      "invalid json",
			setupMock:    func() {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:      "Service Error",
			hasClaims: true,
			managerID: "manager1",
			payload: map[string]string{
				"chat_id": "chat123",
				"text":    "hello world",
			},
			setupMock: func() {
				zaloSvc.EXPECT().SendTextMessage(gomock.Any(), "manager1", "chat123", "hello world").Return(errors.New("api error"))
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			var bodyBytes []byte
			if str, ok := tt.payload.(string); ok {
				bodyBytes = []byte(str)
			} else if tt.payload != nil {
				bodyBytes, _ = json.Marshal(tt.payload)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/zalo/message", bytes.NewBuffer(bodyBytes))
			if tt.hasClaims {
				claims := &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: tt.managerID}}
				req = req.WithContext(security.WithClaims(req.Context(), claims))
			}
			rr := httptest.NewRecorder()

			h.SendMessage(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("expected code %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}

func TestZaloHandler_SendInvoice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	zaloSvc := mock_service.NewMockZaloService(ctrl)
	h := zalo.NewZaloHandler(zaloSvc, "test.local")

	tests := []struct {
		name         string
		hasClaims    bool
		managerID    string
		invoiceID    string
		setupMock    func()
		expectedCode int
	}{
		{
			name:      "Success",
			hasClaims: true,
			managerID: "manager1",
			invoiceID: "inv123",
			setupMock: func() {
				zaloSvc.EXPECT().SendInvoiceToZalo(gomock.Any(), "manager1", "inv123").Return(nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "Unauthorized - Missing Claims",
			hasClaims:    false,
			setupMock:    func() {},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Missing invoice ID",
			hasClaims:    true,
			managerID:    "manager1",
			invoiceID:    "", // missing
			setupMock:    func() {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name:      "Service Error",
			hasClaims: true,
			managerID: "manager1",
			invoiceID: "inv123",
			setupMock: func() {
				zaloSvc.EXPECT().SendInvoiceToZalo(gomock.Any(), "manager1", "inv123").Return(errors.New("failed to send invoice"))
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			req := httptest.NewRequest(http.MethodPost, "/api/v1/zalo/invoice", nil)
			if tt.hasClaims {
				claims := &security.Claims{RegisteredClaims: jwt.RegisteredClaims{Subject: tt.managerID}}
				req = req.WithContext(security.WithClaims(req.Context(), claims))
			}

			rctx := chi.NewRouteContext()
			if tt.invoiceID != "" {
				rctx.URLParams.Add("id", tt.invoiceID)
			}
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()
			h.SendInvoice(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("expected code %d, got %d", tt.expectedCode, rr.Code)
			}
		})
	}
}
