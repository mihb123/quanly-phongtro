package room

import (
	"context"
	"mime/multipart"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/service/shared"
)

type RoomService interface {
	CreateRoom(ctx context.Context, room *model.Room, managerID string) error
	GetRoom(ctx context.Context, id, houseID, managerID string) (*model.Room, error)
	ListRoomsByHouseID(ctx context.Context, managerID string, query ListRoomsQuery) ([]model.Room, int, error)
	GetRoomStats(ctx context.Context, managerID, houseID string) (*model.RoomStats, error)
	ListAvailableRooms(ctx context.Context, managerID string, limit int) ([]model.Room, int, error)
	UpdateRoom(ctx context.Context, id, houseID, managerID string, input UpdateRoomInput) (*model.Room, error)
	UpdateRoomContract(ctx context.Context, id, houseID, managerID string, input UpdateRoomContractInput) (*model.Room, error)
	DeleteRoom(ctx context.Context, id, houseID, managerID string) error
}

type ListRoomsQuery struct {
	HouseID string
	Search  string
	Status  string
	Page    int
	Limit   int
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

// UpdateRoomContractInput carries the contract files of a room.
// KeptContractPaths holds the already-stored paths the manager wants to keep ("" = remove all).
type UpdateRoomContractInput struct {
	ContractFiles     []*multipart.FileHeader
	KeptContractPaths *string
}

type roomInvoiceRecalculator interface {
	RecalculateUnpaidInvoicesByRoom(context.Context, string, string) error
}

type BillingOption func(*RoomServiceImpl)

func WithInvoiceRecalculation(invoices roomInvoiceRecalculator, transaction func(context.Context, func(context.Context) error) error) BillingOption {
	return func(s *RoomServiceImpl) { s.invoices = invoices; s.transaction = transaction }
}

type RoomServiceImpl struct {
	invoices    roomInvoiceRecalculator
	transaction func(context.Context, func(context.Context) error) error
	roomRepo    model.RoomRepository
	houseRepo   model.HouseRepository
}

func NewRoomService(roomRepo model.RoomRepository, houseRepo model.HouseRepository, options ...BillingOption) RoomService {
	s := &RoomServiceImpl{roomRepo: roomRepo, houseRepo: houseRepo}
	for _, option := range options {
		option(s)
	}
	return s
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
		return shared.ErrInvalidHouseID
	}
	if strings.TrimSpace(managerID) == "" {
		return shared.ErrInvalidManagerID
	}
	if _, err := s.houseRepo.GetByID(ctx, room.HouseID, managerID); err != nil {
		return err
	}
	return s.roomRepo.CreateRoom(ctx, room)
}

func (s *RoomServiceImpl) GetRoom(ctx context.Context, id, houseID, managerID string) (*model.Room, error) {
	if strings.TrimSpace(id) == "" {
		return nil, shared.ErrInvalidRoomID
	}
	if strings.TrimSpace(houseID) == "" {
		return nil, shared.ErrInvalidHouseID
	}
	if strings.TrimSpace(managerID) == "" {
		return nil, shared.ErrInvalidManagerID
	}
	if err := s.checkOwnership(ctx, houseID, managerID); err != nil {
		return nil, err
	}
	return s.roomRepo.GetRoomByID(ctx, id, houseID)
}

func (s *RoomServiceImpl) ListRoomsByHouseID(ctx context.Context, managerID string, query ListRoomsQuery) ([]model.Room, int, error) {
	if strings.TrimSpace(query.HouseID) == "" {
		return nil, 0, shared.ErrInvalidHouseID
	}
	if strings.TrimSpace(managerID) == "" {
		return nil, 0, shared.ErrInvalidManagerID
	}
	if err := s.checkOwnership(ctx, query.HouseID, managerID); err != nil {
		return nil, 0, err
	}
	return s.roomRepo.ListRoomsByHouseID(ctx, model.RoomListFilter{
		HouseID: query.HouseID,
		Search:  query.Search,
		Status:  query.Status,
		Limit:   query.Limit,
		Offset:  (query.Page - 1) * query.Limit,
	})
}

