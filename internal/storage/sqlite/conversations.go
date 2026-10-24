// Purpose: Sort discovered conversation metadata without duplicating its summary array.
// Pattern: Filesystem metadata projection.
// Usage: Startup retains the returned metadata only for the lifetime of its picker.
// Related: internal/storage/sqlite/conversation_scan.go, internal/storage/sqlite/catalog.go, internal/tui/startup/resume.go.
package sqlite

import (
	"context"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	"sort"
	"time"
)

func (s *Sessions) Conversations(ctx context.Context, project string) ([]sessionstate.Conversation, error) {
	order := conversationOrder{}
	err := s.scanConversations(ctx, project, func(result conversationResult) {
		order.values = append(order.values, result.conversation)
		order.updated = append(order.updated, result.updated)
	})
	if err != nil {
		return nil, err
	}
	sort.Sort(order)
	return order.values, nil
}

type conversationOrder struct {
	values  []sessionstate.Conversation
	updated []time.Time
}

func (o conversationOrder) Len() int { return len(o.values) }
func (o conversationOrder) Less(i, j int) bool {
	if o.updated[i].Equal(o.updated[j]) {
		return o.values[i].ID > o.values[j].ID
	}
	return o.updated[i].After(o.updated[j])
}
func (o conversationOrder) Swap(i, j int) {
	o.values[i], o.values[j] = o.values[j], o.values[i]
	o.updated[i], o.updated[j] = o.updated[j], o.updated[i]
}
