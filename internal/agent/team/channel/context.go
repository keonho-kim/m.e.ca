// Purpose: Render committed room messages as attributed context with explicit user/peer provenance.
// Pattern: Pure message projection.
// Usage: Inbox admission and shared-payload restoration use identical text and identities.
// Related: internal/agent/team/inbox.go, internal/storage/sqlite/team_payloads.go.
package channel

import (
	"fmt"
	"strconv"
	"strings"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

// Context distinguishes actual user instructions from attributed peer observations.
func (m Message) Context() modelapi.Message {
	if m.From == "user" {
		return modelapi.Message{ID: fmt.Sprintf("room:user/%d", m.Sequence), Role: "user", Content: m.Text}
	}
	text := fmt.Sprintf("Public conversation — %s", m.From)

	return modelapi.Message{ID: fmt.Sprintf("peer:%s/%d", m.From, m.Sequence), Role: "system", Content: text + "\n" + m.Text}
}

// Sequence recognizes context records backed by the public channel.
func Sequence(id string) (int64, bool) {
	if !IsPeer(id) && !strings.HasPrefix(id, "room:") {
		return 0, false
	}
	index := strings.LastIndexByte(id, '/')
	if index < 0 {
		return 0, false
	}
	sequence, err := strconv.ParseInt(id[index+1:], 10, 64)
	return sequence, err == nil && sequence > 0
}

// IsPeer identifies the provenance marker used by private context and compaction.
func IsPeer(id string) bool { return strings.HasPrefix(id, "peer:") }
