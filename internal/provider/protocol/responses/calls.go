// Purpose: Validate Responses item shapes and bind stable function identities.
// Pattern: Provider stream adapter.
// Usage: OpenAI and LM Studio decoding forwards function events and terminal output items here.
// Related: internal/provider/protocol/responses/stream.go, internal/provider/protocol/native/calls.go.
package responses

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
)

type functionItem struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type contentPart struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Refusal string `json:"refusal"`
}

func (p contentPart) value() string {
	if p.Type == "refusal" {
		return p.Refusal
	}
	return p.Text
}

type outputItem struct {
	functionItem
	Summary []contentPart `json:"summary"`
	Content []contentPart `json:"content"`
}

func (s *Decoder) call(state *native.State, index int, item functionItem, final bool) error {
	if s.items == nil {
		s.items = make(map[string]int)
	}
	if previous, ok := s.items[item.ID]; ok && item.ID != "" {
		if previous != index {
			return &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.ResponsesResponsesFunctionIdentityChanged)}
		}
	} else if item.ID != "" && len(s.items) >= toolapi.MaxCalls {
		return &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.ResponsesTooManyResponsesFunctionIdentities)}
	}
	draft, err := state.Calls.At(index)
	if err != nil {
		return err
	}
	if item.CallID == "" || len(item.CallID) > 128 || item.Name == "" || len(item.Name) > 128 || len(item.ID) > 256 {
		return &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.ResponsesIncompleteResponsesFunctionIdentity)}
	}
	if draft.ID != "" && (draft.ID != item.CallID || draft.Name != item.Name) {
		return &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.ResponsesResponsesFunctionIdentityChanged)}
	}
	draft.ID, draft.Name = item.CallID, item.Name
	if item.ID != "" {
		s.items[item.ID] = index
	}
	if final || item.Arguments != "" {
		return draft.Final(item.Arguments)
	}
	return nil
}
