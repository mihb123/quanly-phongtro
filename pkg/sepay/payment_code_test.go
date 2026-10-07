package sepay

import (
	"context"
	"net/url"
	"regexp"
	"testing"
)

// TestBuildPaymentCode keeps codes as prefix + exactly 6 uppercase alphanumerics.
func TestBuildPaymentCode(t *testing.T) {
	const id = "01a11784-18e8-7ad9-80cd-494a3bd7dee8"
	if got := BuildPaymentCode("PH", id, 0); got != "PHD7DEE8" {
		t.Errorf("first attempt = %q, want PHD7DEE8 (tail of the id)", got)
	}
	if got := BuildPaymentCode("p-h", id, 0); got != "PHD7DEE8" {
		t.Errorf("prefix must be uppercased alphanumerics, got %q", got)
	}

	seen := map[string]bool{}
	for attempt := 0; attempt < DefaultCodeAttempts; attempt++ {
		code := BuildPaymentCode("PH", id, attempt)
		if !regexp.MustCompile(`^PH[A-Z0-9]{6}$`).MatchString(code) {
			t.Fatalf("attempt %d code %q must be PH + 6 [A-Z0-9]", attempt, code)
		}
		if seen[code] {
			t.Fatalf("attempt %d repeated code %q", attempt, code)
		}
		seen[code] = true
	}
	if got := BuildPaymentCode("PH", id, 3); got != BuildPaymentCode("PH", id, 3) {
		t.Errorf("codes must be deterministic, got %q", got)
	}
	if got := BuildPaymentCode("PH", "x_y", 0); !regexp.MustCompile(`^PH[A-Z0-9]{6}$`).MatchString(got) {
		t.Errorf("short ids must still yield 6 characters, got %q", got)
	}
}

// TestGenerateUniquePaymentCodeRetriesAndExhausts verifies collisions move to a new fixed-length code.
func TestGenerateUniquePaymentCodeRetriesAndExhausts(t *testing.T) {
	first := ""
	code, err := GenerateUniquePaymentCode(context.Background(), "PT", "abc12345", 0, func(_ context.Context, code string) (bool, error) {
		if first == "" {
			first = code
			return true, nil
		}
		return false, nil
	})
	if err != nil || code == first || len(code) != len(first) {
		t.Fatalf("code, first, err = %q, %q, %v; want a different code of equal length", code, first, err)
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
