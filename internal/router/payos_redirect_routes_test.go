package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/handler"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

// TestPayOSRedirectRoutesArePublic verifies SDK redirect URLs are mounted without auth.
func TestPayOSRedirectRoutesArePublic(t *testing.T) {
	router := New(
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

	tests := []struct {
		name     string
		path     string
		wantBody string
	}{
		{
			name:     "return",
			path:     "/api/v1/payos/return?code=00&id=link-1&cancel=false&status=PAID&orderCode=803347",
			wantBody: "PayOS đã chuyển hướng về hệ thống sau khi thanh toán.",
		},
		{
			name:     "cancel",
			path:     "/api/v1/payos/cancel?code=00&id=link-1&cancel=true&status=CANCELLED&orderCode=803347",
			wantBody: "Bạn đã hủy thanh toán PayOS.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if contentType := rec.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/plain") {
				t.Fatalf("Content-Type = %q, want text/plain", contentType)
			}
			body := rec.Body.String()
			if !strings.Contains(body, tt.wantBody) {
				t.Fatalf("body = %q, want %q", body, tt.wantBody)
			}
			if !strings.Contains(body, "Mã đơn hàng: 803347") {
				t.Fatalf("body = %q, want returned orderCode", body)
			}
		})
	}
}

// TestPayOSRoutesAreNotMountedWithoutHandler verifies disabled PayOS config leaves routes unavailable.
func TestPayOSRoutesAreNotMountedWithoutHandler(t *testing.T) {
	router := New(
		(*handler.AuthHandler)(nil),
		(*handler.HouseHandler)(nil),
		(*handler.RoomHandler)(nil),
		(*security.JWTProvider)(nil),
		(*handler.TenantHandler)(nil),
		(*handler.InvoiceHandler)(nil),
		(*handler.ZaloHandler)(nil),
		(*handler.HouseCostHandler)(nil),
		nil,
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payos/webhook", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
