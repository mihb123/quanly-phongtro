package house

import (
	"context"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

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
		log.Printf("[HouseCostCron] Failed to schedule monthly cost generation: %v", err)
	}

	return svc
}

// Start begins the cron scheduler in a background goroutine.
func (s *HouseCostCronService) Start() {
	s.cron.Start()
	log.Println("[HouseCostCron] Started monthly operating cost scheduler (00:05 on the 1st).")
}

// Stop gracefully shuts down the cron scheduler.
func (s *HouseCostCronService) Stop() {
	s.cron.Stop()
	log.Println("[HouseCostCron] Stopped monthly operating cost scheduler.")
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

	log.Printf("[HouseCostCron] Generating operating costs for %s from %s...", period, previousPeriod)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := s.service.GenerateMonthlyCostsFromPrevious(ctx, previousPeriod, period)
	if err != nil {
		log.Printf("[HouseCostCron] Generation for %s failed: %v", period, err)
		return
	}

	log.Printf("[HouseCostCron] Generation for %s done: %d created, %d skipped, %d failed.",
		period, result.Created, result.Skipped, result.Failed)
}

// formatPeriod renders a time as the yyyy-mm period key used by house_costs.
func formatPeriod(t time.Time) string {
	return t.Format("2006-01")
}
