// Purpose: Assemble bounded native tool call fragments without exposing executable partial calls.
// Pattern: Request-owned stream accumulator.
// Usage: Stream decoders collect calls and release them only at successful completion.
// Related: internal/provider/protocol/chunk.go, internal/provider/protocol/responses/stream.go.
package native

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type Calls struct{ values map[int]*Draft }

type Draft struct {
	ID, Name  string
	Arguments strings.Builder
}

func (s *Calls) Add(body json.RawMessage, ollama, single bool) error {
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
			index = len(s.values)
		} else if !single {
			return &modelapi.ToolCallError{Detail: "streamed tool call has no index"}
		}
		if part.Type != "" && part.Type != "function" {
			return &modelapi.ToolCallError{Detail: "unsupported tool call type"}
		}
		draft, err := s.At(index)
		if err != nil {
			return err
		}
		if part.ID != "" {
			if draft.ID != "" && draft.ID != part.ID {
				return &modelapi.ToolCallError{Detail: "tool call ID changed during streaming"}
			}
			draft.ID = part.ID
		}
		draft.Name += part.Function.Name
		arguments := ""
		if len(part.Function.Arguments) > 0 {
			if ollama {
				arguments = string(part.Function.Arguments)
			} else if err := json.Unmarshal(part.Function.Arguments, &arguments); err != nil {
				return &modelapi.ToolCallError{Detail: "function arguments must be a JSON-encoded string"}
			}
		}
		if err := draft.Append(arguments); err != nil {
			return err
		}
		if len(draft.Name) > 128 || len(draft.ID) > 128 {
			return &modelapi.ToolCallError{Detail: "tool identity exceeds limit"}
		}
		if ollama && draft.ID == "" {
			draft.ID = "call_" + rand.Text()
		}
	}
	return nil
}

func (s *Calls) At(index int) (*Draft, error) {
	if index < 0 || index >= 1024 {
		return nil, &modelapi.ToolCallError{Detail: "invalid tool call index"}
	}
	if s.values == nil {
		s.values = make(map[int]*Draft)
	}
	if draft := s.values[index]; draft != nil {
		return draft, nil
	}
	if len(s.values) >= toolapi.MaxCalls {
		return nil, &modelapi.ToolCallError{Detail: "too many tool calls"}
	}
	draft := &Draft{}
	s.values[index] = draft
	return draft, nil
}

func (d *Draft) Append(text string) error {
	if d.Arguments.Len()+len(text) > toolapi.MaxArgumentsBytes {
		return &modelapi.ToolCallError{Detail: "tool arguments exceed size limit"}
	}
	d.Arguments.WriteString(text)
	return nil
}

func (s *Calls) Complete() ([]toolapi.Call, error) {
	indices := make([]int, 0, len(s.values))
	for index := range s.values {
		indices = append(indices, index)
	}
	sort.Ints(indices)
	ids := map[string]bool{}
	var calls []toolapi.Call
	for _, index := range indices {
		draft := s.values[index]
		call := toolapi.Call{ID: draft.ID, Name: draft.Name, Arguments: json.RawMessage(draft.Arguments.String())}
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
