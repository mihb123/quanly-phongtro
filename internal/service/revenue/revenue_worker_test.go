package revenue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"go.uber.org/mock/gomock"
)

func TestRevenueWorker_ProcessEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSummaryRepo := mock_model.NewMockRevenueSummaryRepository(ctrl)
	mockCostRepo := mock_model.NewMockHouseCostRepository(ctrl)

	worker := NewRevenueWorker(mockSummaryRepo, mockCostRepo)

	event := RevenueSummaryEvent{
		HouseID: "house-1",
		Period:  "2023-10",
	}

	// CalculateRevenue -> 15000
	mockSummaryRepo.EXPECT().
		CalculateRevenue(gomock.Any(), "house-1", "2023-10").
		Return(15000.0, nil)

	// GetByHouseAndPeriod -> 5000
	mockCostRepo.EXPECT().
		GetByHouseAndPeriod(gomock.Any(), "house-1", "2023-10").
		Return(&model.HouseCost{TotalCost: 5000.0}, nil)

	// Upsert -> Profit = 10000
	mockSummaryRepo.EXPECT().Upsert(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, summary *model.HouseRevenueSummary) error {
		if summary.HouseID != "house-1" || summary.TotalRevenue != 15000 || summary.TotalCost != 5000 || summary.Profit != 10000 {
			return errors.New("mismatch")
		}
		return nil
	})

	// Call synchronously
	worker.processEvent(event)
}

func TestRevenueWorker_ProcessEvent_NoCostAndNoRevenue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSummaryRepo := mock_model.NewMockRevenueSummaryRepository(ctrl)
	mockCostRepo := mock_model.NewMockHouseCostRepository(ctrl)

	worker := NewRevenueWorker(mockSummaryRepo, mockCostRepo)

	event := RevenueSummaryEvent{
		HouseID: "house-2",
		Period:  "2023-11",
	}

	// CalculateRevenue -> returns error
	mockSummaryRepo.EXPECT().
		CalculateRevenue(gomock.Any(), "house-2", "2023-11").
		Return(0.0, errors.New("some error"))

	// GetByHouseAndPeriod -> returns house cost not found
	mockCostRepo.EXPECT().
		GetByHouseAndPeriod(gomock.Any(), "house-2", "2023-11").
		Return(nil, errors.New("house cost not found"))

	// Upsert -> Profit = 0
	mockSummaryRepo.EXPECT().Upsert(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, summary *model.HouseRevenueSummary) error {
		if summary.HouseID != "house-2" || summary.TotalRevenue != 0 || summary.TotalCost != 0 || summary.Profit != 0 {
			return errors.New("mismatch")
		}
		return nil
	})

	// Call synchronously
	worker.processEvent(event)
}

// TestRevenueWorker_StartStopAndEnqueue covers the async worker loop.
func TestRevenueWorker_StartStopAndEnqueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSummaryRepo := mock_model.NewMockRevenueSummaryRepository(ctrl)
	mockCostRepo := mock_model.NewMockHouseCostRepository(ctrl)
	worker := NewRevenueWorker(mockSummaryRepo, mockCostRepo)
	done := make(chan struct{}, 1)

	mockSummaryRepo.EXPECT().
		CalculateRevenue(gomock.Any(), "house-1", "2023-10").
		Return(1000.0, nil)
	mockCostRepo.EXPECT().
		GetByHouseAndPeriod(gomock.Any(), "house-1", "2023-10").
		Return(&model.HouseCost{TotalCost: 250.0}, nil)
	mockSummaryRepo.EXPECT().
		Upsert(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, summary *model.HouseRevenueSummary) error {
			if summary.Profit != 750 {
				t.Errorf("expected profit 750, got %v", summary.Profit)
			}
			done <- struct{}{}
			return nil
		})

	worker.Start()
	worker.Enqueue("house-1", "2023-10")

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected worker to process event")
	}

	worker.Stop()
}

// TestRevenueWorker_EnqueueDropsWhenFull covers the non-blocking full-channel branch.
func TestRevenueWorker_EnqueueDropsWhenFull(t *testing.T) {
	worker := NewRevenueWorker(nil, nil)
	for i := 0; i < cap(worker.eventChannel); i++ {
		worker.Enqueue("house-1", "2023-10")
	}

	worker.Enqueue("house-1", "2023-10")
}
