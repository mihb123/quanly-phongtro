package house

import (
	"context"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/logging"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

const houseCostCronTask = "house_cost_cron"

// MonthlyCostCronSchedule fires at 00:05 on the 1st of every month (ICT).
var MonthlyCostCronSchedule = "5 0 1 * *"

// ictLocation is the business timezone used to derive the billing period, matching
// what the frontend computes from the user's local clock.
var ictLocation = time.FixedZone("ICT", 7*60*60)

// HouseCostCronService creates the new month's operating-cost records automatically,
// so managers no longer have to press "Tạo chi phí tháng mới" for every house.
type HouseCostCronService struct {
	cron    *cron.Cron
	service HouseCostService
}

// NewHouseCostCronService constructs the cron service and schedules the monthly job.
func NewHouseCostCronService(service HouseCostService) *HouseCostCronService {
	c := cron.New(cron.WithLocation(ictLocation))
	svc := &HouseCostCronService{cron: c, service: service}

	if _, err := c.AddFunc(MonthlyCostCronSchedule, func() {
		svc.generateForCurrentMonth()
	}); err != nil {
		zap.L().Error("schedule monthly cost generation failed", zap.String("task", houseCostCronTask), zap.Error(err))
	}

	return svc
}

// Start begins the cron scheduler in a background goroutine.
func (s *HouseCostCronService) Start() {
	s.cron.Start()
	zap.L().Info("cron started", zap.String("task", houseCostCronTask), zap.String("schedule", MonthlyCostCronSchedule))
}

// Stop gracefully shuts down the cron scheduler.
func (s *HouseCostCronService) Stop() {
	s.cron.Stop()
	zap.L().Info("cron stopped", zap.String("task", houseCostCronTask))
}

// RunNow immediately generates the current month's records, bypassing the schedule.
func (s *HouseCostCronService) RunNow() {
	s.generateForCurrentMonth()
}

func (s *HouseCostCronService) generateForCurrentMonth() {
	// Normalise to the 1st so AddDate never overflows a short month (e.g. 31 Mar -> 3 Mar).
	now := time.Now().In(ictLocation)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, ictLocation)
	period := formatPeriod(monthStart)
	previousPeriod := formatPeriod(monthStart.AddDate(0, -1, 0))

	ctx, cancel := context.WithTimeout(logging.TaskContext(context.Background(), houseCostCronTask), 5*time.Minute)
	defer cancel()
	log := logging.FromCtx(ctx).With(zap.String("period", period), zap.String("previous_period", previousPeriod))
	log.Info("generating operating costs")

	result, err := s.service.GenerateMonthlyCostsFromPrevious(ctx, previousPeriod, period)
	if err != nil {
		logging.LogTaskError(ctx, "operating cost generation failed", err, zap.String("period", period))
		return
	}
	for _, houseErr := range result.Errors {
		logging.LogTaskError(ctx, "operating cost generation failed for house", houseErr, zap.String("period", period))
	}

	log.Info("operating cost generation done",
		zap.Int("created", result.Created), zap.Int("skipped", result.Skipped), zap.Int("failed", result.Failed))
}

// formatPeriod renders a time as the yyyy-mm period key used by house_costs.
func formatPeriod(t time.Time) string {
	return t.Format("2006-01")
}
