package handler

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mihb123/quanly-phongtro/internal/logger"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service"
)

type TenantHandler struct {
	tenantService service.TenantService
}

func NewTenantHandler(tenantService service.TenantService) *TenantHandler {
	return &TenantHandler{tenantService: tenantService}
}

// POST /api/v1/tenant/
func (h *TenantHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10MB limit for files
	if err != nil {
		logger.Warn(r, http.StatusBadRequest, "invalid multipart form", err)
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, err := claims.GetSubject()
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	
	roomID := r.FormValue("room_id")
	fullName := r.FormValue("full_name")
	phone := r.FormValue("phone")
	email := r.FormValue("email")
	identityCard := r.FormValue("identity_card")
	startDateStr := r.FormValue("start_date")
	
	if roomID == "" || fullName == "" || phone == "" || identityCard == "" || startDateStr == "" {
		writeError(w, http.StatusBadRequest, "missing required fields")
		return
	}
	
	// Normally we'd do time parsing for `startDateStr` here
	// For simplicity in this step, assigning to zero value if parse fails.
	var startDate time.Time
	// Actually let's import time... wait, I need to add time to imports above if use it. I will import it.
	// Oh I missed the time import, let's fix it when it complains or fix it right away
	startDate, err = time.Parse("2006-01-02", startDateStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid start_date format, expected YYYY-MM-DD")
		return
	}

	tenant := &model.Tenant{
		RoomID:       roomID,
		CreatedBy:    userID,
		FullName:     fullName,
		Phone:        phone,
		Email:        email,
		IdentityCard: identityCard,
		StartDate:    startDate,
		Status:       model.TenantStatusActive,
	}

	// Handle file uploads
	cccdFile, cccdHeader, err := r.FormFile("cccd_file")
	if err == nil {
		defer cccdFile.Close()
		fileName := uuid.New().String() + filepath.Ext(cccdHeader.Filename)
		filePath := filepath.Join("uploads", "tenants", fileName)
		if err := saveFile(cccdFile, filePath); err == nil {
			tenant.CCCDPath = "/api/v1/tenant/files/" + fileName
		}
	}

	contractFile, contractHeader, err := r.FormFile("contract_file")
	if err == nil {
		defer contractFile.Close()
		fileName := uuid.New().String() + filepath.Ext(contractHeader.Filename)
		filePath := filepath.Join("uploads", "tenants", fileName)
		if err := saveFile(contractFile, filePath); err == nil {
			tenant.ContractPath = "/api/v1/tenant/files/" + fileName
		}
	}

	if err := h.tenantService.CreateTenant(r.Context(), tenant, claims.Role); err != nil {
		handleTenantError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, tenant, "")
}

func saveFile(file io.Reader, path string) error {
	dst, err := os.Create(path)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, file)
	return err
}

// GET /api/v1/tenant/room/{roomId}
func (h *TenantHandler) GetTenantByRoom(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, err := claims.GetSubject()
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	roomID := chi.URLParam(r, "roomId")
	if roomID == "" {
		writeError(w, http.StatusBadRequest, "roomId is required")
		return
	}

	tenants, err := h.tenantService.GetTenantByRoomID(r.Context(), roomID, claims.Role, userID)
	if err != nil {
		handleTenantError(w, r, err)
		return
	}
	
	writeJSON(w, http.StatusOK, tenants, "")
}

// PATCH /api/v1/tenant/{id}
func (h *TenantHandler) UpdateTenantInfo(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req model.UpdateTenantParams
	// Handle multipart form if needed, or simple JSON. Since frontend might use FormData for files in future, let's parse form dat.
	// But it's easier to just use ParseMultipartForm and grab what changed.
	_ = r.ParseMultipartForm(10 << 20)
	
	fullName := r.FormValue("full_name")
	phone := r.FormValue("phone")
	email := r.FormValue("email")
	identityCard := r.FormValue("identity_card")
	startDateStr := r.FormValue("start_date")
	
	if fullName != "" { req.FullName = &fullName }
	if phone != "" { req.Phone = &phone }
	if email != "" { req.Email = &email }
	if identityCard != "" { req.IdentityCard = &identityCard }
	if startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err == nil {
			req.StartDate = &startDate
		}
	}

	// Handle file updates
	cccdFile, cccdHeader, err := r.FormFile("cccd_file")
	if err == nil {
		defer cccdFile.Close()
		fileName := uuid.New().String() + filepath.Ext(cccdHeader.Filename)
		filePath := filepath.Join("uploads", "tenants", fileName)
		if err := saveFile(cccdFile, filePath); err == nil {
			path := "/api/v1/tenant/files/" + fileName
			req.CCCDPath = &path
		}
	}

	contractFile, contractHeader, err := r.FormFile("contract_file")
	if err == nil {
		defer contractFile.Close()
		fileName := uuid.New().String() + filepath.Ext(contractHeader.Filename)
		filePath := filepath.Join("uploads", "tenants", fileName)
		if err := saveFile(contractFile, filePath); err == nil {
			path := "/api/v1/tenant/files/" + fileName
			req.ContractPath = &path
		}
	}

	tenant, err := h.tenantService.UpdateTenantInfo(r.Context(), id, claims.Role, req)
	if err != nil {
		handleTenantError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, tenant, "")
}

// DELETE /api/v1/tenant/{id}
func (h *TenantHandler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	err := h.tenantService.DeleteTenant(r.Context(), id, claims.Role)
	if err != nil {
		handleTenantError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted successfully"}, "")
}

func handleTenantError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidRoomID):
		logger.Warn(r, http.StatusBadRequest, "invalid room ID", err)
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrRoomFull):
		logger.Warn(r, http.StatusBadRequest, "room full", err)
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrUnauthorized):
		logger.Warn(r, http.StatusUnauthorized, "unauthorized action", err)
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, model.ErrTenantNotFound):
		logger.Warn(r, http.StatusNotFound, "tenant not found", err)
		writeError(w, http.StatusNotFound, "tenant not found")
	default:
		logger.Error(r, http.StatusInternalServerError, "tenant operation failed", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
