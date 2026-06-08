package memory

import (
	"sync"
	"time"
)

type EntryType string

const (
	TypeProject  EntryType = "project"
	TypeDecision EntryType = "decision"
	TypeTool     EntryType = "tool"
)

type Entry struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agent_id"`
	Type      EntryType `json:"type"`
	Content   string    `json:"content"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type Store interface {
	Add(entry Entry) error
	Query(agentID string, entryType EntryType, limit int) ([]Entry, error)
	Recent(limit int) ([]Entry, error)
	Clear(agentID string) error
}

type InMemoryStore struct {
	mu   sync.RWMutex
	data []Entry
}

func NewStore() *InMemoryStore {
	return &InMemoryStore{}
}

func (s *InMemoryStore) Add(entry Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry.Timestamp = time.Now()
	s.data = append(s.data, entry)
	return nil
}

func (s *InMemoryStore) Query(agentID string, entryType EntryType, limit int) ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Entry
	for i := len(s.data) - 1; i >= 0 && len(out) < limit; i-- {
		e := s.data[i]
		if e.AgentID == agentID && e.Type == entryType {
			out = append(out, e)
		}
	}
	return out, nil
}

func (s *InMemoryStore) Recent(limit int) ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit > len(s.data) {
		limit = len(s.data)
	}
	out := make([]Entry, limit)
	copy(out, s.data[len(s.data)-limit:])
	return out, nil
}

func (s *InMemoryStore) Clear(agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var filtered []Entry
	for _, e := range s.data {
		if e.AgentID != agentID {
			filtered = append(filtered, e)
		}
	}
	s.data = filtered
	return nil
}
