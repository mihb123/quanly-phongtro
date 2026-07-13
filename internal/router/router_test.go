package router

import (
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/handler/auth"
	"github.com/mihb123/quanly-phongtro/internal/handler/house"
	"github.com/mihb123/quanly-phongtro/internal/handler/invoice"
	"github.com/mihb123/quanly-phongtro/internal/handler/payment"
	"github.com/mihb123/quanly-phongtro/internal/handler/room"
	"github.com/mihb123/quanly-phongtro/internal/handler/tenant"
	"github.com/mihb123/quanly-phongtro/internal/handler/zalo"
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
		(*auth.AuthHandler)(nil),
		(*house.HouseHandler)(nil),
		(*room.RoomHandler)(nil),
		(*security.JWTProvider)(nil),
		(*tenant.TenantHandler)(nil),
		(*invoice.InvoiceHandler)(nil),
		(*zalo.ZaloHandler)(nil),
		(*house.HouseCostHandler)(nil),
		payment.NewPayOSWebhookHandler(nil),
	)
}
