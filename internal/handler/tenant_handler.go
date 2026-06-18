package handler

import (
	"errors"
	"net/http"
	"strings"
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

type registerTenantRequest struct {
	RoomID       string `validate:"required"`
	FullName     string `validate:"required"`
	Password     string `validate:"omitempty,min=6"`
	Phone        string `validate:"omitempty"`
	Email        string `validate:"omitempty,email"`
	IdentityCard string `validate:"omitempty"`
	StartDate    string `validate:"omitempty"`
}

type updateTenantRequest struct {
	FullName     *string `validate:"omitempty"`
	Phone        *string `validate:"omitempty"`
	Email        *string `validate:"omitempty,email"`
	IdentityCard *string `validate:"omitempty"`
}

type tenantResponse struct {
	TenantID     string `json:"tenant_id"`
	UserID       string `json:"user_id"`
	RoomID       string `json:"room_id"`
	RoomName     string `json:"room_name,omitempty"`
	FullName     string `json:"full_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
	CCCDPath     string `json:"cccd_path"`
	IdentityCard string `json:"identity_card"`
	ContractPath string `json:"contract_path"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date,omitempty"`
	Status       string `json:"status"`
	ZaloUserID   string `json:"zalo_user_id,omitempty"`
}

// newTenantResponse shapes tenant output without exposing manager ownership fields.
func newTenantResponse(tenant model.FullInfoTenant) tenantResponse {
	return tenantResponse{
		TenantID:     tenant.TenantID,
		UserID:       tenant.UserID,
		RoomID:       tenant.RoomID,
		RoomName:     tenant.RoomName,
		FullName:     tenant.FullName,
		Email:        tenant.Email,
		Phone:        tenant.Phone,
		CCCDPath:     tenant.CCCDPath,
		IdentityCard: tenant.IdentityCard,
		ContractPath: tenant.ContractPath,
		StartDate:    tenant.StartDate,
		EndDate:      tenant.EndDate,
		Status:       tenant.Status,
		ZaloUserID:   tenant.ZaloUserID,
	}
}

// newTenantResponses maps tenant service results to external tenant responses.
func newTenantResponses(tenants []model.FullInfoTenant) []tenantResponse {
	responses := make([]tenantResponse, 0, len(tenants))
	for _, tenant := range tenants {
		responses = append(responses, newTenantResponse(tenant))
	}
	return responses
}

func (h *TenantHandler) RegisterTenant(w http.ResponseWriter, r *http.Request) {

	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", nil)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, err := claims.GetSubject()
	if err != nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: invalid claims", err)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	// Prevent DoS: Limit total request body size to 15MB
	r.Body = http.MaxBytesReader(w, r.Body, 15<<20)
	err = r.ParseMultipartForm(10 << 20) // 10MB limit for files in RAM
	if err != nil {
		logger.Warn(r, http.StatusBadRequest, "invalid multipart form", err)
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	roomID := strings.TrimSpace(r.FormValue("room_id"))
	fullName := strings.TrimSpace(r.FormValue("full_name"))
	password := strings.TrimSpace(r.FormValue("password"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	identityCard := strings.TrimSpace(r.FormValue("identity_card"))
	startDateStr := strings.TrimSpace(r.FormValue("start_date"))

	if startDateStr == "" {
		startDateStr = time.Now().Format("2006-01-02")
	}

	req := registerTenantRequest{
		RoomID:       roomID,
		FullName:     fullName,
		Password:     password,
		Phone:        phone,
		Email:        email,
		IdentityCard: identityCard,
		StartDate:    startDateStr,
	}

	if err := validateStruct(req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "request validation failed", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var startDate time.Time
	startDate, err = time.Parse("2006-01-02", startDateStr)
	if err != nil {
		logger.Warn(r, http.StatusBadRequest, "invalid start date", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	cccdFiles := r.MultipartForm.File["cccd_file"]
	if len(cccdFiles) > 10 {
		writeError(w, http.StatusBadRequest, "maximum 10 CCCD files allowed")
		return
	}

	contractFiles := r.MultipartForm.File["contract_file"]
	if len(contractFiles) > 10 {
		writeError(w, http.StatusBadRequest, "maximum 10 contract files allowed")
		return
	}

	registerTenantInput := service.RegisterTenantInput{
		ManagerID:     userID,
		RoomID:        roomID,
		FullName:      fullName,
		Password:      password,
		Phone:         phone,
		Email:         email,
		IdentityCard:  identityCard,
		StartDate:     startDate,
		CCCDFiles:     cccdFiles,
		ContractFiles: contractFiles,
	}

	tenant, err := h.tenantService.RegisterTenant(r.Context(), registerTenantInput)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrMaxTenans):
			logger.Warn(r, http.StatusBadRequest, "room is full", err)
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrInvalidInput):
			logger.Warn(r, http.StatusBadRequest, "invalid input", err)
			writeError(w, http.StatusBadRequest, "invalid input")
		case errors.Is(err, model.ErrPhoneAlreadyExists):
			logger.Warn(r, http.StatusBadRequest, "phone already exists", err)
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			logger.Error(r, http.StatusInternalServerError, "failed to register tenant", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, newTenantResponse(*tenant), "")

}

func (h *TenantHandler) ListTenantByRoomID(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", nil)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	managerID, err := claims.GetSubject()
	if err != nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: invalid claims", err)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	roomID := chi.URLParam(r, "id")
	listTenant, err := h.tenantService.ListTenantByRoomID(r.Context(), managerID, roomID)
	if err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to get list of tenants by room ID", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, newTenantResponses(listTenant), "")
}

func (h *TenantHandler) ListTenantByHouseID(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", nil)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	managerID, err := claims.GetSubject()
	if err != nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: invalid claims", err)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	houseID := chi.URLParam(r, "id")
	listTenant, err := h.tenantService.ListTenantByHouseID(r.Context(), managerID, houseID)
	if err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to get list of tenants by house ID", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, newTenantResponses(listTenant), "")

}

func (h *TenantHandler) UpdateTenantInfo(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", nil)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	managerID, err := claims.GetSubject()
	if err != nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: invalid claims", err)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	tenantID := chi.URLParam(r, "id")

	// Prevent DoS: Limit total request body size to 15MB
	r.Body = http.MaxBytesReader(w, r.Body, 15<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		logger.Warn(r, http.StatusBadRequest, "invalid multipart form", err)
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	// Text fields: empty string means "not provided" → leave field unchanged.
	in := service.UpdateTenantInput{}
	var req updateTenantRequest

	if v := r.FormValue("full_name"); v != "" {
		trimmed := strings.TrimSpace(v)
		in.FullName = &trimmed
		req.FullName = &trimmed
	}
	if v := r.FormValue("phone"); v != "" {
		trimmed := strings.TrimSpace(v)
		in.Phone = &trimmed
		req.Phone = &trimmed
	}
	if v := r.FormValue("email"); v != "" {
		trimmed := strings.TrimSpace(strings.ToLower(v))
		in.Email = &trimmed
		req.Email = &trimmed
	}
	if v := r.FormValue("identity_card"); v != "" {
		trimmed := strings.TrimSpace(v)
		in.IdentityCard = &trimmed
		req.IdentityCard = &trimmed
	}

	if err := validateStruct(req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "update validation failed", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
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
	if len(in.CCCDFiles) > 10 {
		writeError(w, http.StatusBadRequest, "maximum 10 CCCD files allowed")
		return
	}

	in.ContractFiles = r.MultipartForm.File["contract_file"]
	if len(in.ContractFiles) > 10 {
		writeError(w, http.StatusBadRequest, "maximum 10 contract files allowed")
		return
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
		case errors.Is(err, model.ErrPhoneAlreadyExists):
			logger.Warn(r, http.StatusBadRequest, "phone already exists", err)
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			logger.Error(r, http.StatusInternalServerError, "failed to update tenant", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, newTenantResponse(*updated), "tenant updated successfully")
}

func (h *TenantHandler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", nil)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	managerID, err := claims.GetSubject()
	if err != nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: invalid claims", err)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	tenantID := chi.URLParam(r, "id")

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

// DownloadTenantFile serves a tenant upload after manager ownership is verified.
func (h *TenantHandler) DownloadTenantFile(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", nil)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	managerID, err := claims.GetSubject()
	if err != nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: invalid claims", err)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	fileName := chi.URLParam(r, "*")
	filePath, err := h.tenantService.ResolveTenantFilePath(r.Context(), managerID, fileName)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid file path")
		case errors.Is(err, model.ErrTenantNotFound), errors.Is(err, model.ErrUnauthorized):
			writeError(w, http.StatusNotFound, "file not found")
		default:
			logger.Error(r, http.StatusInternalServerError, "failed to resolve tenant file", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	http.ServeFile(w, r, filePath)
}
