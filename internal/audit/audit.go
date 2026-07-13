package audit

import (
	"sync"
	"time"
)

type EventType string

const (
	EventOrderPlaced   EventType = "order_placed"
	EventOrderCanceled EventType = "order_canceled"
	EventFill          EventType = "fill"
)

type Event struct {
	Type        EventType   `json:"type"`
	Venue       string      `json:"venue"`
	Timestamp   time.Time   `json:"timestamp"`
	OrderID     string      `json:"order_id,omitempty"`
	ClientID    string      `json:"client_id,omitempty"`
	Pair        string      `json:"pair,omitempty"`
	Side        string      `json:"side,omitempty"`
	OrderType   string      `json:"order_type,omitempty"`
	Quantity    float64     `json:"quantity,omitempty"`
	Price       float64     `json:"price,omitempty"`
	Fill        *FillDetail `json:"fill,omitempty"`
}

type FillDetail struct {
	TradeID  string    `json:"trade_id"`
	Price    float64   `json:"price"`
	Quantity float64   `json:"quantity"`
	Fee      float64   `json:"fee"`
	FeeAsset string    `json:"fee_asset"`
	TS       time.Time `json:"ts"`
}

type Sink interface {
	Emit(event Event)
}

type InMemorySink struct {
	mu      sync.Mutex
	events  []Event
	onDrop  int
}

func NewInMemorySink() *InMemorySink {
	return &InMemorySink{}
}

func (s *InMemorySink) Emit(event Event) {
	event.Timestamp = time.Now().UTC()
	s.mu.Lock()
	s.events = append(s.events, event)
	s.mu.Unlock()
}

func (s *InMemorySink) Events() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Event, len(s.events))
	copy(out, s.events)
	return out
}

func (s *InMemorySink) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.events)
}