package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/payOSHQ/payos-lib-golang/v2"
)

const (
	maxPayOSOrderCode      int64 = 9007199254740991
	maxOrderCodeAttempts         = 5
	maxOrderCodePartLength       = 4
)

var ErrPayOSVerifiedDataNil = errors.New("verified data is nil")

type PayOSProvider struct{}

// NewPayOSProvider creates the PayOS payment provider adapter.
func NewPayOSProvider() *PayOSProvider {
	return &PayOSProvider{}
}

// Name returns the provider key used in routes and persistence.
func (p *PayOSProvider) Name() string {
	return model.PaymentProviderPayOS
}

// CreatePaymentLink creates a PayOS checkout link from a provider-neutral invoice request.
func (p *PayOSProvider) CreatePaymentLink(ctx context.Context, input PaymentCreateInput) (*PaymentProviderLink, error) {
	client, err := newPayOSClient(input.Credentials)
	if err != nil {
		return nil, err
	}

	amount := int(input.Invoice.TotalAmount)
	if amount <= 0 {
		return nil, errors.New("invoice amount must be greater than 0")
	}

	orderReferenceCode := buildOrderReferenceCode(input.Invoice.ID, input.Invoice.RoomID)
	orderCode, err := generateUniquePayOSOrderCode(ctx, input.Invoice, orderReferenceCode, input.ProviderOrderRefExists)
	if err != nil {
		return nil, err
	}

	req := payos.CreatePaymentLinkRequest{
		OrderCode:   orderCode,
		Amount:      amount,
		Description: payOSDescription(orderReferenceCode, input.Invoice.RoomName),
		ReturnUrl:   fmt.Sprintf("%s/api/v1/payments/providers/payos/return", strings.TrimRight(input.AppURL, "/")),
		CancelUrl:   fmt.Sprintf("%s/api/v1/payments/providers/payos/cancel", strings.TrimRight(input.AppURL, "/")),
		Items: []payos.PaymentLinkItem{
			{
				Name:     fmt.Sprintf("Hóa đơn phòng %s", input.Invoice.RoomName),
				Quantity: 1,
				Price:    amount,
			},
		},
	}
	if input.TenantName != "" {
		req.BuyerName = &input.TenantName
	}

	res, err := client.PaymentRequests.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("payos API create payment link: %w", err)
	}

	return &PaymentProviderLink{
		ProviderOrderRef: fmt.Sprint(orderCode),
		OrderCode:        orderCode,
		PaymentLinkID:    res.PaymentLinkId,
		CheckoutURL:      res.CheckoutUrl,
		QRCode:           res.QrCode,
		Amount:           amount,
	}, nil
}

// CancelPaymentLink cancels a PayOS checkout link through the PayOS SDK.
func (p *PayOSProvider) CancelPaymentLink(ctx context.Context, input PaymentCancelInput) error {
	client, err := newPayOSClient(input.Credentials)
	if err != nil {
		return err
	}
	orderCode, err := strconv.ParseInt(input.ProviderOrderRef, 10, 64)
	if err != nil {
		return fmt.Errorf("parse payos order code: %w", err)
	}
	if _, err := client.PaymentRequests.Cancel(ctx, orderCode, &input.Reason); err != nil {
		return fmt.Errorf("payos API cancel payment link: %w", err)
	}
	return nil
}

// VerifyWebhook verifies a PayOS webhook and maps it into provider-neutral event data.
func (p *PayOSProvider) VerifyWebhook(_ context.Context, input PaymentWebhookInput) (*VerifiedPaymentEvent, error) {
	var webhook payos.WebhookType
	if err := json.Unmarshal(input.Body, &webhook); err != nil {
		return nil, fmt.Errorf("%w: parse payos webhook body: %v", ErrPaymentWebhookInvalid, err)
	}
	if isPayOSTestPing(webhook) {
		return nil, ErrPaymentWebhookIgnored
	}

	verifiedData, err := verifyPayOSWebhookData(webhook, input.Credentials["checksum_key"])
	if err != nil {
		return nil, err
	}
	if verifiedData == nil {
		return nil, ErrPayOSVerifiedDataNil
	}

	return &VerifiedPaymentEvent{
		ProviderOrderRef:     fmt.Sprint(verifiedData.OrderCode),
		OrderCode:            verifiedData.OrderCode,
		Amount:               verifiedData.Amount,
		TransactionReference: verifiedData.Reference,
		PayerAccount:         &verifiedData.AccountNumber,
		RawPayload:           string(input.Body),
		SignatureResult:      paymentSignatureValid,
		MatchingMethod:       paymentMatchOrderRef,
	}, nil
}

// newPayOSClient creates a PayOS SDK client from decrypted provider credentials.
func newPayOSClient(credentials map[string]string) (*payos.PayOS, error) {
	clientID := credentials["client_id"]
	apiKey := credentials["api_key"]
	checksumKey := credentials["checksum_key"]
	if clientID == "" || apiKey == "" || checksumKey == "" {
		return nil, errors.New("missing PayOS credentials")
	}

	client, err := payos.NewPayOS(&payos.PayOSOptions{
		ClientId:    clientID,
		ApiKey:      apiKey,
		ChecksumKey: checksumKey,
	})
	if err != nil {
		return nil, fmt.Errorf("init payos client: %w", err)
	}
	return client, nil
}

