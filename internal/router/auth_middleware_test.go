package router

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"go.uber.org/mock/gomock"
)

func TestAuthMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jwtRepo := mock_model.NewMockAuthSessionRepository(ctrl)
	jwtProvider := security.NewJWTProvider("access", "refresh", time.Hour, jwtRepo)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := authMiddleware(jwtProvider)(handler)

	t.Run("Missing token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("Valid token in header", func(t *testing.T) {
		token, _ := jwtProvider.GenerateAccessToken("manager", "test@test.local", "user-1", true, "")

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("Valid token in cookie", func(t *testing.T) {
		token, _ := jwtProvider.GenerateAccessToken("manager", "test@test.local", "user-1", true, "")

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("Invalid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rec.Code)
		}
	})

	t.Run("Missing DPoP proof", func(t *testing.T) {
		token, _ := jwtProvider.GenerateAccessToken("manager", "test@test.local", "user-1", true, "dummy-jkt")
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("Invalid DPoP proof", func(t *testing.T) {
		token, _ := jwtProvider.GenerateAccessToken("manager", "test@test.local", "user-1", true, "dummy-jkt")
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("DPoP", "invalid-proof")
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("Valid DPoP proof uses absolute htu", func(t *testing.T) {
		dpopPrivateKey, jwk, jkt := newDPoPTestKey(t)
		token, _ := jwtProvider.GenerateAccessToken("manager", "test@test.local", "user-1", true, jkt)
		htu := "https://api.example.test/protected"
		proof := signDPoPProof(t, dpopPrivateKey, jwk, http.MethodGet, htu, token)

		req := httptest.NewRequest(http.MethodGet, "/protected?ignored=true", nil)
		req.Host = "internal.local"
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("DPoP", proof)
		rec := httptest.NewRecorder()

		absoluteMiddleware := authMiddleware(jwtProvider, "https://api.example.test")(handler)
		absoluteMiddleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}

// newDPoPTestKey creates a P-256 key, its public JWK, and matching jkt.
func newDPoPTestKey(t *testing.T) (*ecdsa.PrivateKey, map[string]interface{}, string) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	x := base64.RawURLEncoding.EncodeToString(privateKey.X.Bytes())
	y := base64.RawURLEncoding.EncodeToString(privateKey.Y.Bytes())
	jwk := map[string]interface{}{
		"kty": "EC",
		"crv": "P-256",
		"x":   x,
		"y":   y,
	}

	thumbprintPayload := fmt.Sprintf(`{"crv":"P-256","kty":"EC","x":"%s","y":"%s"}`, x, y)
	sum := sha256.Sum256([]byte(thumbprintPayload))
	return privateKey, jwk, base64.RawURLEncoding.EncodeToString(sum[:])
}

// signDPoPProof signs a DPoP proof for middleware verification.
func signDPoPProof(t *testing.T, privateKey *ecdsa.PrivateKey, jwk map[string]interface{}, method, htu, accessToken string) string {
	t.Helper()

	accessTokenHash := sha256.Sum256([]byte(accessToken))
	claims := security.DPoPClaims{
		Htm: method,
		Htu: htu,
		Jti: uuid.NewString(),
		Ath: base64.RawURLEncoding.EncodeToString(accessTokenHash[:]),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["typ"] = "dpop+jwt"
	token.Header["jwk"] = jwk

	proof, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign proof: %v", err)
	}
	return proof
}

func TestRequireRole(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := requireRole("manager", "admin")(handler)

	t.Run("Missing claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("Has correct role", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		claims := &security.Claims{Role: "manager"}
		ctx := security.WithClaims(req.Context(), claims)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("Has incorrect role", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		claims := &security.Claims{Role: "tenant"}
		ctx := security.WithClaims(req.Context(), claims)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})
}
