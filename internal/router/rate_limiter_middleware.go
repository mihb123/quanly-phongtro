package router

import (
	"net/http"
	"sync"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/security"
	"golang.org/x/time/rate"
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	clients = make(map[string]*client)
	mu      sync.Mutex
)

func RateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := security.ClaimsFromContext(r.Context())
		if !ok {
			http.Error(w, "cannot get used id from tokens", http.StatusInternalServerError)
			return
		}

		userID, err := claims.GetSubject()
		if err != nil {
			http.Error(w, "cannot get used id from tokens", http.StatusInternalServerError)
		}

		mu.Lock()
		if _, exists := clients[userID]; !exists {
			clients[userID] = &client{limiter: rate.NewLimiter(rate.Every(100*time.Second), 1)}
		}
		clients[userID].lastSeen = time.Now()
		limiter := clients[userID].limiter
		mu.Unlock()

		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Background cleanup remains the same
func cleanupClients() {
	for {
		time.Sleep(10 * time.Minute)
		mu.Lock()
		for userID, client := range clients {
			if time.Since(client.lastSeen) > 5*time.Minute {
				delete(clients, userID)
			}
		}
		mu.Unlock()
	}
}

func init() {
	go cleanupClients()
}
