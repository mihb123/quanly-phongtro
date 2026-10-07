package sepay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const MaxWebhookClockSkew = 5 * time.Minute

// WebhookPayload is the incoming-transaction body SePay POSTs to the webhook URL.
type WebhookPayload struct {
	ID              int64  `json:"id"`
	Gateway         string `json:"gateway"`
	TransactionDate string `json:"transactionDate"`
	AccountNumber   string `json:"accountNumber"`
	SubAccount      string `json:"subAccount"`
	Code            string `json:"code"`
	Content         string `json:"content"`
	TransferType    string `json:"transferType"`
	TransferAmount  int    `json:"transferAmount"`
	ReferenceCode   string `json:"referenceCode"`
	Description     string `json:"description"`
}

type IncomingTransfer struct {
	PaymentCode          string
	Amount               int
	TransactionReference string
	AccountNumber        string
	RawPayload           string
}

// VerifyWebhook authenticates the SePay webhook then maps an incoming transfer.
func VerifyWebhook(headers http.Header, body []byte, credentials Credentials, now time.Time) (*IncomingTransfer, error) {
	if err := AuthenticateWebhook(headers, body, credentials, now); err != nil {
		return nil, err
	}

	var webhook WebhookPayload
	if err := json.Unmarshal(body, &webhook); err != nil {
		return nil, fmt.Errorf("%w: parse sepay webhook body: %v", ErrInvalidWebhook, err)
	}

	// Only incoming transfers can settle an invoice; ignore outgoing money.
	if !strings.EqualFold(webhook.TransferType, "in") {
		return nil, ErrIgnoredWebhook
	}

	// No payment code means SePay could not match the configured prefix; nothing to reconcile.
	if webhook.Code == "" {
		return nil, ErrIgnoredWebhook
	}
	if credentials.AccountNumber == "" {
		return nil, fmt.Errorf("%w: sepay account number not configured", ErrMissingCredentials)
	}
	if webhook.AccountNumber != credentials.AccountNumber {
		return nil, fmt.Errorf("%w: sepay webhook account mismatch", ErrInvalidWebhook)
	}

	// SePay's transaction id is stable across retries/replays, so it is the dedup key.
	return &IncomingTransfer{
		PaymentCode:          webhook.Code,
		Amount:               webhook.TransferAmount,
		TransactionReference: strconv.FormatInt(webhook.ID, 10),
		AccountNumber:        webhook.AccountNumber,
		RawPayload:           string(body),
	}, nil
}

// AuthenticateWebhook verifies the request using the configured auth method.
func AuthenticateWebhook(headers http.Header, body []byte, credentials Credentials, now time.Time) error {
	switch credentials.WebhookAuthMethod {
	case AuthMethodHMAC:
		return verifyHMAC(headers, body, credentials.WebhookSecret, now)
	case AuthMethodAPIKey:
		return verifyAPIKey(headers, credentials.WebhookAPIKey)
	case AuthMethodNone, "":
		// Owner opted out of webhook authentication; accept but warn since this is less secure.
		log.Printf("SePay webhook accepted without authentication (webhook_auth_method not set)")
		return nil
	default:
		return fmt.Errorf("%w: unknown sepay webhook auth method", ErrInvalidWebhook)
	}
}

// SignHMAC returns the X-SePay-Signature value SePay sends for a raw body.
func SignHMAC(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + string(body)))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// verifyAPIKey checks the "Authorization: Apikey <key>" header against the configured key.
func verifyAPIKey(headers http.Header, expected string) error {
	if expected == "" {
		return fmt.Errorf("%w: sepay webhook api key not configured", ErrMissingCredentials)
	}
	scheme, value, found := strings.Cut(headers.Get("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, "Apikey") {
		return fmt.Errorf("%w: missing sepay Apikey authorization header", ErrInvalidWebhook)
	}
	if !hmac.Equal([]byte(value), []byte(expected)) {
		return fmt.Errorf("%w: sepay webhook api key mismatch", ErrInvalidWebhook)
	}
	return nil
}

// verifyHMAC verifies the X-SePay-Signature header computed over "{timestamp}.{raw_body}".
func verifyHMAC(headers http.Header, body []byte, secret string, now time.Time) error {
	if secret == "" {
		return fmt.Errorf("%w: sepay webhook secret not configured", ErrMissingCredentials)
	}
	timestamp := headers.Get("X-SePay-Timestamp")
	signature := headers.Get("X-SePay-Signature")
	if timestamp == "" || signature == "" {
		return fmt.Errorf("%w: missing sepay signature headers", ErrInvalidWebhook)
	}
	timestampUnix, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("%w: invalid sepay timestamp", ErrInvalidWebhook)
	}
	clockSkew := now.Sub(time.Unix(timestampUnix, 0))
	if clockSkew > MaxWebhookClockSkew || clockSkew < -MaxWebhookClockSkew {
		return fmt.Errorf("%w: sepay webhook timestamp expired", ErrInvalidWebhook)
	}

	// Sign the raw body bytes exactly as received; never re-marshal the parsed JSON.
	expected := SignHMAC(secret, timestamp, body)
	provided := "sha256=" + strings.TrimPrefix(signature, "sha256=")
	if !hmac.Equal([]byte(provided), []byte(expected)) {
		return fmt.Errorf("%w: sepay webhook signature mismatch", ErrInvalidWebhook)
	}
	return nil
}
