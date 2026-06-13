package model

import (
	"context"
	"time"
)

// ExtraCost represents a custom cost item in the JSONB field
type ExtraCost struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Note   string  `json:"note"`
}

// HouseCost stores monthly operating costs for a house
type HouseCost struct {
	ID          string      `json:"id"`
	HouseID     string      `json:"house_id"`
	Period      string      `json:"period"`
	Rent        float64     `json:"rent"`
	Electricity float64     `json:"electricity"`
	Water       float64     `json:"water"`
	Wifi        float64     `json:"wifi"`
	Cleaning    float64     `json:"cleaning"`
	Maintenance float64     `json:"maintenance"`
	ExtraCosts  []ExtraCost `json:"extra_costs" bun:"type:jsonb"`
	Note        string      `json:"note"`
	TotalCost   float64     `json:"total_cost"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// HouseRevenueSummary stores aggregated profit per house per period
type HouseRevenueSummary struct {
	ID           string    `json:"id"`
	HouseID      string    `json:"house_id"`
	Period       string    `json:"period"`
	TotalRevenue float64   `json:"total_revenue"`
	TotalCost    float64   `json:"total_cost"`
	Profit       float64   `json:"profit"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// HouseCostRepository defines DB operations for operating costs
type HouseCostRepository interface {
	Create(ctx context.Context, cost *HouseCost) error
	GetByHouseAndPeriod(ctx context.Context, houseID, period string) (*HouseCost, error)
	GetLatestByHouseID(ctx context.Context, houseID string) (*HouseCost, error)
	ListByHouseIDs(ctx context.Context, houseIDs []string, period string) ([]HouseCost, error)
	Update(ctx context.Context, cost *HouseCost) error
	Delete(ctx context.Context, id, houseID string) error
}

// RevenueSummaryRepository defines DB operations for revenue summaries
type RevenueSummaryRepository interface {
	Upsert(ctx context.Context, summary *HouseRevenueSummary) error
	GetByHouseAndPeriod(ctx context.Context, houseID, period string) (*HouseRevenueSummary, error)
	ListByHouseIDs(ctx context.Context, houseIDs []string, period string) ([]HouseRevenueSummary, error)
	CalculateRevenue(ctx context.Context, houseID, period string) (float64, error)
}
