package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service"
)

type HouseHandler struct {
	service service.HouseService
}

type createHouseRequest struct {
	Name                    string  `json:"name" validate:"required"`
	Address                 string  `json:"address" validate:"required"`
	DefaultElectricityPrice float64 `json:"default_electricity_price" validate:"gte=0"`
	DefaultWaterPrice       float64 `json:"default_water_price" validate:"gte=0"`
	DefaultWifiPrice        float64 `json:"default_wifi_price" validate:"gte=0"`
	DefaultParikingPrice    float64 `json:"default_parking_price" validate:"gte=0"`
	DefaultServicePrice     float64 `json:"default_service_price" validate:"gte=0"`
}

func NewHouseHandler(service service.HouseService) *HouseHandler {
	return &HouseHandler{service: service}
}

func (h *HouseHandler) CreateHouse(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims.UserID == "" {
		writeError(r, w, http.StatusUnauthorized, "unauthorized", errors.New("missing auth claims"))
		return
	}

	var req createHouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if err := validateStruct(req); err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error(), err)
		return
	}

	output, err := h.service.CreateHouse(r.Context(), service.CreateHouseInput{
		Name:                    req.Name,
		ManagerID:               claims.UserID,
		Address:                 req.Address,
		DefaultElectricityPrice: req.DefaultElectricityPrice,
		DefaultWaterPrice:       req.DefaultWaterPrice,
		DefaultWifiPrice:        req.DefaultWifiPrice,
		DefaultParikingPrice:    req.DefaultParikingPrice,
		DefaultServicePrice:     req.DefaultServicePrice,
	})
	if err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error(), err)
		return
	}
	writeJSON(w, http.StatusCreated, output)
}

func (h *HouseHandler) GetHouseByID(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims.UserID == "" {
		writeError(r, w, http.StatusUnauthorized, "unauthorized", errors.New("missing auth claims"))
		return
	}

	houseID := mux.Vars(r)["id"]
	output, err := h.service.GetHouseByID(r.Context(), houseID, claims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidHouseID), errors.Is(err, service.ErrInvalidManagerID):
			writeError(r, w, http.StatusBadRequest, err.Error(), err)
		case errors.Is(err, model.ErrHouseNotFound):
			writeError(r, w, http.StatusNotFound, "house not found", err)
		default:
			writeError(r, w, http.StatusInternalServerError, "internal server error", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, output)
}
