package handler

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service"
)

type ZaloHandler struct {
	zaloService service.ZaloService
	privateKey  *rsa.PrivateKey
	publicKey   *rsa.PublicKey
	appURL      string
}

func NewZaloHandler(zaloService service.ZaloService, appURL string) *ZaloHandler {
	// Generate RSA key pair for frontend transport encryption
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("failed to generate RSA key for ZaloHandler: %v", err)
	}

	return &ZaloHandler{
		zaloService: zaloService,
		privateKey:  privateKey,
		publicKey:   &privateKey.PublicKey,
		appURL:      appURL,
	}
}

// GetPublicKey returns the base64-encoded SPKI public key for the frontend
func (h *ZaloHandler) GetPublicKey(w http.ResponseWriter, r *http.Request) {
	pubASN1, err := x509.MarshalPKIXPublicKey(h.publicKey)
	if err != nil {
		http.Error(w, "Failed to marshal public key", http.StatusInternalServerError)
		return
	}

	pubB64 := base64.StdEncoding.EncodeToString(pubASN1)
	json.NewEncoder(w).Encode(map[string]string{
		"public_key": pubB64,
	})
}

type ConfigReq struct {
	BotToken      string `json:"bot_token"`       // Base64 encrypted with RSA
}

func (h *ZaloHandler) SaveConfig(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	managerID := claims.Subject

	var req ConfigReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Decrypt botToken
	botTokenCipher, err := base64.StdEncoding.DecodeString(req.BotToken)
	if err != nil {
		http.Error(w, "Invalid bot_token encoding", http.StatusBadRequest)
		return
	}
	botTokenBytes, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, h.privateKey, botTokenCipher, nil)
	if err != nil {
		http.Error(w, "Failed to decrypt bot_token", http.StatusBadRequest)
		return
	}
	botToken := string(botTokenBytes)

	// Generate 12-char webhook secret
	secretToken := uuid.New().String()[:12]
	webhookURL := h.appURL + "/api/v1/zalo/webhooks/" + managerID

	if err := h.zaloService.SaveZaloConfig(r.Context(), managerID, botToken, webhookURL, secretToken); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

func (h *ZaloHandler) GetConfigStatus(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	managerID := claims.Subject

	hasConfig, err := h.zaloService.GetZaloConfigStatus(r.Context(), managerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	webhookURL := h.appURL + "/api/v1/zalo/webhooks/" + managerID

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"has_config":         hasConfig.HasConfig,
		"is_zalo_bot_active": hasConfig.IsActive,
		"webhook_url":        webhookURL,
		"is_linked":          hasConfig.IsLinked,
		"bot_id":             hasConfig.BotID,
		"manager_id":         hasConfig.ManagerID,
	})
}

func (h *ZaloHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	managerID := chi.URLParam(r, "managerID")
	secretTokenHeader := r.Header.Get("X-Bot-Api-Secret-Token") // or X-Zalo-Signature depending on config

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if err := h.zaloService.HandleWebhook(r.Context(), managerID, body, secretTokenHeader); err != nil {
		// Log the error but return 200 OK to Zalo so it doesn't retry
		log.Printf("Webhook error for manager %s: %v", managerID, err)
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ZaloHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	managerID := claims.Subject

	var req struct {
		ChatID string `json:"chat_id"`
		Text   string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.zaloService.SendTextMessage(r.Context(), managerID, req.ChatID, req.Text); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ZaloHandler) SendInvoice(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	managerID := claims.Subject

	invoiceID := chi.URLParam(r, "id")
	if invoiceID == "" {
		http.Error(w, "missing invoice id", http.StatusBadRequest)
		return
	}

	if err := h.zaloService.SendInvoiceToZalo(r.Context(), managerID, invoiceID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}
