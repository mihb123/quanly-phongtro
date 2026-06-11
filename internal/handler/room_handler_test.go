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

func withContextClaims(req *http.Request, userID string) *http.Request {
	claims := &security.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: userID},
	}
	ctx := security.WithClaims(req.Context(), claims)
	return req.WithContext(ctx)
}

func withRouteContext(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestRoomHandler_CreateRoom(t *testing.T) {
	type testCase struct {
		name           string
		setupMocks     func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService)
		buildRequest   func() *http.Request
		expectedStatus int
	}

	tests := []testCase{
		{
			name: "Happy path",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
				roomSvc.EXPECT().CreateRoom(gomock.Any(), gomock.Any(), "user-1").Return(nil)
			},
			buildRequest: func() *http.Request {
				body := map[string]interface{}{
					"house_id":    "house-1",
					"name":        "Room 101",
					"price":       2000000,
					"max_tenants": 3,
					"status":      "AVAILABLE",
				}
				jsonBody, _ := json.Marshal(body)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/room", bytes.NewBuffer(jsonBody))
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Invalid body",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
			},
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/api/v1/room", bytes.NewBuffer([]byte(`{bad json`)))
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Validation error - Missing house_id",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
			},
			buildRequest: func() *http.Request {
				body := map[string]interface{}{
					"name":        "Room 101",
					"price":       2000000,
					"max_tenants": 3,
				}
				jsonBody, _ := json.Marshal(body)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/room", bytes.NewBuffer(jsonBody))
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Unauthorized",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
			},
			buildRequest: func() *http.Request {
				body := map[string]interface{}{
					"house_id":    "house-1",
					"name":        "Room 101",
					"price":       2000000,
					"max_tenants": 3,
				}
				jsonBody, _ := json.Marshal(body)
				return httptest.NewRequest(http.MethodPost, "/api/v1/room", bytes.NewBuffer(jsonBody))
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Service error - House not found",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
				roomSvc.EXPECT().CreateRoom(gomock.Any(), gomock.Any(), "user-1").Return(model.ErrHouseNotFound)
			},
			buildRequest: func() *http.Request {
				body := map[string]interface{}{
					"house_id":    "house-1",
					"name":        "Room 101",
					"price":       2000000,
					"max_tenants": 3,
					"status":      "AVAILABLE",
				}
				jsonBody, _ := json.Marshal(body)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/room", bytes.NewBuffer(jsonBody))
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Service error - Invalid house ID",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
				roomSvc.EXPECT().CreateRoom(gomock.Any(), gomock.Any(), "user-1").Return(service.ErrInvalidHouseID)
			},
			buildRequest: func() *http.Request {
				body := map[string]interface{}{
					"house_id":    "invalid-house",
					"name":        "Room 101",
					"price":       2000000,
					"max_tenants": 3,
					"status":      "AVAILABLE",
				}
				jsonBody, _ := json.Marshal(body)
				req := httptest.NewRequest(http.MethodPost, "/api/v1/room", bytes.NewBuffer(jsonBody))
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			roomSvc := mock_service.NewMockRoomService(ctrl)
			invoiceSvc := mock_service.NewMockInvoiceService(ctrl)
			tc.setupMocks(roomSvc, invoiceSvc)

			roomHandler := handler.NewRoomHandler(roomSvc, invoiceSvc)
			req := tc.buildRequest()
			rec := httptest.NewRecorder()

			roomHandler.CreateRoom(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("expected %d, got %d", tc.expectedStatus, rec.Code)
			}
		})
	}
}

func TestRoomHandler_ListRooms(t *testing.T) {
	type testCase struct {
		name           string
		setupMocks     func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService)
		buildRequest   func() *http.Request
		expectedStatus int
	}

	tests := []testCase{
		{
			name: "Happy path",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
				roomSvc.EXPECT().ListRoomsByHouseID(gomock.Any(), "house-1", "user-1", 1, 25).Return([]model.Room{}, nil)
			},
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/room?house_id=house-1", nil)
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Validation error - Missing house_id",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
			},
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/room", nil)
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Unauthorized",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
			},
			buildRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/v1/room?house_id=house-1", nil)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Service error - House not found",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
				roomSvc.EXPECT().ListRoomsByHouseID(gomock.Any(), "house-1", "user-1", 1, 25).Return(nil, model.ErrHouseNotFound)
			},
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/room?house_id=house-1", nil)
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			roomSvc := mock_service.NewMockRoomService(ctrl)
			invoiceSvc := mock_service.NewMockInvoiceService(ctrl)
			tc.setupMocks(roomSvc, invoiceSvc)

			roomHandler := handler.NewRoomHandler(roomSvc, invoiceSvc)
			req := tc.buildRequest()
			rec := httptest.NewRecorder()

			roomHandler.ListRooms(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("expected %d, got %d", tc.expectedStatus, rec.Code)
			}
		})
	}
}

