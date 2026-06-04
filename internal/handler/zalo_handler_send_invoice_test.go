package handler_test

import (

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/handler"
	"github.com/mihb123/quanly-phongtro/internal/mock/mock_service"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"go.uber.org/mock/gomock"
)

func TestZaloHandler_SendInvoice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	zaloSvc := mock_service.NewMockZaloService(ctrl)
	h := handler.NewZaloHandler(zaloSvc, "test.local")

	// Mock chi router params
	r := chi.NewRouter()
	r.Post("/invoices/{id}/send", h.SendInvoice)

	req := httptest.NewRequest("POST", "/invoices/inv1/send", nil)
	
	claims := &security.Claims{}
	claims.Subject = "manager1"
	ctx := security.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	zaloSvc.EXPECT().SendInvoiceToZalo(gomock.Any(), "manager1", "inv1").Return(nil)

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %v, got %v", http.StatusOK, rr.Code)
	}
}
