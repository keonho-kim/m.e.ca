// Purpose: Validate Responses item shapes and bind stable function identities.
// Pattern: Provider stream adapter.
// Usage: OpenAI and LM Studio decoding forwards function events and terminal output items here.
// Related: internal/provider/protocol/responses/stream.go, internal/provider/protocol/native/calls.go.
package responses

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
)

type functionItem struct {
	Type      string                                      `json:"type"`
	ID        string                                      `json:"id"`
	CallID    string                                      `json:"call_id"`
	Name      string                                      `json:"name"`
	Arguments string                                      `json:"arguments"`
	Summary   []struct{ Type, Text skippedText }          `json:"summary"`
	Content   []struct{ Type, Text, Refusal skippedText } `json:"content"`
}

// Validate repeated text fields without allocating their already-streamed
// contents. encoding/json has checked token syntax before calling this method.
type skippedText struct{}

func (*skippedText) UnmarshalJSON(raw []byte) error {
	if len(raw) > 0 && (raw[0] == '"' || string(raw) == "null") {
		return nil
	}
	return faults.Errorf(diagcatalog.ResponsesResponsesOutputTextMustBeAString)
}

type outputItem struct {
	functionItem
	Summary []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"summary"`
	Content []struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		Refusal string `json:"refusal"`
	} `json:"content"`
}

func (s *Decoder) call(state *native.State, index int, item functionItem, final bool) error {
	if s.items == nil {
		s.items = make(map[string]int)
	}
	if previous, ok := s.items[item.ID]; ok && item.ID != "" {
		index = previous
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
