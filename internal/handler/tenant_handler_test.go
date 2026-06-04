package handler_test

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
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

func buildMultipartRequest(method, url string, fields map[string]string, files map[string]int) (*http.Request, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			return nil, err
		}
	}
	for fieldName, count := range files {
		for i := 0; i < count; i++ {
			part, err := writer.CreateFormFile(fieldName, "test.png")
			if err != nil {
				return nil, err
			}
			part.Write([]byte("file content"))
		}
	}
	writer.Close()
	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

func withClaims(req *http.Request, userID string) *http.Request {
	claims := &security.Claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: userID},
	}
	ctx := security.WithClaims(req.Context(), claims)
	return req.WithContext(ctx)
}

func withChiURLParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestTenantHandler_RegisterTenant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type mockBehavior func(s *mock_service.MockTenantService)

	tests := []struct {
		name           string
		reqBuilder     func() *http.Request
		mockBehavior   mockBehavior
		expectedStatus int
	}{
		{
			name: "Happy path",
			reqBuilder: func() *http.Request {
				req, _ := buildMultipartRequest(http.MethodPost, "/tenant", map[string]string{
					"room_id":    "room-1",
					"full_name":  "Nguyen Van A",
					"start_date": "2026-06-01",
				}, nil)
				return withClaims(req, "user-1")
			},
			mockBehavior: func(s *mock_service.MockTenantService) {
				s.EXPECT().RegisterTenant(gomock.Any(), gomock.Any()).Return(&model.FullInfoTenant{TenantID: "tenant-1"}, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Unauthorized - missing claims",
			reqBuilder: func() *http.Request {
				req, _ := buildMultipartRequest(http.MethodPost, "/tenant", map[string]string{
					"room_id":   "room-1",
					"full_name": "Nguyen Van A",
				}, nil)
				return req
			},
			mockBehavior:   func(s *mock_service.MockTenantService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid body - not multipart",
			reqBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/tenant", bytes.NewBuffer([]byte("plain text")))
				req.Header.Set("Content-Type", "text/plain")
				return withClaims(req, "user-1")
			},
			mockBehavior:   func(s *mock_service.MockTenantService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Validation error - missing room_id",
			reqBuilder: func() *http.Request {
				req, _ := buildMultipartRequest(http.MethodPost, "/tenant", map[string]string{
					"full_name": "Nguyen Van A",
				}, nil)
				return withClaims(req, "user-1")
			},
			mockBehavior:   func(s *mock_service.MockTenantService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Validation error - invalid start_date format",
			reqBuilder: func() *http.Request {
				req, _ := buildMultipartRequest(http.MethodPost, "/tenant", map[string]string{
					"room_id":    "room-1",
					"full_name":  "Nguyen Van A",
					"start_date": "invalid-date",
				}, nil)
				return withClaims(req, "user-1")
			},
			mockBehavior:   func(s *mock_service.MockTenantService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Validation error - too many cccd files",
			reqBuilder: func() *http.Request {
				req, _ := buildMultipartRequest(http.MethodPost, "/tenant", map[string]string{
					"room_id":    "room-1",
					"full_name":  "Nguyen Van A",
				}, map[string]int{"cccd_file": 11})
				return withClaims(req, "user-1")
			},
			mockBehavior:   func(s *mock_service.MockTenantService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Validation error - too many contract files",
			reqBuilder: func() *http.Request {
				req, _ := buildMultipartRequest(http.MethodPost, "/tenant", map[string]string{
					"room_id":    "room-1",
					"full_name":  "Nguyen Van A",
				}, map[string]int{"contract_file": 11})
				return withClaims(req, "user-1")
			},
			mockBehavior:   func(s *mock_service.MockTenantService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Service error - room full",
			reqBuilder: func() *http.Request {
				req, _ := buildMultipartRequest(http.MethodPost, "/tenant", map[string]string{
					"room_id":   "room-1",
					"full_name": "Nguyen Van A",
				}, nil)
				return withClaims(req, "user-1")
			},
			mockBehavior: func(s *mock_service.MockTenantService) {
				s.EXPECT().RegisterTenant(gomock.Any(), gomock.Any()).Return(nil, model.ErrMaxTenans)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Service error - invalid input",
			reqBuilder: func() *http.Request {
				req, _ := buildMultipartRequest(http.MethodPost, "/tenant", map[string]string{
					"room_id":   "room-1",
					"full_name": "Nguyen Van A",
				}, nil)
				return withClaims(req, "user-1")
			},
			mockBehavior: func(s *mock_service.MockTenantService) {
				s.EXPECT().RegisterTenant(gomock.Any(), gomock.Any()).Return(nil, service.ErrInvalidInput)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Service error - internal error",
			reqBuilder: func() *http.Request {
				req, _ := buildMultipartRequest(http.MethodPost, "/tenant", map[string]string{
					"room_id":   "room-1",
					"full_name": "Nguyen Van A",
				}, nil)
				return withClaims(req, "user-1")
			},
			mockBehavior: func(s *mock_service.MockTenantService) {
				s.EXPECT().RegisterTenant(gomock.Any(), gomock.Any()).Return(nil, errors.New("unexpected error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := mock_service.NewMockTenantService(ctrl)
			tt.mockBehavior(s)

			h := handler.NewTenantHandler(s)

			req := tt.reqBuilder()
			rec := httptest.NewRecorder()
			h.RegisterTenant(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestTenantHandler_ListTenantByRoomID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type mockBehavior func(s *mock_service.MockTenantService)

	tests := []struct {
		name           string
		reqBuilder     func() *http.Request
		mockBehavior   mockBehavior
		expectedStatus int
	}{
		{
			name: "Happy path",
			reqBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/tenant/room/room-1", nil)
				req = withChiURLParam(req, "id", "room-1")
				return withClaims(req, "user-1")
			},
			mockBehavior: func(s *mock_service.MockTenantService) {
				s.EXPECT().ListTenantByRoomID(gomock.Any(), "user-1", "room-1").Return([]model.FullInfoTenant{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Unauthorized - missing claims",
			reqBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/tenant/room/room-1", nil)
				return withChiURLParam(req, "id", "room-1")
			},
			mockBehavior:   func(s *mock_service.MockTenantService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Service error - internal error",
			reqBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/tenant/room/room-1", nil)
				req = withChiURLParam(req, "id", "room-1")
				return withClaims(req, "user-1")
			},
			mockBehavior: func(s *mock_service.MockTenantService) {
				s.EXPECT().ListTenantByRoomID(gomock.Any(), "user-1", "room-1").Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := mock_service.NewMockTenantService(ctrl)
			tt.mockBehavior(s)

			h := handler.NewTenantHandler(s)

			req := tt.reqBuilder()
			rec := httptest.NewRecorder()
			h.ListTenantByRoomID(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestTenantHandler_ListTenantByHouseID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type mockBehavior func(s *mock_service.MockTenantService)

	tests := []struct {
		name           string
		reqBuilder     func() *http.Request
		mockBehavior   mockBehavior
		expectedStatus int
	}{
		{
			name: "Happy path",
			reqBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/tenant/house/house-1", nil)
				req = withChiURLParam(req, "id", "house-1")
				return withClaims(req, "user-1")
			},
			mockBehavior: func(s *mock_service.MockTenantService) {
				s.EXPECT().ListTenantByHouseID(gomock.Any(), "user-1", "house-1").Return([]model.FullInfoTenant{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Unauthorized - missing claims",
			reqBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/tenant/house/house-1", nil)
				return withChiURLParam(req, "id", "house-1")
			},
			mockBehavior:   func(s *mock_service.MockTenantService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Service error - internal error",
			reqBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/tenant/house/house-1", nil)
				req = withChiURLParam(req, "id", "house-1")
				return withClaims(req, "user-1")
			},
			mockBehavior: func(s *mock_service.MockTenantService) {
				s.EXPECT().ListTenantByHouseID(gomock.Any(), "user-1", "house-1").Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := mock_service.NewMockTenantService(ctrl)
			tt.mockBehavior(s)

			h := handler.NewTenantHandler(s)

			req := tt.reqBuilder()
			rec := httptest.NewRecorder()
			h.ListTenantByHouseID(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestTenantHandler_UpdateTenantInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type mockBehavior func(s *mock_service.MockTenantService)

	tests := []struct {
		name           string
		reqBuilder     func() *http.Request
		mockBehavior   mockBehavior
		expectedStatus int
	}{
		{
			name: "Happy path",
			reqBuilder: func() *http.Request {
				req, _ := buildMultipartRequest(http.MethodPut, "/tenant/tenant-1", map[string]string{
					"full_name": "Nguyen Van B",
				}, nil)
				req = withChiURLParam(req, "id", "tenant-1")
				return withClaims(req, "user-1")
			},
			mockBehavior: func(s *mock_service.MockTenantService) {
				s.EXPECT().UpdateTenantInfo(gomock.Any(), "user-1", "tenant-1", gomock.Any()).Return(&model.FullInfoTenant{}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Happy path - with kept paths",
			reqBuilder: func() *http.Request {
				req, _ := buildMultipartRequest(http.MethodPut, "/tenant/tenant-1", map[string]string{
					"kept_cccd_paths":       "path1,path2",
					"kept_contract_paths_empty": "true",
				}, nil)
				req = withChiURLParam(req, "id", "tenant-1")
				return withClaims(req, "user-1")
			},
			mockBehavior: func(s *mock_service.MockTenantService) {
				s.EXPECT().UpdateTenantInfo(gomock.Any(), "user-1", "tenant-1", gomock.Any()).DoAndReturn(
					func(ctx context.Context, managerID, tenantID string, in service.UpdateTenantInput) (*model.FullInfoTenant, error) {
						if in.KeptCCCDPaths == nil || *in.KeptCCCDPaths != "path1,path2" {
							t.Errorf("expected kept_cccd_paths to be 'path1,path2', got %v", in.KeptCCCDPaths)
						}
						if in.KeptContractPaths == nil || *in.KeptContractPaths != "" {
							t.Errorf("expected kept_contract_paths to be '', got %v", in.KeptContractPaths)
						}
						return &model.FullInfoTenant{}, nil
					})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Unauthorized - missing claims",
			reqBuilder: func() *http.Request {
				req, _ := buildMultipartRequest(http.MethodPut, "/tenant/tenant-1", map[string]string{}, nil)
				req = withChiURLParam(req, "id", "tenant-1")
				return req
			},
			mockBehavior:   func(s *mock_service.MockTenantService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid body - not multipart",
			reqBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodPut, "/tenant/tenant-1", bytes.NewBuffer([]byte("plain text")))
				req.Header.Set("Content-Type", "text/plain")
				req = withChiURLParam(req, "id", "tenant-1")
				return withClaims(req, "user-1")
			},
			mockBehavior:   func(s *mock_service.MockTenantService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Validation error - invalid email",
			reqBuilder: func() *http.Request {
				req, _ := buildMultipartRequest(http.MethodPut, "/tenant/tenant-1", map[string]string{
					"email": "invalid-email",
				}, nil)
				req = withChiURLParam(req, "id", "tenant-1")
				return withClaims(req, "user-1")
			},
			mockBehavior:   func(s *mock_service.MockTenantService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Validation error - too many cccd files",
			reqBuilder: func() *http.Request {
				req, _ := buildMultipartRequest(http.MethodPut, "/tenant/tenant-1", map[string]string{}, map[string]int{"cccd_file": 11})
				req = withChiURLParam(req, "id", "tenant-1")
				return withClaims(req, "user-1")
			},
			mockBehavior:   func(s *mock_service.MockTenantService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Validation error - too many contract files",
			reqBuilder: func() *http.Request {
				req, _ := buildMultipartRequest(http.MethodPut, "/tenant/tenant-1", map[string]string{}, map[string]int{"contract_file": 11})
				req = withChiURLParam(req, "id", "tenant-1")
				return withClaims(req, "user-1")
			},
			mockBehavior:   func(s *mock_service.MockTenantService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Service error - tenant not found",
			reqBuilder: func() *http.Request {
				req, _ := buildMultipartRequest(http.MethodPut, "/tenant/tenant-1", map[string]string{
					"full_name": "Nguyen Van B",
				}, nil)
				req = withChiURLParam(req, "id", "tenant-1")
				return withClaims(req, "user-1")
			},
			mockBehavior: func(s *mock_service.MockTenantService) {
				s.EXPECT().UpdateTenantInfo(gomock.Any(), "user-1", "tenant-1", gomock.Any()).Return(nil, model.ErrTenantNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Service error - unauthorized manager",
			reqBuilder: func() *http.Request {
				req, _ := buildMultipartRequest(http.MethodPut, "/tenant/tenant-1", map[string]string{}, nil)
				req = withChiURLParam(req, "id", "tenant-1")
				return withClaims(req, "user-1")
			},
			mockBehavior: func(s *mock_service.MockTenantService) {
				s.EXPECT().UpdateTenantInfo(gomock.Any(), "user-1", "tenant-1", gomock.Any()).Return(nil, model.ErrUnauthorized)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "Service error - internal error",
			reqBuilder: func() *http.Request {
				req, _ := buildMultipartRequest(http.MethodPut, "/tenant/tenant-1", map[string]string{}, nil)
				req = withChiURLParam(req, "id", "tenant-1")
				return withClaims(req, "user-1")
			},
			mockBehavior: func(s *mock_service.MockTenantService) {
				s.EXPECT().UpdateTenantInfo(gomock.Any(), "user-1", "tenant-1", gomock.Any()).Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := mock_service.NewMockTenantService(ctrl)
			tt.mockBehavior(s)

			h := handler.NewTenantHandler(s)

			req := tt.reqBuilder()
			rec := httptest.NewRecorder()
			h.UpdateTenantInfo(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestTenantHandler_DeleteTenant(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type mockBehavior func(s *mock_service.MockTenantService)

	tests := []struct {
		name           string
		reqBuilder     func() *http.Request
		mockBehavior   mockBehavior
		expectedStatus int
	}{
		{
			name: "Happy path",
			reqBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodDelete, "/tenant/tenant-1", nil)
				req = withChiURLParam(req, "id", "tenant-1")
				return withClaims(req, "user-1")
			},
			mockBehavior: func(s *mock_service.MockTenantService) {
				s.EXPECT().DeleteTenant(gomock.Any(), "user-1", "tenant-1").Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Unauthorized - missing claims",
			reqBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodDelete, "/tenant/tenant-1", nil)
				return withChiURLParam(req, "id", "tenant-1")
			},
			mockBehavior:   func(s *mock_service.MockTenantService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Service error - tenant not found",
			reqBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodDelete, "/tenant/tenant-1", nil)
				req = withChiURLParam(req, "id", "tenant-1")
				return withClaims(req, "user-1")
			},
			mockBehavior: func(s *mock_service.MockTenantService) {
				s.EXPECT().DeleteTenant(gomock.Any(), "user-1", "tenant-1").Return(model.ErrTenantNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Service error - unauthorized manager",
			reqBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodDelete, "/tenant/tenant-1", nil)
				req = withChiURLParam(req, "id", "tenant-1")
				return withClaims(req, "user-1")
			},
			mockBehavior: func(s *mock_service.MockTenantService) {
				s.EXPECT().DeleteTenant(gomock.Any(), "user-1", "tenant-1").Return(model.ErrUnauthorized)
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "Service error - internal error",
			reqBuilder: func() *http.Request {
				req := httptest.NewRequest(http.MethodDelete, "/tenant/tenant-1", nil)
				req = withChiURLParam(req, "id", "tenant-1")
				return withClaims(req, "user-1")
			},
			mockBehavior: func(s *mock_service.MockTenantService) {
				s.EXPECT().DeleteTenant(gomock.Any(), "user-1", "tenant-1").Return(errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := mock_service.NewMockTenantService(ctrl)
			tt.mockBehavior(s)

			h := handler.NewTenantHandler(s)

			req := tt.reqBuilder()
			rec := httptest.NewRecorder()
			h.DeleteTenant(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}