func TestRoomHandler_GetRoom(t *testing.T) {
	type testCase struct {
		name           string
		setupMocks     func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService)
		buildRequest   func() *http.Request
		expectedStatus int
	}

	tests := []testCase{
		{
			name: "Happy path",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
				roomSvc.EXPECT().GetRoom(gomock.Any(), "room-1", "house-1", "user-1").Return(&model.Room{}, nil)
			},
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/room/room-1?house_id=house-1", nil)
				req = withRouteContext(req, "id", "room-1")
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Validation error - Missing house_id",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
			},
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/room/room-1", nil)
				req = withRouteContext(req, "id", "room-1")
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Unauthorized",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
			},
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/room/room-1?house_id=house-1", nil)
				return withRouteContext(req, "id", "room-1")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Service error - Room not found",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
				roomSvc.EXPECT().GetRoom(gomock.Any(), "room-1", "house-1", "user-1").Return(nil, model.ErrRoomNotFound)
			},
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/room/room-1?house_id=house-1", nil)
				req = withRouteContext(req, "id", "room-1")
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Service error - Generic error",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
				roomSvc.EXPECT().GetRoom(gomock.Any(), "room-1", "house-1", "user-1").Return(nil, errors.New("db error"))
			},
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/room/room-1?house_id=house-1", nil)
				req = withRouteContext(req, "id", "room-1")
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			roomSvc := mock_service.NewMockRoomService(ctrl)
			invoiceSvc := mock_service.NewMockInvoiceService(ctrl)
			tc.setupMocks(roomSvc, invoiceSvc)

			roomHandler := handler.NewRoomHandler(roomSvc, invoiceSvc)
			req := tc.buildRequest()
			rec := httptest.NewRecorder()

			roomHandler.GetRoom(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("expected %d, got %d", tc.expectedStatus, rec.Code)
			}
		})
	}
}

