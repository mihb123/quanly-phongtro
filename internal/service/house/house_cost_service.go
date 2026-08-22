package house

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/service/revenue"

	"github.com/google/uuid"
	"github.com/mihb123/quanly-phongtro/internal/model"
)

type HouseCostService interface {
	CreateMonthlyCost(ctx context.Context, managerID, houseID, period string) (*model.HouseCost, error)
	GetMonthlyCost(ctx context.Context, managerID, houseID, period string) (*model.HouseCost, error)
	UpdateMonthlyCost(ctx context.Context, managerID string, input UpdateCostInput) error
	GetRevenueSummaries(ctx context.Context, managerID string, houseIDs []string, period string) ([]model.HouseRevenueSummary, error)
	GenerateMonthlyCostsFromPrevious(ctx context.Context, previousPeriod, period string) (GenerateMonthlyCostsResult, error)
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
	ExtraCosts  []model.ExtraCost `json:"extra_costs"`
	Note        string            `json:"note"`
}

type houseCostServiceImpl struct {
	costRepo    model.HouseCostRepository
	houseRepo   model.HouseRepository
	eventBus    revenue.EventBus
	summaryRepo model.RevenueSummaryRepository
}

func NewHouseCostService(costRepo model.HouseCostRepository, houseRepo model.HouseRepository, eventBus revenue.EventBus, summaryRepo model.RevenueSummaryRepository) HouseCostService {
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
	total := cost.Rent + cost.Electricity + cost.Water + cost.Wifi + cost.Cleaning
	for _, extra := range cost.ExtraCosts {
		total += extra.Amount
	}
	return total
}

// hasFixedCosts reports whether a cost record carries any fixed-cost information worth carrying forward.
func hasFixedCosts(cost *model.HouseCost) bool {
	return cost != nil && (cost.Rent > 0 || cost.Wifi > 0 || cost.Cleaning > 0)
}

// buildNextCost creates the record for a new period, carrying over the fixed costs
// (rent, wifi, cleaning) from source while leaving variable costs at 0.
func (s *houseCostServiceImpl) buildNextCost(houseID, period string, source *model.HouseCost) *model.HouseCost {
	now := time.Now()
	newCost := &model.HouseCost{
		ID:         uuid.Must(uuid.NewV7()).String(),
		HouseID:    houseID,
		Period:     period,
		ExtraCosts: []model.ExtraCost{},
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if source != nil {
		newCost.Rent = source.Rent
		newCost.Wifi = source.Wifi
		newCost.Cleaning = source.Cleaning
	}

	newCost.TotalCost = s.calculateTotalCost(newCost)
	return newCost
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
	if err != nil {
		latest = nil
	}

	newCost := s.buildNextCost(houseID, period, latest)

	if err := s.costRepo.Create(ctx, newCost); err != nil {
		return nil, err
	}

	// Trigger async recalculation
	if s.eventBus != nil {
		s.eventBus.Publish(revenue.EventHouseCostChanged, revenue.RevenueSummaryPayload{
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
		s.eventBus.Publish(revenue.EventHouseCostChanged, revenue.RevenueSummaryPayload{
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

// GenerateMonthlyCostsResult summarises one batch run of GenerateMonthlyCostsFromPrevious.
type GenerateMonthlyCostsResult struct {
	Created int
	Skipped int
	Failed  int
}

// GenerateMonthlyCostsFromPrevious creates the cost record of `period` for every house that
// already has fixed-cost information in `previousPeriod`. Houses whose previous record carries
// no fixed cost, and houses that already have a record for `period`, are skipped.
// Intended for the monthly cron job, so it performs no per-manager ownership check.
func (s *houseCostServiceImpl) GenerateMonthlyCostsFromPrevious(ctx context.Context, previousPeriod, period string) (GenerateMonthlyCostsResult, error) {
	var result GenerateMonthlyCostsResult

	previousCosts, err := s.costRepo.ListByPeriod(ctx, previousPeriod)
	if err != nil {
		return result, fmt.Errorf("list previous period costs: %w", err)
	}

	for i := range previousCosts {
		previous := &previousCosts[i]

		if !hasFixedCosts(previous) {
			result.Skipped++
			continue
		}

		if existing, err := s.costRepo.GetByHouseAndPeriod(ctx, previous.HouseID, period); err == nil && existing != nil {
			result.Skipped++
			continue
		}

		newCost := s.buildNextCost(previous.HouseID, period, previous)
		if err := s.costRepo.Create(ctx, newCost); err != nil {
			if model.IsHouseCostAlreadyExists(err) {
				result.Skipped++
				continue
			}
			result.Failed++
			log.Printf("[HouseCostCron] failed to create cost for house %s period %s: %v", previous.HouseID, period, err)
			continue
		}

		result.Created++

		if s.eventBus != nil {
			s.eventBus.Publish(revenue.EventHouseCostChanged, revenue.RevenueSummaryPayload{
				HouseID: previous.HouseID,
				Period:  period,
			})
		}
	}

	return result, nil
}
