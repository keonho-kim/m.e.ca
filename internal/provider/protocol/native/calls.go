// Purpose: Assemble bounded native tool call fragments without exposing executable partial calls.
// Pattern: Request-owned stream accumulator.
// Usage: Stream decoders collect calls and release them only at successful completion.
// Related: internal/provider/protocol/chunk.go, internal/provider/protocol/responses/stream.go.
package native

import (
	"crypto/rand"
	"encoding/json"

	"sort"
	"strings"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

type Calls struct{ values map[int]*Draft }

type Draft struct {
	ID, Name  string
	Arguments strings.Builder
}

// Bytes observes retained identity and argument growth without rescanning the
// argument strings. Empty fragments and repeated identities make no progress.
func (s *Calls) Bytes() int {
	bytes := 0
	for _, draft := range s.values {
		bytes += len(draft.ID) + len(draft.Name) + draft.Arguments.Len()
	}
	return bytes
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
		return &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.NativeMalformedNativeFunctionCallArray)}
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
			return &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.NativeStreamedToolCallHasNoIndex)}
		}
		if part.Type != "" && part.Type != "function" {
			return &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.NativeUnsupportedToolCallType)}
		}
		draft, err := s.At(index)
		if err != nil {
			return err
		}
		if part.ID != "" {
			if draft.ID != "" && draft.ID != part.ID {
				return &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.NativeToolCallIDChangedDuringStreaming)}
			}
			draft.ID = part.ID
		}
		draft.Name += part.Function.Name
		arguments := ""
		if len(part.Function.Arguments) > 0 {
			if ollama {
				arguments = string(part.Function.Arguments)
			} else if err := json.Unmarshal(part.Function.Arguments, &arguments); err != nil {
				return &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.NativeFunctionArgumentsMustBeAJSONEncoded)}
			}
		}
		if err := draft.Append(arguments); err != nil {
			return err
		}
		if len(draft.Name) > 128 || len(draft.ID) > 128 {
			return &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.NativeToolIdentityExceedsLimit)}
		}
		if ollama && draft.ID == "" {
			draft.ID = "call_" + rand.Text()
		}
	}
	return nil
}

func (s *Calls) At(index int) (*Draft, error) {
	if index < 0 || index >= 1024 {
		return nil, &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.NativeInvalidToolCallIndex)}
	}
	if s.values == nil {
		s.values = make(map[int]*Draft)
	}
	if draft := s.values[index]; draft != nil {
		return draft, nil
	}
	if len(s.values) >= toolapi.MaxCalls {
		return nil, &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.NativeTooManyToolCalls)}
	}
	draft := &Draft{}
	s.values[index] = draft
	return draft, nil
}

func (d *Draft) Append(text string) error {
	if d.Arguments.Len()+len(text) > toolapi.MaxArgumentsBytes {
		return &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.NativeToolArgumentsExceedSizeLimit)}
	}
	d.Arguments.WriteString(text)
	return nil
}

// Complete consumes the drafts; returned calls own their argument bytes.
func (s *Calls) Complete() ([]toolapi.Call, error) {
	defer func() { s.values = nil }()
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
			return nil, &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.NativeDuplicateToolCallID, call.ID)}
		}
		ids[call.ID] = true
		calls = append(calls, call)
	}
	return calls, nil
}
