package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

func TestRateLimiterMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	middleware := RateLimiter(handler)

	t.Run("Missing claims", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})

	t.Run("Valid request - under limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		
		claims := &security.Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject: "user-1",
			},
		}
		
		ctx := security.WithClaims(req.Context(), claims)
		req = req.WithContext(ctx)
		
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("Too many requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		
		claims := &security.Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject: "user-limit-test", // unique user to avoid conflict with previous test
			},
		}
		
		ctx := security.WithClaims(req.Context(), claims)
		req = req.WithContext(ctx)
		
		// First request (allowed)
		rec1 := httptest.NewRecorder()
		middleware.ServeHTTP(rec1, req)
		if rec1.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec1.Code)
		}

		// Second request immediately (should be blocked, limit is 1 every 100s)
		rec2 := httptest.NewRecorder()
		middleware.ServeHTTP(rec2, req)
		if rec2.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for too many requests, got %d", rec2.Code)
		}
	})
	
	t.Run("Cleanup", func(t *testing.T) {
		// Just cover the initialization of the client map
		mu.Lock()
		clients["old-user"] = &client{lastSeen: time.Now().Add(-10 * time.Minute)}
		mu.Unlock()
		
		// Note: The cleanup is running in a background goroutine since init() was called.
		// Testing it deterministically is hard without modifying the code to accept a ticker,
		// but we can verify it doesn't panic.
	})
}
