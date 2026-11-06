package thuchi

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/plugin"
	"github.com/mihb123/quanly-phongtro/pkg/thuchi"
)

const PluginID = "thu-chi"

var instance = &Plugin{}

func init() { plugin.Register(instance) }

type Plugin struct {
	api    http.Handler
	public http.Handler
	bot    *thuchi.TelegramBot
}

func Use(api, public http.Handler, bot *thuchi.TelegramBot) {
	instance.api = api
	instance.public = public
	instance.bot = bot
}

func (p *Plugin) Manifest() plugin.Manifest {
	return plugin.Manifest{
		ID:             PluginID,
		Name:           "Thu chi cá nhân",
		Version:        "1.0.0",
		Description:    "Sổ thu chi riêng của bạn: ghi tay (cả khi mất mạng), tự nhận giao dịch ngân hàng qua SePay / thông báo / email, phân loại tự học, thống kê, hạn mức, vay nợ, ghi qua bot Telegram.",
		Author:         "quanly-phongtro",
		DefaultEnabled: false,
	}
}

func (p *Plugin) Init(host plugin.Host) error {
	if p.api == nil || p.public == nil {
		return errors.New("thu-chi: module chưa được nối (thiếu thuchi.Use trong cmd/api/main.go)")
	}
	if p.bot != nil {
		p.bot.OwnerActive = func(ctx context.Context, ownerID string) bool {
			enabled, err := host.Enabled(ctx, ownerID)
			return err == nil && enabled
		}
	}
	return nil
}

func (p *Plugin) Routes(r chi.Router) {
	r.Handle("/*", p.api)
}

func (p *Plugin) PublicRoutes(r chi.Router) {
	r.Handle("/*", p.public)
}
