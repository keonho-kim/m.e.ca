// Purpose: Assemble bounded native tool call fragments without exposing executable partial calls.
// Pattern: Request-owned stream accumulator.
// Usage: Stream decoders collect calls and release them only at successful completion.
// Related: internal/provider/protocol/chunk.go, internal/provider/protocol/lmstudio_stream.go.
package protocol

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

type callDraft struct {
	id, name  string
	arguments strings.Builder
}

func (s *streamState) addCalls(body json.RawMessage, ollama, single bool) error {
	var parts []struct {
		Index    *int   `json:"index"`
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Index     *int            `json:"index"`
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"function"`
	}
	if err := json.Unmarshal(body, &parts); err != nil {
		return &modelapi.ToolCallError{Detail: "malformed native function call array"}
	}
	for position, part := range parts {
		index := position
		if part.Index != nil {
			index = *part.Index
		} else if ollama && part.Function.Index != nil {
			index = *part.Function.Index
		} else if ollama {
			index = len(s.calls)
		} else if !single {
			return &modelapi.ToolCallError{Detail: "streamed tool call has no index"}
		}
		if part.Type != "" && part.Type != "function" {
			return &modelapi.ToolCallError{Detail: "unsupported tool call type"}
		}
		draft, err := s.callAt(index)
		if err != nil {
			return err
		}
		if part.ID != "" {
			if draft.id != "" && draft.id != part.ID {
				return &modelapi.ToolCallError{Detail: "tool call ID changed during streaming"}
			}
			draft.id = part.ID
		}
		draft.name += part.Function.Name
		arguments := ""
		if len(part.Function.Arguments) > 0 {
			if ollama {
				arguments = string(part.Function.Arguments)
			} else if err := json.Unmarshal(part.Function.Arguments, &arguments); err != nil {
				return &modelapi.ToolCallError{Detail: "function arguments must be a JSON-encoded string"}
			}
		}
		if err := draft.appendArguments(arguments); err != nil {
			return err
		}
		if len(draft.name) > 128 || len(draft.id) > 128 {
			return &modelapi.ToolCallError{Detail: "tool identity exceeds limit"}
		}
		if ollama && draft.id == "" {
			draft.id = "call_" + rand.Text()
		}
	}
	return nil
}

func (s *streamState) callAt(index int) (*callDraft, error) {
	if index < 0 || index >= 1024 {
		return nil, &modelapi.ToolCallError{Detail: "invalid tool call index"}
	}
	if s.calls == nil {
		s.calls = make(map[int]*callDraft)
	}
	if draft := s.calls[index]; draft != nil {
		return draft, nil
	}
	if len(s.calls) >= toolapi.MaxCalls {
		return nil, &modelapi.ToolCallError{Detail: "too many tool calls"}
	}
	draft := &callDraft{}
	s.calls[index] = draft
	return draft, nil
}

func (d *callDraft) appendArguments(text string) error {
	if d.arguments.Len()+len(text) > toolapi.MaxArgumentsBytes {
		return &modelapi.ToolCallError{Detail: "tool arguments exceed size limit"}
	}
	d.arguments.WriteString(text)
	return nil
}

func (s *streamState) completedCalls() ([]toolapi.Call, error) {
	indices := make([]int, 0, len(s.calls))
	for index := range s.calls {
		indices = append(indices, index)
	}
	sort.Ints(indices)
	ids := map[string]bool{}
	var calls []toolapi.Call
	for _, index := range indices {
		draft := s.calls[index]
		call := toolapi.Call{ID: draft.id, Name: draft.name, Arguments: json.RawMessage(draft.arguments.String())}
		if err := call.Check(); err != nil {
			return nil, &modelapi.ToolCallError{Detail: err.Error()}
		}
		if ids[call.ID] {
			return nil, &modelapi.ToolCallError{Detail: fmt.Sprintf("duplicate tool call ID %s", call.ID)}
		}
		ids[call.ID] = true
		calls = append(calls, call)
	}
	return calls, nil
}
