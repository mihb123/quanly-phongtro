package service_test

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/service"
)

func TestHouseCostService_CreateMonthlyCost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
	mockCostRepo := mock_model.NewMockHouseCostRepository(ctrl)
	mockSummaryRepo := mock_model.NewMockRevenueSummaryRepository(ctrl)

	costService := service.NewHouseCostService(mockCostRepo, mockHouseRepo, nil, mockSummaryRepo)

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
}

func TestHouseCostService_UpdateMonthlyCost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
	mockCostRepo := mock_model.NewMockHouseCostRepository(ctrl)
	mockSummaryRepo := mock_model.NewMockRevenueSummaryRepository(ctrl)

	costService := service.NewHouseCostService(mockCostRepo, mockHouseRepo, nil, mockSummaryRepo)
	ctx := context.Background()

	input := service.UpdateCostInput{
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
}

func TestHouseCostService_GetRevenueSummaries(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHouseRepo := mock_model.NewMockHouseRepository(ctrl)
	mockCostRepo := mock_model.NewMockHouseCostRepository(ctrl)
	mockSummaryRepo := mock_model.NewMockRevenueSummaryRepository(ctrl)

	costService := service.NewHouseCostService(mockCostRepo, mockHouseRepo, nil, mockSummaryRepo)
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
}
