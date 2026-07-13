package room

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	sharedsvc "github.com/mihb123/quanly-phongtro/internal/service/shared"

	"github.com/mihb123/quanly-phongtro/internal/handler/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/mihb123/quanly-phongtro/internal/model"
	invoicesvc "github.com/mihb123/quanly-phongtro/internal/service/invoice"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
	roomsvc "github.com/mihb123/quanly-phongtro/internal/service/room"
)

type RoomHandler struct {
	roomService    roomsvc.RoomService
	invoiceService invoicesvc.InvoiceService
}

func NewRoomHandler(roomService roomsvc.RoomService, invoiceService invoicesvc.InvoiceService) *RoomHandler {
	return &RoomHandler{roomService: roomService, invoiceService: invoiceService}
}

type createRoomRequest struct {
	HouseID               string   `json:"house_id" validate:"required"`
	Name                  string   `json:"name" validate:"required"`
	Price                 int64    `json:"price" validate:"gte=0"`
	MaxTenants            int      `json:"max_tenants" validate:"gte=1"`
	Status                string   `json:"status" validate:"oneof=AVAILABLE OCCUPIED MAINTENANCE"`
	ElectricityPrice      *float64 `json:"electricity_price,omitempty"`
	WaterPrice            *float64 `json:"water_price,omitempty"`
	WifiPrice             *float64 `json:"wifi_price,omitempty"`
	ParkingPrice          *float64 `json:"parking_price,omitempty"`
	ServicePrice          *float64 `json:"service_price,omitempty"`
	ExtraPersonThreshold  *int     `json:"extra_person_threshold,omitempty"`
	ExtraPersonFee        *float64 `json:"extra_person_fee,omitempty"`
	ExtraVehicleThreshold *int     `json:"extra_vehicle_threshold,omitempty"`
	ExtraVehicleFee       *float64 `json:"extra_vehicle_fee,omitempty"`
}

type updateRoomRequest struct {
	HouseID               string   `json:"house_id" validate:"required"`
	Name                  string   `json:"name" validate:"required"`
	Price                 int64    `json:"price" validate:"gte=0"`
	MaxTenants            int      `json:"max_tenants" validate:"gte=1"`
	Status                string   `json:"status" validate:"oneof=AVAILABLE OCCUPIED MAINTENANCE"`
	ElectricityPrice      *float64 `json:"electricity_price,omitempty"`
	WaterPrice            *float64 `json:"water_price,omitempty"`
	WifiPrice             *float64 `json:"wifi_price,omitempty"`
	ParkingPrice          *float64 `json:"parking_price,omitempty"`
	ServicePrice          *float64 `json:"service_price,omitempty"`
	ExtraPersonThreshold  *int     `json:"extra_person_threshold,omitempty"`
	ExtraPersonFee        *float64 `json:"extra_person_fee,omitempty"`
	ExtraVehicleThreshold *int     `json:"extra_vehicle_threshold,omitempty"`
	ExtraVehicleFee       *float64 `json:"extra_vehicle_fee,omitempty"`
	GroupChatID           *string  `json:"group_chat_id,omitempty"`
}

