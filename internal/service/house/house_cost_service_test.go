package house_test

import (
	"context"
	"errors"
	"testing"
	"time"

	housesvc "github.com/mihb123/quanly-phongtro/internal/service/house"
	revenuesvc "github.com/mihb123/quanly-phongtro/internal/service/revenue"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"go.uber.org/mock/gomock"
)

func TestHouseCostService_CreateMonthlyCost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
	mockCostRepo := mock_model.NewMockHouseCostRepository(ctrl)
	mockSummaryRepo := mock_model.NewMockRevenueSummaryRepository(ctrl)

	costService := housesvc.NewHouseCostService(mockCostRepo, mockHouseRepo, nil, mockSummaryRepo)

	ctx := context.Background()

	// Forbidden case
	mockHouseRepo.EXPECT().
		IsHouseOwnedBy(gomock.Any(), "house-1", "manager-1").
		Return(false, nil)

	_, err := costService.CreateMonthlyCost(ctx, "manager-1", "house-1", "2023-10")
	if err == nil || err.Error() != "forbidden: you do not own this house" {
		t.Errorf("expected forbidden error, got %v", err)
	}

	// Success case
	mockHouseRepo.EXPECT().
		IsHouseOwnedBy(gomock.Any(), "house-1", "manager-1").
		Return(true, nil)

	mockCostRepo.EXPECT().
		GetByHouseAndPeriod(gomock.Any(), "house-1", "2023-10").
		Return(nil, errors.New("house cost not found"))

	mockCostRepo.EXPECT().
		GetLatestByHouseID(gomock.Any(), "house-1").
		Return(&model.HouseCost{
			Rent:     2000,
			Wifi:     100,
			Cleaning: 50,
		}, nil)

	mockCostRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cost *model.HouseCost) error {
			if cost.Rent != 2000 || cost.TotalCost != 2150 {
				t.Errorf("unexpected cost calculation: %+v", cost)
			}
			return nil
		})

	newCost, err := costService.CreateMonthlyCost(ctx, "manager-1", "house-1", "2023-10")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if newCost.Period != "2023-10" {
		t.Errorf("unexpected period")
	}

	// Existing cost
	mockHouseRepo.EXPECT().
		IsHouseOwnedBy(gomock.Any(), "house-1", "manager-1").
		Return(true, nil)
	mockCostRepo.EXPECT().
		GetByHouseAndPeriod(gomock.Any(), "house-1", "2023-10").
		Return(&model.HouseCost{ID: "cost-1"}, nil)

	_, err = costService.CreateMonthlyCost(ctx, "manager-1", "house-1", "2023-10")
	if err == nil || err.Error() != "house cost record already exists for this period" {
		t.Errorf("expected existing cost error, got %v", err)
	}

	// Create error
	mockHouseRepo.EXPECT().
		IsHouseOwnedBy(gomock.Any(), "house-1", "manager-1").
		Return(true, nil)
	mockCostRepo.EXPECT().
		GetByHouseAndPeriod(gomock.Any(), "house-1", "2023-10").
		Return(nil, errors.New("house cost not found"))
	mockCostRepo.EXPECT().
		GetLatestByHouseID(gomock.Any(), "house-1").
		Return(nil, errors.New("house cost not found"))
	mockCostRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(errors.New("db error"))

	_, err = costService.CreateMonthlyCost(ctx, "manager-1", "house-1", "2023-10")
	if err == nil || err.Error() != "db error" {
		t.Errorf("expected db error, got %v", err)
	}
}

// TestHouseCostService_CreateMonthlyCostPublishesEvent verifies recalculation events are emitted.
func TestHouseCostService_CreateMonthlyCostPublishesEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
	mockCostRepo := mock_model.NewMockHouseCostRepository(ctrl)
	mockSummaryRepo := mock_model.NewMockRevenueSummaryRepository(ctrl)
	eventBus := revenuesvc.NewEventBus()
	received := make(chan interface{}, 1)
	eventBus.Subscribe(revenuesvc.EventHouseCostChanged, func(payload interface{}) {
		received <- payload
	})

	costService := housesvc.NewHouseCostService(mockCostRepo, mockHouseRepo, eventBus, mockSummaryRepo)
	ctx := context.Background()

	mockHouseRepo.EXPECT().IsHouseOwnedBy(gomock.Any(), "house-1", "manager-1").Return(true, nil)
	mockCostRepo.EXPECT().GetByHouseAndPeriod(gomock.Any(), "house-1", "2023-10").Return(nil, errors.New("house cost not found"))
	mockCostRepo.EXPECT().GetLatestByHouseID(gomock.Any(), "house-1").Return(nil, errors.New("house cost not found"))
	mockCostRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

	_, err := costService.CreateMonthlyCost(ctx, "manager-1", "house-1", "2023-10")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	select {
	case value := <-received:
		payload, ok := value.(revenuesvc.RevenueSummaryPayload)
		if !ok || payload.HouseID != "house-1" || payload.Period != "2023-10" {
			t.Fatalf("unexpected payload: %+v", value)
		}
	case <-time.After(time.Second):
		t.Fatal("expected house cost changed event")
	}
}

