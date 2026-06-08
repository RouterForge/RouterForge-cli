package event

import (
	"sync"
	"testing"
)

func TestInMemoryBus_Publish(t *testing.T) {
	b := NewBus()
	var mu sync.Mutex
	received := 0

	b.Subscribe("task.started", func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		received++
	})

	b.Publish(EvtTaskStarted, Event{Source: "test", Payload: "hello"})

	mu.Lock()
	if received != 1 {
		t.Fatalf("expected 1 event, got %d", received)
	}
	mu.Unlock()
}

func TestInMemoryBus_Wildcard(t *testing.T) {
	b := NewBus()
	var mu sync.Mutex
	count := 0

	b.Subscribe("*", func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		count++
	})

	b.Publish(EvtPhaseChanged, Event{Source: "s1"})
	b.Publish(EvtTaskStarted, Event{Source: "s2"})

	mu.Lock()
	if count != 2 {
		t.Fatalf("expected 2 events, got %d", count)
	}
	mu.Unlock()
}

func TestInMemoryBus_WildcardPrefix(t *testing.T) {
	b := NewBus()
	var mu sync.Mutex
	count := 0

	b.Subscribe("task.*", func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		count++
	})

	b.Publish(EvtTaskStarted, Event{Source: "s1"})
	b.Publish(EvtTaskCompleted, Event{Source: "s2"})
	b.Publish(EvtPhaseChanged, Event{Source: "s3"})

	mu.Lock()
	if count != 2 {
		t.Fatalf("expected 2 events, got %d", count)
	}
	mu.Unlock()
}

func TestInMemoryBus_Unsubscribe(t *testing.T) {
	b := NewBus()
	var mu sync.Mutex
	count := 0

	sub := b.Subscribe("test", func(e Event) {
		mu.Lock()
		defer mu.Unlock()
		count++
	})

	b.Publish(EventType("test"), Event{Source: "s1"})
	b.Unsubscribe(sub)
	b.Publish(EventType("test"), Event{Source: "s2"})

	mu.Lock()
	if count != 1 {
		t.Fatalf("expected 1 event after unsubscribe, got %d", count)
	}
	mu.Unlock()
}

func TestMatchTopic(t *testing.T) {
	tests := []struct {
		pattern string
		topic   string
		want    bool
	}{
		{"*", "anything", true},
		{"task.*", "task.started", true},
		{"task.*", "phase.changed", false},
		{"task.started", "task.started", true},
		{"task.started", "task.completed", false},
	}
	for _, tt := range tests {
		got := matchTopic(tt.pattern, tt.topic)
		if got != tt.want {
			t.Errorf("matchTopic(%q, %q) = %v, want %v", tt.pattern, tt.topic, got, tt.want)
		}
	}
}
