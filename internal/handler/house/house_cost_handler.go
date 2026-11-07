package house

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/handler/httpx"
	"github.com/mihb123/quanly-phongtro/pkg/apperr"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/model"
	housesvc "github.com/mihb123/quanly-phongtro/internal/service/house"
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
		httpx.FailWith(w, r, http.StatusConflict, apperr.Message(err), err)
	case model.IsHouseCostForbidden(err):
		httpx.FailWith(w, r, http.StatusForbidden, apperr.Message(err), err)
	case model.IsHouseCostNotFound(err):
		httpx.FailWith(w, r, http.StatusNotFound, "house cost not found", err)
	case model.IsHouseCostInvalidID(err):
		httpx.FailWith(w, r, http.StatusNotFound, apperr.Message(err), err)
	default:
		httpx.Fail(w, r, err)
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
		httpx.FailWith(w, r, http.StatusBadRequest, apperr.Message(err), err)
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
		httpx.FailWith(w, r, http.StatusBadRequest, apperr.Message(err), err)
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
		httpx.FailWith(w, r, http.StatusBadRequest, apperr.Message(err), err)
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

	if periodsParam := r.URL.Query().Get("periods"); periodsParam != "" && houseIDsParam != "" {
		h.getRevenueTrend(w, r, managerID, houseIDsParam, periodsParam)
		return
	}

	if houseIDsParam == "" || period == "" {
		httpx.WriteError(w, http.StatusBadRequest, "house_ids and period are required")
		return
	}
	if err := httpx.ValidateMonthPeriodValue(period); err != nil {
		httpx.FailWith(w, r, http.StatusBadRequest, apperr.Message(err), err)
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

const maxRevenueTrendPeriods = 24

func (h *HouseCostHandler) getRevenueTrend(w http.ResponseWriter, r *http.Request, managerID, houseIDsParam, periodsParam string) {
	houseIDs := splitTrimmed(houseIDsParam)
	periods := splitTrimmed(periodsParam)
	if len(periods) > maxRevenueTrendPeriods {
		httpx.WriteError(w, http.StatusBadRequest, "too many periods")
		return
	}
	for _, p := range periods {
		if err := httpx.ValidateMonthPeriodValue(p); err != nil {
			httpx.FailWith(w, r, http.StatusBadRequest, apperr.Message(err), err)
			return
		}
	}

	summaries, err := h.costService.GetRevenueTrend(r.Context(), managerID, houseIDs, periods)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "failed to get revenue summaries")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, summaries, "success")
}

func splitTrimmed(raw string) []string {
	var out []string
	for _, v := range strings.Split(raw, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
