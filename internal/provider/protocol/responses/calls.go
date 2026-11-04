// Purpose: Bind Responses function items and incremental arguments to stable call identities.
// Pattern: Provider stream adapter.
// Usage: LM Studio decoding forwards function events and terminal output items here.
// Related: internal/provider/protocol/responses/stream.go, internal/provider/protocol/native/calls.go.
package responses

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
)

type Item struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Content   []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

func (s *Decoder) call(state *native.State, index int, item Item, final bool) error {
	if s.items == nil {
		s.items = make(map[string]int)
	}
	if previous, ok := s.items[item.ID]; ok && item.ID != "" {
		index = previous
	}
	draft, err := state.Calls.At(index)
	if err != nil {
		return err
	}
	if item.CallID == "" || item.Name == "" || len(item.ID) > 256 {
		return &modelapi.ToolCallError{Detail: "incomplete Responses function identity"}
	}
	if draft.ID != "" && (draft.ID != item.CallID || draft.Name != item.Name) {
		return &modelapi.ToolCallError{Detail: "Responses function identity changed"}
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
