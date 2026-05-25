package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
)

type TenantHandler struct {
	tenantService service.TenantService
}

func NewTenantHandler(service service.TenantService) *TenantHandler {
	return &TenantHandler{
		tenantService: service,
	}
}

func handleTenantError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, model.ErrMaxTenans):
		logger.Warn(r, http.StatusBadRequest, "room is max", err)
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrInvalidInput):
		logger.Warn(r, http.StatusBadRequest, "invalid input", err)
		writeError(w, http.StatusBadRequest, "invalid input")
	default:
		logger.Error(r, http.StatusInternalServerError, "tenant operation failed", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *TenantHandler) RegisterTenant(w http.ResponseWriter, r *http.Request) {

	claims, ok := security.ClaimsFromContext(r.Context())
	userID, err := claims.GetSubject()
	if !ok || err != nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", err)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	err = r.ParseMultipartForm(10 << 20) // 10MB limit for files
	if err != nil {
		logger.Warn(r, http.StatusBadRequest, "invalid multipart form", err)
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	roomID := r.FormValue("room_id")
	fullName := r.FormValue("full_name")
	password := r.FormValue("password")
	phone := r.FormValue("phone")
	email := r.FormValue("email")
	identityCard := r.FormValue("identity_card")
	startDateStr := r.FormValue("start_date")

	if email == "" && phone != "" {
		email = phone + "@tenant.local"
	}

	if password == "" && phone != "" {
		password = phone
	}

	if roomID == "" || fullName == "" || phone == "" || identityCard == "" || startDateStr == "" || email == "" {
		writeError(w, http.StatusBadRequest, "missing required fields")
		return
	}

	if len(password) < 6 {
		writeError(w, http.StatusBadRequest, "password is too short")
		return
	}

	var startDate time.Time
	startDate, err = time.Parse("2006-01-02", startDateStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	cccdFiles := r.MultipartForm.File["cccd_file"]
	contractFiles := r.MultipartForm.File["contract_file"]

	registerTenantInput := service.RegisterTenantInput{
		ManagerID:          userID,
		RoomID:             roomID,
		FullName:           fullName,
		Password:           password,
		Phone:              phone,
		Email:              email,
		IdentityCard:       identityCard,
		StartDate:          startDate,
		CCCDFiles:          cccdFiles,
		ContractFiles:      contractFiles,
	}

	user, err := h.tenantService.RegisterTenant(r.Context(), registerTenantInput)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, user, "")

}

func (h *TenantHandler) ListTenantByRoomID(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	managerID, err := claims.GetSubject()
	if !ok || err != nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", err)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	roomID := chi.URLParam(r, "id")
	if roomID == "" {
		logger.Warn(r, http.StatusBadRequest, "missing roomID", err)
		writeError(w, http.StatusBadRequest, "missing room ID")
		return
	}
	listTenant, err := h.tenantService.ListTenantByRoomID(r.Context(), managerID, roomID)
	if err != nil {
		logger.Warn(r, http.StatusInternalServerError, "can not get list of tenant by room id", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, listTenant, "")

}

func (h *TenantHandler) UpdateTenantInfo(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	managerID, err := claims.GetSubject()
	if !ok || err != nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", err)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	tenantID := chi.URLParam(r, "id")
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		logger.Warn(r, http.StatusBadRequest, "invalid multipart form", err)
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	// Text fields: empty string means "not provided" → leave field unchanged.
	in := service.UpdateTenantInput{}

	if v := r.FormValue("full_name"); v != "" {
		in.FullName = &v
	}
	if v := r.FormValue("phone"); v != "" {
		in.Phone = &v
	}
	if v := r.FormValue("email"); v != "" {
		in.Email = &v
	}
	if v := r.FormValue("identity_card"); v != "" {
		in.IdentityCard = &v
	}

	if v := r.FormValue("kept_cccd_paths"); v != "" {
		in.KeptCCCDPaths = &v
	} else if r.FormValue("kept_cccd_paths_empty") == "true" {
        empty := ""
		in.KeptCCCDPaths = &empty
	}

	if v := r.FormValue("kept_contract_paths"); v != "" {
		in.KeptContractPaths = &v
	} else if r.FormValue("kept_contract_paths_empty") == "true" {
        empty := ""
		in.KeptContractPaths = &empty
	}

	in.CCCDFiles = r.MultipartForm.File["cccd_file"]
	in.ContractFiles = r.MultipartForm.File["contract_file"]

	updated, err := h.tenantService.UpdateTenantInfo(r.Context(), managerID, tenantID, in)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrTenantNotFound):
			logger.Warn(r, http.StatusNotFound, "tenant not found", err)
			writeError(w, http.StatusNotFound, "tenant not found")
		case errors.Is(err, model.ErrUnauthorized):
			logger.Warn(r, http.StatusForbidden, "manager does not own tenant", err)
			writeError(w, http.StatusForbidden, "forbidden: you do not manage this tenant")
		default:
			logger.Error(r, http.StatusInternalServerError, "failed to update tenant", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, updated, "tenant updated successfully")
}

func (h *TenantHandler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	managerID, err := claims.GetSubject()
	if !ok || err != nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", err)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	tenantID := chi.URLParam(r, "id")
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "missing tenant ID")
		return
	}

	if err := h.tenantService.DeleteTenant(r.Context(), managerID, tenantID); err != nil {
		switch {
		case errors.Is(err, model.ErrTenantNotFound):
			logger.Warn(r, http.StatusNotFound, "tenant not found", err)
			writeError(w, http.StatusNotFound, "tenant not found")
		case errors.Is(err, model.ErrUnauthorized):
			logger.Warn(r, http.StatusForbidden, "manager does not own tenant", err)
			writeError(w, http.StatusForbidden, "forbidden: you do not manage this tenant")
		default:
			logger.Error(r, http.StatusInternalServerError, "failed to delete tenant", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, nil, "tenant deleted successfully")
}
