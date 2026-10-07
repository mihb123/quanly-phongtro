package sepay

import (
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"
)

const incomingBody = `{"id":92704,"gateway":"Vietcombank","transactionDate":"2024-07-02 11:08:33","accountNumber":"123456789","subAccount":"","code":"PTABC123","content":"PTABC123 chuyen tien","transferType":"in","transferAmount":500000,"referenceCode":"FT24012345678"}`

func testCredentials(authMethod string) Credentials {
	return Credentials{
		Environment:       EnvironmentSandbox,
		BankShortName:     "MBBank",
		AccountNumber:     "123456789",
		CodePrefix:        "PT",
		WebhookAuthMethod: authMethod,
		WebhookAPIKey:     "secret-api-key",
		WebhookSecret:     "secret-hmac",
	}
}

func hmacHeaders(secret string, at time.Time, body []byte) http.Header {
	timestamp := strconv.FormatInt(at.Unix(), 10)
	headers := http.Header{}
	headers.Set("X-SePay-Timestamp", timestamp)
	headers.Set("X-SePay-Signature", SignHMAC(secret, timestamp, body))
	return headers
}

// TestVerifyWebhookHMAC covers the signed-request happy path and payload mapping.
func TestVerifyWebhookHMAC(t *testing.T) {
	now := time.Now()
	body := []byte(incomingBody)

	transfer, err := VerifyWebhook(hmacHeaders("secret-hmac", now, body), body, testCredentials(AuthMethodHMAC), now)
	if err != nil {
		t.Fatalf("VerifyWebhook() error = %v", err)
	}
	if transfer.PaymentCode != "PTABC123" || transfer.Amount != 500000 || transfer.TransactionReference != "92704" || transfer.AccountNumber != "123456789" {
		t.Errorf("transfer = %+v", transfer)
	}
	if transfer.RawPayload != incomingBody {
		t.Errorf("RawPayload must keep the raw body")
	}
}

// TestVerifyWebhookRejects covers every authentication and payload rejection path.
func TestVerifyWebhookRejects(t *testing.T) {
	now := time.Now()
	body := []byte(incomingBody)
	expired := now.Add(-MaxWebhookClockSkew - time.Second)
	future := now.Add(MaxWebhookClockSkew + time.Second)

	tests := []struct {
		name        string
		headers     http.Header
		body        []byte
		credentials Credentials
		want        error
	}{
		{"tampered body", hmacHeaders("secret-hmac", now, []byte(`{"id":1}`)), body, testCredentials(AuthMethodHMAC), ErrInvalidWebhook},
		{"wrong secret", hmacHeaders("other", now, body), body, testCredentials(AuthMethodHMAC), ErrInvalidWebhook},
		{"expired timestamp", hmacHeaders("secret-hmac", expired, body), body, testCredentials(AuthMethodHMAC), ErrInvalidWebhook},
		{"future timestamp", hmacHeaders("secret-hmac", future, body), body, testCredentials(AuthMethodHMAC), ErrInvalidWebhook},
		{"missing hmac headers", http.Header{}, body, testCredentials(AuthMethodHMAC), ErrInvalidWebhook},
		{"missing hmac secret", hmacHeaders("secret-hmac", now, body), body, Credentials{WebhookAuthMethod: AuthMethodHMAC}, ErrMissingCredentials},
		{"wrong api key", http.Header{"Authorization": []string{"Apikey wrong"}}, body, testCredentials(AuthMethodAPIKey), ErrInvalidWebhook},
		{"missing api key header", http.Header{}, body, testCredentials(AuthMethodAPIKey), ErrInvalidWebhook},
		{"missing api key config", http.Header{"Authorization": []string{"Apikey secret-api-key"}}, body, Credentials{WebhookAuthMethod: AuthMethodAPIKey}, ErrMissingCredentials},
		{"unknown method", http.Header{}, body, Credentials{WebhookAuthMethod: "basic"}, ErrInvalidWebhook},
		{"invalid json", http.Header{}, []byte("{bad json"), testCredentials(AuthMethodNone), ErrInvalidWebhook},
		{"wrong account", http.Header{}, []byte(`{"id":1,"transferType":"in","transferAmount":500000,"code":"PTABC123","accountNumber":"987654321"}`), testCredentials(AuthMethodNone), ErrInvalidWebhook},
		{"outgoing transfer", http.Header{}, []byte(`{"id":1,"transferType":"out","transferAmount":100,"code":"PTX"}`), testCredentials(AuthMethodNone), ErrIgnoredWebhook},
		{"missing code", http.Header{}, []byte(`{"id":2,"transferType":"in","transferAmount":100,"code":""}`), testCredentials(AuthMethodNone), ErrIgnoredWebhook},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := VerifyWebhook(tt.headers, tt.body, tt.credentials, now); !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

// TestVerifyWebhookAPIKey accepts the documented "Apikey <key>" header.
func TestVerifyWebhookAPIKey(t *testing.T) {
	headers := http.Header{"Authorization": []string{"Apikey secret-api-key"}}
	if _, err := VerifyWebhook(headers, []byte(incomingBody), testCredentials(AuthMethodAPIKey), time.Now()); err != nil {
		t.Fatalf("VerifyWebhook() error = %v", err)
	}
}