func handleRoomError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, sharedsvc.ErrInvalidRoomID), errors.Is(err, sharedsvc.ErrInvalidHouseID):
		logger.Warn(r, http.StatusBadRequest, "invalid room or house ID", err)
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, model.ErrRoomNotFound):
		logger.Warn(r, http.StatusNotFound, "room or house not found", err)
		httpx.WriteError(w, http.StatusNotFound, "room or house not found")
	case errors.Is(err, model.ErrHouseNotFound):
		logger.Warn(r, http.StatusNotFound, "house not found", err)
		httpx.WriteError(w, http.StatusNotFound, "house not found")
	default:
		logger.Error(r, http.StatusInternalServerError, "room operation failed", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

// POST /api/v1/room
func (h *RoomHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		return
	}

	var req createRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "invalid request body", err)
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := httpx.ValidateStruct(req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "request validation failed", err)
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	status := req.Status
	if status == "" {
		status = "AVAILABLE"
	}

	room := &model.Room{
		HouseID:               req.HouseID,
		Name:                  req.Name,
		Price:                 req.Price,
		MaxTenants:            req.MaxTenants,
		Status:                status,
		ElectricityPrice:      req.ElectricityPrice,
		WaterPrice:            req.WaterPrice,
		WifiPrice:             req.WifiPrice,
		ParkingPrice:          req.ParkingPrice,
		ServicePrice:          req.ServicePrice,
		ExtraPersonThreshold:  req.ExtraPersonThreshold,
		ExtraPersonFee:        req.ExtraPersonFee,
		ExtraVehicleThreshold: req.ExtraVehicleThreshold,
		ExtraVehicleFee:       req.ExtraVehicleFee,
	}

	if err := h.roomService.CreateRoom(r.Context(), room, managerID); err != nil {
		handleRoomError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, room, "")
}

// GET /api/v1/room?house_id=&page=&limit=
func (h *RoomHandler) ListRooms(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		return
	}

	houseID := r.URL.Query().Get("house_id")
	if houseID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "house_id query param is required")
		return
	}

	pageInt, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || pageInt < 1 {
		pageInt = 1
	}
	limitInt, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limitInt < 1 {
		limitInt = 25
	}

	rooms, err := h.roomService.ListRoomsByHouseID(r.Context(), houseID, managerID, pageInt, limitInt)
	if err != nil {
		handleRoomError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, rooms, "")
}

// GET /api/v1/room/{id}?house_id=
func (h *RoomHandler) GetRoom(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	houseID := r.URL.Query().Get("house_id")
	if houseID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "house_id query param is required")
		return
	}

	room, err := h.roomService.GetRoom(r.Context(), id, houseID, managerID)
	if err != nil {
		handleRoomError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, room, "")
}

// PATCH /api/v1/room/{id}
func (h *RoomHandler) UpdateRoom(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")

	var req updateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "invalid request body", err)
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := httpx.ValidateStruct(req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "request validation failed", err)
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	room, err := h.roomService.UpdateRoom(r.Context(), id, req.HouseID, managerID, roomsvc.UpdateRoomInput{
		Name:                  req.Name,
		Price:                 req.Price,
		MaxTenants:            req.MaxTenants,
		Status:                req.Status,
		ElectricityPrice:      req.ElectricityPrice,
		WaterPrice:            req.WaterPrice,
		WifiPrice:             req.WifiPrice,
		ParkingPrice:          req.ParkingPrice,
		ServicePrice:          req.ServicePrice,
		ExtraPersonThreshold:  req.ExtraPersonThreshold,
		ExtraPersonFee:        req.ExtraPersonFee,
		ExtraVehicleThreshold: req.ExtraVehicleThreshold,
		ExtraVehicleFee:       req.ExtraVehicleFee,
		GroupChatID:           req.GroupChatID,
	})
	if err != nil {
		handleRoomError(w, r, err)
		return
	}

	// Trigger recalculation for UNPAID invoices when room is updated
	err = h.invoiceService.RecalculateUnpaidInvoicesByRoom(r.Context(), managerID, id)
	if err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to recalculate invoices after room update", err)
		httpx.WriteError(w, http.StatusInternalServerError, "failed to recalculate invoices after room update")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, room, "")
}

// DELETE /api/v1/room/{id}?house_id=
func (h *RoomHandler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	managerID, ok := httpx.GetManagerID(r, w)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	houseID := r.URL.Query().Get("house_id")
	if houseID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "house_id query param is required")
		return
	}

	if err := h.roomService.DeleteRoom(r.Context(), id, houseID, managerID); err != nil {
		handleRoomError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, nil, "room deleted successfully")
}
