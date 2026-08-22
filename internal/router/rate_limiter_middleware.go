package router

import (
	"net/http"
	"net/netip"
	"sync"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/security"
	"golang.org/x/time/rate"
)

const (
	// limiterIdleTTL: key không dùng trong khoảng này sẽ bị dọn khi prune.
	limiterIdleTTL = 5 * time.Minute
	// limiterPruneEvery: chu kỳ prune tối thiểu, chạy ngay trong allow nên không cần goroutine nền.
	limiterPruneEvery = 10 * time.Minute
	// limiterMaxKeys chặn map phình vô hạn khi client sau proxy tin cậy giả mạo X-Forwarded-For.
	limiterMaxKeys = 20000
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// limiterStore giữ token bucket theo key (user ID hoặc IP) và tự dọn key nguội.
type limiterStore struct {
	mu        sync.Mutex
	clients   map[string]*client
	limit     rate.Limit
	burst     int
	lastPrune time.Time
}

func newLimiterStore(limit rate.Limit, burst int) *limiterStore {
	return &limiterStore{
		clients:   make(map[string]*client),
		limit:     limit,
		burst:     burst,
		lastPrune: time.Now(),
	}
}

// allow lấy một token của key, đồng thời prune định kỳ các key đã nguội.
func (s *limiterStore) allow(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if now.Sub(s.lastPrune) > limiterPruneEvery || len(s.clients) >= limiterMaxKeys {
		s.pruneLocked(now)
	}

	c, exists := s.clients[key]
	if !exists {
		c = &client{limiter: rate.NewLimiter(s.limit, s.burst)}
		s.clients[key] = c
	}
	c.lastSeen = now

	return c.limiter.Allow()
}

// pruneLocked xóa key đã nguội; nếu vẫn chạm trần thì bỏ bớt key bất kỳ để giữ map có biên.
func (s *limiterStore) pruneLocked(now time.Time) {
	s.lastPrune = now
	for key, c := range s.clients {
		if now.Sub(c.lastSeen) > limiterIdleTTL {
			delete(s.clients, key)
		}
	}
	for key := range s.clients {
		if len(s.clients) < limiterMaxKeys {
			break
		}
		delete(s.clients, key)
	}
}

// userRateLimiter giới hạn theo user ID cho endpoint gửi lại OTP.
var userRateLimiter = newLimiterStore(rate.Every(100*time.Second), 1)

func RateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := security.ClaimsFromContext(r.Context())
		if !ok || claims == nil {
			writeError(w, http.StatusInternalServerError, "cannot parse token")
			return
		}

		userID, err := claims.GetSubject()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "cannot parse token")
			return
		}

		if !userRateLimiter.allow(userID) {
			writeError(w, http.StatusBadRequest, "too many request")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ipRateLimiter giới hạn theo IP cho các endpoint công khai, nơi chưa có token để định danh.
// Mỗi route dùng một store riêng nên hạn mức không dùng chung giữa các route.
func ipRateLimiter(store *limiterStore, trustedProxies []netip.Prefix, retryAfter string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !store.allow(security.ClientIP(r, trustedProxies)) {
				w.Header().Set("Retry-After", retryAfter)
				writeError(w, http.StatusTooManyRequests, "too many requests, please try again later")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// authRateLimiters gom các middleware giới hạn theo IP cho nhóm endpoint auth công khai.
type authRateLimiters struct {
	login     func(http.Handler) http.Handler
	register  func(http.Handler) http.Handler
	verifyOTP func(http.Handler) http.Handler
}

// newAuthRateLimiters tạo store riêng cho từng route: gõ sai mật khẩu vài lần vẫn thoải mái,
// nhưng dò mật khẩu / tạo tài khoản hàng loạt từ một IP thì bị chặn.
func newAuthRateLimiters(trustedProxies []netip.Prefix) authRateLimiters {
	return authRateLimiters{
		// 10 lần liên tiếp, sau đó 4 lần/phút.
		login: ipRateLimiter(newLimiterStore(rate.Every(15*time.Second), 10), trustedProxies, "15"),
		// Đăng ký là hành vi hiếm: 5 lần liên tiếp, sau đó 1 lần/phút.
		register: ipRateLimiter(newLimiterStore(rate.Every(time.Minute), 5), trustedProxies, "60"),
		// Lớp chặn theo IP bổ sung cho bộ đếm otp_checks vốn tính theo email.
		verifyOTP: ipRateLimiter(newLimiterStore(rate.Every(10*time.Second), 10), trustedProxies, "10"),
	}
}
