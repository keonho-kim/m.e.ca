// Purpose: Encode Claude history fragments while preserving exact signed-prefix binding.
// Pattern: Native dialect adapter for the execution-owned history cache.
// Usage: encodeCloud constructs a fresh header and checkpoints normalized message prefixes.
// Related: internal/provider/protocol/claude_request.go, internal/provider/protocol/cloud_history.go.
package protocol

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/protocol/claude"
)

func newClaudeCloud(base string, request modelapi.Request, messages []Message, controls Controls) (cloudRequest, error) {
	body, prefix, err := claudeBase(request, messages, controls)
	if err != nil {
		return cloudRequest{}, err
	}
	type fields claude.Request
	header, err := wireJSON.Marshal(struct {
		fields
		Messages []claude.Message `json:"messages,omitempty"`
	}{fields: fields(body)})
	return cloudRequest{endpoint: base + "/messages", field: `,"messages":`, header: header, prefix: prefix, initial: prefix.Value()}, err
}

func (r *cloudRequest) claudeEntry(message Message) (cloudEntry, error) {
	digest := ""
	if len(message.continuation) > 0 {
		digest = r.prefix.Value()
	}
	role, blocks, err := claudeMessage(message, digest)
	if err != nil {
		return cloudEntry{}, err
	}
	entry := cloudEntry{role: role}
	if len(blocks) == 0 {
		return entry, nil
	}
	if err := r.prefix.Append(role, blocks); err != nil {
		return cloudEntry{}, err
	}
	encoded, err := wireJSON.Marshal(blocks)
	if err != nil {
		return cloudEntry{}, err
	}
	entry.parts = encoded[1 : len(encoded)-1]
	return entry, nil
}
