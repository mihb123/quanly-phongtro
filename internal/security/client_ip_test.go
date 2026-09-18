package security

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func TestClientIP(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}

	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		proxies    []netip.Prefix
		want       string
	}{
		{
			name:       "no proxy configured ignores forwarded headers",
			remoteAddr: "203.0.113.9:1234",
			headers:    map[string]string{"X-Forwarded-For": "198.51.100.1", "CF-Connecting-IP": "198.51.100.2"},
			want:       "203.0.113.9",
		},
		{
			name:       "untrusted peer ignores forwarded headers",
			remoteAddr: "203.0.113.9:1234",
			headers:    map[string]string{"X-Forwarded-For": "198.51.100.1"},
			proxies:    trusted,
			want:       "203.0.113.9",
		},
		{
			name:       "trusted peer prefers CF-Connecting-IP",
			remoteAddr: "127.0.0.1:1234",
			headers:    map[string]string{"CF-Connecting-IP": "198.51.100.7", "X-Forwarded-For": "1.2.3.4, 198.51.100.7"},
			proxies:    trusted,
			want:       "198.51.100.7",
		},
		// Cloudflare nối IP thật vào cuối chuỗi client tự gửi, nên lấy mục đầu là lấy phải giá trị giả mạo.
		{
			name:       "trusted peer uses last X-Forwarded-For entry",
			remoteAddr: "127.0.0.1:1234",
			headers:    map[string]string{"X-Forwarded-For": "1.2.3.4, 5.6.7.8, 198.51.100.7"},
			proxies:    trusted,
			want:       "198.51.100.7",
		},
		{
			name:       "trusted peer uses X-Real-IP from nginx",
			remoteAddr: "127.0.0.1:1234",
			headers:    map[string]string{"X-Real-IP": "198.51.100.8"},
			proxies:    trusted,
			want:       "198.51.100.8",
		},
		{
			name:       "trusted peer uses True-Client-IP",
			remoteAddr: "127.0.0.1:1234",
			headers:    map[string]string{"True-Client-IP": "198.51.100.9"},
			proxies:    trusted,
			want:       "198.51.100.9",
		},
		{
			name:       "trusted peer uses RFC 7239 Forwarded header",
			remoteAddr: "127.0.0.1:1234",
			headers:    map[string]string{"Forwarded": `for="198.51.100.10:4321";proto=https`},
			proxies:    trusted,
			want:       "198.51.100.10",
		},
		{
			name:       "untrusted peer ignores X-Real-IP",
			remoteAddr: "203.0.113.9:1234",
			headers:    map[string]string{"X-Real-IP": "198.51.100.8"},
			proxies:    trusted,
			want:       "203.0.113.9",
		},
		{
			name:       "trusted peer prefers X-Forwarded-For over X-Real-IP",
			remoteAddr: "127.0.0.1:1234",
			headers: map[string]string{
				"X-Real-IP":       "198.51.100.8",
				"X-Forwarded-For": "203.0.113.5, 198.51.100.11",
			},
			proxies: trusted,
			want:    "198.51.100.11",
		},
		{
			name:       "trusted peer reads IPv6 node in Forwarded header",
			remoteAddr: "127.0.0.1:1234",
			headers:    map[string]string{"Forwarded": `for="[2001:db8::1]:8080";proto=https`},
			proxies:    trusted,
			want:       "2001:db8::1",
		},
		{
			name:       "trusted peer ignores comma inside a quoted Forwarded node",
			remoteAddr: "127.0.0.1:1234",
			headers:    map[string]string{"Forwarded": `for=192.0.2.60;host="a,b", for=198.51.100.12`},
			proxies:    trusted,
			want:       "198.51.100.12",
		},
		{
			name:       "trusted peer skips obfuscated Forwarded node",
			remoteAddr: "127.0.0.1:1234",
			headers:    map[string]string{"Forwarded": `for=198.51.100.13, for=_hidden`},
			proxies:    trusted,
			want:       "198.51.100.13",
		},
		{
			name:       "trusted peer falls back to remote addr on garbage header",
			remoteAddr: "127.0.0.1:1234",
			headers:    map[string]string{"X-Forwarded-For": "not-an-ip"},
			proxies:    trusted,
			want:       "127.0.0.1",
		},
		{
			name:       "unparsable remote addr is returned as-is",
			remoteAddr: "pipe",
			want:       "pipe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			if got := ClientIP(req, tt.proxies); got != tt.want {
				t.Errorf("ClientIP() = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("ClientIP ignores context and honors the trust list it is given", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		req = req.WithContext(WithClientIP(req.Context(), "113.161.50.20"))

		if got := ClientIP(req, nil); got != "127.0.0.1" {
			t.Errorf("ClientIP() = %q, want 127.0.0.1", got)
		}
	})

	t.Run("ResolvedClientIP reuses the IP cached in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		req = req.WithContext(WithClientIP(req.Context(), "113.161.50.20"))

		if got := ResolvedClientIP(req, nil); got != "113.161.50.20" {
			t.Errorf("ResolvedClientIP() = %q, want 113.161.50.20", got)
		}
	})

	t.Run("ResolvedClientIP falls back to ClientIP without context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		req.Header.Set("CF-Connecting-IP", "113.161.50.20")

		if got := ResolvedClientIP(req, trusted); got != "113.161.50.20" {
			t.Errorf("ResolvedClientIP() = %q, want 113.161.50.20", got)
		}
	})
}
