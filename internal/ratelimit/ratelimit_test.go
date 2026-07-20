package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestWeightedLimiterAllow(t *testing.T) {
	l := NewWeightedLimiter("binance", 1200)
	for i := 0; i < 10; i++ {
		if err := l.Wait(context.Background(), 1); err != nil {
			t.Fatalf("wait %d: %v", i, err)
		}
	}
	if l.Used() != 10 {
		t.Fatalf("used: %d", l.Used())
	}
}

func TestWeightedLimiterExhausted(t *testing.T) {
	l := NewWeightedLimiter("binance", 5)
	for i := 0; i < 5; i++ {
		_ = l.Wait(context.Background(), 1)
	}
	if err := l.Wait(context.Background(), 1); err != ErrBudgetExhausted {
		t.Fatalf("expected exhausted, got %v", err)
	}
}

func TestWeightedLimiterReplenish(t *testing.T) {
	l := NewWeightedLimiter("binance", 5)
	l.refill = 50 * time.Millisecond
	for i := 0; i < 5; i++ {
		_ = l.Wait(context.Background(), 1)
	}
	time.Sleep(60 * time.Millisecond)
	if err := l.Wait(context.Background(), 1); err != nil {
		t.Fatalf("expected replenish, got %v", err)
	}
}

func TestWeightedLimiterUpdateUsed(t *testing.T) {
	l := NewWeightedLimiter("binance", 1200)
	l.UpdateUsed("binance", 600)
	if l.Used() != 600 {
		t.Fatalf("used: %d", l.Used())
	}
	if l.Headroom() > 0.5+0.01 || l.Headroom() < 0.5-0.01 {
		t.Fatalf("headroom: %v", l.Headroom())
	}
}

func TestWeightedLimiterBackoff(t *testing.T) {
	l := NewWeightedLimiter("binance", 100)
	l.Backoff("binance", 100*time.Millisecond)
	if err := l.Wait(context.Background(), 1); err != ErrBudgetExhausted {
		t.Fatalf("expected exhausted during backoff, got %v", err)
	}
	time.Sleep(120 * time.Millisecond)
	if err := l.Wait(context.Background(), 1); err != nil {
		t.Fatalf("expected ok after backoff, got %v", err)
	}
}

func TestCounterLimiterAllow(t *testing.T) {
	l := NewCounterLimiter("kraken", 20)
	for i := 0; i < 20; i++ {
		if err := l.Wait(context.Background(), 1); err != nil {
			t.Fatalf("wait %d: %v", i, err)
		}
	}
	if err := l.Wait(context.Background(), 1); err != ErrBudgetExhausted {
		t.Fatalf("expected exhausted, got %v", err)
	}
}

func TestCounterLimiterDecay(t *testing.T) {
	l := NewCounterLimiter("kraken", 20)
	l.decay = 20 * time.Millisecond
	for i := 0; i < 10; i++ {
		_ = l.Wait(context.Background(), 1)
	}
	time.Sleep(120 * time.Millisecond)
	if err := l.Wait(context.Background(), 15); err != nil {
		t.Fatalf("expected ok after decay, got %v", err)
	}
}

func TestRPSLimiterAllow(t *testing.T) {
	l := NewRPSLimiter("otc", 5)
	for i := 0; i < 5; i++ {
		if err := l.Wait(context.Background(), 1); err != nil {
			t.Fatalf("wait %d: %v", i, err)
		}
	}
	if err := l.Wait(context.Background(), 1); err != ErrBudgetExhausted {
		t.Fatalf("expected exhausted, got %v", err)
	}
}

func TestRPSLimiterReset(t *testing.T) {
	l := NewRPSLimiter("otc", 5)
	l.window = 50 * time.Millisecond
	for i := 0; i < 5; i++ {
		_ = l.Wait(context.Background(), 1)
	}
	time.Sleep(60 * time.Millisecond)
	if err := l.Wait(context.Background(), 1); err != nil {
		t.Fatalf("expected reset, got %v", err)
	}
}

