package invoice

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	sharedsvc "github.com/mihb123/quanly-phongtro/internal/service/shared"

	"github.com/mihb123/quanly-phongtro/internal/handler/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/model"
	invoicesvc "github.com/mihb123/quanly-phongtro/internal/service/invoice"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
)

type InvoiceHandler struct {
	invoiceService invoicesvc.InvoiceService
	imageService   invoicesvc.ImageService
}

func NewInvoiceHandler(invoiceService invoicesvc.InvoiceService, imageService invoicesvc.ImageService) *InvoiceHandler {
	return &InvoiceHandler{invoiceService: invoiceService, imageService: imageService}
}

type createInvoiceRequest struct {
	RoomID              string  `json:"room_id" validate:"required"`
	Period              string  `json:"period" validate:"required,monthperiod"`
	OldElectricityIndex *int    `json:"old_electricity_index"`
	NewElectricityIndex int     `json:"new_electricity_index" validate:"gte=0"`
	OldWaterIndex       *int    `json:"old_water_index"`
	NewWaterIndex       int     `json:"new_water_index" validate:"gte=0"`
	OtherFee            float64 `json:"other_fee" validate:"gte=0"`
	Discount            float64 `json:"discount" validate:"gte=0"`
	VehicleCount        int     `json:"vehicle_count" validate:"gte=0"`
	TenantCount         *int    `json:"tenant_count"`
}

type invoiceResponse struct {
	ID                   string    `json:"id"`
	RoomID               string    `json:"room_id"`
	HouseID              string    `json:"house_id,omitempty"`
	RoomName             string    `json:"room_name,omitempty"`
	Period               string    `json:"period"`
	RoomFee              float64   `json:"room_fee"`
	OldElectricityIndex  int       `json:"old_electricity_index"`
	NewElectricityIndex  int       `json:"new_electricity_index"`
	ElectricityFee       float64   `json:"electricity_fee"`
	OldWaterIndex        int       `json:"old_water_index"`
	NewWaterIndex        int       `json:"new_water_index"`
	WaterFee             float64   `json:"water_fee"`
	WifiFee              float64   `json:"wifi_fee"`
	ParkingFee           float64   `json:"parking_fee"`
	ServiceFee           float64   `json:"service_fee"`
	OtherFee             float64   `json:"other_fee"`
	Discount             float64   `json:"discount"`
	TenantCount          int       `json:"tenant_count"`
	VehicleCount         int       `json:"vehicle_count"`
	ExtraPersonFee       float64   `json:"extra_person_fee"`
	ExtraVehicleFee      float64   `json:"extra_vehicle_fee"`
	TotalAmount          float64   `json:"total_amount"`
	Status               string    `json:"status"`
	PaymentMethod        *string   `json:"payment_method"`
	TransactionImagePath *string   `json:"transaction_image_path"`
	CreatedAt            time.Time `json:"created_at"`
}

// newInvoiceResponse shapes invoice output without exposing persistence ownership fields.
func newInvoiceResponse(invoice model.InvoiceWithRoom) invoiceResponse {
	return invoiceResponse{
		ID:                   invoice.ID,
		RoomID:               invoice.RoomID,
		HouseID:              invoice.HouseID,
		RoomName:             invoice.RoomName,
		Period:               invoice.Period,
		RoomFee:              invoice.RoomFee,
		OldElectricityIndex:  invoice.OldElectricityIndex,
		NewElectricityIndex:  invoice.NewElectricityIndex,
		ElectricityFee:       invoice.ElectricityFee,
		OldWaterIndex:        invoice.OldWaterIndex,
		NewWaterIndex:        invoice.NewWaterIndex,
		WaterFee:             invoice.WaterFee,
		WifiFee:              invoice.WifiFee,
		ParkingFee:           invoice.ParkingFee,
		ServiceFee:           invoice.ServiceFee,
		OtherFee:             invoice.OtherFee,
		Discount:             invoice.Discount,
		TenantCount:          invoice.TenantCount,
		VehicleCount:         invoice.VehicleCount,
		ExtraPersonFee:       invoice.ExtraPersonFee,
		ExtraVehicleFee:      invoice.ExtraVehicleFee,
		TotalAmount:          invoice.TotalAmount,
		Status:               invoice.Status,
		PaymentMethod:        invoice.PaymentMethod,
		TransactionImagePath: invoice.TransactionImagePath,
		CreatedAt:            invoice.CreatedAt,
	}
}

