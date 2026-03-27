package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
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
	DefaultParkingPrice     float64 `json:"default_parking_price" validate:"gte=0"`
	DefaultServicePrice     float64 `json:"default_service_price" validate:"gte=0"`
}

func NewHouseHandler(service service.HouseService) *HouseHandler {
	return &HouseHandler{service: service}
}

func (h *HouseHandler) CreateHouse(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	userID, err := claims.GetSubject()
	if !ok || err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createHouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateStruct(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	output, err := h.service.CreateHouse(r.Context(), service.CreateHouseInput{
		Name:                    req.Name,
		ManagerID:               userID,
		Address:                 req.Address,
		DefaultElectricityPrice: req.DefaultElectricityPrice,
		DefaultWaterPrice:       req.DefaultWaterPrice,
		DefaultWifiPrice:        req.DefaultWifiPrice,
		DefaultParkingPrice:     req.DefaultParkingPrice,
		DefaultServicePrice:     req.DefaultServicePrice,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, output, "")
}

func (h *HouseHandler) GetHouseByID(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	userID, err := claims.GetSubject()
	if !ok || err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	houseID := chi.URLParam(r, "id")
	output, err := h.service.GetHouseByID(r.Context(), houseID, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidHouseID), errors.Is(err, service.ErrInvalidManagerID):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, model.ErrHouseNotFound):
			writeError(w, http.StatusNotFound, "house not found")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, output, "")
}