// TestHouseCostService_GetMonthlyCost covers ownership, lookup, and JSON-safe extra costs.
func TestHouseCostService_GetMonthlyCost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
	mockCostRepo := mock_model.NewMockHouseCostRepository(ctrl)
	mockSummaryRepo := mock_model.NewMockRevenueSummaryRepository(ctrl)

	costService := housesvc.NewHouseCostService(mockCostRepo, mockHouseRepo, nil, mockSummaryRepo)
	ctx := context.Background()

	mockHouseRepo.EXPECT().
		IsHouseOwnedBy(gomock.Any(), "house-1", "manager-1").
		Return(true, nil)
	mockCostRepo.EXPECT().
		GetByHouseAndPeriod(gomock.Any(), "house-1", "2023-10").
		Return(&model.HouseCost{ID: "cost-1", HouseID: "house-1", Period: "2023-10"}, nil)

	cost, err := costService.GetMonthlyCost(ctx, "manager-1", "house-1", "2023-10")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cost.ExtraCosts == nil {
		t.Errorf("expected empty extra costs slice")
	}

	mockHouseRepo.EXPECT().
		IsHouseOwnedBy(gomock.Any(), "house-2", "manager-1").
		Return(false, nil)

	_, err = costService.GetMonthlyCost(ctx, "manager-1", "house-2", "2023-10")
	if err == nil || err.Error() != "forbidden: you do not own this house" {
		t.Errorf("expected forbidden error, got %v", err)
	}

	mockHouseRepo.EXPECT().
		IsHouseOwnedBy(gomock.Any(), "house-3", "manager-1").
		Return(false, errors.New("db error"))

	_, err = costService.GetMonthlyCost(ctx, "manager-1", "house-3", "2023-10")
	if err == nil {
		t.Errorf("expected ownership error")
	}

	mockHouseRepo.EXPECT().
		IsHouseOwnedBy(gomock.Any(), "house-4", "manager-1").
		Return(true, nil)
	mockCostRepo.EXPECT().
		GetByHouseAndPeriod(gomock.Any(), "house-4", "2023-10").
		Return(nil, errors.New("db error"))

	_, err = costService.GetMonthlyCost(ctx, "manager-1", "house-4", "2023-10")
	if err == nil || err.Error() != "db error" {
		t.Errorf("expected db error, got %v", err)
	}
}

func TestHouseCostService_UpdateMonthlyCost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
	mockCostRepo := mock_model.NewMockHouseCostRepository(ctrl)
	mockSummaryRepo := mock_model.NewMockRevenueSummaryRepository(ctrl)

	costService := housesvc.NewHouseCostService(mockCostRepo, mockHouseRepo, nil, mockSummaryRepo)
	ctx := context.Background()

	input := housesvc.UpdateCostInput{
		ID:          "cost-1",
		HouseID:     "house-1",
		Period:      "2023-10",
		Rent:        3000,
		Electricity: 100,
	}

	// Success
	mockHouseRepo.EXPECT().
		IsHouseOwnedBy(gomock.Any(), "house-1", "manager-1").
		Return(true, nil)

	mockCostRepo.EXPECT().
		GetByHouseAndPeriod(gomock.Any(), "house-1", "2023-10").
		Return(&model.HouseCost{ID: "cost-1", HouseID: "house-1", Period: "2023-10"}, nil)

	mockCostRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cost *model.HouseCost) error {
			if cost.TotalCost != 3100 {
				t.Errorf("expected 3100 total cost, got %v", cost.TotalCost)
			}
			return nil
		})

	err := costService.UpdateMonthlyCost(ctx, "manager-1", input)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Invalid cost ID
	mockHouseRepo.EXPECT().
		IsHouseOwnedBy(gomock.Any(), "house-1", "manager-1").
		Return(true, nil)
	mockCostRepo.EXPECT().
		GetByHouseAndPeriod(gomock.Any(), "house-1", "2023-10").
		Return(&model.HouseCost{ID: "different", HouseID: "house-1", Period: "2023-10"}, nil)

	err = costService.UpdateMonthlyCost(ctx, "manager-1", input)
	if err == nil || err.Error() != "invalid cost ID" {
		t.Errorf("expected invalid cost ID, got %v", err)
	}

	// Repository update error
	mockHouseRepo.EXPECT().
		IsHouseOwnedBy(gomock.Any(), "house-1", "manager-1").
		Return(true, nil)
	mockCostRepo.EXPECT().
		GetByHouseAndPeriod(gomock.Any(), "house-1", "2023-10").
		Return(&model.HouseCost{ID: "cost-1", HouseID: "house-1", Period: "2023-10"}, nil)
	mockCostRepo.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		Return(errors.New("db error"))

	err = costService.UpdateMonthlyCost(ctx, "manager-1", input)
	if err == nil || err.Error() != "db error" {
		t.Errorf("expected db error, got %v", err)
	}

	// Ownership check error
	mockHouseRepo.EXPECT().
		IsHouseOwnedBy(gomock.Any(), "house-1", "manager-1").
		Return(false, errors.New("ownership error"))

	err = costService.UpdateMonthlyCost(ctx, "manager-1", input)
	if err == nil {
		t.Errorf("expected ownership error")
	}

	// Cost lookup error
	mockHouseRepo.EXPECT().
		IsHouseOwnedBy(gomock.Any(), "house-1", "manager-1").
		Return(true, nil)
	mockCostRepo.EXPECT().
		GetByHouseAndPeriod(gomock.Any(), "house-1", "2023-10").
		Return(nil, errors.New("lookup error"))

	err = costService.UpdateMonthlyCost(ctx, "manager-1", input)
	if err == nil || err.Error() != "lookup error" {
		t.Errorf("expected lookup error, got %v", err)
	}
}

