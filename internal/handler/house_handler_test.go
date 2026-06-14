package handler_test

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
	"github.com/mihb123/quanly-phongtro/internal/handler"
	"github.com/mihb123/quanly-phongtro/internal/mock/mock_service"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service"
	"go.uber.org/mock/gomock"
)

func withValidClaims(req *http.Request) *http.Request {
	claims := &security.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: "user-1"},
	}
	ctx := security.WithClaims(req.Context(), claims)
	return req.WithContext(ctx)
}

func TestHouseHandler_CreateHouse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	houseSvc := mock_service.NewMockHouseService(ctrl)
	invoiceSvc := mock_service.NewMockInvoiceService(ctrl)
	h := handler.NewHouseHandler(houseSvc, invoiceSvc)

	tests := []struct {
		name           string
		setupAuth      func(*http.Request) *http.Request
		reqBody        interface{}
		rawBody        string
		mockBehavior   func(svc *mock_service.MockHouseService)
		expectedStatus int
	}{
		{
			name:      "Happy path",
			setupAuth: withValidClaims,
			reqBody: map[string]interface{}{
				"name":                     "House 1",
				"house_code":               "h1",
				"address":                  "123 Street",
				"electricity_billing_type": "USAGE",
				"water_billing_type":       "USAGE",
				"electricity_billing_unit": "ROOM",
				"water_billing_unit":       "PERSON",
			},
			mockBehavior: func(svc *mock_service.MockHouseService) {
				svc.EXPECT().CreateHouse(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Unauthorized",
			setupAuth:      func(req *http.Request) *http.Request { return req },
			reqBody:        map[string]interface{}{},
			mockBehavior:   func(svc *mock_service.MockHouseService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid body",
			setupAuth:      withValidClaims,
			rawBody:        "{invalid json",
			mockBehavior:   func(svc *mock_service.MockHouseService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "Validation error",
			setupAuth: withValidClaims,
			reqBody: map[string]interface{}{
				"name": "House 1", // missing address and other required fields
			},
			mockBehavior:   func(svc *mock_service.MockHouseService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "Service error",
			setupAuth: withValidClaims,
			reqBody: map[string]interface{}{
				"name":                     "House 1",
				"house_code":               "h1",
				"address":                  "123 Street",
				"electricity_billing_type": "USAGE",
				"water_billing_type":       "USAGE",
				"electricity_billing_unit": "ROOM",
				"water_billing_unit":       "PERSON",
			},
			mockBehavior: func(svc *mock_service.MockHouseService) {
				svc.EXPECT().CreateHouse(gomock.Any(), gomock.Any()).Return(errors.New("db error"))
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if tt.rawBody != "" {
				bodyBytes = []byte(tt.rawBody)
			} else if tt.reqBody != nil {
				bodyBytes, _ = json.Marshal(tt.reqBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/house", bytes.NewBuffer(bodyBytes))
			req = tt.setupAuth(req)

			rec := httptest.NewRecorder()
			tt.mockBehavior(houseSvc)

			h.CreateHouse(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHouseHandler_GetHouseByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	houseSvc := mock_service.NewMockHouseService(ctrl)
	h := handler.NewHouseHandler(houseSvc, nil)

	tests := []struct {
		name           string
		setupAuth      func(*http.Request) *http.Request
		houseID        string
		mockBehavior   func(svc *mock_service.MockHouseService)
		expectedStatus int
	}{
		{
			name:      "Happy path",
			setupAuth: withValidClaims,
			houseID:   "house-1",
			mockBehavior: func(svc *mock_service.MockHouseService) {
				svc.EXPECT().GetHouseByID(gomock.Any(), "house-1", "user-1").Return(&model.House{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Unauthorized",
			setupAuth:      func(req *http.Request) *http.Request { return req },
			houseID:        "house-1",
			mockBehavior:   func(svc *mock_service.MockHouseService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:      "Service error - Not found",
			setupAuth: withValidClaims,
			houseID:   "house-not-found",
			mockBehavior: func(svc *mock_service.MockHouseService) {
				svc.EXPECT().GetHouseByID(gomock.Any(), "house-not-found", "user-1").Return(nil, model.ErrHouseNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:      "Service error - Invalid ID",
			setupAuth: withValidClaims,
			houseID:   "invalid-id",
			mockBehavior: func(svc *mock_service.MockHouseService) {
				svc.EXPECT().GetHouseByID(gomock.Any(), "invalid-id", "user-1").Return(nil, service.ErrInvalidHouseID)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "Service error - Internal Server Error",
			setupAuth: withValidClaims,
			houseID:   "house-1",
			mockBehavior: func(svc *mock_service.MockHouseService) {
				svc.EXPECT().GetHouseByID(gomock.Any(), "house-1", "user-1").Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/house/"+tt.houseID, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.houseID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			req = tt.setupAuth(req)

			rec := httptest.NewRecorder()
			tt.mockBehavior(houseSvc)

			h.GetHouseByID(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHouseHandler_ListHouseByManagerID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	houseSvc := mock_service.NewMockHouseService(ctrl)
	h := handler.NewHouseHandler(houseSvc, nil)

	tests := []struct {
		name           string
		setupAuth      func(*http.Request) *http.Request
		queryParams    string
		mockBehavior   func(svc *mock_service.MockHouseService)
		expectedStatus int
	}{
		{
			name:        "Happy path",
			setupAuth:   withValidClaims,
			queryParams: "?page=2&limit=10&search=test",
			mockBehavior: func(svc *mock_service.MockHouseService) {
				svc.EXPECT().ListHouseByManagerID(gomock.Any(), "user-1", 2, 10, "test").Return([]model.House{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Happy path with default query params",
			setupAuth:   withValidClaims,
			queryParams: "",
			mockBehavior: func(svc *mock_service.MockHouseService) {
				svc.EXPECT().ListHouseByManagerID(gomock.Any(), "user-1", 1, 5, "").Return([]model.House{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Unauthorized",
			setupAuth:      func(req *http.Request) *http.Request { return req },
			queryParams:    "",
			mockBehavior:   func(svc *mock_service.MockHouseService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:        "Service error",
			setupAuth:   withValidClaims,
			queryParams: "",
			mockBehavior: func(svc *mock_service.MockHouseService) {
				svc.EXPECT().ListHouseByManagerID(gomock.Any(), "user-1", 1, 5, "").Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/house"+tt.queryParams, nil)
			req = tt.setupAuth(req)

			rec := httptest.NewRecorder()
			tt.mockBehavior(houseSvc)

			h.ListHouseByManagerID(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHouseHandler_UpdateHouse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	houseSvc := mock_service.NewMockHouseService(ctrl)
	invoiceSvc := mock_service.NewMockInvoiceService(ctrl)
	h := handler.NewHouseHandler(houseSvc, invoiceSvc)

	tests := []struct {
		name           string
		setupAuth      func(*http.Request) *http.Request
		houseID        string
		reqBody        interface{}
		rawBody        string
		mockBehavior   func(hSvc *mock_service.MockHouseService, iSvc *mock_service.MockInvoiceService)
		expectedStatus int
	}{
		{
			name:      "Happy path",
			setupAuth: withValidClaims,
			houseID:   "house-1",
			reqBody: map[string]interface{}{
				"name":       "Updated House",
				"house_code": "h1",
				"address":    "456 Street",
			},
			mockBehavior: func(hSvc *mock_service.MockHouseService, iSvc *mock_service.MockInvoiceService) {
				hSvc.EXPECT().UpdateHouse(gomock.Any(), "house-1", "user-1", gomock.Any()).Return(&model.House{}, nil)
				iSvc.EXPECT().RecalculateUnpaidInvoicesByHouse(gomock.Any(), "user-1", "house-1").Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "Happy path - Invoice recalculation fails (still 200)",
			setupAuth: withValidClaims,
			houseID:   "house-1",
			reqBody: map[string]interface{}{
				"name":       "Updated House",
				"house_code": "h1",
				"address":    "456 Street",
			},
			mockBehavior: func(hSvc *mock_service.MockHouseService, iSvc *mock_service.MockInvoiceService) {
				hSvc.EXPECT().UpdateHouse(gomock.Any(), "house-1", "user-1", gomock.Any()).Return(&model.House{}, nil)
				iSvc.EXPECT().RecalculateUnpaidInvoicesByHouse(gomock.Any(), "user-1", "house-1").Return(errors.New("invoice error"))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Unauthorized",
			setupAuth:      func(req *http.Request) *http.Request { return req },
			houseID:        "house-1",
			reqBody:        map[string]interface{}{},
			mockBehavior:   func(hSvc *mock_service.MockHouseService, iSvc *mock_service.MockInvoiceService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid body",
			setupAuth:      withValidClaims,
			houseID:        "house-1",
			rawBody:        "{invalid json",
			mockBehavior:   func(hSvc *mock_service.MockHouseService, iSvc *mock_service.MockInvoiceService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "Validation error",
			setupAuth: withValidClaims,
			houseID:   "house-1",
			reqBody: map[string]interface{}{
				"name": "Updated House", // missing address
			},
			mockBehavior:   func(hSvc *mock_service.MockHouseService, iSvc *mock_service.MockInvoiceService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "Service error",
			setupAuth: withValidClaims,
			houseID:   "house-1",
			reqBody: map[string]interface{}{
				"name":       "Updated House",
				"house_code": "h1",
				"address":    "456 Street",
			},
			mockBehavior: func(hSvc *mock_service.MockHouseService, iSvc *mock_service.MockInvoiceService) {
				hSvc.EXPECT().UpdateHouse(gomock.Any(), "house-1", "user-1", gomock.Any()).Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if tt.rawBody != "" {
				bodyBytes = []byte(tt.rawBody)
			} else if tt.reqBody != nil {
				bodyBytes, _ = json.Marshal(tt.reqBody)
			}

			req := httptest.NewRequest(http.MethodPut, "/house/"+tt.houseID, bytes.NewBuffer(bodyBytes))
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.houseID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			req = tt.setupAuth(req)

			rec := httptest.NewRecorder()
			tt.mockBehavior(houseSvc, invoiceSvc)

			h.UpdateHouse(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHouseHandler_DeleteHouse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	houseSvc := mock_service.NewMockHouseService(ctrl)
	h := handler.NewHouseHandler(houseSvc, nil)

	tests := []struct {
		name           string
		setupAuth      func(*http.Request) *http.Request
		houseID        string
		mockBehavior   func(svc *mock_service.MockHouseService)
		expectedStatus int
	}{
		{
			name:      "Happy path",
			setupAuth: withValidClaims,
			houseID:   "house-1",
			mockBehavior: func(svc *mock_service.MockHouseService) {
				svc.EXPECT().DeleteHouse(gomock.Any(), "house-1", "user-1").Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Unauthorized",
			setupAuth:      func(req *http.Request) *http.Request { return req },
			houseID:        "house-1",
			mockBehavior:   func(svc *mock_service.MockHouseService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:      "Service error - Not found",
			setupAuth: withValidClaims,
			houseID:   "house-not-found",
			mockBehavior: func(svc *mock_service.MockHouseService) {
				svc.EXPECT().DeleteHouse(gomock.Any(), "house-not-found", "user-1").Return(model.ErrHouseNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:      "Service error - Invalid ID",
			setupAuth: withValidClaims,
			houseID:   "invalid-id",
			mockBehavior: func(svc *mock_service.MockHouseService) {
				svc.EXPECT().DeleteHouse(gomock.Any(), "invalid-id", "user-1").Return(service.ErrInvalidHouseID)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "Service error - Internal Server Error",
			setupAuth: withValidClaims,
			houseID:   "house-1",
			mockBehavior: func(svc *mock_service.MockHouseService) {
				svc.EXPECT().DeleteHouse(gomock.Any(), "house-1", "user-1").Return(errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/house/"+tt.houseID, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.houseID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			req = tt.setupAuth(req)

			rec := httptest.NewRecorder()
			tt.mockBehavior(houseSvc)

			h.DeleteHouse(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}
