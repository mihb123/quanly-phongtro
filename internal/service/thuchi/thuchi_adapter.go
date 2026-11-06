package thuchi

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/pkg/thuchi"
)

const (
	APIPrefix    = "/api/v1/plugins/thu-chi"
	PublicPrefix = APIPrefix + "/public"
)

type aesCrypto struct{ key []byte }

func (c aesCrypto) Encrypt(plaintext string) (string, error) {
	return security.Encrypt(plaintext, c.key)
}

func (c aesCrypto) Decrypt(ciphertext string) (string, error) {
	return security.Decrypt(ciphertext, c.key)
}

type TelegramConfig struct {
	BotToken string
	ChatIDs  []int64
	APIBase  string
}

type Module struct {
	Service  *thuchi.Service
	Telegram *thuchi.TelegramBot
	API      http.Handler
	Public   http.Handler
}

func New(db *sql.DB, secretKey []byte, publicBaseURL string, tg TelegramConfig) *Module {
	svc := thuchi.NewService(thuchi.NewPGStore(db), aesCrypto{key: secretKey})
	var bot *thuchi.TelegramBot
	if tg.BotToken != "" {
		bot = thuchi.NewTelegramBot(svc, tg.BotToken, tg.ChatIDs, "")
		if tg.APIBase != "" {
			bot.APIBase = tg.APIBase
		}
	}
	h := thuchi.NewHandler(svc, managerFromClaims, publicBaseURL+PublicPrefix)
	return &Module{
		Service:  svc,
		Telegram: bot,
		API:      http.StripPrefix(APIPrefix, h.API()),
		Public:   http.StripPrefix(PublicPrefix, h.Public()),
	}
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