func TestRPSLimiterBackoff(t *testing.T) {
	l := NewRPSLimiter("otc", 100)
	l.Backoff("otc", 50*time.Millisecond)
	if err := l.Wait(context.Background(), 1); err != ErrBudgetExhausted {
		t.Fatalf("expected backoff, got %v", err)
	}
	time.Sleep(60 * time.Millisecond)
	if err := l.Wait(context.Background(), 1); err != nil {
		t.Fatalf("expected ok after backoff, got %v", err)
	}
}

func TestWeightedLimiterBudget(t *testing.T) {
	l := NewWeightedLimiter("binance", 1200)
	if l.Budget() != 1200 {
		t.Fatalf("budget: %d", l.Budget())
	}
	l.SetBudget(2400)
	if l.Budget() != 2400 {
		t.Fatalf("budget after set: %d", l.Budget())
	}
	if l.Headroom() != 1 {
		t.Fatalf("headroom full: %v", l.Headroom())
	}
}

func TestWeightedLimiterHeadroomZeroBudget(t *testing.T) {
	l := NewWeightedLimiter("binance", 0)
	if l.Headroom() != 0 {
		t.Fatalf("headroom zero budget: %v", l.Headroom())
	}
}

func TestCounterLimiterHeadroom(t *testing.T) {
	l := NewCounterLimiter("kraken", 10)
	if l.Headroom() != 1 {
		t.Fatalf("headroom full: %v", l.Headroom())
	}
	if l.Current() != 0 {
		t.Fatalf("current: %d", l.Current())
	}
	_ = l.Wait(context.Background(), 4)
	if l.Headroom() > 0.7 || l.Headroom() < 0.5 {
		t.Fatalf("headroom after 4: %v", l.Headroom())
	}
	if l.Current() != 4 {
		t.Fatalf("current after 4: %d", l.Current())
	}
}

func TestCounterLimiterHeadroomZeroBudget(t *testing.T) {
	l := NewCounterLimiter("kraken", 0)
	if l.Headroom() != 0 {
		t.Fatalf("headroom zero budget: %v", l.Headroom())
	}
	if err := l.Wait(context.Background(), 1); err != ErrBudgetExhausted {
		t.Fatalf("expected exhausted with zero budget, got %v", err)
	}
	if l.Current() != 0 {
		t.Fatalf("current should remain 0 when budget is 0: %d", l.Current())
	}
}

func TestCounterLimiterBackoff(t *testing.T) {
	l := NewCounterLimiter("kraken", 10)
	l.Backoff("kraken", 50*time.Millisecond)
	if err := l.Wait(context.Background(), 1); err != ErrBudgetExhausted {
		t.Fatalf("expected backoff, got %v", err)
	}
	time.Sleep(60 * time.Millisecond)
	if err := l.Wait(context.Background(), 1); err != nil {
		t.Fatalf("expected ok after backoff, got %v", err)
	}
}

func TestRPSLimiterHeadroom(t *testing.T) {
	l := NewRPSLimiter("otc", 10)
	if l.Headroom() != 1 {
		t.Fatalf("headroom full: %v", l.Headroom())
	}
	// RPSLimiter.Wait increments count by 1 per call regardless of cost.
	for i := 0; i < 5; i++ {
		_ = l.Wait(context.Background(), 1)
	}
	// count == 5, limit == 10, headroom == 0.5
	if l.Headroom() > 0.51 || l.Headroom() < 0.49 {
		t.Fatalf("headroom after 5: %v", l.Headroom())
	}
}

func TestRPSLimiterHeadroomZeroLimit(t *testing.T) {
	l := NewRPSLimiter("otc", 0)
	if l.Headroom() != 0 {
		t.Fatalf("headroom zero limit: %v", l.Headroom())
	}
}

func TestWeightedLimiterUpdateUsedZeroBudget(t *testing.T) {
	l := NewWeightedLimiter("binance", 0)
	l.UpdateUsed("binance", 100)
	if l.Used() != 100 {
		t.Fatalf("used: %d", l.Used())
	}
}