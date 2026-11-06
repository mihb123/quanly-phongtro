package thuchi

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mihb123/quanly-phongtro/pkg/sepay"
)

const (
	maxSyncBody   = 4 << 20
	maxNotifyBody = 64 << 10
	maxEmailBody  = 8 << 20
	maxSePayBody  = 1 << 20
	signatureSkew = 5 * time.Minute
)

type OwnerFunc func(r *http.Request) (string, bool)

type Handler struct {
	svc       *Service
	owner     OwnerFunc
	publicURL string
}

func NewHandler(svc *Service, owner OwnerFunc, publicBaseURL string) *Handler {
	return &Handler{svc: svc, owner: owner, publicURL: strings.TrimRight(publicBaseURL, "/")}
}

func (h *Handler) API() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /sync/pull", h.pull)
	mux.HandleFunc("POST /sync/push", h.push)
	mux.HandleFunc("GET /ingest/info", h.ingestInfo)
	mux.HandleFunc("POST /ingest/tokens", h.createToken)
	mux.HandleFunc("DELETE /ingest/tokens/{id}", h.revokeToken)
	mux.HandleFunc("PUT /ingest/sepay", h.saveSePay)
	mux.HandleFunc("GET /telegram/info", h.telegramInfo)
	mux.HandleFunc("POST /telegram/link", h.telegramLink)
	mux.HandleFunc("PUT /telegram/links/{id}", h.telegramUpdateLink)
	mux.HandleFunc("DELETE /telegram/links/{id}", h.telegramUnlink)
	return mux
}

func (h *Handler) Public() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /{ownerID}/ingest/{source}", h.ingest)
	mux.HandleFunc("POST /telegram/webhook", h.telegramWebhook)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"status": status, "data": data})
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"status": status, "message": msg})
}

func (h *Handler) ownerOr401(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := h.owner(r)
	if !ok || !ValidID(id) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return "", false
	}
	return id, true
}

func (h *Handler) pull(w http.ResponseWriter, r *http.Request) {
	owner, ok := h.ownerOr401(w, r)
	if !ok {
		return
	}
	since, _ := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 2000 {
		limit = 1000
	}
	res, err := h.svc.Pull(r.Context(), owner, since, limit)
	if err != nil {
		log.Printf("thuchi: pull: %v", err)
		writeErr(w, http.StatusInternalServerError, "pull failed")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *Handler) push(w http.ResponseWriter, r *http.Request) {
	owner, ok := h.ownerOr401(w, r)
	if !ok {
		return
	}
	var body struct {
		Mutations []Mutation `json:"mutations"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxSyncBody)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(body.Mutations) > 500 {
		writeErr(w, http.StatusRequestEntityTooLarge, "too many mutations")
		return
	}
	res, err := h.svc.Push(r.Context(), owner, body.Mutations)
	if err != nil {
		log.Printf("thuchi: push: %v", err)
		writeErr(w, http.StatusInternalServerError, "push failed")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

type ingestInfo struct {
	OwnerID  string            `json:"owner_id"`
	URLs     map[string]string `json:"urls"`
	Tokens   []*IngestToken    `json:"tokens"`
	SePay    *IngestToken      `json:"sepay"`
	Packages map[string]string `json:"packages"`
	Parsers  []string          `json:"parsers"`
}

func (h *Handler) ingestURL(owner, source string) string {
	return h.publicURL + "/" + owner + "/ingest/" + source
}

func (h *Handler) ingestInfo(w http.ResponseWriter, r *http.Request) {
	owner, ok := h.ownerOr401(w, r)
	if !ok {
		return
	}
	tokens, err := h.svc.Store.ListIngestTokens(r.Context(), owner)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "load tokens failed")
		return
	}
	info := ingestInfo{
		OwnerID:  owner,
		URLs:     map[string]string{},
		Tokens:   []*IngestToken{},
		Packages: NotifyPackages,
		Parsers:  []string{"sepay"},
	}
	for code := range NotifyParsers {
		info.Parsers = append(info.Parsers, "notify:"+code)
	}
	for domain := range EmailParsers {
		info.Parsers = append(info.Parsers, "email:"+domain)
	}
	for _, s := range []string{SourceSePay, SourceNotify, SourceEmail} {
		info.URLs[s] = h.ingestURL(owner, s)
	}
	for _, t := range tokens {
		if t.Kind == SourceSePay {
			info.SePay = t
			continue
		}
		info.Tokens = append(info.Tokens, t)
	}
	writeJSON(w, http.StatusOK, info)
}

func randomSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (h *Handler) createToken(w http.ResponseWriter, r *http.Request) {
	owner, ok := h.ownerOr401(w, r)
	if !ok {
		return
	}
	var body struct {
		Kind string `json:"kind"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil || !oneOf(body.Kind, SourceNotify, SourceEmail) {
		writeErr(w, http.StatusBadRequest, "kind must be notify or email")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" || len([]rune(name)) > 60 {
		writeErr(w, http.StatusBadRequest, "invalid name")
		return
	}
	secret := randomSecret()
	enc, err := h.svc.Crypto.Encrypt(secret)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "encrypt failed")
		return
	}
	t := &IngestToken{ID: NewID(), OwnerID: owner, Kind: body.Kind, Name: name, SecretEnc: enc}
	if err := h.svc.Store.CreateIngestToken(r.Context(), t); err != nil {
		writeErr(w, http.StatusInternalServerError, "create token failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"token":  t,
		"secret": t.ID + "." + secret,
		"url":    h.ingestURL(owner, body.Kind),
	})
}

