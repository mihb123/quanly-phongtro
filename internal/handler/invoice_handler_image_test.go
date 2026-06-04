package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mihb123/quanly-phongtro/internal/handler"
	"github.com/mihb123/quanly-phongtro/internal/mock/mock_service"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"go.uber.org/mock/gomock"
)

func TestInvoiceHandler_DownloadInvoiceImage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invSvc := mock_service.NewMockInvoiceService(ctrl)
	imgSvc := mock_service.NewMockImageService(ctrl)
	h := handler.NewInvoiceHandler(invSvc, imgSvc)

	r := chi.NewRouter()
	r.Get("/invoices/{id}/image", h.DownloadInvoiceImage)

	req := httptest.NewRequest("GET", "/invoices/inv1/image", nil)
	
	claims := &security.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "manager1",
		},
	}
	ctx := security.WithClaims(req.Context(), claims)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	invoice := &model.InvoiceWithRoom{
		Invoice: model.Invoice{
			ID: "inv1",
		},
	}
	invSvc.EXPECT().GetInvoice(gomock.Any(), "manager1", "inv1").Return(invoice, nil)
	imgSvc.EXPECT().GenerateInvoiceImage(gomock.Any(), invoice).Return([]byte("fake-png"), nil)

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %v, got %v", http.StatusOK, rr.Code)
	}
	if rr.Header().Get("Content-Type") != "image/png" {
		t.Errorf("expected content type image/png, got %v", rr.Header().Get("Content-Type"))
	}
	if rr.Body.String() != "fake-png" {
		t.Errorf("expected body fake-png, got %v", rr.Body.String())
	}
}
