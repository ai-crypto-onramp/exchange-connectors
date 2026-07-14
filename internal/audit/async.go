package audit

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/ai-crypto-onramp/exchange-connectors/internal/metrics"
)

type AsyncSink struct {
	mu       sync.Mutex
	queue    chan Event
	delegate Sink
	closed   bool
	venue    string
	dropped  int64
}

type gRPCClient struct {
	url   string
	venue string
}

func NewgRPCClient(url, venue string) *gRPCClient {
	return &gRPCClient{url: url, venue: venue}
}

func (g *gRPCClient) Emit(event Event) {
	metrics.AuditEventsEmittedTotal.WithLabelValues(g.venue, string(event.Type)).Inc()
}

func NewAsyncSink(delegate Sink, queueSize int, venue string) *AsyncSink {
	s := &AsyncSink{
		queue:    make(chan Event, queueSize),
		delegate: delegate,
		venue:    venue,
	}
	go s.loop()
	return s
}

func (s *AsyncSink) Emit(event Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	select {
	case s.queue <- event:
	default:
		s.mu.Lock()
		s.dropped++
		s.mu.Unlock()
		metrics.AuditDroppedTotal.WithLabelValues(s.venue, string(event.Type)).Inc()
	}
}

func (s *AsyncSink) loop() {
	for ev := range s.queue {
		if s.delegate != nil {
			s.delegate.Emit(ev)
		}
		metrics.AuditEventsEmittedTotal.WithLabelValues(s.venue, string(ev.Type)).Inc()
	}
}

func (s *AsyncSink) Dropped() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dropped
}

func (s *AsyncSink) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()
	close(s.queue)
}

type TracePropagator interface {
	Extract(ctx context.Context) string
}

type noopTrace struct{}

func NewNoopTrace() TracePropagator { return &noopTrace{} }

func (n *noopTrace) Extract(ctx context.Context) string { return "" }

func EmitWithTrace(ctx context.Context, sink Sink, event Event, tp TracePropagator) {
	if tp != nil {
		event.TraceID = tp.Extract(ctx)
	}
	if sink != nil {
		sink.Emit(event)
	}
}

type logSink struct{ logger *log.Logger }

func NewLogSink(logger *log.Logger) Sink {
	if logger == nil {
		logger = log.Default()
	}
	return &logSink{logger: logger}
}

func (l *logSink) Emit(event Event) {
	l.logger.Printf("audit: type=%s venue=%s order=%s", event.Type, event.Venue, event.OrderID)
}