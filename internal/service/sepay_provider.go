package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

type SePayProvider struct{}

// NewSePayProvider initializes a new SePay provider adapter.
func NewSePayProvider() *SePayProvider {
	return &SePayProvider{}
}

// Name returns the standard string identifier for this provider.
func (p *SePayProvider) Name() string {
	return model.PaymentProviderSePay
}

// CreatePaymentLink generates a SePay QR transfer link for the invoice.
func (p *SePayProvider) CreatePaymentLink(ctx context.Context, input PaymentCreateInput) (*PaymentProviderLink, error) {
	amount := int(input.Invoice.TotalAmount)
	if amount <= 0 {
		return nil, errors.New("invoice amount must be greater than 0")
	}

	bank := input.Credentials["bank_short_name"]
	account := input.Credentials["account_number"]
	prefix := input.Credentials["code_prefix"]

	if bank == "" || account == "" || prefix == "" {
		return nil, errors.New("missing SePay credentials")
	}

	paymentCode, err := generateUniqueSePayCode(ctx, prefix, input.Invoice.ID, input.ProviderOrderRefExists)
	if err != nil {
		return nil, err
	}

	base := "https://qr.sepay.vn/img"
	values := url.Values{}
	values.Add("acc", account)
	values.Add("bank", bank)
	values.Add("amount", strconv.Itoa(amount))
	values.Add("des", paymentCode)

	qrURL := base + "?" + values.Encode()

	return &PaymentProviderLink{
		ProviderOrderRef: paymentCode,
		OrderCode:        0,
		PaymentLinkID:    paymentCode,
		CheckoutURL:      qrURL,
		QRCode:           qrURL,
		Amount:           amount,
	}, nil
}

// CancelPaymentLink cancels the remote provider link and marks the local link cancelled.
// For SePay, QR transfer does not have a remote link to cancel.
func (p *SePayProvider) CancelPaymentLink(ctx context.Context, input PaymentCancelInput) error {
	return nil
}

// sePayWebhookPayload is the incoming-transaction body SePay POSTs to the webhook URL.
type sePayWebhookPayload struct {
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

// VerifyWebhook authenticates the SePay webhook then maps an incoming transfer into a neutral event.
func (p *SePayProvider) VerifyWebhook(_ context.Context, input PaymentWebhookInput) (*VerifiedPaymentEvent, error) {
	if err := authenticateSePayWebhook(input); err != nil {
		return nil, err
	}

	var webhook sePayWebhookPayload
	if err := json.Unmarshal(input.Body, &webhook); err != nil {
		return nil, fmt.Errorf("%w: parse sepay webhook body: %v", ErrPaymentWebhookInvalid, err)
	}

	// Only incoming transfers can settle an invoice; ignore outgoing money.
	if !strings.EqualFold(webhook.TransferType, "in") {
		return nil, ErrPaymentWebhookIgnored
	}

	// No payment code means SePay could not match the configured prefix; nothing to reconcile.
	if webhook.Code == "" {
		return nil, ErrPaymentWebhookIgnored
	}

	// SePay's transaction id is stable across retries/replays, so it is the dedup key.
	transactionReference := strconv.FormatInt(webhook.ID, 10)
	counterAccount := webhook.AccountNumber

	return &VerifiedPaymentEvent{
		ProviderOrderRef:     webhook.Code,
		Amount:               webhook.TransferAmount,
		TransactionReference: transactionReference,
		CounterAccount:       &counterAccount,
		RawPayload:           string(input.Body),
		SignatureResult:      paymentSignatureValid,
		MatchingMethod:       paymentMatchOrderRef,
	}, nil
}

// authenticateSePayWebhook verifies the request using the manager-configured auth method.
func authenticateSePayWebhook(input PaymentWebhookInput) error {
	switch input.Credentials["webhook_auth_method"] {
	case "hmac":
		return verifySePayHMAC(input)
	case "apikey":
		return verifySePayAPIKey(input)
	case "none", "":
		// Manager opted out of webhook authentication; accept but warn since this is less secure.
		log.Printf("SePay webhook accepted without authentication (webhook_auth_method not set)")
		return nil
	default:
		return fmt.Errorf("%w: unknown sepay webhook auth method", ErrPaymentWebhookInvalid)
	}
}

// verifySePayAPIKey checks the "Authorization: Apikey <key>" header against the configured key.
func verifySePayAPIKey(input PaymentWebhookInput) error {
	expected := input.Credentials["webhook_api_key"]
	if expected == "" {
		return fmt.Errorf("%w: sepay webhook api key not configured", ErrPaymentCredentialsNotFound)
	}
	scheme, value, found := strings.Cut(input.Headers.Get("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, "Apikey") {
		return fmt.Errorf("%w: missing sepay Apikey authorization header", ErrPaymentWebhookInvalid)
	}
	if !hmac.Equal([]byte(value), []byte(expected)) {
		return fmt.Errorf("%w: sepay webhook api key mismatch", ErrPaymentWebhookInvalid)
	}
	return nil
}

// verifySePayHMAC verifies the X-SePay-Signature header computed over "{timestamp}.{raw_body}".
func verifySePayHMAC(input PaymentWebhookInput) error {
	secret := input.Credentials["webhook_secret"]
	if secret == "" {
		return fmt.Errorf("%w: sepay webhook secret not configured", ErrPaymentCredentialsNotFound)
	}
	timestamp := input.Headers.Get("X-SePay-Timestamp")
	signature := input.Headers.Get("X-SePay-Signature")
	if timestamp == "" || signature == "" {
		return fmt.Errorf("%w: missing sepay signature headers", ErrPaymentWebhookInvalid)
	}
	providedHex := strings.TrimPrefix(signature, "sha256=")

	mac := hmac.New(sha256.New, []byte(secret))
	// Sign the raw body bytes exactly as received; never re-marshal the parsed JSON.
	mac.Write([]byte(timestamp + "." + string(input.Body)))
	expectedHex := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(providedHex), []byte(expectedHex)) {
		return fmt.Errorf("%w: sepay webhook signature mismatch", ErrPaymentWebhookInvalid)
	}
	return nil
}

// generateUniqueSePayCode generates a new SePay payment code avoiding collisions.
func generateUniqueSePayCode(ctx context.Context, prefix, invoiceID string, exists func(context.Context, string) (bool, error)) (string, error) {
	for attempt := 0; attempt < maxOrderCodeAttempts; attempt++ {
		code := buildSePayPaymentCode(prefix, invoiceID, attempt)
		if exists == nil {
			return code, nil
		}
		found, err := exists(ctx, code)
		if err != nil {
			return "", err
		}
		if !found {
			return code, nil
		}
	}
	return "", errors.New("could not generate unique SePay payment code")
}

// buildSePayPaymentCode creates a formatted string from prefix and an invoice identifier.
func buildSePayPaymentCode(prefix, invoiceID string, attempt int) string {
	cleanID := strings.ReplaceAll(invoiceID, "-", "")
	cleanID = strings.ToUpper(cleanID)
	if len(cleanID) > 8 {
		cleanID = cleanID[:8]
	}

	// Uppercase the whole code so a lowercase prefix still survives the [A-Z0-9] filter below;
	// SePay must receive the prefix intact to auto-extract the payment code.
	code := strings.ToUpper(prefix + cleanID)
	if attempt > 0 {
		code += strconv.Itoa(attempt)
	}

	// Keep only [A-Z0-9] so the code is safe inside a bank transfer memo.
	var safeCode strings.Builder
	for _, ch := range code {
		if (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			safeCode.WriteRune(ch)
		}
	}

	return safeCode.String()
}
