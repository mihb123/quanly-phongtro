package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/logger"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service"
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

	if roomID == "" || fullName == "" || phone == "" || identityCard == "" || startDateStr == "" || email == "" {
		writeError(w, http.StatusBadRequest, "missing required fields")
		return
	}

	if len(password) < 6 {
		writeError(w, http.StatusBadRequest, "password is too short")
		return
	}

	// Normally we'd do time parsing for `startDateStr` here
	// For simplicity in this step, assigning to zero value if parse fails.
	var startDate time.Time
	// Actually let's import time... wait, I need to add time to imports above if use it. I will import it.
	// Oh I missed the time import, let's fix it when it complains or fix it right away
	startDate, err = time.Parse("Mon Jan 02 2006 15:04:05 MST-0700", startDateStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	cccdFile, cccdHeader, err := r.FormFile("cccd_file")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer cccdFile.Close()
	contractFile, contractHeader, err := r.FormFile("contract_file")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer contractFile.Close()
	registerTenantInput := service.RegisterTenantInput{
		ManagerID:          userID,
		RoomID:             roomID,
		FullName:           fullName,
		Password:           password,
		Phone:              phone,
		Email:              email,
		IdentityCard:       identityCard,
		StartDate:          startDate,
		CCCDFile:           cccdFile,
		CCCDFileHeader:     cccdHeader,
		ContractFile:       contractFile,
		ContractFileHeader: contractHeader,
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

	// File fields are optional: ignore the error when no file was uploaded.
	if cccdFile, cccdHeader, err := r.FormFile("cccd_file"); err == nil {
		defer cccdFile.Close()
		in.CCCDFile = cccdFile
		in.CCCDFileHeader = cccdHeader
	}
	if contractFile, contractHeader, err := r.FormFile("contract_file"); err == nil {
		defer contractFile.Close()
		in.ContractFile = contractFile
		in.ContractFileHeader = contractHeader
	}

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
