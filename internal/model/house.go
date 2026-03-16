package model

import (
	"context"
	"errors"
	"time"
)

var (
	ErrHouseNotFound = errors.New("house not found")
)

type House struct {
	ID                      string
	ManagerID               string
	Name                    string
	Address                 string
	DefaultElectricityPrice float64
	DefaultWaterPrice       float64
	DefaultParikingPrice    float64
	DefaultServicePrice     float64
	DefaultWifiPrice        float64
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type HouseRepository interface {
	CreateHouse(ctx context.Context, house *House) error
	GetByID(ctx context.Context, id, managerID string) (*House, error)
}