// newInvoiceStatusResponse shapes status-update output returned by pay/unpay endpoints.
func newInvoiceStatusResponse(invoice *model.Invoice) invoiceResponse {
	return invoiceResponse{
		ID:                   invoice.ID,
		RoomID:               invoice.RoomID,
		Period:               invoice.Period,
		RoomFee:              invoice.RoomFee,
		OldElectricityIndex:  invoice.OldElectricityIndex,
		NewElectricityIndex:  invoice.NewElectricityIndex,
		ElectricityFee:       invoice.ElectricityFee,
		OldWaterIndex:        invoice.OldWaterIndex,
		NewWaterIndex:        invoice.NewWaterIndex,
		WaterFee:             invoice.WaterFee,
		WifiFee:              invoice.WifiFee,
		ParkingFee:           invoice.ParkingFee,
		ServiceFee:           invoice.ServiceFee,
		OtherFee:             invoice.OtherFee,
		Discount:             invoice.Discount,
		TenantCount:          invoice.TenantCount,
		VehicleCount:         invoice.VehicleCount,
		ExtraPersonFee:       invoice.ExtraPersonFee,
		ExtraVehicleFee:      invoice.ExtraVehicleFee,
		TotalAmount:          invoice.TotalAmount,
		Status:               invoice.Status,
		PaymentMethod:        invoice.PaymentMethod,
		TransactionImagePath: invoice.TransactionImagePath,
		CreatedAt:            invoice.CreatedAt,
	}
}

// newInvoiceResponses maps repository results to external invoice responses.
func newInvoiceResponses(invoices []model.InvoiceWithRoom) []invoiceResponse {
	responses := make([]invoiceResponse, 0, len(invoices))
	for _, invoice := range invoices {
		responses = append(responses, newInvoiceResponse(invoice))
	}
	return responses
}

// writeInvoiceResponse encodes one invoice response with the current invoice handler shape.
func writeInvoiceResponse(w http.ResponseWriter, r *http.Request, status int, invoice invoiceResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(invoice); err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to encode response", err)
	}
}

// writeInvoiceListResponse encodes invoice list responses with the current invoice handler shape.
func writeInvoiceListResponse(w http.ResponseWriter, r *http.Request, invoices []invoiceResponse) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(invoices); err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to encode response", err)
	}
}

func handleInvoiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, model.ErrInvoiceNotFound):
		logger.Warn(r, http.StatusNotFound, "invoice not found", err)
		httpx.WriteError(w, http.StatusNotFound, "invoice not found")
	case errors.Is(err, model.ErrRoomNotFound):
		logger.Warn(r, http.StatusNotFound, "room not found", err)
		httpx.WriteError(w, http.StatusNotFound, "room not found")
	case errors.Is(err, model.ErrDuplicateInvoice):
		logger.Warn(r, http.StatusConflict, "duplicate invoice", err)
		httpx.WriteError(w, http.StatusConflict, "duplicate invoice for this room and period")
	case errors.Is(err, model.ErrInvalidElectricityIndex), errors.Is(err, model.ErrInvalidWaterIndex):
		logger.Warn(r, http.StatusBadRequest, "invalid indices", err)
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
	case model.IsInvoiceStateError(err):
		logger.Warn(r, http.StatusBadRequest, "invalid invoice state transition", err)
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		logger.Error(r, http.StatusInternalServerError, "internal server error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *InvoiceHandler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		return
	}

	var req createInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "invalid request body", err)
		httpx.WriteError(w, http.StatusBadRequest, "invalid request format")
		return
	}
	defer r.Body.Close()

	if err := httpx.ValidateStruct(req); err != nil {
		logger.Warn(r, http.StatusUnprocessableEntity, "validation failed", err)
		httpx.WriteError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	input := invoicesvc.CreateInvoiceInput{
		RoomID:              req.RoomID,
		Period:              req.Period,
		OldElectricityIndex: req.OldElectricityIndex,
		NewElectricityIndex: req.NewElectricityIndex,
		OldWaterIndex:       req.OldWaterIndex,
		NewWaterIndex:       req.NewWaterIndex,
		OtherFee:            req.OtherFee,
		Discount:            req.Discount,
		VehicleCount:        req.VehicleCount,
		TenantCount:         req.TenantCount,
	}

	invoice, err := h.invoiceService.CreateInvoice(r.Context(), managerID, input)
	if err != nil {
		handleInvoiceError(w, r, err)
		return
	}

	writeInvoiceResponse(w, r, http.StatusCreated, newInvoiceResponse(*invoice))
}

