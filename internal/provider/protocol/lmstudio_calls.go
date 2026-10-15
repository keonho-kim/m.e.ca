// Purpose: Bind Responses function items and incremental arguments to stable call identities.
// Pattern: Provider stream adapter.
// Usage: LM Studio decoding forwards function events and terminal output items here.
// Related: internal/provider/protocol/lmstudio_stream.go, internal/provider/protocol/tool_calls.go.
package protocol

import modelapi "github.com/keonho-kim/v-code/internal/agent/model"

type lmItem struct {
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

func (s *streamState) lmCall(index int, item lmItem, final bool) error {
	if s.lmItems == nil {
		s.lmItems = make(map[string]int)
	}
	if previous, ok := s.lmItems[item.ID]; ok && item.ID != "" {
		index = previous
	}
	draft, err := s.callAt(index)
	if err != nil {
		return err
	}
	if item.CallID == "" || item.Name == "" || len(item.ID) > 256 {
		return &modelapi.ToolCallError{Detail: "incomplete Responses function identity"}
	}
	if draft.id != "" && (draft.id != item.CallID || draft.name != item.Name) {
		return &modelapi.ToolCallError{Detail: "Responses function identity changed"}
	}
	draft.id, draft.name = item.CallID, item.Name
	if item.ID != "" {
		s.lmItems[item.ID] = index
	}
	if final || item.Arguments != "" {
		return draft.finalArguments(item.Arguments)
	}
	return nil
}

func (d *callDraft) finalArguments(value string) error {
	if d.arguments.Len() == 0 {
		return d.appendArguments(value)
	}
	if d.arguments.String() != value && !equivalentArguments(d.arguments.String(), value) {
		return &modelapi.ToolCallError{Detail: "final function arguments differ from streamed arguments"}
	}
	return nil
}
