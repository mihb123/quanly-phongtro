package config

import (
	"net/netip"
	"testing"
	"time"
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

// setRequiredEnv sets the minimum env vars Load needs so a test can focus on one setting.
func setRequiredEnv(t *testing.T) {
	t.Helper()
	for key, value := range map[string]string{
		"SMTP_USERNAME":            "user",
		"SMTP_PASSWORD":            "pass",
		"ZALO_BOT_ENCRYPTION_KEY":  "key",
		"POSTGRES_DSN":             "postgres://localhost:5432/db",
		"ACCESS_TOKEN_JWT_SECRET":  "access",
		"REFRESH_TOKEN_JWT_SECRET": "refresh",
		"OTP_EXPIRE_MINUTES":       "5",
	} {
		t.Setenv(key, value)
	}
}

// TestLoadTrustedProxies covers the default, the explicit list and the opt-out values.
func TestLoadTrustedProxies(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		wantTrusted []string
		wantNone    bool
	}{
		{name: "defaults to localhost when unset", value: "", wantTrusted: []string{"127.0.0.1", "::1"}},
		{name: "explicit list is honored", value: "10.0.0.0/8", wantTrusted: []string{"10.1.2.3"}},
		{name: "none disables every proxy", value: "none", wantNone: true},
		{name: "off is accepted case-insensitively", value: "OFF", wantNone: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv("TRUSTED_PROXIES", tt.value)

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if tt.wantNone {
				if len(cfg.TrustedProxyCIDRs) != 0 {
					t.Fatalf("TrustedProxyCIDRs = %v, want empty", cfg.TrustedProxyCIDRs)
				}
				return
			}

			for _, addr := range tt.wantTrusted {
				if !containsAddr(cfg.TrustedProxyCIDRs, netip.MustParseAddr(addr)) {
					t.Errorf("TrustedProxyCIDRs %v does not contain %s", cfg.TrustedProxyCIDRs, addr)
				}
			}
		})
	}
}

func containsAddr(prefixes []netip.Prefix, addr netip.Addr) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// TestLoadSlowAPIDefaults đảm bảo ba biến SLOW_API_* là tùy chọn: không khai báo,
// để trống hay chỉ có khoảng trắng đều rơi về giá trị mặc định.
func TestLoadSlowAPIDefaults(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		setEnv    bool
		wantValue time.Duration
	}{
		{name: "unset uses defaults", setEnv: false, wantValue: time.Second},
		{
			name:      "empty values fall back to defaults",
			setEnv:    true,
			env:       map[string]string{"SLOW_API_THRESHOLD": "", "SLOW_API_LOG_FILE": "", "SLOW_API_LOG_MAX_DAYS": ""},
			wantValue: time.Second,
		},
		{
			name:      "whitespace only values fall back to defaults",
			setEnv:    true,
			env:       map[string]string{"SLOW_API_THRESHOLD": "  ", "SLOW_API_LOG_FILE": "  ", "SLOW_API_LOG_MAX_DAYS": " "},
			wantValue: time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequiredEnv(t)
			if tt.setEnv {
				for key, value := range tt.env {
					t.Setenv(key, value)
				}
			}

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if cfg.SlowAPIThreshold != tt.wantValue {
				t.Errorf("SlowAPIThreshold = %v, want %v", cfg.SlowAPIThreshold, tt.wantValue)
			}
			if cfg.SlowAPILogFile != defaultSlowAPILogFile {
				t.Errorf("SlowAPILogFile = %q, want %q", cfg.SlowAPILogFile, defaultSlowAPILogFile)
			}
			if cfg.SlowAPILogMaxDays != defaultSlowAPILogMaxDays {
				t.Errorf("SlowAPILogMaxDays = %d, want %d", cfg.SlowAPILogMaxDays, defaultSlowAPILogMaxDays)
			}
		})
	}
}

// TestLoadSlowAPIExplicitValues covers the opt-out and custom values.
func TestLoadSlowAPIExplicitValues(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("SLOW_API_THRESHOLD", "0")
	t.Setenv("SLOW_API_LOG_FILE", "tmp/custom_slow.log")
	t.Setenv("SLOW_API_LOG_MAX_DAYS", "30")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.SlowAPIThreshold != 0 {
		t.Errorf("SlowAPIThreshold = %v, want 0 (disabled)", cfg.SlowAPIThreshold)
	}
	if cfg.SlowAPILogFile != "tmp/custom_slow.log" || cfg.SlowAPILogMaxDays != 30 {
		t.Errorf("unexpected slow api config: %q / %d", cfg.SlowAPILogFile, cfg.SlowAPILogMaxDays)
	}
}

// TestLoadSlowAPIInvalidValues đảm bảo giá trị sai bị báo lỗi thay vì âm thầm bỏ qua.
func TestLoadSlowAPIInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "threshold is not a number", key: "SLOW_API_THRESHOLD", value: "fast"},
		{name: "threshold is negative", key: "SLOW_API_THRESHOLD", value: "-1"},
		{name: "max days is zero", key: "SLOW_API_LOG_MAX_DAYS", value: "0"},
		{name: "max days is not a number", key: "SLOW_API_LOG_MAX_DAYS", value: "week"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv(tt.key, tt.value)

			if _, err := Load(); err == nil {
				t.Fatalf("Load() error = nil, want error for %s=%q", tt.key, tt.value)
			}
		})
	}
}