// generateUniquePayOSOrderCode finds a PayOS-safe numeric order code before creating the remote link.
func generateUniquePayOSOrderCode(ctx context.Context, invoice *model.InvoiceWithRoom, referenceCode string, exists func(context.Context, string) (bool, error)) (int64, error) {
	for attempt := 0; attempt < maxOrderCodeAttempts; attempt++ {
		orderCode := payOSOrderCode(referenceCode, invoice.ID, invoice.RoomID, attempt)
		if exists == nil {
			return orderCode, nil
		}
		found, err := exists(ctx, fmt.Sprint(orderCode))
		if err != nil {
			return 0, fmt.Errorf("check payment order code: %w", err)
		}
		if !found {
			return orderCode, nil
		}
	}

	return 0, errors.New("could not generate unique PayOS order code")
}

// payOSDescription creates the short PayOS invoice description.
func payOSDescription(referenceCode, roomName string) string {
	description := fmt.Sprintf("HD %s P%s", referenceCode, roomName)
	if len(description) > 25 {
		return description[:25]
	}
	return description
}

// verifyPayOSWebhookData validates the webhook signature with the configured checksum key.
func verifyPayOSWebhookData(webhook payos.WebhookType, checksumKey string) (*payos.WebhookDataType, error) {
	if checksumKey == "" {
		return nil, fmt.Errorf("%w: payos checksum key is required", ErrPaymentCredentialsNotFound)
	}
	if webhook.Data == nil {
		return nil, ErrPayOSVerifiedDataNil
	}
	if webhook.Signature == "" {
		return nil, fmt.Errorf("%w: payos webhook signature is required", ErrPaymentWebhookInvalid)
	}

	expectedSignature, err := createPayOSSignature(webhook.Data, checksumKey)
	if err != nil {
		return nil, fmt.Errorf("create payos signature: %w", err)
	}
	if !hmac.Equal([]byte(expectedSignature), []byte(webhook.Signature)) {
		return nil, fmt.Errorf("%w: payos webhook signature mismatch", ErrPaymentWebhookInvalid)
	}

	return webhook.Data, nil
}

// createPayOSSignature signs PayOS webhook data using the library-compatible HMAC format.
func createPayOSSignature(data any, checksumKey string) (string, error) {
	sortedPayload, err := sortPayOSObjectByKey(data)
	if err != nil {
		return "", err
	}

	hasher := hmac.New(sha256.New, []byte(checksumKey))
	if _, err := hasher.Write([]byte(sortedPayload)); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// sortPayOSObjectByKey returns key=value pairs sorted like the PayOS Go SDK.
func sortPayOSObjectByKey(data any) (string, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	var payload map[string]any
	if err := json.Unmarshal(jsonBytes, &payload); err != nil {
		return "", err
	}

	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		value, err := payOSSignatureValue(payload[key])
		if err != nil {
			return "", err
		}
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, value))
	}

	return strings.Join(pairs, "&"), nil
}

// payOSSignatureValue normalizes JSON values the same way PayOS signs webhook data.
func payOSSignatureValue(value any) (string, error) {
	switch v := value.(type) {
	case nil:
		return "", nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case string:
		return v, nil
	default:
		resultBytes, err := json.Marshal(value)
		return string(resultBytes), err
	}
}

// isPayOSTestPing reports whether PayOS sent a success ping without a real order.
func isPayOSTestPing(webhook payos.WebhookType) bool {
	if webhook.Success != nil && *webhook.Success {
		return webhook.Data != nil && webhook.Data.OrderCode == 0
	}

	return false
}

// buildOrderReferenceCode creates the human-readable invoice-room code used in PayOS descriptions.
func buildOrderReferenceCode(invoiceID, roomID string) string {
	return fmt.Sprintf("%s-%s", orderCodePart(invoiceID), orderCodePart(roomID))
}

// orderCodePart returns up to four base36-compatible characters from an identifier.
func orderCodePart(value string) string {
	var builder strings.Builder
	for _, char := range strings.ToLower(strings.TrimSpace(value)) {
		if !isBase36Char(char) {
			continue
		}
		builder.WriteRune(char)
		if builder.Len() == maxOrderCodePartLength {
			break
		}
	}
	if builder.Len() == 0 {
		return "0"
	}

	return builder.String()
}

// payOSOrderCode converts the reference code to PayOS numeric orderCode and hashes fallback attempts.
func payOSOrderCode(referenceCode, invoiceID, roomID string, attempt int) int64 {
	if attempt == 0 {
		if orderCode := base36OrderCode(referenceCode); orderCode > 0 {
			return orderCode
		}
	}

	hash := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%d", invoiceID, roomID, attempt)))
	return int64(binary.BigEndian.Uint64(hash[:8])%uint64(maxPayOSOrderCode-1)) + 1
}

// base36OrderCode encodes the invoice-room reference code into PayOS' numeric orderCode field.
func base36OrderCode(referenceCode string) int64 {
	var orderCode int64
	for _, char := range strings.ToLower(referenceCode) {
		digit, ok := base36Digit(char)
		if !ok {
			continue
		}
		orderCode = orderCode*36 + int64(digit)
		if orderCode > maxPayOSOrderCode {
			return 0
		}
	}

	return orderCode
}

// base36Digit converts one base36 character to its numeric value.
func base36Digit(char rune) (int, bool) {
	switch {
	case char >= '0' && char <= '9':
		return int(char - '0'), true
	case char >= 'a' && char <= 'z':
		return int(char-'a') + 10, true
	default:
		return 0, false
	}
}

// isBase36Char reports whether a rune can be represented in the PayOS order code.
func isBase36Char(char rune) bool {
	_, ok := base36Digit(char)
	return ok
}
