package thuchi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
)

type telegramInfo struct {
	Enabled            bool              `json:"enabled"`
	BotUsername        string            `json:"bot_username"`
	Mode               string            `json:"mode"`
	ReadsGroupMessages bool              `json:"reads_group_messages"`
	Links              []*TelegramLink   `json:"links"`
	Examples           []TelegramExample `json:"examples"`
}

func (h *Handler) telegramWebhook(w http.ResponseWriter, r *http.Request) {
	if h.svc.Telegram == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	h.svc.Telegram.ServeWebhook(w, r)
}

func (h *Handler) telegramInfo(w http.ResponseWriter, r *http.Request) {
	owner, ok := h.ownerOr401(w, r)
	if !ok {
		return
	}
	links, err := h.svc.Store.ListTelegramLinks(r.Context(), owner)
	if err != nil {
		log.Printf("thuchi: list telegram links: %v", err)
		writeErr(w, http.StatusInternalServerError, "load failed")
		return
	}
	info := telegramInfo{Links: links, Examples: h.svc.preset().Examples}
	if bot := h.svc.Telegram; bot != nil {
		info.Enabled = true
		info.BotUsername = bot.Username()
		info.Mode = bot.Mode()
		info.ReadsGroupMessages = bot.ReadsGroupMessages()
	}
	writeJSON(w, http.StatusOK, info)
}

func (h *Handler) telegramLink(w http.ResponseWriter, r *http.Request) {
	owner, ok := h.ownerOr401(w, r)
	if !ok {
		return
	}
	bot := h.svc.Telegram
	if bot == nil || bot.Username() == "" {
		writeErr(w, http.StatusServiceUnavailable, "Bot Telegram chưa được cấu hình")
		return
	}
	code := NewTelegramCode()
	expires := h.svc.now().Add(telegramCodeTTL).UTC()
	if err := h.svc.Store.CreateTelegramCode(r.Context(), owner, code, expires); err != nil {
		log.Printf("thuchi: create telegram code: %v", err)
		writeErr(w, http.StatusInternalServerError, "create failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"code":       code,
		"command":    "/link " + code,
		"url":        "https://t.me/" + bot.Username() + "?start=" + code,
		"expires_at": expires.Format(time.RFC3339),
	})
}

func (h *Handler) telegramUpdateLink(w http.ResponseWriter, r *http.Request) {
	owner, ok := h.ownerOr401(w, r)
	if !ok {
		return
	}
	var body struct {
		DefaultAccountID *string `json:"default_account_id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil || (body.DefaultAccountID != nil && !ValidID(*body.DefaultAccountID)) {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	err := h.svc.Store.SetTelegramLinkAccount(r.Context(), owner, r.PathValue("id"), body.DefaultAccountID)
	if errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusNotFound, "link not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "update failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) telegramUnlink(w http.ResponseWriter, r *http.Request) {
	owner, ok := h.ownerOr401(w, r)
	if !ok {
		return
	}
	err := h.svc.Store.DeleteTelegramLink(r.Context(), owner, r.PathValue("id"))
	if errors.Is(err, ErrNotFound) {
		writeErr(w, http.StatusNotFound, "link not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "unlink failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
