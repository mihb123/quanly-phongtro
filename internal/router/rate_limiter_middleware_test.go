package router

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"golang.org/x/time/rate"

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
		userRateLimiter.mu.Lock()
		userRateLimiter.clients["old-user"] = &client{lastSeen: time.Now().Add(-10 * time.Minute)}
		userRateLimiter.lastPrune = time.Now().Add(-limiterPruneEvery - time.Minute)
		userRateLimiter.mu.Unlock()

		// Lần allow tiếp theo phải dọn key đã nguội.
		userRateLimiter.allow("fresh-user")

		userRateLimiter.mu.Lock()
		_, stillThere := userRateLimiter.clients["old-user"]
		userRateLimiter.mu.Unlock()
		if stillThere {
			t.Errorf("expected idle client to be pruned")
		}
	})
}

func TestIPRateLimiterMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("Blocks after burst is exhausted", func(t *testing.T) {
		middleware := ipRateLimiter(newLimiterStore(rate.Every(time.Minute), 2), nil, "60")(handler)

		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.RemoteAddr = "203.0.113.5:1234"
			rec := httptest.NewRecorder()
			middleware.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
			}
		}

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "203.0.113.5:1234"
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)
		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("expected 429, got %d", rec.Code)
		}
		if rec.Header().Get("Retry-After") != "60" {
			t.Errorf("expected Retry-After header, got %q", rec.Header().Get("Retry-After"))
		}
	})

	t.Run("Limits are per IP", func(t *testing.T) {
		middleware := ipRateLimiter(newLimiterStore(rate.Every(time.Minute), 1), nil, "60")(handler)

		for _, addr := range []string{"203.0.113.10:1", "203.0.113.11:1"} {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.RemoteAddr = addr
			rec := httptest.NewRecorder()
			middleware.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("%s: expected 200, got %d", addr, rec.Code)
			}
		}
	})

	// X-Forwarded-For chỉ được tin khi peer nằm trong dải proxy tin cậy, nếu không
	// client tự đặt header là bypass được rate limit.
	t.Run("Ignores untrusted X-Forwarded-For", func(t *testing.T) {
		middleware := ipRateLimiter(newLimiterStore(rate.Every(time.Minute), 1), nil, "60")(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "203.0.113.20:1"
		req.Header.Set("X-Forwarded-For", "198.51.100.1")
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		req2 := httptest.NewRequest(http.MethodPost, "/", nil)
		req2.RemoteAddr = "203.0.113.20:1"
		req2.Header.Set("X-Forwarded-For", "198.51.100.2")
		rec2 := httptest.NewRecorder()
		middleware.ServeHTTP(rec2, req2)
		if rec2.Code != http.StatusTooManyRequests {
			t.Errorf("expected 429 despite spoofed X-Forwarded-For, got %d", rec2.Code)
		}
	})

	// Cloudflare nối IP thật vào cuối X-Forwarded-For, nên đổi phần đầu không tách được bucket.
	t.Run("Rotating spoofed X-Forwarded-For prefix does not bypass the limit", func(t *testing.T) {
		trusted := []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}
		middleware := ipRateLimiter(newLimiterStore(rate.Every(time.Minute), 1), trusted, "60")(handler)

		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = "127.0.0.1:1"
		req.Header.Set("X-Forwarded-For", "1.1.1.1, 198.51.100.50")
		rec := httptest.NewRecorder()
		middleware.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		req2 := httptest.NewRequest(http.MethodPost, "/", nil)
		req2.RemoteAddr = "127.0.0.1:1"
		req2.Header.Set("X-Forwarded-For", "2.2.2.2, 198.51.100.50")
		rec2 := httptest.NewRecorder()
		middleware.ServeHTTP(rec2, req2)
		if rec2.Code != http.StatusTooManyRequests {
			t.Errorf("expected 429, got %d", rec2.Code)
		}
	})

	t.Run("Honors X-Forwarded-For from trusted proxy", func(t *testing.T) {
		trusted := []netip.Prefix{netip.MustParsePrefix("203.0.113.30/32")}
		middleware := ipRateLimiter(newLimiterStore(rate.Every(time.Minute), 1), trusted, "60")(handler)

		for _, forwarded := range []string{"198.51.100.10", "198.51.100.11"} {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.RemoteAddr = "203.0.113.30:1"
			req.Header.Set("X-Forwarded-For", forwarded)
			rec := httptest.NewRecorder()
			middleware.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("%s: expected 200, got %d", forwarded, rec.Code)
			}
		}
	})
}
