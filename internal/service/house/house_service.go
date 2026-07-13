package house

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/service/shared"
)

type HouseService interface {
	CreateHouse(ctx context.Context, house *model.House) error
	GetHouseByID(ctx context.Context, id, managerID string) (*model.House, error)
	ListHouseByManagerID(ctx context.Context, managerID string, page, limit int, search string) ([]model.House, error)
	UpdateHouse(ctx context.Context, id, managerID string, input UpdateHouseInput) (*model.House, error)
	DeleteHouse(ctx context.Context, id, managerID string) error
	// IsHouseCodeAvailable báo house_code còn dùng được không trong toàn hệ thống, bỏ qua excludeHouseID.
	IsHouseCodeAvailable(ctx context.Context, houseCode, excludeHouseID string) (bool, error)
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

// IsHouseCodeAvailable báo house_code còn dùng được không (chưa bị nhà nào trong toàn hệ thống chiếm); bỏ qua nhà excludeHouseID khi cập nhật.
func (h *HouseServiceImpl) IsHouseCodeAvailable(ctx context.Context, houseCode, excludeHouseID string) (bool, error) {
	taken, err := h.houseRepo.IsHouseCodeTaken(ctx, houseCode, excludeHouseID)
	if err != nil {
		return false, err
	}
	return !taken, nil
}

// ensureHouseCodeUnique đảm bảo house_code chưa bị nhà nào khác trong toàn hệ thống dùng (khớp unique index uq_houses_code); bỏ qua nhà excludeID khi cập nhật.
func (h *HouseServiceImpl) ensureHouseCodeUnique(ctx context.Context, houseCode, excludeID string) error {
	available, err := h.IsHouseCodeAvailable(ctx, houseCode, excludeID)
	if err != nil {
		return err
	}
	if !available {
		return model.ErrHouseCodeExists
	}
	return nil
}

func (h *HouseServiceImpl) CreateHouse(ctx context.Context, house *model.House) error {
	if err := h.ensureHouseCodeUnique(ctx, house.HouseCode, ""); err != nil {
		return err
	}

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
		ID:         uuid.Must(uuid.NewV7()).String(),
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
		return nil, shared.ErrInvalidHouseID
	}

	ownerID := strings.TrimSpace(managerID)
	if ownerID == "" {
		return nil, shared.ErrInvalidManagerID
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
	if err := h.ensureHouseCodeUnique(ctx, input.HouseCode, id); err != nil {
		return nil, err
	}

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
		return shared.ErrInvalidHouseID
	}
	ownerID := strings.TrimSpace(managerID)
	if ownerID == "" {
		return shared.ErrInvalidManagerID
	}
	return h.houseRepo.DeleteHouse(ctx, houseID, ownerID)
}
