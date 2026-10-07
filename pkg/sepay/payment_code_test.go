package sepay

import (
	"context"
	"net/url"
	"testing"
)

// TestBuildPaymentCode keeps codes uppercase alphanumeric with the prefix intact.
func TestBuildPaymentCode(t *testing.T) {
	tests := []struct {
		prefix, id string
		attempt    int
		want       string
	}{
		{"PH", "01a11784-18e8-7ad9-80cd-494a3bd7dee8", 0, "PH01A11784"},
		{"ph", "abc-12", 0, "PHABC12"},
		{"PH", "01a11784-18e8", 2, "PH01A117842"},
		{"P-H", "x_y", 0, "PHXY"},
	}
	for _, tt := range tests {
		if got := BuildPaymentCode(tt.prefix, tt.id, tt.attempt); got != tt.want {
			t.Errorf("BuildPaymentCode(%q, %q, %d) = %q, want %q", tt.prefix, tt.id, tt.attempt, got, tt.want)
		}
	}
}

// TestGenerateUniquePaymentCodeRetriesAndExhausts verifies collisions advance the attempt suffix.
func TestGenerateUniquePaymentCodeRetriesAndExhausts(t *testing.T) {
	seen := 0
	code, err := GenerateUniquePaymentCode(context.Background(), "PT", "abc12345", 0, func(context.Context, string) (bool, error) {
		seen++
		return seen == 1, nil
	})
	if err != nil || code != "PTABC123451" {
		t.Fatalf("code, err = %q, %v; want PTABC123451", code, err)
	}

	if _, err := GenerateUniquePaymentCode(context.Background(), "PT", "abc12345", 3, func(context.Context, string) (bool, error) {
		return true, nil
	}); err == nil {
		t.Fatal("expected error after all generated codes collide, got nil")
	}
}

// TestBuildQRURL verifies the SePay QR image query contract.
func TestBuildQRURL(t *testing.T) {
	parsed, err := url.Parse(BuildQRURL("0000000001", "Vietcombank", 1000000, "PH01A11784"))
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if parsed.Host != "qr.sepay.vn" || query.Get("acc") != "0000000001" || query.Get("bank") != "Vietcombank" || query.Get("amount") != "1000000" || query.Get("des") != "PH01A11784" {
		t.Errorf("QR url = %s", parsed)
	}
}
