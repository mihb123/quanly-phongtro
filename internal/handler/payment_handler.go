package handler

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service"
)

const maxPaymentWebhookBodyBytes = 1 << 20

type PaymentHandler struct {
	paymentService    service.PaymentService
	credentialService service.PaymentCredentialService
	privateKey        *rsa.PrivateKey
	publicKey         *rsa.PublicKey
	appURL            string
}

type payOSConfigRequest struct {
	ClientID    string `json:"client_id"`
	APIKey      string `json:"api_key"`
	ChecksumKey string `json:"checksum_key"`
}

// NewPaymentHandler creates HTTP handlers for payment provider config and callbacks.
func NewPaymentHandler(paymentService service.PaymentService, credentialService service.PaymentCredentialService, appURL string) *PaymentHandler {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("failed to generate RSA key for PaymentHandler: %v", err)
	}

	return &PaymentHandler{
		paymentService:    paymentService,
		credentialService: credentialService,
		privateKey:        privateKey,
		publicKey:         &privateKey.PublicKey,
		appURL:            strings.TrimRight(appURL, "/"),
	}
}

// NewPayOSWebhookHandler preserves old router tests and legacy call sites while using PaymentHandler.
func NewPayOSWebhookHandler(paymentService service.PaymentService) *PaymentHandler {
	return NewPaymentHandler(paymentService, nil, "")
}

