package config

import (
	"net/netip"
	"testing"
)

// TestGetCookieSecureDefaultsByEnvironment verifies production cookies default to Secure.
func TestGetCookieSecureDefaultsByEnvironment(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "")

	secure, err := getCookieSecure("prod")
	if err != nil {
		t.Fatalf("getCookieSecure prod error = %v", err)
	}
	if !secure {
		t.Fatal("prod CookieSecure = false, want true")
	}

	secure, err = getCookieSecure("dev")
	if err != nil {
		t.Fatalf("getCookieSecure dev error = %v", err)
	}
	if secure {
		t.Fatal("dev CookieSecure = true, want false")
	}
}

// TestParseTrustedProxyCIDRs accepts both CIDR and single-IP entries.
func TestParseTrustedProxyCIDRs(t *testing.T) {
	prefixes, err := parseTrustedProxyCIDRs("10.0.0.0/8, 192.0.2.10")
	if err != nil {
		t.Fatalf("parseTrustedProxyCIDRs error = %v", err)
	}
	if len(prefixes) != 2 {
		t.Fatalf("len(prefixes) = %d, want 2", len(prefixes))
	}
	if !prefixes[0].Contains(netip.MustParseAddr("10.1.2.3")) {
		t.Fatal("10.0.0.0/8 prefix did not match expected address")
	}
	if !prefixes[1].Contains(netip.MustParseAddr("192.0.2.10")) {
		t.Fatal("single IP prefix did not match expected address")
	}
}
