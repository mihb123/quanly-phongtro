package router

import (
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/handler"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

// TestNewRegistersRoutesWithoutPanic guards against duplicate chi route mounts.
func TestNewRegistersRoutesWithoutPanic(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("New panicked while registering routes: %v", recovered)
		}
	}()

	New(
		(*handler.AuthHandler)(nil),
		(*handler.HouseHandler)(nil),
		(*handler.RoomHandler)(nil),
		(*security.JWTProvider)(nil),
		(*handler.TenantHandler)(nil),
		(*handler.InvoiceHandler)(nil),
		(*handler.ZaloHandler)(nil),
		(*handler.HouseCostHandler)(nil),
		handler.NewPayOSWebhookHandler(nil),
	)
}