// GetPublicKey returns the base64-encoded SPKI public key for frontend secret transport.
func (h *PaymentHandler) GetPublicKey(w http.ResponseWriter, _ *http.Request) {
	pubASN1, err := x509.MarshalPKIXPublicKey(h.publicKey)
	if err != nil {
		http.Error(w, "Failed to marshal public key", http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{
		"public_key": base64.StdEncoding.EncodeToString(pubASN1),
	})
}

// GetPayOSConfig returns manager-specific PayOS config status.
func (h *PaymentHandler) GetPayOSConfig(w http.ResponseWriter, r *http.Request) {
	managerID, ok := authenticatedManagerID(w, r)
	if !ok {
		return
	}

	status, err := h.credentialService.GetPayOSConfig(r.Context(), managerID, h.appURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(status)
}

// SavePayOSConfig decrypts and stores manager-specific PayOS credentials.
func (h *PaymentHandler) SavePayOSConfig(w http.ResponseWriter, r *http.Request) {
	managerID, ok := authenticatedManagerID(w, r)
	if !ok {
		return
	}

	var req payOSConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	credentials, err := h.decryptPayOSConfig(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if credentials.ClientID == "" || credentials.APIKey == "" || credentials.ChecksumKey == "" {
		http.Error(w, "missing PayOS credentials", http.StatusBadRequest)
		return
	}

	if err := h.credentialService.SavePayOSConfig(r.Context(), managerID, credentials); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// DeletePayOSConfig deactivates manager-specific PayOS credentials.
func (h *PaymentHandler) DeletePayOSConfig(w http.ResponseWriter, r *http.Request) {
	managerID, ok := authenticatedManagerID(w, r)
	if !ok {
		return
	}
	if err := h.credentialService.DeletePayOSConfig(r.Context(), managerID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// HandleProviderWebhook processes incoming webhooks for a manager/provider pair.
func (h *PaymentHandler) HandleProviderWebhook(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	managerID := chi.URLParam(r, "managerID")
	h.handleWebhook(w, r, provider, managerID)
}

// HandleLegacyPayOSWebhook processes legacy app-level PayOS webhooks.
func (h *PaymentHandler) HandleLegacyPayOSWebhook(w http.ResponseWriter, r *http.Request) {
	h.handleWebhook(w, r, "payos", "")
}

// HandleProviderReturn acknowledges browser redirects after provider checkout completion.
func (*PaymentHandler) HandleProviderReturn(w http.ResponseWriter, r *http.Request) {
	writePaymentRedirectText(w, r, "Cổng thanh toán đã chuyển hướng về hệ thống sau khi thanh toán.")
}

// HandleProviderCancel acknowledges browser redirects after provider checkout cancellation.
func (*PaymentHandler) HandleProviderCancel(w http.ResponseWriter, r *http.Request) {
	writePaymentRedirectText(w, r, "Bạn đã hủy thanh toán.")
}

// HandlePayOSReturn acknowledges the legacy PayOS return route.
func (h *PaymentHandler) HandlePayOSReturn(w http.ResponseWriter, r *http.Request) {
	writePaymentRedirectText(w, r, "PayOS đã chuyển hướng về hệ thống sau khi thanh toán.")
}

// HandlePayOSCancel acknowledges the legacy PayOS cancel route.
func (h *PaymentHandler) HandlePayOSCancel(w http.ResponseWriter, r *http.Request) {
	writePaymentRedirectText(w, r, "Bạn đã hủy thanh toán PayOS.")
}

// handleWebhook reads and forwards a bounded webhook body to the payment service.
func (h *PaymentHandler) handleWebhook(w http.ResponseWriter, r *http.Request, provider, managerID string) {
	r.Body = http.MaxBytesReader(w, r.Body, maxPaymentWebhookBodyBytes)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read payment webhook body: %v", err)
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := h.paymentService.HandleWebhook(r.Context(), provider, managerID, body); err != nil {
		log.Printf("Error processing %s webhook: %v", provider, err)
		switch {
		case errors.Is(err, service.ErrPayOSVerifiedDataNil), errors.Is(err, service.ErrPaymentWebhookInvalid):
			http.Error(w, "invalid webhook", http.StatusBadRequest)
			return
		case errors.Is(err, service.ErrPaymentCredentialsNotFound):
			http.Error(w, "payment credentials not found", http.StatusUnauthorized)
			return
		case errors.Is(err, service.ErrPaymentProviderNotFound):
			http.Error(w, "payment provider not found", http.StatusBadRequest)
			return
		}
		http.Error(w, "webhook processing failed", http.StatusInternalServerError)
		return
	}
	writePaymentWebhookSuccess(w)
}

// decryptPayOSConfig decrypts RSA-encrypted PayOS credential fields.
func (h *PaymentHandler) decryptPayOSConfig(req payOSConfigRequest) (service.PayOSCredentials, error) {
	clientID, err := h.decryptSecret(req.ClientID, "client_id")
	if err != nil {
		return service.PayOSCredentials{}, err
	}
	apiKey, err := h.decryptSecret(req.APIKey, "api_key")
	if err != nil {
		return service.PayOSCredentials{}, err
	}
	checksumKey, err := h.decryptSecret(req.ChecksumKey, "checksum_key")
	if err != nil {
		return service.PayOSCredentials{}, err
	}
	return service.PayOSCredentials{
		ClientID:    clientID,
		APIKey:      apiKey,
		ChecksumKey: checksumKey,
	}, nil
}

// decryptSecret decodes a base64 RSA-OAEP encrypted request field.
func (h *PaymentHandler) decryptSecret(cipherText, fieldName string) (string, error) {
	cipherBytes, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", errors.New("Invalid " + fieldName + " encoding")
	}
	plainBytes, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, h.privateKey, cipherBytes, nil)
	if err != nil {
		return "", errors.New("Failed to decrypt " + fieldName)
	}
	return string(plainBytes), nil
}

// authenticatedManagerID extracts the current manager user ID from request claims.
func authenticatedManagerID(w http.ResponseWriter, r *http.Request) (string, bool) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return "", false
	}
	return claims.Subject, true
}

// writePaymentWebhookSuccess writes the acknowledgement body expected by payment providers.
func writePaymentWebhookSuccess(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"success": true}`)); err != nil {
		log.Printf("Failed to write payment webhook response: %v", err)
	}
}

// writePaymentRedirectText renders provider redirect data without updating payment state.
func writePaymentRedirectText(w http.ResponseWriter, r *http.Request, message string) {
	query := r.URL.Query()
	lines := []string{message}
	lines = appendPaymentRedirectParam(lines, "Mã kết quả", query.Get("code"))
	lines = appendPaymentRedirectParam(lines, "Mã link thanh toán", query.Get("id"))
	lines = appendPaymentRedirectParam(lines, "Trạng thái hủy", query.Get("cancel"))
	lines = appendPaymentRedirectParam(lines, "Trạng thái thanh toán", query.Get("status"))
	lines = appendPaymentRedirectParam(lines, "Mã đơn hàng", query.Get("orderCode"))
	lines = append(lines, "", "Hóa đơn sẽ được cập nhật khi webhook hợp lệ được xử lý.")

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(strings.Join(lines, "\n") + "\n")); err != nil {
		log.Printf("Failed to write payment redirect response: %v", err)
	}
}

// appendPaymentRedirectParam adds a returned provider query param when it is present.
func appendPaymentRedirectParam(lines []string, label, value string) []string {
	if value == "" {
		return lines
	}
	return append(lines, label+": "+value)
}
