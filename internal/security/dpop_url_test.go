package security

import (
	"net/http"
	"net/netip"
	"testing"
)

// TestBuildDPoPHTU verifies the htu mirrors the public origin and trusts
// X-Forwarded-* only from configured proxies.
func TestBuildDPoPHTU(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")}

	tests := []struct {
		name           string
		remoteAddr     string
		host           string
		forwardedProto string
		forwardedHost  string
		trustedProxies []netip.Prefix
		want           string
	}{
		{
			name:       "localhost direct uses request host and http",
			remoteAddr: "127.0.0.1:54321",
			host:       "localhost:8080",
			want:       "http://localhost:8080/api/v1/auth/me",
		},
		{
			name:           "trusted proxy honors forwarded proto and host",
			remoteAddr:     "192.0.2.10:5555",
			host:           "internal.local",
			forwardedProto: "https",
			forwardedHost:  "pt.mvpc.site",
			trustedProxies: trusted,
			want:           "https://pt.mvpc.site/api/v1/auth/me",
		},
		{
			name:           "untrusted peer ignores forwarded headers",
			remoteAddr:     "203.0.113.5:5555",
			host:           "localhost:8080",
			forwardedProto: "https",
			forwardedHost:  "evil.example",
			trustedProxies: trusted,
			want:           "http://localhost:8080/api/v1/auth/me",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/api/v1/auth/me?token=secret", nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			req.RemoteAddr = tc.remoteAddr
			req.Host = tc.host
			if tc.forwardedProto != "" {
				req.Header.Set("X-Forwarded-Proto", tc.forwardedProto)
			}
			if tc.forwardedHost != "" {
				req.Header.Set("X-Forwarded-Host", tc.forwardedHost)
			}

			got := BuildDPoPHTU(req, tc.trustedProxies)
			if got != tc.want {
				t.Errorf("BuildDPoPHTU = %q, want %q", got, tc.want)
			}
		})
	}
}
