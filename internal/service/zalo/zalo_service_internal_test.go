package zalo

import (
	"testing"
)

// TestNewZaloService_PublicBaseURL covers optional public base URL normalization.
func TestNewZaloService_PublicBaseURL(t *testing.T) {
	svc, err := NewZaloService(nil, nil, nil, nil, nil, nil, nil, nil, "z123456789abcdef0123456789abcdef", "https://example.com/")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	impl := svc.(*zaloServiceImpl)
	if impl.publicBaseURL != "https://example.com" {
		t.Errorf("expected trimmed base URL, got %s", impl.publicBaseURL)
	}
}
