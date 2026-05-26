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

type RoomHandler struct {
	roomService service.RoomService
}

func NewRoomHandler(roomService service.RoomService) *RoomHandler {
	return &RoomHandler{roomService: roomService}
}

type createRoomRequest struct {
	HouseID    string `json:"house_id" validate:"required"`
	Name       string `json:"name" validate:"required"`
	Price      int64  `json:"price" validate:"gte=0"`
	MaxTenants int    `json:"max_tenants" validate:"gte=1"`
	Status     string `json:"status" validate:"omitempty,oneof=AVAILABLE OCCUPIED MAINTENANCE"`
}

type updateRoomRequest struct {
	HouseID    string  `json:"house_id" validate:"required"`
	Name       *string `json:"name"`
	Price      *int64  `json:"price" validate:"omitempty,gte=0"`
	MaxTenants *int    `json:"max_tenants" validate:"omitempty,gte=1"`
	Status     *string `json:"status" validate:"omitempty,oneof=AVAILABLE OCCUPIED MAINTENANCE"`
}

// getManagerID extracts the authenticated manager's user ID from the JWT claims.
func getManagerID(r *http.Request, w http.ResponseWriter) (string, bool) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", nil)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return "", false
	}
	userID, err := claims.GetSubject()
	if err != nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: cannot read subject", err)
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return "", false
	}
	return userID, true
}

func handleRoomError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidRoomID), errors.Is(err, service.ErrInvalidHouseID):
		logger.Warn(r, http.StatusBadRequest, "invalid room or house ID", err)
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, model.ErrRoomNotFound):
		logger.Warn(r, http.StatusNotFound, "room or house not found", err)
		writeError(w, http.StatusNotFound, "room or house not found")
	case errors.Is(err, model.ErrHouseNotFound):
		logger.Warn(r, http.StatusNotFound, "house not found", err)
		writeError(w, http.StatusNotFound, "house not found")
	default:
		logger.Error(r, http.StatusInternalServerError, "room operation failed", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

// POST /api/v1/room
func (h *RoomHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		return
	}

	var req createRoomRequest
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

	status := req.Status
	if status == "" {
		status = "AVAILABLE"
	}

	room := &model.Room{
		HouseID:    req.HouseID,
		Name:       req.Name,
		Price:      req.Price,
		MaxTenants: req.MaxTenants,
		Status:     status,
	}

	if err := h.roomService.CreateRoom(r.Context(), room, managerID); err != nil {
		handleRoomError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, room, "")
}

// GET /api/v1/room?house_id=&page=&limit=
func (h *RoomHandler) ListRooms(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		return
	}

	houseID := r.URL.Query().Get("house_id")
	if houseID == "" {
		writeError(w, http.StatusBadRequest, "house_id query param is required")
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
	writeJSON(w, http.StatusOK, rooms, "")
}

// GET /api/v1/room/{id}?house_id=
func (h *RoomHandler) GetRoom(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	houseID := r.URL.Query().Get("house_id")
	if houseID == "" {
		writeError(w, http.StatusBadRequest, "house_id query param is required")
		return
	}

	room, err := h.roomService.GetRoom(r.Context(), id, houseID, managerID)
	if err != nil {
		handleRoomError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, room, "")
}

// PATCH /api/v1/room/{id}
func (h *RoomHandler) UpdateRoom(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")

	var req updateRoomRequest
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

	room, err := h.roomService.UpdateRoom(r.Context(), id, req.HouseID, managerID, service.UpdateRoomInput{
		Name:       req.Name,
		Price:      req.Price,
		MaxTenants: req.MaxTenants,
		Status:     req.Status,
	})
	if err != nil {
		handleRoomError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, room, "")
}

// DELETE /api/v1/room/{id}?house_id=
func (h *RoomHandler) DeleteRoom(w http.ResponseWriter, r *http.Request) {
	managerID, ok := getManagerID(r, w)
	if !ok {
		return
	}

	id := chi.URLParam(r, "id")
	houseID := r.URL.Query().Get("house_id")
	if houseID == "" {
		writeError(w, http.StatusBadRequest, "house_id query param is required")
		return
	}

	if err := h.roomService.DeleteRoom(r.Context(), id, houseID, managerID); err != nil {
		handleRoomError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, nil, "room deleted successfully")
}
