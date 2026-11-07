// Purpose: Decode Gemini text, thought signatures, completed function calls and measured usage.
// Pattern: Response-owned GenerateContent stream adapter.
// Usage: Protocol.ReadResponse dispatches each native Gemini JSON candidate here.
// Related: internal/provider/protocol/gemini/types.go, internal/provider/protocol/native/state.go.
package gemini

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
	"github.com/keonho-kim/meca/internal/provider/transport"
)

type Decoder struct {
	body    []byte
	callIDs []string
}

func (d *Decoder) Consume(content []byte, _ bool, state *native.State, emit func(modelapi.Event) error) error {
	var frame struct {
		Error      json.RawMessage `json:"error"`
		Candidates []struct {
			Content Content `json:"content"`
			Finish  string  `json:"finishReason"`
			Index   int     `json:"index"`
		} `json:"candidates"`
		Feedback struct {
			Block string `json:"blockReason"`
		} `json:"promptFeedback"`
		Usage *usageMetadata `json:"usageMetadata"`
	}
	if err := json.Unmarshal(content, &frame); err != nil {
		return fmt.Errorf("invalid Gemini response: %w", err)
	}
	if len(frame.Error) > 0 && string(frame.Error) != "null" {
		return transport.ParseError(0, content, "")
	}
	if frame.Feedback.Block != "" {
		if err := d.usage(frame.Usage, state); err != nil {
			return err
		}
		return state.EmitFailedUsage(&transport.PermanentError{Cause: fmt.Errorf("gemini prompt was blocked: %s", frame.Feedback.Block)}, emit)
	}
	if len(frame.Candidates) > 1 {
		return fmt.Errorf("multiple Gemini candidates are not supported")
	}
	if len(frame.Candidates) == 1 {
		candidate := frame.Candidates[0]
		if candidate.Index != 0 {
			return fmt.Errorf("invalid Gemini candidate index")
		}
		state.Received = true
		for _, part := range candidate.Content.Parts {
			if part.Inline != nil || part.Response != nil {
				return fmt.Errorf("unsupported Gemini output part")
			}
			raw, err := json.Marshal(part)
			if err != nil {
				return err
			}
			if len(d.body) == 0 {
				d.body = append(d.body, `{"parts":[`...)
			} else {
				d.body = append(d.body, ',')
			}
			if len(raw) > modelapi.MaxResponseBytes-len(d.body) {
				return &modelapi.OutputLimitError{Reason: "Gemini continuation byte limit"}
			}
			d.body = append(d.body, raw...)
			if part.Call != nil {
				call, err := state.Calls.At(len(d.callIDs))
				if err != nil {
					return err
				}
				call.ID, call.Name = part.Call.ID, part.Call.Name
				if call.ID == "" {
					call.ID = "call_" + rand.Text()
				}
				if err := call.Append(string(part.Call.Arguments())); err != nil {
					return err
				}
				d.callIDs = append(d.callIDs, call.ID)
			}
			if part.Text != nil && *part.Text != "" {
				state.Usage = nil
				event := modelapi.Event{Streaming: state.Streaming}
				if part.Thought {
					event.Thinking = *part.Text
				} else {
					event.Delta = *part.Text
				}
				if err := emit(event); err != nil {
					return err
				}
			}
		}
		if err := d.usage(frame.Usage, state); err != nil {
			return err
		}
		if candidate.Finish != "" {
			defer func() { *d = Decoder{} }()
		}
		switch candidate.Finish {
		case "":
			return nil
		case "STOP":
			if len(d.body) > 0 {
				ids, err := json.Marshal(d.callIDs)
				if err != nil {
					return err
				}
				if len(d.body)+len(`],"call_ids":`)+len(ids)+1 > modelapi.MaxResponseBytes {
					return &modelapi.OutputLimitError{Reason: "Gemini continuation byte limit"}
				}
				d.body = append(d.body, `],"call_ids":`...)
				d.body = append(d.body, ids...)
				state.Continuation = append(d.body, '}')
			}
			state.Finished = true
		case "MAX_TOKENS":
			return state.EmitFailedUsage(&modelapi.OutputLimitError{Reason: "Gemini output token limit"}, emit)
		default:
			return state.EmitFailedUsage(&transport.PermanentError{Cause: fmt.Errorf("gemini response did not complete: %s", candidate.Finish)}, emit)
		}
	} else if err := d.usage(frame.Usage, state); err != nil {
		return err
	}
	return nil
}

type usageMetadata struct {
	Input    int64 `json:"promptTokenCount"`
	Output   int64 `json:"candidatesTokenCount"`
	Thoughts int64 `json:"thoughtsTokenCount"`
}

func (d *Decoder) usage(value *usageMetadata, state *native.State) error {
	if value == nil {
		return nil
	}
	if value.Thoughts < 0 || value.Thoughts > modelapi.MaxCallTokens {
		return fmt.Errorf("invalid Gemini thought usage")
	}
	// These are non-optional protobuf scalars: an omitted counter is exactly zero.
	// A missing usageMetadata object still means usage was not reported.
	if value.Output < 0 || value.Output > modelapi.MaxCallTokens {
		return fmt.Errorf("invalid Gemini output usage")
	}
	state.Usage = &modelapi.TokenUsage{Input: value.Input, Output: value.Output + value.Thoughts}
	return state.Usage.Check()
}
