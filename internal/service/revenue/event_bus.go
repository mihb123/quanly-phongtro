package revenue

import "sync"

type EventType string

const (
	EventInvoiceChanged   EventType = "EventInvoiceChanged"
	EventHouseCostChanged EventType = "EventHouseCostChanged"
)

type RevenueSummaryPayload struct {
	HouseID string
	Period  string
}

type EventBus interface {
	Subscribe(eventType EventType, handler func(payload interface{}))
	Publish(eventType EventType, payload interface{})
}

type EventBusImpl struct {
	handlers map[EventType][]func(payload interface{})
	mu       sync.RWMutex
}

func NewEventBus() *EventBusImpl {
	return &EventBusImpl{
		handlers: make(map[EventType][]func(payload interface{})),
	}
}

func (b *EventBusImpl) Subscribe(eventType EventType, handler func(payload interface{})) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

func (b *EventBusImpl) Publish(eventType EventType, payload interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if handlers, ok := b.handlers[eventType]; ok {
		for _, handler := range handlers {
			// Execute handler asynchronously so Publish doesn't block
			go handler(payload)
		}
	}
}
