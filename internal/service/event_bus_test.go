package service

import (
	"testing"
	"time"
)

// TestEventBus_PublishDispatchesSubscribedHandlers verifies asynchronous event delivery.
func TestEventBus_PublishDispatchesSubscribedHandlers(t *testing.T) {
	bus := NewEventBus()
	received := make(chan interface{}, 1)
	payload := RevenueSummaryPayload{HouseID: "house-1", Period: "2023-10"}

	bus.Subscribe(EventHouseCostChanged, func(value interface{}) {
		received <- value
	})
	bus.Publish(EventHouseCostChanged, payload)

	select {
	case value := <-received:
		got, ok := value.(RevenueSummaryPayload)
		if !ok {
			t.Fatalf("expected RevenueSummaryPayload, got %T", value)
		}
		if got.HouseID != payload.HouseID || got.Period != payload.Period {
			t.Errorf("unexpected payload: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("expected subscribed handler to receive payload")
	}
}

// TestEventBus_PublishWithoutSubscribers verifies unused events do not block.
func TestEventBus_PublishWithoutSubscribers(t *testing.T) {
	bus := NewEventBus()
	bus.Publish(EventInvoiceChanged, RevenueSummaryPayload{HouseID: "house-1"})
}
