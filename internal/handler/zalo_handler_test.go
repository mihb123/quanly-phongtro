package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/handler"
	"github.com/mihb123/quanly-phongtro/internal/mock/mock_service"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service"
	"go.uber.org/mock/gomock"
)

func TestZaloHandler_GetPublicKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	zaloSvc := mock_service.NewMockZaloService(ctrl)
	zaloHandler := handler.NewZaloHandler(zaloSvc, "test.local")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/zalo/public-key", nil)
	rr := httptest.NewRecorder()

	zaloHandler.GetPublicKey(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var res struct {
		PublicKey string `json:"public_key"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if res.PublicKey == "" {
		t.Errorf("expected non-empty public key")
	}
}

func TestZaloHandler_GetConfigStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	zaloSvc := mock_service.NewMockZaloService(ctrl)
	zaloHandler := handler.NewZaloHandler(zaloSvc, "test.local")

	managerID := "manager1"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/zalo/config", nil)
	claims := &security.Claims{}
	claims.Subject = managerID
	req = req.WithContext(security.WithClaims(req.Context(), claims))
	rr := httptest.NewRecorder()

	zaloSvc.EXPECT().GetZaloConfigStatus(req.Context(), managerID).Return(service.ZaloBotStatus{
		HasConfig: true,
		IsActive:  true,
	}, nil)

	zaloHandler.GetConfigStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var res struct {
		HasConfig     bool `json:"has_config"`
		IsZaloBotActive bool `json:"is_zalo_bot_active"`
	}
	json.NewDecoder(rr.Body).Decode(&res)
	
	if !res.HasConfig {
		t.Errorf("expected has_config to be true")
	}
	if !res.IsZaloBotActive {
		t.Errorf("expected is_zalo_bot_active to be true")
	}
}

func TestZaloHandler_SaveConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	zaloSvc := mock_service.NewMockZaloService(ctrl)
	zaloHandler := handler.NewZaloHandler(zaloSvc, "test.local")

	// First we need to get the public key to encrypt the payload, since the handler expects RSA encrypted payload
	reqPK := httptest.NewRequest(http.MethodGet, "/api/v1/zalo/public-key", nil)
	rrPK := httptest.NewRecorder()
	zaloHandler.GetPublicKey(rrPK, reqPK)

	// Since we mock the service, we can bypass RSA encryption by directly parsing out how the handler does it
	// Actually, wait, handler generates its own RSA key internally, we can't easily mock that part.
	// We'll just pass invalid payload and expect Bad Request? 
	// No, let's just test Webhook and ConfigStatus, and skip SaveConfig's complex RSA setup, OR extract the private key via reflection.
	// Let's actually encrypt correctly by reusing the exact public key bytes from the handler response? It's ASN1 format.
	// Let's just test the basic error path for now to save time, or we can parse the public key.
	// We can use the GetPublicKey response!
}

func TestZaloHandler_Webhook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	zaloSvc := mock_service.NewMockZaloService(ctrl)
	zaloHandler := handler.NewZaloHandler(zaloSvc, "test.local")

	managerID := "manager1"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/zalo/webhooks/"+managerID, bytes.NewBuffer([]byte(`{"event":"test"}`)))
	req.Header.Set("X-Bot-Api-Secret-Token", "secret123")
	
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("managerID", managerID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()

	zaloSvc.EXPECT().HandleWebhook(gomock.Any(), managerID, []byte(`{"event":"test"}`), "secret123").Return(nil)

	zaloHandler.Webhook(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rr.Code)
	}
}

func TestZaloHandler_SaveConfig_WithEncryption(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	zaloSvc := mock_service.NewMockZaloService(ctrl)
	zaloHandler := handler.NewZaloHandler(zaloSvc, "test.local")
	managerID := "manager1"

	// 1. Get the public key
	reqPK := httptest.NewRequest(http.MethodGet, "/api/v1/zalo/public-key", nil)
	rrPK := httptest.NewRecorder()
	zaloHandler.GetPublicKey(rrPK, reqPK)
	var resPK struct {
		PublicKey string `json:"public_key"`
	}
	json.NewDecoder(rrPK.Body).Decode(&resPK)

	pubASN1, _ := base64.StdEncoding.DecodeString(resPK.PublicKey)
	pubKey, _ := x509.ParsePKIXPublicKey(pubASN1)
	rsaPubKey := pubKey.(*rsa.PublicKey)

	// 2. Encrypt token
	tokenBytes, _ := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPubKey, []byte("bot_token_123"), nil)

	payload := map[string]string{
		"bot_token":      base64.StdEncoding.EncodeToString(tokenBytes),
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/zalo/config", bytes.NewBuffer(body))
	claims := &security.Claims{}
	claims.Subject = managerID
	req = req.WithContext(security.WithClaims(req.Context(), claims))
	rr := httptest.NewRecorder()

	zaloSvc.EXPECT().SaveZaloConfig(req.Context(), managerID, "bot_token_123", "test.local/api/v1/zalo/webhooks/"+managerID, gomock.Any()).Return(nil)

	zaloHandler.SaveConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rr.Code)
	}
}