func (h *InvoiceHandler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
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
	if filter.Period != "" {
		if err := httpx.ValidateMonthPeriodValue(filter.Period); err != nil {
			logger.Warn(r, http.StatusBadRequest, "invalid invoice period", err)
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	invoices, err := h.invoiceService.ListInvoices(r.Context(), managerID, filter)
	if err != nil {
		handleInvoiceError(w, r, err)
		return
	}

	writeInvoiceListResponse(w, r, newInvoiceResponses(invoices))
}

func (h *InvoiceHandler) GetInvoice(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing invoice id")
		return
	}

	invoice, err := h.invoiceService.GetInvoice(r.Context(), managerID, id)
	if err != nil {
		handleInvoiceError(w, r, err)
		return
	}

	writeInvoiceResponse(w, r, http.StatusOK, newInvoiceResponse(*invoice))
}

func (h *InvoiceHandler) PayInvoice(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing invoice id")
		return
	}

	invoice, err := h.invoiceService.PayInvoice(r.Context(), managerID, id)
	if err != nil {
		handleInvoiceError(w, r, err)
		return
	}

	writeInvoiceResponse(w, r, http.StatusOK, newInvoiceStatusResponse(invoice))
}

func (h *InvoiceHandler) UnpayInvoice(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing invoice id")
		return
	}

	invoice, err := h.invoiceService.UnpayInvoice(r.Context(), managerID, id)
	if err != nil {
		handleInvoiceError(w, r, err)
		return
	}

	writeInvoiceResponse(w, r, http.StatusOK, newInvoiceStatusResponse(invoice))
}

func (h *InvoiceHandler) DeleteInvoice(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing invoice id")
		return
	}

	err := h.invoiceService.DeleteInvoice(r.Context(), managerID, id)
	if err != nil {
		handleInvoiceError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *InvoiceHandler) DownloadInvoiceImage(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "missing invoice id")
		return
	}

	invoice, err := h.invoiceService.GetInvoice(r.Context(), managerID, id)
	if err != nil {
		handleInvoiceError(w, r, err)
		return
	}

	imageBytes, err := h.imageService.GenerateInvoiceImage(r.Context(), invoice)
	if err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to generate image", err)
		httpx.WriteError(w, http.StatusInternalServerError, "failed to generate image")
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"phongtro_hoadon_%s_%s.png\"", invoice.RoomName, invoice.Period))
	w.Write(imageBytes)
}

// DownloadTransactionImage serves a transaction proof image after invoice ownership is verified.
func (h *InvoiceHandler) DownloadTransactionImage(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		return
	}

	fileName := chi.URLParam(r, "*")
	filePath, err := h.invoiceService.ResolveTransactionImagePath(r.Context(), managerID, fileName)
	if err != nil {
		switch {
		case errors.Is(err, sharedsvc.ErrInvalidInput):
			httpx.WriteError(w, http.StatusBadRequest, "invalid file path")
		case errors.Is(err, model.ErrInvoiceNotFound):
			httpx.WriteError(w, http.StatusNotFound, "file not found")
		default:
			logger.Error(r, http.StatusInternalServerError, "failed to resolve transaction image", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	http.ServeFile(w, r, filePath)
}
