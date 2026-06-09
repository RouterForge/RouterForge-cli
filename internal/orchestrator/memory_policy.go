package orchestrator

import (
	"fmt"
	"strings"
	"sync"
)

type AccessLevel int

const (
	AccessDenied  AccessLevel = 0
	AccessRead    AccessLevel = 1
	AccessWrite   AccessLevel = 2
	AccessAdmin   AccessLevel = 3
)

type MemoryPolicy struct {
	mu      sync.RWMutex
	rules   []MemoryAccessRule
	defaultLevel AccessLevel
}

type MemoryAccessRule struct {
	AgentID     string      `json:"agent_id"`
	Scope       string      `json:"scope"`
	AccessLevel AccessLevel `json:"access_level"`
	Reason      string      `json:"reason"`
}

func NewMemoryPolicy() *MemoryPolicy {
	return &MemoryPolicy{
		defaultLevel: AccessRead,
	}
}

func (mp *MemoryPolicy) SetDefault(level AccessLevel) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.defaultLevel = level
}

func (mp *MemoryPolicy) Grant(agentID, scope string, level AccessLevel, reason string) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.rules = append(mp.rules, MemoryAccessRule{
		AgentID:     agentID,
		Scope:       scope,
		AccessLevel: level,
		Reason:      reason,
	})
}

func (mp *MemoryPolicy) Revoke(agentID, scope string) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	var kept []MemoryAccessRule
	for _, r := range mp.rules {
		if r.AgentID == agentID && r.Scope == scope {
			continue
		}
		kept = append(kept, r)
	}
	mp.rules = kept
}

func (mp *MemoryPolicy) CheckAccess(agentID, scope string, requested AccessLevel) bool {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	for _, r := range mp.rules {
		if r.AgentID == agentID && scopeMatches(r.Scope, scope) {
			return r.AccessLevel >= requested
		}
	}

	if mp.defaultLevel >= requested {
		return true
	}

	return false
}

func (mp *MemoryPolicy) GetAccessLevel(agentID, scope string) AccessLevel {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	for _, r := range mp.rules {
		if r.AgentID == agentID && scopeMatches(r.Scope, scope) {
			return r.AccessLevel
		}
	}
	return mp.defaultLevel
}

func (mp *MemoryPolicy) Rules() []MemoryAccessRule {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	out := make([]MemoryAccessRule, len(mp.rules))
	copy(out, mp.rules)
	return out
}

func (mp *MemoryPolicy) Validate() []string {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	var warnings []string
	for _, r := range mp.rules {
		if r.AccessLevel < AccessRead {
			warnings = append(warnings, fmt.Sprintf("deny rule for %s on %s has no effect", r.AgentID, r.Scope))
		}
	}
	return warnings
}

func scopeMatches(ruleScope, requestScope string) bool {
	if ruleScope == "*" {
		return true
	}
	if strings.HasSuffix(ruleScope, "*") {
		prefix := strings.TrimSuffix(ruleScope, "*")
		return strings.HasPrefix(requestScope, prefix)
	}
	return ruleScope == requestScope
}

type MemoryPolicyEnforcer struct {
	policy *MemoryPolicy
}

func NewMemoryPolicyEnforcer(policy *MemoryPolicy) *MemoryPolicyEnforcer {
	return &MemoryPolicyEnforcer{policy: policy}
}

func (e *MemoryPolicyEnforcer) CanRead(agentID, scope string) bool {
	return e.policy.CheckAccess(agentID, scope, AccessRead)
}

func (e *MemoryPolicyEnforcer) CanWrite(agentID, scope string) bool {
	return e.policy.CheckAccess(agentID, scope, AccessWrite)
}

func (e *MemoryPolicyEnforcer) CanAdmin(agentID, scope string) bool {
	return e.policy.CheckAccess(agentID, scope, AccessAdmin)
}

func (e *MemoryPolicyEnforcer) RequireRead(agentID, scope string) error {
	if !e.CanRead(agentID, scope) {
		return fmt.Errorf("agent %s denied read access to scope %s", agentID, scope)
	}
	return nil
}

func (e *MemoryPolicyEnforcer) RequireWrite(agentID, scope string) error {
	if !e.CanWrite(agentID, scope) {
		return fmt.Errorf("agent %s denied write access to scope %s", agentID, scope)
	}
	return nil
}
