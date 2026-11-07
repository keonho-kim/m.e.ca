// Purpose: Encode Gemini history fragments while retaining native call/result identity pairing.
// Pattern: Native dialect adapter for the execution-owned history cache.
// Usage: encodeCloud restores only the IDs introduced by each unchanged cached message.
// Related: internal/provider/protocol/gemini_request.go, internal/provider/protocol/cloud_history.go.
package protocol

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/protocol/gemini"
)

func newGeminiCloud(base string, request modelapi.Request, messages []Message, controls Controls) (cloudRequest, error) {
	body := geminiBase(request, messages, controls)
	type fields gemini.Request
	header, err := wireJSON.Marshal(struct {
		fields
		Contents []gemini.Content `json:"contents,omitempty"`
	}{fields: fields(body)})
	return cloudRequest{endpoint: geminiEndpoint(base, request.ModelID), field: `,"contents":`, header: header, wireIDs: map[string]string{}}, err
}

func (r *cloudRequest) geminiEntry(message Message) (cloudEntry, error) {
	role, parts, err := geminiMessage(message, r.wireIDs)
	if err != nil {
		return cloudEntry{}, err
	}
	entry := cloudEntry{role: role}
	if len(parts) == 0 {
		return entry, nil
	}
	encoded, err := wireJSON.Marshal(parts)
	if err != nil {
		return cloudEntry{}, err
	}
	entry.parts = encoded[1 : len(encoded)-1]
	if message.Role == "assistant" {
		calls, err := message.functionCalls()
		if err != nil {
			return cloudEntry{}, err
		}
		for _, call := range calls {
			entry.ids = append(entry.ids, cloudCallID{local: call.ID, wire: r.wireIDs[call.ID]})
		}
	}
	return entry, nil
}
