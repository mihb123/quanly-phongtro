package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/service"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
)

type HouseCostHandler struct {
	costService service.HouseCostService
}

func NewHouseCostHandler(costService service.HouseCostService) *HouseCostHandler {
	return &HouseCostHandler{costService: costService}
}

// handleHouseCostError maps domain errors to stable HTTP responses.
func handleHouseCostError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case model.IsHouseCostAlreadyExists(err):
		writeError(w, http.StatusConflict, err.Error())
	case model.IsHouseCostForbidden(err):
		writeError(w, http.StatusForbidden, err.Error())
	case model.IsHouseCostNotFound(err):
		writeError(w, http.StatusNotFound, "house cost not found")
	case model.IsHouseCostInvalidID(err):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		logger.Error(r, http.StatusInternalServerError, "house cost operation failed", err)
		writeError(w, http.StatusInternalServerError, "failed to process house cost")
	}
}

func (h *HouseCostHandler) CreateMonthlyCost(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		HouseID string `json:"house_id"`
		Period  string `json:"period"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.HouseID = strings.TrimSpace(req.HouseID)
	req.Period = strings.TrimSpace(req.Period)
	if req.HouseID == "" || req.Period == "" {
		writeError(w, http.StatusBadRequest, "house_id and period are required")
		return
	}
	if err := validateMonthPeriodValue(req.Period); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cost, err := h.costService.CreateMonthlyCost(r.Context(), managerID, req.HouseID, req.Period)
	if err != nil {
		handleHouseCostError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, cost, "success")
}

func (h *HouseCostHandler) GetMonthlyCost(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	houseID := r.URL.Query().Get("house_id")
	period := r.URL.Query().Get("period")

	if houseID == "" || period == "" {
		writeError(w, http.StatusBadRequest, "house_id and period are required")
		return
	}
	if err := validateMonthPeriodValue(period); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cost, err := h.costService.GetMonthlyCost(r.Context(), managerID, houseID, period)
	if err != nil {
		handleHouseCostError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, cost, "success")
}

func (h *HouseCostHandler) UpdateMonthlyCost(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req service.UpdateCostInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.ID = id

	if req.HouseID == "" || req.Period == "" {
		writeError(w, http.StatusBadRequest, "house_id and period are required")
		return
	}
	if err := validateMonthPeriodValue(req.Period); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.costService.UpdateMonthlyCost(r.Context(), managerID, req); err != nil {
		handleHouseCostError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, nil, "success")
}

func (h *HouseCostHandler) GetRevenueSummaries(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	houseIDsParam := r.URL.Query().Get("house_ids")
	period := r.URL.Query().Get("period")

	if houseIDsParam == "" || period == "" {
		writeError(w, http.StatusBadRequest, "house_ids and period are required")
		return
	}
	if err := validateMonthPeriodValue(period); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	houseIDs := strings.Split(houseIDsParam, ",")
	for i := range houseIDs {
		houseIDs[i] = strings.TrimSpace(houseIDs[i])
	}

	summaries, err := h.costService.GetRevenueSummaries(r.Context(), managerID, houseIDs, period)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get revenue summaries")
		return
	}

	writeJSON(w, http.StatusOK, summaries, "success")
}
