package service

import (
	"context"
	"errors"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

var (
	ErrInvalidRoomID = errors.New("invalid room id")
)

type RoomService interface {
	CreateRoom(ctx context.Context, room *model.Room, managerID string) error
	GetRoom(ctx context.Context, id, houseID, managerID string) (*model.Room, error)
	ListRoomsByHouseID(ctx context.Context, houseID, managerID string, page, limit int) ([]model.Room, error)
	UpdateRoom(ctx context.Context, id, houseID, managerID string, input UpdateRoomInput) (*model.Room, error)
	DeleteRoom(ctx context.Context, id, houseID, managerID string) error
}

type UpdateRoomInput struct {
	Name                  string
	Price                 int64
	MaxTenants            int
	Status                string
	ElectricityPrice      *float64
	WaterPrice            *float64
	WifiPrice             *float64
	ParkingPrice          *float64
	ServicePrice          *float64
	ExtraPersonThreshold  *int
	ExtraPersonFee        *float64
	ExtraVehicleThreshold *int
	ExtraVehicleFee       *float64
	GroupChatID           *string
}

type RoomServiceImpl struct {
	roomRepo  model.RoomRepository
	houseRepo model.HouseRepository
}

func NewRoomService(roomRepo model.RoomRepository, houseRepo model.HouseRepository) RoomService {
	return &RoomServiceImpl{roomRepo: roomRepo, houseRepo: houseRepo}
}

// checkOwnership verifies that houseID belongs to managerID.
// Returns model.ErrHouseNotFound if not owned or not found.
func (s *RoomServiceImpl) checkOwnership(ctx context.Context, houseID, managerID string) error {
	owned, err := s.houseRepo.IsHouseOwnedBy(ctx, houseID, managerID)
	if err != nil {
		return err
	}
	if !owned {
		return model.ErrHouseNotFound
	}
	return nil
}

func (s *RoomServiceImpl) CreateRoom(ctx context.Context, room *model.Room, managerID string) error {
	if strings.TrimSpace(room.HouseID) == "" {
		return ErrInvalidHouseID
	}
	if strings.TrimSpace(managerID) == "" {
		return ErrInvalidManagerID
	}
	if _, err := s.houseRepo.GetByID(ctx, room.HouseID, managerID); err != nil {
		return err
	}
	return s.roomRepo.CreateRoom(ctx, room)
}

func (s *RoomServiceImpl) GetRoom(ctx context.Context, id, houseID, managerID string) (*model.Room, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrInvalidRoomID
	}
	if strings.TrimSpace(houseID) == "" {
		return nil, ErrInvalidHouseID
	}
	if strings.TrimSpace(managerID) == "" {
		return nil, ErrInvalidManagerID
	}
	if err := s.checkOwnership(ctx, houseID, managerID); err != nil {
		return nil, err
	}
	return s.roomRepo.GetRoomByID(ctx, id, houseID)
}

func (s *RoomServiceImpl) ListRoomsByHouseID(ctx context.Context, houseID, managerID string, page, limit int) ([]model.Room, error) {
	if strings.TrimSpace(houseID) == "" {
		return nil, ErrInvalidHouseID
	}
	if strings.TrimSpace(managerID) == "" {
		return nil, ErrInvalidManagerID
	}
	if err := s.checkOwnership(ctx, houseID, managerID); err != nil {
		return nil, err
	}
	offset := (page - 1) * limit
	return s.roomRepo.ListRoomsByHouseID(ctx, houseID, limit, offset)
}

func (s *RoomServiceImpl) UpdateRoom(ctx context.Context, id, houseID, managerID string, input UpdateRoomInput) (*model.Room, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrInvalidRoomID
	}
	if strings.TrimSpace(houseID) == "" {
		return nil, ErrInvalidHouseID
	}
	if strings.TrimSpace(managerID) == "" {
		return nil, ErrInvalidManagerID
	}
	if err := s.checkOwnership(ctx, houseID, managerID); err != nil {
		return nil, err
	}
	params := model.UpdateRoomParams{
		Name:                  input.Name,
		Price:                 input.Price,
		MaxTenants:            input.MaxTenants,
		Status:                input.Status,
		ElectricityPrice:      input.ElectricityPrice,
		WaterPrice:            input.WaterPrice,
		WifiPrice:             input.WifiPrice,
		ParkingPrice:          input.ParkingPrice,
		ServicePrice:          input.ServicePrice,
		ExtraPersonThreshold:  input.ExtraPersonThreshold,
		ExtraPersonFee:        input.ExtraPersonFee,
		ExtraVehicleThreshold: input.ExtraVehicleThreshold,
		ExtraVehicleFee:       input.ExtraVehicleFee,
		GroupChatID:           input.GroupChatID,
	}

	return s.roomRepo.UpdateRoom(ctx, id, houseID, params)
}

func (s *RoomServiceImpl) DeleteRoom(ctx context.Context, id, houseID, managerID string) error {
	if strings.TrimSpace(id) == "" {
		return ErrInvalidRoomID
	}
	if strings.TrimSpace(houseID) == "" {
		return ErrInvalidHouseID
	}
	if strings.TrimSpace(managerID) == "" {
		return ErrInvalidManagerID
	}
	if err := s.checkOwnership(ctx, houseID, managerID); err != nil {
		return err
	}
	return s.roomRepo.DeleteRoom(ctx, id, houseID)
}
