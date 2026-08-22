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
}
