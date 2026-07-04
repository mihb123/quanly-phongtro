package house

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/handler/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/model"
	housesvc "github.com/mihb123/quanly-phongtro/internal/service/house"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
)

type HouseCostHandler struct {
	costService housesvc.HouseCostService
}

func NewHouseCostHandler(costService housesvc.HouseCostService) *HouseCostHandler {
	return &HouseCostHandler{costService: costService}
}

// handleHouseCostError maps domain errors to stable HTTP responses.
func handleHouseCostError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case model.IsHouseCostAlreadyExists(err):
		httpx.WriteError(w, http.StatusConflict, err.Error())
	case model.IsHouseCostForbidden(err):
		httpx.WriteError(w, http.StatusForbidden, err.Error())
	case model.IsHouseCostNotFound(err):
		httpx.WriteError(w, http.StatusNotFound, "house cost not found")
	case model.IsHouseCostInvalidID(err):
		httpx.WriteError(w, http.StatusNotFound, err.Error())
	default:
		logger.Error(r, http.StatusInternalServerError, "house cost operation failed", err)
		httpx.WriteError(w, http.StatusInternalServerError, "failed to process house cost")
	}
}

func (h *HouseCostHandler) CreateMonthlyCost(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		HouseID string `json:"house_id"`
		Period  string `json:"period"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.HouseID = strings.TrimSpace(req.HouseID)
	req.Period = strings.TrimSpace(req.Period)
	if req.HouseID == "" || req.Period == "" {
		httpx.WriteError(w, http.StatusBadRequest, "house_id and period are required")
		return
	}
	if err := httpx.ValidateMonthPeriodValue(req.Period); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	cost, err := h.costService.CreateMonthlyCost(r.Context(), managerID, req.HouseID, req.Period)
	if err != nil {
		handleHouseCostError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, cost, "success")
}

func (h *HouseCostHandler) GetMonthlyCost(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	houseID := r.URL.Query().Get("house_id")
	period := r.URL.Query().Get("period")

	if houseID == "" || period == "" {
		httpx.WriteError(w, http.StatusBadRequest, "house_id and period are required")
		return
	}
	if err := httpx.ValidateMonthPeriodValue(period); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	cost, err := h.costService.GetMonthlyCost(r.Context(), managerID, houseID, period)
	if err != nil {
		handleHouseCostError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, cost, "success")
}

func (h *HouseCostHandler) UpdateMonthlyCost(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req housesvc.UpdateCostInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.ID = id

	if req.HouseID == "" || req.Period == "" {
		httpx.WriteError(w, http.StatusBadRequest, "house_id and period are required")
		return
	}
	if err := httpx.ValidateMonthPeriodValue(req.Period); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.costService.UpdateMonthlyCost(r.Context(), managerID, req); err != nil {
		handleHouseCostError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, nil, "success")
}

func (h *HouseCostHandler) GetRevenueSummaries(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	houseIDsParam := r.URL.Query().Get("house_ids")
	period := r.URL.Query().Get("period")

	if houseIDsParam == "" || period == "" {
		httpx.WriteError(w, http.StatusBadRequest, "house_ids and period are required")
		return
	}
	if err := httpx.ValidateMonthPeriodValue(period); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	houseIDs := strings.Split(houseIDsParam, ",")
	for i := range houseIDs {
		houseIDs[i] = strings.TrimSpace(houseIDs[i])
	}

	summaries, err := h.costService.GetRevenueSummaries(r.Context(), managerID, houseIDs, period)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to get revenue summaries")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, summaries, "success")
}
