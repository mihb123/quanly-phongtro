package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/service"
)

type HouseCostHandler struct {
	costService service.HouseCostService
}

func NewHouseCostHandler(costService service.HouseCostService) *HouseCostHandler {
	return &HouseCostHandler{costService: costService}
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

	cost, err := h.costService.CreateMonthlyCost(r.Context(), managerID, req.HouseID, req.Period)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			writeError(w, http.StatusConflict, err.Error())
		} else if strings.Contains(err.Error(), "forbidden") {
			writeError(w, http.StatusForbidden, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "failed to create house cost: "+err.Error())
		}
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

	cost, err := h.costService.GetMonthlyCost(r.Context(), managerID, houseID, period)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "house cost not found")
		} else if strings.Contains(err.Error(), "forbidden") {
			writeError(w, http.StatusForbidden, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "failed to get house cost")
		}
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

	if err := h.costService.UpdateMonthlyCost(r.Context(), managerID, req); err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "invalid cost ID") {
			writeError(w, http.StatusNotFound, err.Error())
		} else if strings.Contains(err.Error(), "forbidden") {
			writeError(w, http.StatusForbidden, err.Error())
		} else {
			writeError(w, http.StatusInternalServerError, "failed to update house cost: "+err.Error())
		}
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
