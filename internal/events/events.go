package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ai-crypto-onramp/exchange-connectors/internal/metrics"
	"github.com/ai-crypto-onramp/exchange-connectors/internal/venue"
	"github.com/shopspring/decimal"
)

type FillEvent struct {
	VenueOrderID string          `json:"venue_order_id"`
	InternalID   string          `json:"internal_order_id"`
	Venue        string          `json:"venue"`
	Pair         string          `json:"pair"`
	Price        decimal.Decimal `json:"price"`
	Size         decimal.Decimal `json:"size"`
	Fee          decimal.Decimal `json:"fee"`
	FeeAsset     string          `json:"fee_asset"`
	TradeID      string          `json:"trade_id"`
	Timestamp    time.Time       `json:"timestamp"`
	EmittedAt    time.Time       `json:"emitted_at"`
}

type BalanceEvent struct {
	Venue      string          `json:"venue"`
	Asset      string          `json:"asset"`
	Free       decimal.Decimal `json:"free"`
	Locked     decimal.Decimal `json:"locked"`
	Timestamp  time.Time       `json:"timestamp"`
	EmittedAt  time.Time       `json:"emitted_at"`
}

type Publisher interface {
	Publish(ctx context.Context, subject string, data []byte) error
}

type NATSPublisher struct {
	conn NATSConn
}

type NATSConn interface {
	Publish(subject string, data []byte) error
	Close()
}

func NewNATSPublisher(conn NATSConn) *NATSPublisher {
	return &NATSPublisher{conn: conn}
}

func (p *NATSPublisher) Publish(ctx context.Context, subject string, data []byte) error {
	if p.conn == nil {
		return errors.New("events: nats not connected")
	}
	if err := p.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("events: publish: %w", err)
	}
	return nil
}

type Bus struct {
	mu        sync.Mutex
	publisher Publisher
	subject   string
	maxRetries int
	deadLetter func(FillEvent, error)
}

func NewBus(publisher Publisher, subject string) *Bus {
	return &Bus{
		publisher:  publisher,
		subject:     subject,
		maxRetries: 3,
		deadLetter: func(FillEvent, error) {},
	}
}

func (b *Bus) SetDeadLetterHandler(h func(FillEvent, error)) {
	b.mu.Lock()
	b.deadLetter = h
	b.mu.Unlock()
}

func (b *Bus) PublishFill(ctx context.Context, ev FillEvent) error {
	if ev.VenueOrderID == "" {
		return errors.New("events: venue_order_id required")
	}
	if ev.InternalID == "" {
		ev.InternalID = ev.VenueOrderID
	}
	if ev.EmittedAt.IsZero() {
		ev.EmittedAt = time.Now().UTC()
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	var lastErr error
	for attempt := 0; attempt <= b.maxRetries; attempt++ {
		err := b.publisher.Publish(ctx, b.subject+".fills", data)
		if err == nil {
			metrics.EventsPublishedTotal.WithLabelValues(ev.Venue, "fill", "ok").Inc()
			latency := time.Since(ev.Timestamp).Seconds()
			if latency < 0 {
				latency = 0
			}
			metrics.FillLatencySeconds.WithLabelValues(ev.Venue).Observe(latency)
			return nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 100 * time.Millisecond):
		}
	}
	metrics.EventsPublishedTotal.WithLabelValues(ev.Venue, "fill", "failed").Inc()
	b.mu.Lock()
	b.deadLetter(ev, lastErr)
	b.mu.Unlock()
	return lastErr
}

func (b *Bus) PublishBalance(ctx context.Context, ev BalanceEvent) error {
	if ev.Venue == "" || ev.Asset == "" {
		return errors.New("events: venue and asset required")
	}
	if ev.EmittedAt.IsZero() {
		ev.EmittedAt = time.Now().UTC()
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	var lastErr error
	for attempt := 0; attempt <= b.maxRetries; attempt++ {
		err := b.publisher.Publish(ctx, b.subject+".balances", data)
		if err == nil {
			metrics.EventsPublishedTotal.WithLabelValues(ev.Venue, "balance", "ok").Inc()
			lag := time.Since(ev.Timestamp).Seconds()
			if lag < 0 {
				lag = 0
			}
			metrics.BalanceSyncLagSeconds.WithLabelValues(ev.Venue).Observe(lag)
			return nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 100 * time.Millisecond):
		}
	}
	metrics.EventsPublishedTotal.WithLabelValues(ev.Venue, "balance", "failed").Inc()
	return lastErr
}

func FillFromVenue(venueName, internalID, pair string, f venue.Fill) FillEvent {
	return FillEvent{
		VenueOrderID: f.VenueOrderID,
		InternalID:   internalID,
		Venue:        venueName,
		Pair:         pair,
		Price:        f.Price,
		Size:         f.Quantity,
		Fee:          f.Fee,
		FeeAsset:     f.FeeAsset,
		TradeID:      f.TradeID,
		Timestamp:    f.Timestamp,
	}
}

func BalanceFromVenue(venueName string, b venue.Balance) BalanceEvent {
	return BalanceEvent{
		Venue:     venueName,
		Asset:     b.Asset,
		Free:      b.Free,
		Locked:    b.Locked,
		Timestamp: time.Now().UTC(),
	}
}