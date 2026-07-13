package audit

import (
	"testing"
)

func TestInMemorySinkEmitAndCount(t *testing.T) {
	s := NewInMemorySink()
	if s.Count() != 0 {
		t.Fatalf("expected 0, got %d", s.Count())
	}
	s.Emit(Event{Type: EventOrderPlaced, Venue: "dummy", OrderID: "o1"})
	s.Emit(Event{Type: EventFill, Venue: "dummy", OrderID: "o1"})
	if s.Count() != 2 {
		t.Fatalf("expected 2, got %d", s.Count())
	}
	evs := s.Events()
	if len(evs) != 2 {
		t.Fatalf("expected 2 events, got %d", len(evs))
	}
	if evs[0].OrderID != "o1" {
		t.Fatalf("order id: %s", evs[0].OrderID)
	}
	if evs[0].Timestamp.IsZero() {
		t.Fatalf("timestamp not set")
	}
}

func TestInMemorySinkEventsCopy(t *testing.T) {
	s := NewInMemorySink()
	s.Emit(Event{Type: EventOrderCanceled, Venue: "dummy"})
	evs := s.Events()
	evs[0].OrderID = "mutated"
	evs2 := s.Events()
	if evs2[0].OrderID == "mutated" {
		t.Fatalf("Events() returned a live reference, expected a copy")
	}
}

func TestSinkInterfaceSatisfaction(t *testing.T) {
	var _ Sink = (*InMemorySink)(nil)
}