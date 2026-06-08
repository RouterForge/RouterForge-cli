package memory

import (
	"fmt"
	"strings"
)

type Compressor struct {
	store Store
}

func NewCompressor(store Store) *Compressor {
	return &Compressor{store: store}
}

func (c *Compressor) BuildContext(agentID string) string {
	entries, _ := c.store.Query(agentID, TypeDecision, 5)
	if len(entries) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Past decisions:\n")
	for _, e := range entries {
		b.WriteString(fmt.Sprintf("- %s\n", e.Content))
	}
	return b.String()
}

func (c *Compressor) TrimContext(ctx string, maxLen int) string {
	if len(ctx) <= maxLen {
		return ctx
	}
	return ctx[:maxLen] + "\n... [context truncated]"
}
