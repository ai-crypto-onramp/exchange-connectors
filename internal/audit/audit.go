package audit

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

type EventType string

const (
	EventOrderPlaced     EventType = "order_placed"
	EventOrderCanceled   EventType = "order_canceled"
	EventOrderUpdated    EventType = "order_updated"
	EventFill            EventType = "fill"
	EventBalanceSnapshot EventType = "balance_snapshot"
	EventCredRotation    EventType = "credential_rotation"
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
	Quantity    decimal.Decimal `json:"quantity,omitempty"`
	Price       decimal.Decimal `json:"price,omitempty"`
	Fill        *FillDetail `json:"fill,omitempty"`
	Balance     *BalanceDetail `json:"balance,omitempty"`
	TraceID     string      `json:"trace_id,omitempty"`
}

type FillDetail struct {
	TradeID  string          `json:"trade_id"`
	Price    decimal.Decimal `json:"price"`
	Quantity decimal.Decimal `json:"quantity"`
	Fee      decimal.Decimal `json:"fee"`
	FeeAsset string          `json:"fee_asset"`
	TS       time.Time       `json:"ts"`
}

type BalanceDetail struct {
	Asset  string          `json:"asset"`
	Free   decimal.Decimal `json:"free"`
	Locked decimal.Decimal `json:"locked"`
}

type Sink interface {
	Emit(event Event)
}

type InMemorySink struct {
	mu     sync.Mutex
	events []Event
}

func NewInMemorySink() *InMemorySink {
	return &InMemorySink{}
}

func (s *InMemorySink) Emit(event Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
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