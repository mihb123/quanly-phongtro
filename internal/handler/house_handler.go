package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
)

type HouseHandler struct {
	houseService   service.HouseService
	invoiceService service.InvoiceService
}

type createHouseRequest struct {
	Name                    string  `json:"name" validate:"required"`
	HouseCode               string  `json:"house_code" validate:"required,max=12,housecode"`
	Address                 string  `json:"address" validate:"required"`
	DefaultElectricityPrice float64 `json:"default_electricity_price" validate:"gte=0"`
	DefaultWaterPrice       float64 `json:"default_water_price" validate:"gte=0"`
	DefaultWifiPrice        float64 `json:"default_wifi_price" validate:"gte=0"`
	DefaultParkingPrice     float64 `json:"default_parking_price" validate:"gte=0"`
	DefaultServicePrice     float64 `json:"default_service_price" validate:"gte=0"`
	ElectricityBillingType  string  `json:"electricity_billing_type" validate:"required,oneof=USAGE FIXED"`
	WaterBillingType        string  `json:"water_billing_type" validate:"required,oneof=USAGE FIXED"`
	ElectricityBillingUnit  string  `json:"electricity_billing_unit" validate:"required,oneof=ROOM PERSON"`
	WaterBillingUnit        string  `json:"water_billing_unit" validate:"required,oneof=ROOM PERSON"`
	ExtraPersonThreshold    int     `json:"extra_person_threshold" validate:"gte=0"`
	ExtraPersonFee          float64 `json:"extra_person_fee" validate:"gte=0"`
	ExtraVehicleThreshold   int     `json:"extra_vehicle_threshold" validate:"gte=0"`
	ExtraVehicleFee         float64 `json:"extra_vehicle_fee" validate:"gte=0"`
}

type updateHouseRequest struct {
	Name                    string  `json:"name" validate:"required"`
	HouseCode               string  `json:"house_code" validate:"required,max=12,housecode"`
	Address                 string  `json:"address" validate:"required"`
	DefaultElectricityPrice float64 `json:"default_electricity_price" validate:"gte=0"`
	DefaultWaterPrice       float64 `json:"default_water_price" validate:"gte=0"`
	DefaultWifiPrice        float64 `json:"default_wifi_price" validate:"gte=0"`
	DefaultParkingPrice     float64 `json:"default_parking_price" validate:"gte=0"`
	DefaultServicePrice     float64 `json:"default_service_price" validate:"gte=0"`
	ElectricityBillingType  string  `json:"electricity_billing_type" validate:"omitempty,oneof=USAGE FIXED"`
	WaterBillingType        string  `json:"water_billing_type" validate:"omitempty,oneof=USAGE FIXED"`
	ElectricityBillingUnit  string  `json:"electricity_billing_unit" validate:"omitempty,oneof=ROOM PERSON"`
	WaterBillingUnit        string  `json:"water_billing_unit" validate:"omitempty,oneof=ROOM PERSON"`
	ExtraPersonThreshold    int     `json:"extra_person_threshold" validate:"gte=0"`
	ExtraPersonFee          float64 `json:"extra_person_fee" validate:"gte=0"`
	ExtraVehicleThreshold   int     `json:"extra_vehicle_threshold" validate:"gte=0"`
	ExtraVehicleFee         float64 `json:"extra_vehicle_fee" validate:"gte=0"`
}

func NewHouseHandler(service service.HouseService, invoiceService service.InvoiceService) *HouseHandler {
	return &HouseHandler{houseService: service, invoiceService: invoiceService}
}