func TestRoomHandler_UpdateRoom(t *testing.T) {
	type testCase struct {
		name           string
		setupMocks     func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService)
		buildRequest   func() *http.Request
		expectedStatus int
	}

	tests := []testCase{
		{
			name: "Happy path",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
				roomSvc.EXPECT().UpdateRoom(gomock.Any(), "room-1", "house-1", "user-1", gomock.Any()).Return(&model.Room{}, nil)
				invoiceSvc.EXPECT().RecalculateUnpaidInvoicesByRoom(gomock.Any(), "user-1", "room-1").Return(nil)
			},
			buildRequest: func() *http.Request {
				body := map[string]interface{}{
					"house_id":    "house-1",
					"name":        "Room 101",
					"price":       2500000,
					"max_tenants": 4,
					"status":      "AVAILABLE",
				}
				jsonBody, _ := json.Marshal(body)
				req := httptest.NewRequest(http.MethodPatch, "/api/v1/room/room-1", bytes.NewBuffer(jsonBody))
				req = withRouteContext(req, "id", "room-1")
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Happy path with recalculate invoice error",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
				roomSvc.EXPECT().UpdateRoom(gomock.Any(), "room-1", "house-1", "user-1", gomock.Any()).Return(&model.Room{}, nil)
				invoiceSvc.EXPECT().RecalculateUnpaidInvoicesByRoom(gomock.Any(), "user-1", "room-1").Return(errors.New("db error"))
			},
			buildRequest: func() *http.Request {
				body := map[string]interface{}{
					"house_id":    "house-1",
					"name":        "Room 101",
					"price":       2500000,
					"max_tenants": 4,
					"status":      "AVAILABLE",
				}
				jsonBody, _ := json.Marshal(body)
				req := httptest.NewRequest(http.MethodPatch, "/api/v1/room/room-1", bytes.NewBuffer(jsonBody))
				req = withRouteContext(req, "id", "room-1")
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusOK, // still 200 OK
		},
		{
			name: "Invalid body",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
			},
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodPatch, "/api/v1/room/room-1", bytes.NewBuffer([]byte(`{bad json`)))
				req = withRouteContext(req, "id", "room-1")
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Validation error - Missing name",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
			},
			buildRequest: func() *http.Request {
				body := map[string]interface{}{
					"house_id":    "house-1",
					"price":       2500000,
					"max_tenants": 4,
					"status":      "AVAILABLE",
				}
				jsonBody, _ := json.Marshal(body)
				req := httptest.NewRequest(http.MethodPatch, "/api/v1/room/room-1", bytes.NewBuffer(jsonBody))
				req = withRouteContext(req, "id", "room-1")
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Unauthorized",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
			},
			buildRequest: func() *http.Request {
				body := map[string]interface{}{
					"house_id":    "house-1",
					"name":        "Room 101",
					"price":       2500000,
					"max_tenants": 4,
					"status":      "AVAILABLE",
				}
				jsonBody, _ := json.Marshal(body)
				req := httptest.NewRequest(http.MethodPatch, "/api/v1/room/room-1", bytes.NewBuffer(jsonBody))
				return withRouteContext(req, "id", "room-1")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Service error - Room not found",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
				roomSvc.EXPECT().UpdateRoom(gomock.Any(), "room-1", "house-1", "user-1", gomock.Any()).Return(nil, model.ErrRoomNotFound)
			},
			buildRequest: func() *http.Request {
				body := map[string]interface{}{
					"house_id":    "house-1",
					"name":        "Room 101",
					"price":       2500000,
					"max_tenants": 4,
					"status":      "AVAILABLE",
				}
				jsonBody, _ := json.Marshal(body)
				req := httptest.NewRequest(http.MethodPatch, "/api/v1/room/room-1", bytes.NewBuffer(jsonBody))
				req = withRouteContext(req, "id", "room-1")
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			roomSvc := mock_service.NewMockRoomService(ctrl)
			invoiceSvc := mock_service.NewMockInvoiceService(ctrl)
			tc.setupMocks(roomSvc, invoiceSvc)

			roomHandler := handler.NewRoomHandler(roomSvc, invoiceSvc)
			req := tc.buildRequest()
			rec := httptest.NewRecorder()

			roomHandler.UpdateRoom(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("expected %d, got %d", tc.expectedStatus, rec.Code)
			}
		})
	}
}

func TestRoomHandler_DeleteRoom(t *testing.T) {
	type testCase struct {
		name           string
		setupMocks     func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService)
		buildRequest   func() *http.Request
		expectedStatus int
	}

	tests := []testCase{
		{
			name: "Happy path",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
				roomSvc.EXPECT().DeleteRoom(gomock.Any(), "room-1", "house-1", "user-1").Return(nil)
			},
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodDelete, "/api/v1/room/room-1?house_id=house-1", nil)
				req = withRouteContext(req, "id", "room-1")
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Validation error - Missing house_id",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
			},
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodDelete, "/api/v1/room/room-1", nil)
				req = withRouteContext(req, "id", "room-1")
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Unauthorized",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
			},
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodDelete, "/api/v1/room/room-1?house_id=house-1", nil)
				return withRouteContext(req, "id", "room-1")
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Service error - Room not found",
			setupMocks: func(roomSvc *mock_service.MockRoomService, invoiceSvc *mock_service.MockInvoiceService) {
				roomSvc.EXPECT().DeleteRoom(gomock.Any(), "room-1", "house-1", "user-1").Return(model.ErrRoomNotFound)
			},
			buildRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodDelete, "/api/v1/room/room-1?house_id=house-1", nil)
				req = withRouteContext(req, "id", "room-1")
				return withContextClaims(req, "user-1")
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			roomSvc := mock_service.NewMockRoomService(ctrl)
			invoiceSvc := mock_service.NewMockInvoiceService(ctrl)
			tc.setupMocks(roomSvc, invoiceSvc)

			roomHandler := handler.NewRoomHandler(roomSvc, invoiceSvc)
			req := tc.buildRequest()
			rec := httptest.NewRecorder()

			roomHandler.DeleteRoom(rec, req)

			if rec.Code != tc.expectedStatus {
				t.Errorf("expected %d, got %d", tc.expectedStatus, rec.Code)
			}
		})
	}
}
