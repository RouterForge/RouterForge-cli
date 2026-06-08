package event

import (
	"strings"
	"sync"
)

type EventType string

const (
	EvtTaskStarted    EventType = "task.started"
	EvtTaskCompleted  EventType = "task.completed"
	EvtTaskFailed     EventType = "task.failed"
	EvtToolExecuted   EventType = "tool.executed"
	EvtModelCalled    EventType = "model.called"
	EvtAgentCreated   EventType = "agent.created"
	EvtAgentDied      EventType = "agent.died"
	EvtPhaseChanged   EventType = "phase.changed"
	EvtEscalation     EventType = "escalation.raised"
)

type Event struct {
	Type    EventType   `json:"type"`
	Source  string      `json:"source"`
	Payload interface{} `json:"payload"`
}

type Handler func(Event)

type Subscription struct {
	Topic   string
	Handler Handler
}

type Bus interface {
	Publish(topic EventType, event Event)
	Send(target string, event Event)
	Subscribe(topic string, handler Handler) *Subscription
	Unsubscribe(sub *Subscription)
}

type InMemoryBus struct {
	mu           sync.RWMutex
	subscriptions []*Subscription
}

func NewBus() *InMemoryBus {
	return &InMemoryBus{}
}

func (b *InMemoryBus) Publish(topic EventType, event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	event.Type = topic
	for _, sub := range b.subscriptions {
		if matchTopic(sub.Topic, string(topic)) {
			sub.Handler(event)
		}
	}
}

func (b *InMemoryBus) Send(target string, event Event) {
	// Direct send — same as publish for in-memory
	b.Publish(EventType("direct."+target), event)
}

func (b *InMemoryBus) Subscribe(topic string, handler Handler) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()
	sub := &Subscription{Topic: topic, Handler: handler}
	b.subscriptions = append(b.subscriptions, sub)
	return sub
}

func (b *InMemoryBus) Unsubscribe(sub *Subscription) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i, s := range b.subscriptions {
		if s == sub {
			b.subscriptions = append(b.subscriptions[:i], b.subscriptions[i+1:]...)
			return
		}
	}
}

func matchTopic(pattern, topic string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.HasPrefix(topic, prefix)
	}
	return pattern == topic
}
