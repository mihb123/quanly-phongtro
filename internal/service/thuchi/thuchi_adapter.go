package thuchi

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/pkg/thuchi"
)

const (
	APIPrefix          = "/api/v1/plugins/thu-chi"
	PublicPrefix       = APIPrefix + "/public"
	HookPrefix         = APIPrefix + "/hook"
	TelegramWebhookURL = HookPrefix + "/telegram/webhook"
)

type aesCrypto struct{ key []byte }

func (c aesCrypto) Encrypt(plaintext string) (string, error) {
	return security.Encrypt(plaintext, c.key)
}

func (c aesCrypto) Decrypt(ciphertext string) (string, error) {
	return security.Decrypt(ciphertext, c.key)
}

type TelegramConfig struct {
	BotToken   string
	ChatIDs    []int64
	APIBase    string
	WebhookURL string
}

type Module struct {
	Service  *thuchi.Service
	Telegram *thuchi.TelegramBot
	API      http.Handler
	Public   http.Handler
	Hook     http.Handler
}

func New(db *sql.DB, secretKey []byte, publicBaseURL string, tg TelegramConfig) *Module {
	svc := thuchi.NewService(thuchi.NewPGStore(db), aesCrypto{key: secretKey})
	svc.Preset = &thuchi.LandlordPreset
	var bot *thuchi.TelegramBot
	if tg.BotToken != "" {
		bot = thuchi.NewTelegramBot(svc, tg.BotToken, tg.ChatIDs, tg.WebhookURL)
		bot.SetupPath = setupPath(publicBaseURL)
		if tg.APIBase != "" {
			bot.APIBase = tg.APIBase
		}
	}
	h := thuchi.NewHandler(svc, managerFromClaims, publicBaseURL+PublicPrefix)
	hook := http.NewServeMux()
	if bot != nil && tg.WebhookURL != "" {
		hook.HandleFunc("POST "+TelegramWebhookURL, bot.ServeWebhook)
	}
	return &Module{
		Service:  svc,
		Telegram: bot,
		API:      http.StripPrefix(APIPrefix, h.API()),
		Public:   http.StripPrefix(PublicPrefix, h.Public()),
		Hook:     hook,
	}
}

func setupPath(publicBaseURL string) string {
	site := "trang quản lý phòng trọ"
	if u, err := url.Parse(publicBaseURL); err == nil && u.Host != "" {
		site = u.Host
	}
	return site + " → Tiện ích → Thu chi cá nhân → Cài đặt → Telegram"
}

func managerFromClaims(r *http.Request) (string, bool) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		return "", false
	}
	sub, err := claims.GetSubject()
	return sub, err == nil && sub != ""
}

func (m *Module) Start(ctx context.Context) {
	go m.Service.RunWorker(ctx)
	if m.Telegram != nil {
		go m.Telegram.Run(ctx)
	}
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			if err := m.Service.Store.PurgeTombstones(ctx, 30*24*time.Hour); err != nil {
				log.Printf("thuchi: purge tombstones: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