func (h *HouseHandler) CreateHouse(w http.ResponseWriter, r *http.Request) {
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

	var req createHouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "invalid request body", err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateStruct(req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "request validation failed", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	house := &model.House{
		Name:                    req.Name,
		HouseCode:               req.HouseCode,
		ManagerID:               userID,
		Address:                 req.Address,
		DefaultElectricityPrice: req.DefaultElectricityPrice,
		DefaultWaterPrice:       req.DefaultWaterPrice,
		DefaultWifiPrice:        req.DefaultWifiPrice,
		DefaultParkingPrice:     req.DefaultParkingPrice,
		DefaultServicePrice:     req.DefaultServicePrice,
		ElectricityBillingType:  req.ElectricityBillingType,
		WaterBillingType:        req.WaterBillingType,
		ElectricityBillingUnit:  req.ElectricityBillingUnit,
		WaterBillingUnit:        req.WaterBillingUnit,
		ExtraPersonThreshold:    req.ExtraPersonThreshold,
		ExtraPersonFee:          req.ExtraPersonFee,
		ExtraVehicleThreshold:   req.ExtraVehicleThreshold,
		ExtraVehicleFee:         req.ExtraVehicleFee,
	}

	err = h.houseService.CreateHouse(r.Context(), house)
	if err != nil {
		logger.Error(r, http.StatusBadRequest, "failed to create house", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, house, "")
}

func (h *HouseHandler) GetHouseByID(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", nil)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, err := claims.GetSubject()
	if err != nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", err)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	houseID := chi.URLParam(r, "id")
	houses, err := h.houseService.GetHouseByID(r.Context(), houseID, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidHouseID), errors.Is(err, service.ErrInvalidManagerID):
			logger.Warn(r, http.StatusBadRequest, "invalid house or manager ID", err)
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, model.ErrHouseNotFound):
			logger.Warn(r, http.StatusNotFound, "house not found", err)
			writeError(w, http.StatusNotFound, "house not found")
		default:
			logger.Error(r, http.StatusInternalServerError, "unexpected error getting house by ID", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, houses, "")
}

func (h *HouseHandler) ListHouseByManagerID(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", nil)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, err := claims.GetSubject()
	if err != nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", err)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")
	search := r.URL.Query().Get("search")
	pageInt, err := strconv.Atoi(page)
	if err != nil {
		pageInt = 1
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		limitInt = 5
	}

	houses, err := h.houseService.ListHouseByManagerID(r.Context(), userID, pageInt, limitInt, search)
	if err != nil {
		logger.Error(r, http.StatusInternalServerError, "cannot list house", err)
		writeError(w, http.StatusInternalServerError, "cannot list houseByID")
	}

	writeJSON(w, http.StatusOK, houses, "")
}

func (h *HouseHandler) UpdateHouse(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", nil)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, err := claims.GetSubject()
	if err != nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", err)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	var req updateHouseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "invalid request body", err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateStruct(req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "request validation failed", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	updateHouseInput := service.UpdateHouseInput{
		Name:                    req.Name,
		HouseCode:               req.HouseCode,
		Address:                 req.Address,
		DefaultElectricityPrice: req.DefaultElectricityPrice,
		DefaultWaterPrice:       req.DefaultWaterPrice,
		DefaultWifiPrice:        req.DefaultWifiPrice,
		DefaultParkingPrice:     req.DefaultParkingPrice,
		DefaultServicePrice:     req.DefaultServicePrice,
		ElectricityBillingType:  req.ElectricityBillingType,
		WaterBillingType:        req.WaterBillingType,
		ElectricityBillingUnit:  req.ElectricityBillingUnit,
		WaterBillingUnit:        req.WaterBillingUnit,
		ExtraPersonThreshold:    req.ExtraPersonThreshold,
		ExtraPersonFee:          req.ExtraPersonFee,
		ExtraVehicleThreshold:   req.ExtraVehicleThreshold,
		ExtraVehicleFee:         req.ExtraVehicleFee,
	}

	house, err := h.houseService.UpdateHouse(r.Context(), id, userID, updateHouseInput)
	if err != nil {
		logger.Warn(r, http.StatusBadRequest, "cannot update house", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Trigger recalculation for unpaid invoices when house is updated
	err = h.invoiceService.RecalculateUnpaidInvoicesByHouse(r.Context(), userID, id)
	if err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to recalculate invoices after house update", err)
		writeError(w, http.StatusInternalServerError, "failed to recalculate invoices after house update")
		return
	}

	writeJSON(w, http.StatusOK, house, "")
}

func (h *HouseHandler) DeleteHouse(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", nil)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	userID, err := claims.GetSubject()
	if err != nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", err)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	err = h.houseService.DeleteHouse(r.Context(), id, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidHouseID), errors.Is(err, service.ErrInvalidManagerID):
			logger.Warn(r, http.StatusBadRequest, "invalid house or manager ID", err)
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, model.ErrHouseNotFound):
			logger.Warn(r, http.StatusNotFound, "house not found", err)
			writeError(w, http.StatusNotFound, "house not found")
		default:
			logger.Error(r, http.StatusInternalServerError, "failed to delete house", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, nil, "house deleted successfully")
}
