package house_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/handler/house"
	"github.com/mihb123/quanly-phongtro/internal/mock/mock_service"
	"github.com/mihb123/quanly-phongtro/internal/model"
	housesvc "github.com/mihb123/quanly-phongtro/internal/service/house"
	"go.uber.org/mock/gomock"
)

func TestHouseCostHandler_CreateMonthlyCost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mock_service.NewMockHouseCostService(ctrl)
	h := house.NewHouseCostHandler(mockSvc)

	tests := []struct {
		name           string
		setupAuth      func(*http.Request) *http.Request
		reqBody        interface{}
		mockBehavior   func(svc *mock_service.MockHouseCostService)
		expectedStatus int
	}{
		{
			name:      "Happy path",
			setupAuth: withValidClaims,
			reqBody: map[string]interface{}{
				"house_id": "house-1",
				"period":   "2023-10",
			},
			mockBehavior: func(svc *mock_service.MockHouseCostService) {
				svc.EXPECT().
					CreateMonthlyCost(gomock.Any(), "user-1", "house-1", "2023-10").
					Return(&model.HouseCost{ID: "cost-1"}, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:      "Validation error - Missing HouseID",
			setupAuth: withValidClaims,
			reqBody: map[string]interface{}{
				"period": "2023-10",
			},
			mockBehavior:   func(svc *mock_service.MockHouseCostService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Unauthorized",
			setupAuth:      func(req *http.Request) *http.Request { return req },
			reqBody:        map[string]interface{}{"house_id": "house-1", "period": "2023-10"},
			mockBehavior:   func(svc *mock_service.MockHouseCostService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid body",
			setupAuth:      withValidClaims,
			reqBody:        "invalid-json",
			mockBehavior:   func(svc *mock_service.MockHouseCostService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:      "Service error - Forbidden",
			setupAuth: withValidClaims,
			reqBody: map[string]interface{}{
				"house_id": "house-1",
				"period":   "2023-10",
			},
			mockBehavior: func(svc *mock_service.MockHouseCostService) {
				svc.EXPECT().
					CreateMonthlyCost(gomock.Any(), "user-1", "house-1", "2023-10").
					Return(nil, errors.New("forbidden: you do not own this house"))
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:      "Service error - Already Exists",
			setupAuth: withValidClaims,
			reqBody: map[string]interface{}{
				"house_id": "house-1",
				"period":   "2023-10",
			},
			mockBehavior: func(svc *mock_service.MockHouseCostService) {
				svc.EXPECT().
					CreateMonthlyCost(gomock.Any(), "user-1", "house-1", "2023-10").
					Return(nil, errors.New("house cost record already exists for this period"))
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:      "Service error - Internal",
			setupAuth: withValidClaims,
			reqBody: map[string]interface{}{
				"house_id": "house-1",
				"period":   "2023-10",
			},
			mockBehavior: func(svc *mock_service.MockHouseCostService) {
				svc.EXPECT().
					CreateMonthlyCost(gomock.Any(), "user-1", "house-1", "2023-10").
					Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.reqBody)
			if raw, ok := tt.reqBody.(string); ok {
				bodyBytes = []byte(raw)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/house-cost", bytes.NewBuffer(bodyBytes))
			req = tt.setupAuth(req)

			rec := httptest.NewRecorder()
			tt.mockBehavior(mockSvc)

			h.CreateMonthlyCost(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

// TestHouseCostHandler_GetMonthlyCost covers query validation and service error mapping.
func TestHouseCostHandler_GetMonthlyCost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mock_service.NewMockHouseCostService(ctrl)
	h := house.NewHouseCostHandler(mockSvc)

	tests := []struct {
		name           string
		setupAuth      func(*http.Request) *http.Request
		queryParams    string
		mockBehavior   func(svc *mock_service.MockHouseCostService)
		expectedStatus int
	}{
		{
			name:        "Happy path",
			setupAuth:   withValidClaims,
			queryParams: "?house_id=house-1&period=2023-10",
			mockBehavior: func(svc *mock_service.MockHouseCostService) {
				svc.EXPECT().
					GetMonthlyCost(gomock.Any(), "user-1", "house-1", "2023-10").
					Return(&model.HouseCost{ID: "cost-1"}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Unauthorized",
			setupAuth:      func(req *http.Request) *http.Request { return req },
			queryParams:    "?house_id=house-1&period=2023-10",
			mockBehavior:   func(svc *mock_service.MockHouseCostService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing parameters",
			setupAuth:      withValidClaims,
			queryParams:    "?house_id=house-1",
			mockBehavior:   func(svc *mock_service.MockHouseCostService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "Not found",
			setupAuth:   withValidClaims,
			queryParams: "?house_id=house-1&period=2023-10",
			mockBehavior: func(svc *mock_service.MockHouseCostService) {
				svc.EXPECT().
					GetMonthlyCost(gomock.Any(), "user-1", "house-1", "2023-10").
					Return(nil, errors.New("house cost not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:        "Forbidden",
			setupAuth:   withValidClaims,
			queryParams: "?house_id=house-1&period=2023-10",
			mockBehavior: func(svc *mock_service.MockHouseCostService) {
				svc.EXPECT().
					GetMonthlyCost(gomock.Any(), "user-1", "house-1", "2023-10").
					Return(nil, errors.New("forbidden: you do not own this house"))
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:        "Service error",
			setupAuth:   withValidClaims,
			queryParams: "?house_id=house-1&period=2023-10",
			mockBehavior: func(svc *mock_service.MockHouseCostService) {
				svc.EXPECT().
					GetMonthlyCost(gomock.Any(), "user-1", "house-1", "2023-10").
					Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/house-cost"+tt.queryParams, nil)
			req = tt.setupAuth(req)

			rec := httptest.NewRecorder()
			tt.mockBehavior(mockSvc)

			h.GetMonthlyCost(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHouseCostHandler_UpdateMonthlyCost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mock_service.NewMockHouseCostService(ctrl)
	h := house.NewHouseCostHandler(mockSvc)

	tests := []struct {
		name           string
		setupAuth      func(*http.Request) *http.Request
		costID         string
		reqBody        interface{}
		mockBehavior   func(svc *mock_service.MockHouseCostService)
		expectedStatus int
	}{
		{
			name:      "Happy path",
			setupAuth: withValidClaims,
			costID:    "cost-1",
			reqBody: map[string]interface{}{
				"id":       "cost-1",
				"house_id": "house-1",
				"period":   "2023-10",
				"rent":     3000,
			},
			mockBehavior: func(svc *mock_service.MockHouseCostService) {
				svc.EXPECT().
					UpdateMonthlyCost(gomock.Any(), "user-1", gomock.Any()).
					DoAndReturn(func(ctx context.Context, managerID string, input housesvc.UpdateCostInput) error {
						if input.Rent != 3000 {
							t.Errorf("unexpected rent %f", input.Rent)
						}
						return nil
					})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "Service error",
			setupAuth: withValidClaims,
			costID:    "cost-1",
			reqBody: map[string]interface{}{
				"id":       "cost-1",
				"house_id": "house-1",
				"period":   "2023-10",
			},
			mockBehavior: func(svc *mock_service.MockHouseCostService) {
				svc.EXPECT().
					UpdateMonthlyCost(gomock.Any(), "user-1", gomock.Any()).
					Return(errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:      "Service error - Not found",
			setupAuth: withValidClaims,
			costID:    "cost-1",
			reqBody: map[string]interface{}{
				"house_id": "house-1",
				"period":   "2023-10",
			},
			mockBehavior: func(svc *mock_service.MockHouseCostService) {
				svc.EXPECT().
					UpdateMonthlyCost(gomock.Any(), "user-1", gomock.Any()).
					Return(errors.New("house cost not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:      "Service error - Forbidden",
			setupAuth: withValidClaims,
			costID:    "cost-1",
			reqBody: map[string]interface{}{
				"house_id": "house-1",
				"period":   "2023-10",
			},
			mockBehavior: func(svc *mock_service.MockHouseCostService) {
				svc.EXPECT().
					UpdateMonthlyCost(gomock.Any(), "user-1", gomock.Any()).
					Return(errors.New("forbidden: you do not own this house"))
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Unauthorized",
			setupAuth:      func(req *http.Request) *http.Request { return req },
			costID:         "cost-1",
			reqBody:        map[string]interface{}{"house_id": "house-1", "period": "2023-10"},
			mockBehavior:   func(svc *mock_service.MockHouseCostService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing cost id",
			setupAuth:      withValidClaims,
			costID:         "",
			reqBody:        map[string]interface{}{"house_id": "house-1", "period": "2023-10"},
			mockBehavior:   func(svc *mock_service.MockHouseCostService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid body",
			setupAuth:      withValidClaims,
			costID:         "cost-1",
			reqBody:        "invalid-json",
			mockBehavior:   func(svc *mock_service.MockHouseCostService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing required fields",
			setupAuth:      withValidClaims,
			costID:         "cost-1",
			reqBody:        map[string]interface{}{"house_id": "house-1"},
			mockBehavior:   func(svc *mock_service.MockHouseCostService) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.reqBody)
			if raw, ok := tt.reqBody.(string); ok {
				bodyBytes = []byte(raw)
			}
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/house-cost/"+tt.costID, bytes.NewBuffer(bodyBytes))

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.costID)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			req = tt.setupAuth(req)

			rec := httptest.NewRecorder()
			tt.mockBehavior(mockSvc)

			h.UpdateMonthlyCost(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestHouseCostHandler_GetRevenueSummaries(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mock_service.NewMockHouseCostService(ctrl)
	h := house.NewHouseCostHandler(mockSvc)

	tests := []struct {
		name           string
		setupAuth      func(*http.Request) *http.Request
		queryParams    string
		mockBehavior   func(svc *mock_service.MockHouseCostService)
		expectedStatus int
	}{
		{
			name:        "Happy path",
			setupAuth:   withValidClaims,
			queryParams: "?house_ids=h1,h2&period=2023-10",
			mockBehavior: func(svc *mock_service.MockHouseCostService) {
				svc.EXPECT().
					GetRevenueSummaries(gomock.Any(), "user-1", []string{"h1", "h2"}, "2023-10").
					Return([]model.HouseRevenueSummary{{Profit: 1000}}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Missing parameters",
			setupAuth:   withValidClaims,
			queryParams: "",
			mockBehavior: func(svc *mock_service.MockHouseCostService) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Unauthorized",
			setupAuth:      func(req *http.Request) *http.Request { return req },
			queryParams:    "?house_ids=h1&period=2023-10",
			mockBehavior:   func(svc *mock_service.MockHouseCostService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:        "Service error",
			setupAuth:   withValidClaims,
			queryParams: "?house_ids=h1&period=2023-10",
			mockBehavior: func(svc *mock_service.MockHouseCostService) {
				svc.EXPECT().
					GetRevenueSummaries(gomock.Any(), "user-1", []string{"h1"}, "2023-10").
					Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/revenue-summary"+tt.queryParams, nil)
			req = tt.setupAuth(req)

			rec := httptest.NewRecorder()
			tt.mockBehavior(mockSvc)

			h.GetRevenueSummaries(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}
