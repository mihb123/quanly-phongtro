package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mihb123/quanly-phongtro/internal/model"
)

type HouseCostService interface {
	CreateMonthlyCost(ctx context.Context, managerID, houseID, period string) (*model.HouseCost, error)
	GetMonthlyCost(ctx context.Context, managerID, houseID, period string) (*model.HouseCost, error)
	UpdateMonthlyCost(ctx context.Context, managerID string, input UpdateCostInput) error
	GetRevenueSummaries(ctx context.Context, managerID string, houseIDs []string, period string) ([]model.HouseRevenueSummary, error)
}

type UpdateCostInput struct {
	ID          string            `json:"id"`
	HouseID     string            `json:"house_id"`
	Period      string            `json:"period"`
	Rent        float64           `json:"rent"`
	Electricity float64           `json:"electricity"`
	Water       float64           `json:"water"`
	Wifi        float64           `json:"wifi"`
	Cleaning    float64           `json:"cleaning"`
	Maintenance float64           `json:"maintenance"`
	ExtraCosts  []model.ExtraCost `json:"extra_costs"`
	Note        string            `json:"note"`
}

type houseCostServiceImpl struct {
	costRepo    model.HouseCostRepository
	houseRepo   model.HouseRepository
	eventBus    EventBus
	summaryRepo model.RevenueSummaryRepository
}

func NewHouseCostService(costRepo model.HouseCostRepository, houseRepo model.HouseRepository, eventBus EventBus, summaryRepo model.RevenueSummaryRepository) HouseCostService {
	return &houseCostServiceImpl{
		costRepo:    costRepo,
		houseRepo:   houseRepo,
		eventBus:    eventBus,
		summaryRepo: summaryRepo,
	}
}

func (s *houseCostServiceImpl) verifyHouseOwnership(ctx context.Context, managerID, houseID string) error {
	owned, err := s.houseRepo.IsHouseOwnedBy(ctx, houseID, managerID)
	if err != nil {
		return fmt.Errorf("verify house ownership: %w", err)
	}
	if !owned {
		return model.ErrHouseCostForbidden
	}
	return nil
}

func (s *houseCostServiceImpl) calculateTotalCost(cost *model.HouseCost) float64 {
	total := cost.Rent + cost.Electricity + cost.Water + cost.Wifi + cost.Cleaning + cost.Maintenance
	for _, extra := range cost.ExtraCosts {
		total += extra.Amount
	}
	return total
}

func (s *houseCostServiceImpl) CreateMonthlyCost(ctx context.Context, managerID, houseID, period string) (*model.HouseCost, error) {
	if err := s.verifyHouseOwnership(ctx, managerID, houseID); err != nil {
		return nil, err
	}

	// Check if already exists
	existing, err := s.costRepo.GetByHouseAndPeriod(ctx, houseID, period)
	if err == nil && existing != nil {
		return nil, model.ErrHouseCostAlreadyExists
	}

	// Get latest to copy defaults
	latest, err := s.costRepo.GetLatestByHouseID(ctx, houseID)

	newCost := &model.HouseCost{
		ID:         uuid.New().String(),
		HouseID:    houseID,
		Period:     period,
		ExtraCosts: []model.ExtraCost{},
	}

	if err == nil && latest != nil {
		// Copy fixed costs
		newCost.Rent = latest.Rent
		newCost.Wifi = latest.Wifi
		newCost.Cleaning = latest.Cleaning
		// Variable costs default to 0
		newCost.Electricity = 0
		newCost.Water = 0
		newCost.Maintenance = 0
		// Optional: could copy extra_costs template with 0 amounts if desired, but starting empty is safer
	}

	newCost.TotalCost = s.calculateTotalCost(newCost)
	newCost.CreatedAt = time.Now()
	newCost.UpdatedAt = time.Now()

	if err := s.costRepo.Create(ctx, newCost); err != nil {
		return nil, err
	}

	// Trigger async recalculation
	if s.eventBus != nil {
		s.eventBus.Publish(EventHouseCostChanged, RevenueSummaryPayload{
			HouseID: houseID,
			Period:  period,
		})
	}

	return newCost, nil
}

func (s *houseCostServiceImpl) GetMonthlyCost(ctx context.Context, managerID, houseID, period string) (*model.HouseCost, error) {
	if err := s.verifyHouseOwnership(ctx, managerID, houseID); err != nil {
		return nil, err
	}

	cost, err := s.costRepo.GetByHouseAndPeriod(ctx, houseID, period)
	if err != nil {
		return nil, err
	}

	// Ensure ExtraCosts is never nil for JSON marshaling
	if cost.ExtraCosts == nil {
		cost.ExtraCosts = []model.ExtraCost{}
	}

	return cost, nil
}

func (s *houseCostServiceImpl) UpdateMonthlyCost(ctx context.Context, managerID string, input UpdateCostInput) error {
	if err := s.verifyHouseOwnership(ctx, managerID, input.HouseID); err != nil {
		return err
	}

	cost, err := s.costRepo.GetByHouseAndPeriod(ctx, input.HouseID, input.Period)
	if err != nil {
		return err
	}

	if cost.ID != input.ID {
		return model.ErrHouseCostInvalidID
	}

	cost.Rent = input.Rent
	cost.Electricity = input.Electricity
	cost.Water = input.Water
	cost.Wifi = input.Wifi
	cost.Cleaning = input.Cleaning
	cost.Maintenance = input.Maintenance
	cost.ExtraCosts = input.ExtraCosts
	if cost.ExtraCosts == nil {
		cost.ExtraCosts = []model.ExtraCost{}
	}
	cost.Note = input.Note
	cost.TotalCost = s.calculateTotalCost(cost)
	cost.UpdatedAt = time.Now()

	if err := s.costRepo.Update(ctx, cost); err != nil {
		return err
	}

	// Trigger async recalculation
	if s.eventBus != nil {
		s.eventBus.Publish(EventHouseCostChanged, RevenueSummaryPayload{
			HouseID: cost.HouseID,
			Period:  cost.Period,
		})
	}

	return nil
}

func (s *houseCostServiceImpl) GetRevenueSummaries(ctx context.Context, managerID string, houseIDs []string, period string) ([]model.HouseRevenueSummary, error) {
	// Filter to only owned houses
	ownedHouseIDs := []string{}
	for _, id := range houseIDs {
		owned, err := s.houseRepo.IsHouseOwnedBy(ctx, id, managerID)
		if err == nil && owned {
			ownedHouseIDs = append(ownedHouseIDs, id)
		}
	}

	if len(ownedHouseIDs) == 0 {
		return []model.HouseRevenueSummary{}, nil
	}

	summaries, err := s.summaryRepo.ListByHouseIDs(ctx, ownedHouseIDs, period)
	if err != nil {
		return nil, err
	}
	return summaries, nil
}
