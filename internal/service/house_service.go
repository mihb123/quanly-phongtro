package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mihb123/quanly-phongtro/internal/model"
)

var (
	ErrInvalidHouseID   = errors.New("invalid house id")
	ErrInvalidManagerID = errors.New("invalid manager id")
)

type HouseService interface {
	CreateHouse(ctx context.Context, house *model.House) error
	GetHouseByID(ctx context.Context, id, managerID string) (*model.House, error)
	ListHouseByManagerID(ctx context.Context, managerID string, page, limit int, search string) ([]model.House, error)
	UpdateHouse(ctx context.Context, id, managerID string, input UpdateHouseInput) (*model.House, error)
	DeleteHouse(ctx context.Context, id, managerID string) error
}

type HouseServiceImpl struct {
	houseRepo     model.HouseRepository
	houseCostRepo model.HouseCostRepository
}

func NewHouseServiceImpt(houseRepo model.HouseRepository, houseCostRepo model.HouseCostRepository) HouseService {
	return &HouseServiceImpl{
		houseRepo:     houseRepo,
		houseCostRepo: houseCostRepo,
	}
}

type UpdateHouseInput struct {
	Name                    string
	HouseCode               string
	Address                 string
	DefaultElectricityPrice float64
	DefaultWaterPrice       float64
	DefaultWifiPrice        float64
	DefaultParkingPrice     float64
	DefaultServicePrice     float64
	ElectricityBillingType  string
	WaterBillingType        string
	ElectricityBillingUnit  string
	WaterBillingUnit        string
	ExtraPersonThreshold    int
	ExtraPersonFee          float64
	ExtraVehicleThreshold   int
	ExtraVehicleFee         float64
}

func (h *HouseServiceImpl) CreateHouse(ctx context.Context, house *model.House) error {
	err := h.houseRepo.CreateHouse(ctx, house)
	if err != nil {
		return err
	}
	if h.houseCostRepo == nil {
		return nil
	}

	now := time.Now()
	period := now.Format("2006-01")
	cost := &model.HouseCost{
		ID:         uuid.New().String(),
		HouseID:    house.ID,
		Period:     period,
		ExtraCosts: []model.ExtraCost{},
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := h.houseCostRepo.Create(ctx, cost); err != nil {
		return err
	}
	return nil
}

func (h *HouseServiceImpl) GetHouseByID(ctx context.Context, id, managerID string) (*model.House, error) {
	houseID := strings.TrimSpace(id)
	if houseID == "" {
		return nil, ErrInvalidHouseID
	}

	ownerID := strings.TrimSpace(managerID)
	if ownerID == "" {
		return nil, ErrInvalidManagerID
	}

	house, err := h.houseRepo.GetByID(ctx, houseID, ownerID)
	if err != nil {
		return nil, err
	}

	return house, nil
}

func (h *HouseServiceImpl) ListHouseByManagerID(ctx context.Context, managerID string, page, limit int, search string) ([]model.House, error) {
	offset := (page - 1) * limit
	return h.houseRepo.ListHouseByManagerID(ctx, managerID, limit, offset, search)
}

func (h *HouseServiceImpl) UpdateHouse(ctx context.Context, id, managerID string, input UpdateHouseInput) (*model.House, error) {
	updateHouseParams := model.UpdateHouseParams{
		Name:                    input.Name,
		HouseCode:               input.HouseCode,
		Address:                 input.Address,
		DefaultElectricityPrice: input.DefaultElectricityPrice,
		DefaultWaterPrice:       input.DefaultWaterPrice,
		DefaultWifiPrice:        input.DefaultWifiPrice,
		DefaultParkingPrice:     input.DefaultParkingPrice,
		DefaultServicePrice:     input.DefaultServicePrice,
		ElectricityBillingType:  input.ElectricityBillingType,
		WaterBillingType:        input.WaterBillingType,
		ElectricityBillingUnit:  input.ElectricityBillingUnit,
		WaterBillingUnit:        input.WaterBillingUnit,
		ExtraPersonThreshold:    input.ExtraPersonThreshold,
		ExtraPersonFee:          input.ExtraPersonFee,
		ExtraVehicleThreshold:   input.ExtraVehicleThreshold,
		ExtraVehicleFee:         input.ExtraVehicleFee,
	}
	house, err := h.houseRepo.UpdateHouse(ctx, id, managerID, updateHouseParams)
	if err != nil {
		return nil, err
	}
	return house, nil
}

func (h *HouseServiceImpl) DeleteHouse(ctx context.Context, id, managerID string) error {
	houseID := strings.TrimSpace(id)
	if houseID == "" {
		return ErrInvalidHouseID
	}
	ownerID := strings.TrimSpace(managerID)
	if ownerID == "" {
		return ErrInvalidManagerID
	}
	return h.houseRepo.DeleteHouse(ctx, houseID, ownerID)
}
