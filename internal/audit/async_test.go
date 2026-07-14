package audit

import (
	"context"
	"log"
	"sync/atomic"
	"testing"
	"time"
)

type countingSink struct {
	count int64
}

func (c *countingSink) Emit(event Event) {
	atomic.AddInt64(&c.count, 1)
}

func (c *countingSink) Count() int64 {
	return atomic.LoadInt64(&c.count)
}

func TestAsyncSinkEmits(t *testing.T) {
	delegate := &countingSink{}
	s := NewAsyncSink(delegate, 16, "binance")
	defer s.Close()
	for i := 0; i < 5; i++ {
		s.Emit(Event{Type: EventOrderPlaced, Venue: "binance"})
	}
	time.Sleep(200 * time.Millisecond)
	if delegate.Count() != 5 {
		t.Fatalf("count: %d", delegate.Count())
	}
}

func TestAsyncSinkDropOnOverflow(t *testing.T) {
	delegate := &blockingSink{}
	s := NewAsyncSink(delegate, 2, "binance")
	defer s.Close()
	for i := 0; i < 10; i++ {
		s.Emit(Event{Type: EventOrderPlaced, Venue: "binance"})
	}
	if s.Dropped() == 0 {
		t.Fatalf("expected drops, got %d", s.Dropped())
	}
}

type blockingSink struct{}

func (b *blockingSink) Emit(event Event) {
	time.Sleep(50 * time.Millisecond)
}

func TestAsyncSinkClose(t *testing.T) {
	delegate := &countingSink{}
	s := NewAsyncSink(delegate, 4, "binance")
	s.Emit(Event{Type: EventOrderPlaced, Venue: "binance"})
	s.Close()
}

func TestEmitWithTrace(t *testing.T) {
	s := NewInMemorySink()
	tp := &fakeTrace{}
	EmitWithTrace(context.Background(), s, Event{Type: EventOrderPlaced, Venue: "binance"}, tp)
	evs := s.Events()
	if len(evs) != 1 || evs[0].TraceID != "trace-123" {
		t.Fatalf("trace: %+v", evs)
	}
}

type fakeTrace struct{}

func (f *fakeTrace) Extract(ctx context.Context) string { return "trace-123" }

func TestEmitWithTraceNoSink(t *testing.T) {
	tp := &fakeTrace{}
	EmitWithTrace(context.Background(), nil, Event{Type: EventOrderPlaced}, tp)
}

func TestLogSink(t *testing.T) {
	s := NewLogSink(log.Default())
	s.Emit(Event{Type: EventOrderPlaced, Venue: "binance", OrderID: "O1"})
}

func TestGRPCClient(t *testing.T) {
	g := NewgRPCClient("audit:9000", "binance")
	g.Emit(Event{Type: EventOrderPlaced, Venue: "binance"})
}

func TestNoopTrace(t *testing.T) {
	tp := NewNoopTrace()
	if tp.Extract(context.Background()) != "" {
		t.Fatalf("expected empty trace")
	}
}

func TestAsyncSinkNonBlocking(t *testing.T) {
	delegate := &blockingSink{}
	s := NewAsyncSink(delegate, 1, "binance")
	defer s.Close()
	done := make(chan struct{})
	go func() {
		s.Emit(Event{Type: EventOrderPlaced, Venue: "binance"})
		s.Emit(Event{Type: EventOrderPlaced, Venue: "binance"})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("Emit blocked")
	}
}