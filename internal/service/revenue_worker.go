package service

import (
	"context"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/mihb123/quanly-phongtro/internal/model"
)

type RevenueSummaryEvent struct {
	HouseID string
	Period  string
}

type RevenueWorker struct {
	eventChannel chan RevenueSummaryEvent
	summaryRepo  model.RevenueSummaryRepository
	costRepo     model.HouseCostRepository
	wg           sync.WaitGroup
	quit         chan struct{}
}

func NewRevenueWorker(summaryRepo model.RevenueSummaryRepository, costRepo model.HouseCostRepository) *RevenueWorker {
	return &RevenueWorker{
		eventChannel: make(chan RevenueSummaryEvent, 1000), // Buffer to avoid blocking
		summaryRepo:  summaryRepo,
		costRepo:     costRepo,
		quit:         make(chan struct{}),
	}
}

// Start spawns a background goroutine that listens for events
func (w *RevenueWorker) Start() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		for {
			select {
			case event := <-w.eventChannel:
				w.processEvent(event)
			case <-w.quit:
				return
			}
		}
	}()
}

// Stop gracefully shuts down the worker
func (w *RevenueWorker) Stop() {
	close(w.quit)
	w.wg.Wait()
}

// Enqueue sends a recalculation event (non-blocking if buffer isn't full)
func (w *RevenueWorker) Enqueue(houseID, period string) {
	select {
	case w.eventChannel <- RevenueSummaryEvent{HouseID: houseID, Period: period}:
		// Successfully enqueued
	default:
		// Channel is full, log error or handle appropriately.
		// For our scale, 1000 buffer is plenty.
		log.Printf("warning: revenue worker event channel is full, dropped event for house %s period %s", houseID, period)
	}
}

// processEvent recalculates and upserts the summary
func (w *RevenueWorker) processEvent(event RevenueSummaryEvent) {
	ctx := context.Background() // Background context since it's an async job

	// 1. Get total revenue from paid invoices
	totalRevenue, err := w.summaryRepo.CalculateRevenue(ctx, event.HouseID, event.Period)
	if err != nil {
		log.Printf("error calculating revenue for house %s period %s: %v", event.HouseID, event.Period, err)
		// We proceed, but revenue might be 0 or old depending on logic
	}

	// 2. Get total cost from house_costs
	var totalCost float64 = 0
	cost, err := w.costRepo.GetByHouseAndPeriod(ctx, event.HouseID, event.Period)
	if err == nil && cost != nil {
		totalCost = cost.TotalCost
	} else if err != nil && err.Error() != "house cost not found" {
		log.Printf("error fetching house cost for house %s period %s: %v", event.HouseID, event.Period, err)
	}

	// 3. Calculate profit
	profit := totalRevenue - totalCost

	// 4. Upsert summary
	summary := &model.HouseRevenueSummary{
		ID:           uuid.New().String(),
		HouseID:      event.HouseID,
		Period:       event.Period,
		TotalRevenue: totalRevenue,
		TotalCost:    totalCost,
		Profit:       profit,
	}

	err = w.summaryRepo.Upsert(ctx, summary)
	if err != nil {
		log.Printf("error upserting revenue summary for house %s period %s: %v", event.HouseID, event.Period, err)
	}
}
