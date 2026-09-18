package router

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/security"
)

func TestRequestLogger(t *testing.T) {
	trusted := []netip.Prefix{
		netip.MustParsePrefix("127.0.0.1/32"),
		netip.MustParsePrefix("10.0.0.0/8"),
	}

	tests := []struct {
		name        string
		remoteAddr  string
		headers     map[string]string
		proxies     []netip.Prefix
		wantRemote  string
		wantScheme  string
		expectCtxIP string
	}{
		{
			name:       "cloudflare tunnel request logs real IP and https scheme",
			remoteAddr: "127.0.0.1:37274",
			headers: map[string]string{
				"CF-Connecting-IP":  "113.161.50.20",
				"X-Forwarded-Proto": "https",
			},
			proxies:     trusted,
			wantRemote:  "from 113.161.50.20 ",
			wantScheme:  "https://",
			expectCtxIP: "113.161.50.20",
		},
		{
			name:       "nginx reverse proxy with X-Real-IP logs real IP and https scheme",
			remoteAddr: "127.0.0.1:41234",
			headers: map[string]string{
				"X-Real-IP":         "14.232.208.10",
				"X-Forwarded-Proto": "https",
			},
			proxies:     trusted,
			wantRemote:  "from 14.232.208.10 ",
			wantScheme:  "https://",
			expectCtxIP: "14.232.208.10",
		},
		{
			name:       "load balancer in private VPC network with X-Forwarded-For",
			remoteAddr: "10.0.1.5:52341",
			headers: map[string]string{
				"X-Forwarded-For":   "203.0.113.88",
				"X-Forwarded-Proto": "https",
			},
			proxies:     trusted,
			wantRemote:  "from 203.0.113.88 ",
			wantScheme:  "https://",
			expectCtxIP: "203.0.113.88",
		},
		{
			name:       "traefik style Forwarded header logs real IP and https scheme",
			remoteAddr: "10.0.1.5:44120",
			headers: map[string]string{
				"Forwarded":         `for="203.0.113.77:41000";proto=https`,
				"X-Forwarded-Proto": "https",
			},
			proxies:     trusted,
			wantRemote:  "from 203.0.113.77 ",
			wantScheme:  "https://",
			expectCtxIP: "203.0.113.77",
		},
		{
			name:       "untrusted direct request ignores spoofed headers",
			remoteAddr: "198.51.100.99:12345",
			headers: map[string]string{
				"CF-Connecting-IP":  "1.1.1.1",
				"X-Real-IP":         "2.2.2.2",
				"X-Forwarded-For":   "3.3.3.3",
				"X-Forwarded-Proto": "https",
			},
			proxies:     trusted,
			wantRemote:  "from 198.51.100.99 ",
			wantScheme:  "http://",
			expectCtxIP: "198.51.100.99",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			loggerMw := customRequestLogger(tt.proxies, &buf)

			var capturedCtxIP string
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if ip, ok := security.ClientIPFromContext(r.Context()); ok {
					capturedCtxIP = ip
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("ok"))
			})

			req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()

			loggerMw(handler).ServeHTTP(rec, req)

			logOutput := buf.String()

			if !strings.Contains(logOutput, tt.wantRemote) {
				t.Errorf("expected log output to contain remote %q, got: %s", tt.wantRemote, logOutput)
			}
			if !strings.Contains(logOutput, tt.wantScheme) {
				t.Errorf("expected log output to contain scheme %q, got: %s", tt.wantScheme, logOutput)
			}
			if capturedCtxIP != tt.expectCtxIP {
				t.Errorf("expected context client IP %q, got %q", tt.expectCtxIP, capturedCtxIP)
			}
		})
	}
}
