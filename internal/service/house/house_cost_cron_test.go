package house_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_service"
	housesvc "github.com/mihb123/quanly-phongtro/internal/service/house"
	"go.uber.org/mock/gomock"
)

// TestHouseCostCronService_RunNow verifies the job derives the current and previous
// periods and forwards them to the service.
func TestHouseCostCronService_RunNow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_service.NewMockHouseCostService(ctrl)

	var gotPrevious, gotPeriod string
	mockService.EXPECT().
		GenerateMonthlyCostsFromPrevious(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, previousPeriod, period string) (housesvc.GenerateMonthlyCostsResult, error) {
			gotPrevious, gotPeriod = previousPeriod, period
			return housesvc.GenerateMonthlyCostsResult{Created: 2}, nil
		})

	housesvc.NewHouseCostCronService(mockService).RunNow()

	if len(gotPeriod) != 7 || len(gotPrevious) != 7 {
		t.Fatalf("expected yyyy-mm periods, got %q and %q", gotPrevious, gotPeriod)
	}
	if gotPrevious >= gotPeriod {
		t.Errorf("previous period %q should precede %q", gotPrevious, gotPeriod)
	}
}

// TestHouseCostCronService_RunNowError ensures a failing run does not panic.
func TestHouseCostCronService_RunNowError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_service.NewMockHouseCostService(ctrl)
	mockService.EXPECT().
		GenerateMonthlyCostsFromPrevious(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(housesvc.GenerateMonthlyCostsResult{}, errors.New("db down"))

	cron := housesvc.NewHouseCostCronService(mockService)
	cron.Start()
	cron.RunNow()
	cron.Stop()
}