// TestHouseCostService_UpdateMonthlyCostPublishesEvent verifies update recalculation events.
func TestHouseCostService_UpdateMonthlyCostPublishesEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
	mockCostRepo := mock_model.NewMockHouseCostRepository(ctrl)
	mockSummaryRepo := mock_model.NewMockRevenueSummaryRepository(ctrl)
	eventBus := revenuesvc.NewEventBus()
	received := make(chan interface{}, 1)
	eventBus.Subscribe(revenuesvc.EventHouseCostChanged, func(payload interface{}) {
		received <- payload
	})

	costService := housesvc.NewHouseCostService(mockCostRepo, mockHouseRepo, eventBus, mockSummaryRepo)
	ctx := context.Background()
	input := housesvc.UpdateCostInput{
		ID:         "cost-1",
		HouseID:    "house-1",
		Period:     "2023-10",
		ExtraCosts: []model.ExtraCost{{Name: "repair", Amount: 25}},
	}

	mockHouseRepo.EXPECT().IsHouseOwnedBy(gomock.Any(), "house-1", "manager-1").Return(true, nil)
	mockCostRepo.EXPECT().GetByHouseAndPeriod(gomock.Any(), "house-1", "2023-10").Return(&model.HouseCost{ID: "cost-1", HouseID: "house-1", Period: "2023-10"}, nil)
	mockCostRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)

	err := costService.UpdateMonthlyCost(ctx, "manager-1", input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	select {
	case value := <-received:
		payload, ok := value.(revenuesvc.RevenueSummaryPayload)
		if !ok || payload.HouseID != "house-1" || payload.Period != "2023-10" {
			t.Fatalf("unexpected payload: %+v", value)
		}
	case <-time.After(time.Second):
		t.Fatal("expected house cost changed event")
	}
}

func TestHouseCostService_GetRevenueSummaries(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
	mockCostRepo := mock_model.NewMockHouseCostRepository(ctrl)
	mockSummaryRepo := mock_model.NewMockRevenueSummaryRepository(ctrl)

	costService := housesvc.NewHouseCostService(mockCostRepo, mockHouseRepo, nil, mockSummaryRepo)
	ctx := context.Background()

	// User requests house-1 and house-2
	mockHouseRepo.EXPECT().IsHouseOwnedBy(gomock.Any(), "house-1", "manager-1").Return(true, nil)
	mockHouseRepo.EXPECT().IsHouseOwnedBy(gomock.Any(), "house-2", "manager-1").Return(false, nil) // not owned

	mockSummaryRepo.EXPECT().
		ListByHouseIDs(gomock.Any(), []string{"house-1"}, "2023-10").
		Return([]model.HouseRevenueSummary{
			{HouseID: "house-1", Profit: 5000},
		}, nil)

	summaries, err := costService.GetRevenueSummaries(ctx, "manager-1", []string{"house-1", "house-2"}, "2023-10")
	if err != nil {
		t.Errorf("error not expected: %v", err)
	}
	if len(summaries) != 1 {
		t.Errorf("expected only 1 summary due to ownership filter")
	}

	mockHouseRepo.EXPECT().IsHouseOwnedBy(gomock.Any(), "house-3", "manager-1").Return(false, errors.New("db error"))
	mockHouseRepo.EXPECT().IsHouseOwnedBy(gomock.Any(), "house-4", "manager-1").Return(false, nil)

	summaries, err = costService.GetRevenueSummaries(ctx, "manager-1", []string{"house-3", "house-4"}, "2023-10")
	if err != nil {
		t.Errorf("error not expected: %v", err)
	}
	if len(summaries) != 0 {
		t.Errorf("expected no summaries")
	}

	mockHouseRepo.EXPECT().IsHouseOwnedBy(gomock.Any(), "house-1", "manager-1").Return(true, nil)
	mockSummaryRepo.EXPECT().
		ListByHouseIDs(gomock.Any(), []string{"house-1"}, "2023-10").
		Return(nil, errors.New("db error"))

	_, err = costService.GetRevenueSummaries(ctx, "manager-1", []string{"house-1"}, "2023-10")
	if err == nil || err.Error() != "db error" {
		t.Errorf("expected db error, got %v", err)
	}
}