func (s *RoomServiceImpl) GetRoomStats(ctx context.Context, managerID, houseID string) (*model.RoomStats, error) {
	if strings.TrimSpace(managerID) == "" {
		return nil, shared.ErrInvalidManagerID
	}
	if houseID != "" {
		if err := s.checkOwnership(ctx, houseID, managerID); err != nil {
			return nil, err
		}
	}
	return s.roomRepo.GetRoomStats(ctx, managerID, houseID)
}

func (s *RoomServiceImpl) ListAvailableRooms(ctx context.Context, managerID string, limit int) ([]model.Room, int, error) {
	if strings.TrimSpace(managerID) == "" {
		return nil, 0, shared.ErrInvalidManagerID
	}
	return s.roomRepo.ListAvailableRooms(ctx, managerID, limit)
}

func (s *RoomServiceImpl) UpdateRoom(ctx context.Context, id, houseID, managerID string, input UpdateRoomInput) (*model.Room, error) {
	if strings.TrimSpace(id) == "" {
		return nil, shared.ErrInvalidRoomID
	}
	if strings.TrimSpace(houseID) == "" {
		return nil, shared.ErrInvalidHouseID
	}
	if strings.TrimSpace(managerID) == "" {
		return nil, shared.ErrInvalidManagerID
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

	var room *model.Room
	update := func(ctx context.Context) error {
		var err error
		room, err = s.roomRepo.UpdateRoom(ctx, id, houseID, params)
		if err != nil {
			return err
		}
		if s.invoices != nil {
			return s.invoices.RecalculateUnpaidInvoicesByRoom(ctx, managerID, id)
		}
		return nil
	}
	var err error
	if s.transaction != nil {
		err = s.transaction(ctx, update)
	} else {
		err = update(ctx)
	}
	if err != nil {
		return nil, err
	}
	return room, nil
}

// UpdateRoomContract lưu hợp đồng thuê của phòng: giữ lại các file cũ được chọn và nối thêm file mới.
func (s *RoomServiceImpl) UpdateRoomContract(ctx context.Context, id, houseID, managerID string, input UpdateRoomContractInput) (*model.Room, error) {
	if strings.TrimSpace(id) == "" {
		return nil, shared.ErrInvalidRoomID
	}
	if strings.TrimSpace(houseID) == "" {
		return nil, shared.ErrInvalidHouseID
	}
	if strings.TrimSpace(managerID) == "" {
		return nil, shared.ErrInvalidManagerID
	}
	if err := s.checkOwnership(ctx, houseID, managerID); err != nil {
		return nil, err
	}

	existing, err := s.roomRepo.GetRoomByID(ctx, id, houseID)
	if err != nil {
		return nil, err
	}

	contractPath := existing.ContractPath
	if input.KeptContractPaths != nil {
		contractPath = *input.KeptContractPaths
	}
	if len(input.ContractFiles) > 0 {
		newPaths, err := shared.SaveUploadedFiles(shared.TenantUploadDir, input.ContractFiles)
		if err != nil {
			return nil, err
		}
		if contractPath != "" {
			contractPath += "," + newPaths
		} else {
			contractPath = newPaths
		}
	}

	return s.roomRepo.UpdateRoomContract(ctx, id, houseID, contractPath)
}

func (s *RoomServiceImpl) DeleteRoom(ctx context.Context, id, houseID, managerID string) error {
	if strings.TrimSpace(id) == "" {
		return shared.ErrInvalidRoomID
	}
	if strings.TrimSpace(houseID) == "" {
		return shared.ErrInvalidHouseID
	}
	if strings.TrimSpace(managerID) == "" {
		return shared.ErrInvalidManagerID
	}
	if err := s.checkOwnership(ctx, houseID, managerID); err != nil {
		return err
	}
	return s.roomRepo.DeleteRoom(ctx, id, houseID)
}
