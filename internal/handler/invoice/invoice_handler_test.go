package invoice_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mihb123/quanly-phongtro/internal/handler/invoice"
	"github.com/mihb123/quanly-phongtro/internal/mock/mock_service"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"go.uber.org/mock/gomock"
)

func setClaims(req *http.Request, userID string) *http.Request {
	claims := &security.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: userID},
	}
	ctx := security.WithClaims(req.Context(), claims)
	return req.WithContext(ctx)
}

func setURLParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

type failingResponseWriter struct {
	header http.Header
	status int
}

// Header returns writable headers for handlers before the forced write failure.
func (w *failingResponseWriter) Header() http.Header {
	return w.header
}

// Write always fails so JSON encoder error paths can be exercised.
func (w *failingResponseWriter) Write([]byte) (int, error) {
	return 0, errors.New("write error")
}

// WriteHeader records the status that the handler attempted to send.
func (w *failingResponseWriter) WriteHeader(status int) {
	w.status = status
}

// newFailingResponseWriter builds a response writer that fails body writes.
func newFailingResponseWriter() *failingResponseWriter {
	return &failingResponseWriter{header: make(http.Header)}
}

func TestInvoiceHandler_CreateInvoice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoiceSvc := mock_service.NewMockInvoiceService(ctrl)
	imageSvc := mock_service.NewMockImageService(ctrl)
	h := invoice.NewInvoiceHandler(invoiceSvc, imageSvc)

	tests := []struct {
		name           string
		body           interface{}
		setupAuth      func(*http.Request) *http.Request
		mock           func()
		expectedStatus int
	}{
		{
			name: "Happy path",
			body: map[string]interface{}{
				"room_id":               "room-1",
				"period":                "2026-06",
				"new_electricity_index": 100,
				"new_water_index":       10,
				"other_fee":             50000,
				"discount":              0,
				"vehicle_count":         2,
			},
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					CreateInvoice(gomock.Any(), "user-1", gomock.Any()).
					Return(&model.InvoiceWithRoom{}, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Invalid body",
			body: "invalid-json",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Validation error - missing room_id",
			body: map[string]interface{}{
				"period":                "2026-06",
				"new_electricity_index": 100,
				"new_water_index":       10,
			},
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock:           func() {},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Validation error - invalid period",
			body: map[string]interface{}{
				"room_id":               "room-1",
				"period":                "2026-6",
				"new_electricity_index": 100,
				"new_water_index":       10,
			},
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock:           func() {},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "Unauthorized",
			body: map[string]interface{}{},
			setupAuth: func(r *http.Request) *http.Request {
				return r
			},
			mock:           func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Service error - Duplicate Invoice",
			body: map[string]interface{}{
				"room_id":               "room-1",
				"period":                "2026-06",
				"new_electricity_index": 100,
				"new_water_index":       10,
			},
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					CreateInvoice(gomock.Any(), "user-1", gomock.Any()).
					Return(nil, model.ErrDuplicateInvoice)
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "Service error - Room Not Found",
			body: map[string]interface{}{
				"room_id":               "room-1",
				"period":                "2026-06",
				"new_electricity_index": 100,
				"new_water_index":       10,
			},
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					CreateInvoice(gomock.Any(), "user-1", gomock.Any()).
					Return(nil, model.ErrRoomNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Service error - invalid electricity index",
			body: map[string]interface{}{
				"room_id":               "room-1",
				"period":                "2026-06",
				"new_electricity_index": 100,
				"new_water_index":       10,
			},
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					CreateInvoice(gomock.Any(), "user-1", gomock.Any()).
					Return(nil, model.ErrInvalidElectricityIndex)
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()

			var buf bytes.Buffer
			if strBody, ok := tt.body.(string); ok {
				buf.WriteString(strBody)
			} else {
				json.NewEncoder(&buf).Encode(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/invoice", &buf)
			req = tt.setupAuth(req)
			rec := httptest.NewRecorder()

			h.CreateInvoice(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

// TestInvoiceHandler_ResponseEncodingErrors covers JSON encoder failure branches.
func TestInvoiceHandler_ResponseEncodingErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoiceSvc := mock_service.NewMockInvoiceService(ctrl)
	imageSvc := mock_service.NewMockImageService(ctrl)
	h := invoice.NewInvoiceHandler(invoiceSvc, imageSvc)

	invoiceSvc.EXPECT().CreateInvoice(gomock.Any(), "user-1", gomock.Any()).Return(&model.InvoiceWithRoom{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/invoice", bytes.NewBufferString(`{
		"room_id":"room-1",
		"period":"2026-06",
		"new_electricity_index":100,
		"new_water_index":10
	}`))
	h.CreateInvoice(newFailingResponseWriter(), setClaims(req, "user-1"))

	invoiceSvc.EXPECT().ListInvoices(gomock.Any(), "user-1", model.InvoiceListFilter{Page: 1, Limit: 20}).Return([]model.InvoiceWithRoom{}, 0, nil)
	req = httptest.NewRequest(http.MethodGet, "/invoice", nil)
	h.ListInvoices(newFailingResponseWriter(), setClaims(req, "user-1"))

	invoiceSvc.EXPECT().GetInvoice(gomock.Any(), "user-1", "inv-1").Return(&model.InvoiceWithRoom{}, nil)
	req = setURLParam(httptest.NewRequest(http.MethodGet, "/invoice/inv-1", nil), "id", "inv-1")
	h.GetInvoice(newFailingResponseWriter(), setClaims(req, "user-1"))

	invoiceSvc.EXPECT().PayInvoice(gomock.Any(), "user-1", "inv-1").Return(&model.Invoice{}, nil)
	req = setURLParam(httptest.NewRequest(http.MethodPatch, "/invoice/inv-1/pay", nil), "id", "inv-1")
	h.PayInvoice(newFailingResponseWriter(), setClaims(req, "user-1"))

	invoiceSvc.EXPECT().UnpayInvoice(gomock.Any(), "user-1", "inv-1").Return(&model.Invoice{}, nil)
	req = setURLParam(httptest.NewRequest(http.MethodPatch, "/invoice/inv-1/unpay", nil), "id", "inv-1")
	h.UnpayInvoice(newFailingResponseWriter(), setClaims(req, "user-1"))
}

func TestInvoiceHandler_ListInvoices(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoiceSvc := mock_service.NewMockInvoiceService(ctrl)
	imageSvc := mock_service.NewMockImageService(ctrl)
	h := invoice.NewInvoiceHandler(invoiceSvc, imageSvc)

	tests := []struct {
		name           string
		url            string
		setupAuth      func(*http.Request) *http.Request
		mock           func()
		expectedStatus int
		expectedTotal  string
	}{
		{
			name: "Happy path",
			url:  "/invoice?house_id=house-1&room_id=room-1&period=2026-06&status=UNPAID&page=2&limit=10",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				filter := model.InvoiceListFilter{
					HouseID: "house-1",
					RoomID:  "room-1",
					Period:  "2026-06",
					Status:  "UNPAID",
					Page:    2,
					Limit:   10,
				}
				invoiceSvc.EXPECT().
					ListInvoices(gomock.Any(), "user-1", filter).
					Return([]model.InvoiceWithRoom{}, 42, nil)
			},
			expectedStatus: http.StatusOK,
			expectedTotal:  "42",
		},
		{
			name: "Multiple statuses",
			url:  "/invoice?status=UNPAID,partially_paid,PENDING_VERIFICATION",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				filter := model.InvoiceListFilter{
					Status: "UNPAID,partially_paid,PENDING_VERIFICATION",
					Page:   1,
					Limit:  20,
				}
				invoiceSvc.EXPECT().
					ListInvoices(gomock.Any(), "user-1", filter).
					Return([]model.InvoiceWithRoom{{}}, 1, nil)
			},
			expectedStatus: http.StatusOK,
			expectedTotal:  "1",
		},
		{
			name: "Invalid status",
			url:  "/invoice?status=UNPAID,BOGUS",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid page/limit fallback",
			url:  "/invoice?page=invalid&limit=999",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				filter := model.InvoiceListFilter{
					Page:  1,
					Limit: 20, // default when limit > 100 or invalid
				}
				invoiceSvc.EXPECT().
					ListInvoices(gomock.Any(), "user-1", filter).
					Return([]model.InvoiceWithRoom{}, 0, nil)
			},
			expectedStatus: http.StatusOK,
			expectedTotal:  "0",
		},
		{
			name: "Unauthorized",
			url:  "/invoice",
			setupAuth: func(r *http.Request) *http.Request {
				return r
			},
			mock:           func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid period",
			url:  "/invoice?period=2026-6",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Service error",
			url:  "/invoice",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					ListInvoices(gomock.Any(), "user-1", gomock.Any()).
					Return(nil, 0, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req = tt.setupAuth(req)
			rec := httptest.NewRecorder()

			h.ListInvoices(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
			if got := rec.Header().Get("X-Total-Count"); got != tt.expectedTotal {
				t.Errorf("expected X-Total-Count %q, got %q", tt.expectedTotal, got)
			}
		})
	}
}

func TestInvoiceHandler_GetInvoiceBreakdown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoiceSvc := mock_service.NewMockInvoiceService(ctrl)
	imageSvc := mock_service.NewMockImageService(ctrl)
	h := invoice.NewInvoiceHandler(invoiceSvc, imageSvc)

	t.Run("Happy path aggregates expected and collected", func(t *testing.T) {
		filter := model.InvoiceBreakdownFilter{
			HouseIDs: []string{"house-1", "house-2"},
			RoomID:   "room-1",
			Period:   "2026-06",
		}
		invoiceSvc.EXPECT().
			SumInvoicesByStatus(gomock.Any(), "user-1", filter).
			Return([]model.InvoiceStatusTotals{
				{Status: "PAID", InvoiceCount: 2, RoomFee: 6000000, ElectricityFee: 400000, WaterFee: 100000, Discount: 50000, TotalAmount: 6450000},
				{Status: "UNPAID", InvoiceCount: 1, RoomFee: 3000000, ElectricityFee: 200000, OtherFee: 10000, TotalAmount: 3210000},
				{Status: "PENDING_VERIFICATION", InvoiceCount: 1, RoomFee: 2500000, TotalAmount: 2500000},
			}, nil)

		req := httptest.NewRequest(http.MethodGet, "/invoice/breakdown?house_id=house-1&house_ids=house-2,&room_id=room-1&period=2026-06", nil)
		rec := httptest.NewRecorder()
		h.GetInvoiceBreakdown(rec, setClaims(req, "user-1"))

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		var body struct {
			Expected  map[string]float64 `json:"expected"`
			Collected map[string]float64 `json:"collected"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body.Expected["invoice_count"] != 4 || body.Expected["room_fee"] != 11500000 || body.Expected["total_amount"] != 12160000 {
			t.Errorf("unexpected expected totals: %v", body.Expected)
		}
		if body.Collected["invoice_count"] != 2 || body.Collected["room_fee"] != 6000000 || body.Collected["discount"] != 50000 || body.Collected["total_amount"] != 6450000 {
			t.Errorf("unexpected collected totals: %v", body.Collected)
		}
	})

	t.Run("Invalid period", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/invoice/breakdown?period=2026-6", nil)
		rec := httptest.NewRecorder()
		h.GetInvoiceBreakdown(rec, setClaims(req, "user-1"))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("Invalid status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/invoice/breakdown?status=DONE", nil)
		rec := httptest.NewRecorder()
		h.GetInvoiceBreakdown(rec, setClaims(req, "user-1"))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("Unauthorized", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.GetInvoiceBreakdown(rec, httptest.NewRequest(http.MethodGet, "/invoice/breakdown", nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("Service error", func(t *testing.T) {
		invoiceSvc.EXPECT().
			SumInvoicesByStatus(gomock.Any(), "user-1", gomock.Any()).
			Return(nil, errors.New("db error"))
		rec := httptest.NewRecorder()
		h.GetInvoiceBreakdown(rec, setClaims(httptest.NewRequest(http.MethodGet, "/invoice/breakdown", nil), "user-1"))
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rec.Code)
		}
	})
}

func TestInvoiceHandler_GetInvoice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoiceSvc := mock_service.NewMockInvoiceService(ctrl)
	imageSvc := mock_service.NewMockImageService(ctrl)
	h := invoice.NewInvoiceHandler(invoiceSvc, imageSvc)

	tests := []struct {
		name           string
		id             string
		setupAuth      func(*http.Request) *http.Request
		mock           func()
		expectedStatus int
	}{
		{
			name: "Happy path",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					GetInvoice(gomock.Any(), "user-1", "inv-1").
					Return(&model.InvoiceWithRoom{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Unauthorized",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return r
			},
			mock:           func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Missing ID",
			id:   "",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invoice not found",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					GetInvoice(gomock.Any(), "user-1", "inv-1").
					Return(nil, model.ErrInvoiceNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Service error",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					GetInvoice(gomock.Any(), "user-1", "inv-1").
					Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			req := httptest.NewRequest(http.MethodGet, "/invoice/"+tt.id, nil)
			req = tt.setupAuth(req)
			if tt.id != "" {
				req = setURLParam(req, "id", tt.id)
			}
			rec := httptest.NewRecorder()

			h.GetInvoice(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestInvoiceHandler_PayInvoice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoiceSvc := mock_service.NewMockInvoiceService(ctrl)
	imageSvc := mock_service.NewMockImageService(ctrl)
	h := invoice.NewInvoiceHandler(invoiceSvc, imageSvc)

	tests := []struct {
		name           string
		id             string
		setupAuth      func(*http.Request) *http.Request
		mock           func()
		expectedStatus int
	}{
		{
			name: "Happy path",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					PayInvoice(gomock.Any(), "user-1", "inv-1").
					Return(&model.Invoice{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Unauthorized",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return r
			},
			mock:           func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Missing ID",
			id:   "",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Already paid",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					PayInvoice(gomock.Any(), "user-1", "inv-1").
					Return(nil, errors.New("invoice is already paid"))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invoice not found",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					PayInvoice(gomock.Any(), "user-1", "inv-1").
					Return(nil, model.ErrInvoiceNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Service error",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					PayInvoice(gomock.Any(), "user-1", "inv-1").
					Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			req := httptest.NewRequest(http.MethodPatch, "/invoice/"+tt.id+"/pay", nil)
			req = tt.setupAuth(req)
			if tt.id != "" {
				req = setURLParam(req, "id", tt.id)
			}
			rec := httptest.NewRecorder()

			h.PayInvoice(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestInvoiceHandler_UnpayInvoice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoiceSvc := mock_service.NewMockInvoiceService(ctrl)
	imageSvc := mock_service.NewMockImageService(ctrl)
	h := invoice.NewInvoiceHandler(invoiceSvc, imageSvc)

	tests := []struct {
		name           string
		id             string
		setupAuth      func(*http.Request) *http.Request
		mock           func()
		expectedStatus int
	}{
		{
			name: "Happy path",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					UnpayInvoice(gomock.Any(), "user-1", "inv-1").
					Return(&model.Invoice{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Unauthorized",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return r
			},
			mock:           func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Missing ID",
			id:   "",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Already unpaid",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					UnpayInvoice(gomock.Any(), "user-1", "inv-1").
					Return(nil, errors.New("invoice is already unpaid"))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invoice not found",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					UnpayInvoice(gomock.Any(), "user-1", "inv-1").
					Return(nil, model.ErrInvoiceNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Service error",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					UnpayInvoice(gomock.Any(), "user-1", "inv-1").
					Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			req := httptest.NewRequest(http.MethodPatch, "/invoice/"+tt.id+"/unpay", nil)
			req = tt.setupAuth(req)
			if tt.id != "" {
				req = setURLParam(req, "id", tt.id)
			}
			rec := httptest.NewRecorder()

			h.UnpayInvoice(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestInvoiceHandler_DeleteInvoice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoiceSvc := mock_service.NewMockInvoiceService(ctrl)
	imageSvc := mock_service.NewMockImageService(ctrl)
	h := invoice.NewInvoiceHandler(invoiceSvc, imageSvc)

	tests := []struct {
		name           string
		id             string
		setupAuth      func(*http.Request) *http.Request
		mock           func()
		expectedStatus int
	}{
		{
			name: "Happy path",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					DeleteInvoice(gomock.Any(), "user-1", "inv-1").
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "Unauthorized",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return r
			},
			mock:           func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Missing ID",
			id:   "",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Cannot delete paid",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					DeleteInvoice(gomock.Any(), "user-1", "inv-1").
					Return(errors.New("cannot delete a paid invoice"))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invoice not found",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					DeleteInvoice(gomock.Any(), "user-1", "inv-1").
					Return(model.ErrInvoiceNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Service error",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					DeleteInvoice(gomock.Any(), "user-1", "inv-1").
					Return(errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			req := httptest.NewRequest(http.MethodDelete, "/invoice/"+tt.id, nil)
			req = tt.setupAuth(req)
			if tt.id != "" {
				req = setURLParam(req, "id", tt.id)
			}
			rec := httptest.NewRecorder()

			h.DeleteInvoice(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestInvoiceHandler_DownloadInvoiceImage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	invoiceSvc := mock_service.NewMockInvoiceService(ctrl)
	imageSvc := mock_service.NewMockImageService(ctrl)
	h := invoice.NewInvoiceHandler(invoiceSvc, imageSvc)

	tests := []struct {
		name           string
		id             string
		setupAuth      func(*http.Request) *http.Request
		mock           func()
		expectedStatus int
	}{
		{
			name: "Happy path",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				inv := &model.InvoiceWithRoom{
					Invoice:  model.Invoice{Period: "2026-06"},
					RoomName: "101",
				}
				invoiceSvc.EXPECT().
					GetInvoice(gomock.Any(), "user-1", "inv-1").
					Return(inv, nil)
				imageSvc.EXPECT().
					GenerateInvoiceImage(gomock.Any(), inv).
					Return([]byte("image-data"), nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Unauthorized",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return r
			},
			mock:           func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Missing ID",
			id:   "",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock:           func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invoice not found",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				invoiceSvc.EXPECT().
					GetInvoice(gomock.Any(), "user-1", "inv-1").
					Return(nil, model.ErrInvoiceNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Generate image error",
			id:   "inv-1",
			setupAuth: func(r *http.Request) *http.Request {
				return setClaims(r, "user-1")
			},
			mock: func() {
				inv := &model.InvoiceWithRoom{}
				invoiceSvc.EXPECT().
					GetInvoice(gomock.Any(), "user-1", "inv-1").
					Return(inv, nil)
				imageSvc.EXPECT().
					GenerateInvoiceImage(gomock.Any(), inv).
					Return(nil, errors.New("generate error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			req := httptest.NewRequest(http.MethodGet, "/invoice/"+tt.id+"/download", nil)
			req = tt.setupAuth(req)
			if tt.id != "" {
				req = setURLParam(req, "id", tt.id)
			}
			rec := httptest.NewRecorder()

			h.DownloadInvoiceImage(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}
