package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

var (
	ErrInvalidHouseID   = errors.New("invalid house id")
	ErrInvalidManagerID = errors.New("invalid manager id")
)

type HouseService interface {
	CreateHouse(ctx context.Context, in CreateHouseInput) (*HouseOutput, error)
	GetHouseByID(ctx context.Context, id, managerID string) (*HouseOutput, error)
}

type HouseServiceImpl struct {
	houseRepo model.HouseRepository
}

func NewHouseServiceImpt(houseRepo model.HouseRepository) HouseService {
	return &HouseServiceImpl{houseRepo: houseRepo}
}

type CreateHouseInput struct {
	ManagerID               string
	Name                    string
	Address                 string
	DefaultElectricityPrice float64
	DefaultWaterPrice       float64
	DefaultWifiPrice        float64
	DefaultParkingPrice     float64
	DefaultServicePrice     float64
}

type HouseOutput struct {
	HouseID                 string    `json:"house_id"`
	ManagerID               string    `json:"manager_id"`
	Name                    string    `json:"name"`
	Address                 string    `json:"address"`
	DefaultElectricityPrice float64   `json:"default_electricity_price"`
	DefaultWaterPrice       float64   `json:"default_water_price"`
	DefaultWifiPrice        float64   `json:"default_wifi_price"`
	DefaultParkingPrice     float64   `json:"default_parking_price"`
	DefaultServicePrice     float64   `json:"default_service_price"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

func (h *HouseServiceImpl) CreateHouse(ctx context.Context, in CreateHouseInput) (*HouseOutput, error) {

	newHouse := &model.House{
		Name:                    in.Name,
		ManagerID:               in.ManagerID,
		Address:                 in.Address,
		DefaultElectricityPrice: in.DefaultElectricityPrice,
		DefaultWaterPrice:       in.DefaultWaterPrice,
		DefaultParkingPrice:     in.DefaultParkingPrice,
		DefaultWifiPrice:        in.DefaultWifiPrice,
		DefaultServicePrice:     in.DefaultServicePrice,
	}

	err := h.houseRepo.CreateHouse(ctx, newHouse)
	if err != nil {
		return nil, err
	}

	return toHouseOutput(newHouse), nil
}

func (h *HouseServiceImpl) GetHouseByID(ctx context.Context, id, managerID string) (*HouseOutput, error) {
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

	return toHouseOutput(house), nil
}

func toHouseOutput(house *model.House) *HouseOutput {
	return &HouseOutput{
		HouseID:                 house.ID,
		ManagerID:               house.ManagerID,
		Address:                 house.Address,
		Name:                    house.Name,
		DefaultElectricityPrice: house.DefaultElectricityPrice,
		DefaultWifiPrice:        house.DefaultWifiPrice,
		DefaultWaterPrice:       house.DefaultWaterPrice,
		DefaultServicePrice:     house.DefaultServicePrice,
		DefaultParkingPrice:     house.DefaultParkingPrice,
		CreatedAt:               house.CreatedAt,
		UpdatedAt:               house.UpdatedAt,
	}
}