func (h *Handler) revokeToken(w http.ResponseWriter, r *http.Request) {
	owner, ok := h.ownerOr401(w, r)
	if !ok {
		return
	}
	if err := h.svc.Store.RevokeIngestToken(r.Context(), owner, r.PathValue("id")); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeErr(w, http.StatusNotFound, "token not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, "revoke failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) saveSePay(w http.ResponseWriter, r *http.Request) {
	owner, ok := h.ownerOr401(w, r)
	if !ok {
		return
	}
	var body struct {
		AuthMethod string `json:"auth_method"`
		Secret     string `json:"secret"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	body.Secret = strings.TrimSpace(body.Secret)
	if !oneOf(body.AuthMethod, sepay.AuthMethodHMAC, sepay.AuthMethodAPIKey) || len(body.Secret) < 8 || len(body.Secret) > 512 {
		writeErr(w, http.StatusBadRequest, "auth_method must be hmac or apikey and secret is required")
		return
	}
	enc, err := h.svc.Crypto.Encrypt(body.Secret)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "encrypt failed")
		return
	}
	tokens, err := h.svc.Store.ListIngestTokens(r.Context(), owner)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "load tokens failed")
		return
	}
	t := &IngestToken{ID: NewID(), OwnerID: owner, Kind: SourceSePay, Name: "SePay", SecretEnc: enc, Config: map[string]string{"auth_method": body.AuthMethod}}
	if err := h.svc.Store.CreateIngestToken(r.Context(), t); err != nil {
		writeErr(w, http.StatusInternalServerError, "save failed")
		return
	}
	for _, old := range tokens {
		if old.Kind == SourceSePay {
			_ = h.svc.Store.RevokeIngestToken(r.Context(), owner, old.ID)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"sepay": t, "url": h.ingestURL(owner, SourceSePay)})
}

func (h *Handler) ingest(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("ownerID")
	source := r.PathValue("source")
	limit := map[string]int64{SourceSePay: maxSePayBody, SourceNotify: maxNotifyBody, SourceEmail: maxEmailBody}[source]
	if limit == 0 || !ValidID(owner) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
	if err != nil {
		writeErr(w, http.StatusRequestEntityTooLarge, "body too large")
		return
	}
	ctx := r.Context()
	tokens, err := h.svc.Store.ListIngestTokens(ctx, owner)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	tokenID, err := h.authenticate(r, source, body, tokens)
	if err != nil {
		log.Printf("thuchi: ingest %s for %s rejected: %v", source, owner, err)
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	_ = h.svc.Store.TouchIngestToken(ctx, tokenID)

	key := ""
	switch source {
	case SourceNotify:
		if p, err := DecodeNotify(body); err == nil {
			key = p.Package
		}
	case SourceSePay:
		var p SePayPayload
		if json.Unmarshal(body, &p) == nil {
			key = strings.TrimSpace(p.Gateway + " " + Last4(p.AccountNumber))
		}
	case SourceEmail:
		if m, err := ParseEmail(body); err == nil {
			key = m.From
		}
	}
	err = h.svc.Store.RunOwner(ctx, owner, func(ctx2 context.Context, tx Tx) error {
		return tx.SaveRawEvent(ctx2, &RawEvent{Source: source, SourceKey: truncate(key, 200), Payload: string(body), Status: RawPending})
	})
	if err != nil {
		log.Printf("thuchi: store raw event: %v", err)
		writeErr(w, http.StatusInternalServerError, "store failed")
		return
	}
	h.svc.Kick()
	if source == SourceSePay {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":true}`))
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"accepted": true})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func (h *Handler) authenticate(r *http.Request, source string, body []byte, tokens []*IngestToken) (string, error) {
	now := h.svc.now()
	if source == SourceSePay {
		for _, t := range tokens {
			if t.Kind != SourceSePay {
				continue
			}
			secret, err := h.svc.Crypto.Decrypt(t.SecretEnc)
			if err != nil {
				return "", err
			}
			creds := sepay.Credentials{WebhookAuthMethod: t.Config["auth_method"]}
			if creds.WebhookAuthMethod == sepay.AuthMethodAPIKey {
				creds.WebhookAPIKey = secret
			} else {
				creds.WebhookAuthMethod = sepay.AuthMethodHMAC
				creds.WebhookSecret = secret
			}
			if err := sepay.AuthenticateWebhook(r.Header, body, creds, now); err != nil {
				return "", err
			}
			return t.ID, nil
		}
		return "", errors.New("sepay not configured")
	}

	tokenID, secret := "", ""
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		tokenID, secret, _ = strings.Cut(strings.TrimSpace(strings.TrimPrefix(auth, "Bearer ")), ".")
	} else {
		tokenID = r.Header.Get("X-ThuChi-Token")
	}
	var token *IngestToken
	for _, t := range tokens {
		if t.ID == tokenID && t.Kind == source {
			token = t
		}
	}
	if token == nil {
		return "", errors.New("unknown token")
	}
	stored, err := h.svc.Crypto.Decrypt(token.SecretEnc)
	if err != nil {
		return "", err
	}
	if secret != "" {
		if !hmac.Equal([]byte(secret), []byte(stored)) {
			return "", errors.New("bad secret")
		}
		return token.ID, nil
	}
	ts := r.Header.Get("X-ThuChi-Timestamp")
	sig := strings.TrimPrefix(r.Header.Get("X-ThuChi-Signature"), "sha256=")
	unix, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || sig == "" {
		return "", errors.New("missing signature")
	}
	if skew := now.Sub(time.Unix(unix, 0)); skew > signatureSkew || skew < -signatureSkew {
		return "", errors.New("timestamp skew")
	}
	if !hmac.Equal([]byte(sig), []byte(SignBody(stored, ts, body))) {
		return "", fmt.Errorf("bad signature")
	}
	return token.ID, nil
}

func SignBody(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + string(body)))
	return hex.EncodeToString(mac.Sum(nil))
}
