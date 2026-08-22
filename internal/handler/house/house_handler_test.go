package house_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	sharedsvc "github.com/mihb123/quanly-phongtro/internal/service/shared"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mihb123/quanly-phongtro/internal/handler/house"
	"github.com/mihb123/quanly-phongtro/internal/mock/mock_service"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	housesvc "github.com/mihb123/quanly-phongtro/internal/service/house"
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
	h := house.NewHouseHandler(houseSvc, invoiceSvc)

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
	h := house.NewHouseHandler(houseSvc, nil)

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
				svc.EXPECT().GetHouseByID(gomock.Any(), "invalid-id", "user-1").Return(nil, sharedsvc.ErrInvalidHouseID)
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
	h := house.NewHouseHandler(houseSvc, nil)

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
	h := house.NewHouseHandler(houseSvc, invoiceSvc)

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
			name:      "Invoice recalculation fails",
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
			expectedStatus: http.StatusInternalServerError,
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
	h := house.NewHouseHandler(houseSvc, nil)

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
				svc.EXPECT().DeleteHouse(gomock.Any(), "invalid-id", "user-1").Return(sharedsvc.ErrInvalidHouseID)
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

// manyFileNames sinh n tên file hợp lệ cho test vượt hạn mức upload.
func manyFileNames(n int) []string {
	names := make([]string, n)
	for i := range names {
		names[i] = "file-" + strconv.Itoa(i) + ".png"
	}
	return names
}

// TestHouseHandler_UpdateHouseDocuments kiểm tra cách handler đọc multipart: file mới, danh sách file cũ giữ lại và cờ xóa hết.
func TestHouseHandler_UpdateHouseDocuments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	houseSvc := mock_service.NewMockHouseService(ctrl)
	h := house.NewHouseHandler(houseSvc, mock_service.NewMockInvoiceService(ctrl))

	// buildForm dựng multipart body từ các field text và file (fieldName -> danh sách tên file).
	buildForm := func(fields map[string]string, files map[string][]string) (*bytes.Buffer, string) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		for k, v := range fields {
			_ = writer.WriteField(k, v)
		}
		for field, names := range files {
			for _, name := range names {
				part, _ := writer.CreateFormFile(field, name)
				_, _ = part.Write([]byte("data"))
			}
		}
		writer.Close()
		return body, writer.FormDataContentType()
	}

	tests := []struct {
		name           string
		setupAuth      func(*http.Request) *http.Request
		fields         map[string]string
		files          map[string][]string
		mockBehavior   func(svc *mock_service.MockHouseService)
		expectedStatus int
	}{
		{
			name:      "Uploads new files and keeps selected ones",
			setupAuth: withValidClaims,
			fields:    map[string]string{"kept_owner_cccd_paths": "old-cccd.png"},
			files:     map[string][]string{"owner_cccd_file": {"new.png"}, "owner_contract_file": {"hd.pdf", "phuluc.pdf"}},
			mockBehavior: func(svc *mock_service.MockHouseService) {
				svc.EXPECT().UpdateHouseDocuments(gomock.Any(), "house-1", "user-1", gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _ string, in housesvc.UpdateHouseDocumentsInput) (*model.House, error) {
						if in.KeptCCCDPaths == nil || *in.KeptCCCDPaths != "old-cccd.png" {
							t.Errorf("KeptCCCDPaths = %v, want old-cccd.png", in.KeptCCCDPaths)
						}
						if in.KeptContractPaths != nil {
							t.Errorf("KeptContractPaths = %v, want nil (giữ nguyên)", *in.KeptContractPaths)
						}
						if len(in.CCCDFiles) != 1 || len(in.ContractFiles) != 2 {
							t.Errorf("files = %d cccd / %d contract, want 1/2", len(in.CCCDFiles), len(in.ContractFiles))
						}
						return &model.House{ID: "house-1"}, nil
					})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "Empty flag clears a group",
			setupAuth: withValidClaims,
			fields:    map[string]string{"kept_owner_contract_paths_empty": "true"},
			mockBehavior: func(svc *mock_service.MockHouseService) {
				svc.EXPECT().UpdateHouseDocuments(gomock.Any(), "house-1", "user-1", gomock.Any()).
					DoAndReturn(func(_ context.Context, _, _ string, in housesvc.UpdateHouseDocumentsInput) (*model.House, error) {
						if in.KeptContractPaths == nil || *in.KeptContractPaths != "" {
							t.Errorf("KeptContractPaths = %v, want empty string", in.KeptContractPaths)
						}
						return &model.House{ID: "house-1"}, nil
					})
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Too many files",
			setupAuth:      withValidClaims,
			files:          map[string][]string{"owner_cccd_file": manyFileNames(11)},
			mockBehavior:   func(svc *mock_service.MockHouseService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Unauthorized",
			setupAuth:      func(req *http.Request) *http.Request { return req },
			mockBehavior:   func(svc *mock_service.MockHouseService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:      "House not found",
			setupAuth: withValidClaims,
			mockBehavior: func(svc *mock_service.MockHouseService) {
				svc.EXPECT().UpdateHouseDocuments(gomock.Any(), "house-1", "user-1", gomock.Any()).
					Return(nil, model.ErrHouseNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockBehavior(houseSvc)

			body, contentType := buildForm(tt.fields, tt.files)
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/house/house-1/documents", body)
			req.Header.Set("Content-Type", contentType)
			req = tt.setupAuth(req)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", "house-1")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rr := httptest.NewRecorder()
			h.UpdateHouseDocuments(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d (body: %s)", rr.Code, tt.expectedStatus, rr.Body.String())
			}
		})
	}
}
