package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/service"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
)

type InvoiceHandler struct {
	invoiceService service.InvoiceService
}

func NewInvoiceHandler(invoiceService service.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{invoiceService: invoiceService}
}

type createInvoiceRequest struct {
	RoomID              string  `json:"room_id" validate:"required"`
	Period              string  `json:"period" validate:"required"`
	NewElectricityIndex int     `json:"new_electricity_index" validate:"gte=0"`
	NewWaterIndex       int     `json:"new_water_index" validate:"gte=0"`
	OtherFee            float64 `json:"other_fee" validate:"gte=0"`
	Discount            float64 `json:"discount" validate:"gte=0"`
	VehicleCount        int     `json:"vehicle_count" validate:"gte=0"`
}

func handleInvoiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, model.ErrInvoiceNotFound):
		logger.Warn(r, http.StatusNotFound, "invoice not found", err)
		writeError(w, http.StatusNotFound, "invoice not found")
	case errors.Is(err, model.ErrRoomNotFound):
		logger.Warn(r, http.StatusNotFound, "room not found", err)
		writeError(w, http.StatusNotFound, "room not found")
	case errors.Is(err, model.ErrDuplicateInvoice):
		logger.Warn(r, http.StatusConflict, "duplicate invoice", err)
		writeError(w, http.StatusConflict, "duplicate invoice for this room and period")
	case errors.Is(err, model.ErrInvalidElectricityIndex), errors.Is(err, model.ErrInvalidWaterIndex):
		logger.Warn(r, http.StatusBadRequest, "invalid indices", err)
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		logger.Error(r, http.StatusInternalServerError, "internal server error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *InvoiceHandler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		return
	}

	var req createInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "invalid request body", err)
		writeError(w, http.StatusBadRequest, "invalid request format")
		return
	}
	defer r.Body.Close()

	if err := validateStruct(req); err != nil {
		logger.Warn(r, http.StatusUnprocessableEntity, "validation failed", err)
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	input := service.CreateInvoiceInput{
		RoomID:              req.RoomID,
		Period:              req.Period,
		NewElectricityIndex: req.NewElectricityIndex,
		NewWaterIndex:       req.NewWaterIndex,
		OtherFee:            req.OtherFee,
		Discount:            req.Discount,
		VehicleCount:        req.VehicleCount,
	}

	invoice, err := h.invoiceService.CreateInvoice(r.Context(), managerID, input)
	if err != nil {
		handleInvoiceError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(invoice); err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to encode response", err)
	}
}

func (h *InvoiceHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		return
	}

	filter := model.InvoiceListFilter{
		HouseID: r.URL.Query().Get("house_id"),
		RoomID:  r.URL.Query().Get("room_id"),
		Period:  r.URL.Query().Get("period"),
		Status:  r.URL.Query().Get("status"),
		Page:    1,
		Limit:   20,
	}

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			filter.Page = p
		}
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			filter.Limit = l
		}
	}

	invoices, err := h.invoiceService.ListInvoices(r.Context(), managerID, filter)
	if err != nil {
		handleInvoiceError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(invoices); err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to encode response", err)
	}
}

func (h *InvoiceHandler) GetInvoice(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing invoice id")
		return
	}

	invoice, err := h.invoiceService.GetInvoice(r.Context(), managerID, id)
	if err != nil {
		handleInvoiceError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(invoice); err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to encode response", err)
	}
}

func (h *InvoiceHandler) PayInvoice(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing invoice id")
		return
	}

	invoice, err := h.invoiceService.PayInvoice(r.Context(), managerID, id)
	if err != nil {
		if err.Error() == "invoice is already paid" {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		handleInvoiceError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(invoice); err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to encode response", err)
	}
}

func (h *InvoiceHandler) UnpayInvoice(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing invoice id")
		return
	}

	invoice, err := h.invoiceService.UnpayInvoice(r.Context(), managerID, id)
	if err != nil {
		if err.Error() == "invoice is already unpaid" {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		handleInvoiceError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(invoice); err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to encode response", err)
	}
}
