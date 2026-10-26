// Purpose: Preserve ordered OpenRouter reasoning blocks and expose only their display text.
// Pattern: Provider continuation adapter.
// Usage: The response decoder retains opaque blocks; Messages restores completed history.
// Related: internal/provider/protocol/chunk.go, internal/provider/protocol/tools.go.
package protocol

import (
	"encoding/json"
	"fmt"
	"strings"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func (state *streamState) routerDetails(blocks []json.RawMessage, streaming bool) (string, error) {
	var text strings.Builder
	for _, raw := range blocks {
		var block routerHeader
		if err := json.Unmarshal(raw, &block); err != nil || block.Type == "" {
			return "", fmt.Errorf("invalid OpenRouter reasoning block")
		}
		state.reasoningBytes += len(raw)
		if state.reasoningBytes > modelapi.MaxResponseBytes {
			return "", &modelapi.OutputLimitError{Reason: "provider continuation state byte limit"}
		}
		if !streaming {
			state.reasoningDetails = append(state.reasoningDetails, raw)
		} else if draft := state.reasoningDraft; draft != nil && draft.accepts(block) {
			if err := draft.append(raw, block); err != nil {
				return "", err
			}
		} else {
			if err := state.flushRouterDraft(); err != nil {
				return "", err
			}
			state.reasoningDraft = &routerDraft{header: block, raw: raw}
			state.reasoningDraft.text.WriteString(block.Text + block.Summary)
		}
		if block.Type == "reasoning.text" {
			text.WriteString(block.Text)
		} else if block.Type == "reasoning.summary" {
			text.WriteString(block.Summary)
		}
	}
	return text.String(), nil
}

func routerHistory(content []byte) ([]json.RawMessage, error) {
	var state streamState
	var blocks []json.RawMessage
	if err := json.Unmarshal(content, &blocks); err != nil || blocks == nil {
		return nil, fmt.Errorf("invalid OpenRouter continuation state")
	}
	if _, err := state.routerDetails(blocks, false); err != nil {
		return nil, err
	}
	return state.reasoningDetails, nil
}

type routerHeader struct{ Type, Text, Summary, Signature, ID, Format string }

type routerDraft struct {
	header routerHeader
	raw    json.RawMessage
	fields map[string]json.RawMessage
	text   strings.Builder
}

// A signature closes a text block. Stream indices can be reused across types;
// encrypted blocks always remain independent opaque objects.
func (draft *routerDraft) accepts(next routerHeader) bool {
	prior := draft.header
	return prior.Type == next.Type && (prior.Type == "reasoning.text" || prior.Type == "reasoning.summary") && prior.Signature == "" &&
		(prior.ID == "" || next.ID == "" || prior.ID == next.ID) && (prior.Format == "" || next.Format == "" || prior.Format == next.Format)
}

func (draft *routerDraft) append(raw json.RawMessage, header routerHeader) error {
	if draft.fields == nil {
		if err := json.Unmarshal(draft.raw, &draft.fields); err != nil {
			return err
		}
	}
	var next map[string]json.RawMessage
	if err := json.Unmarshal(raw, &next); err != nil {
		return err
	}
	for key, value := range next {
		if old, exists := draft.fields[key]; !exists || string(old) == "null" || string(old) == `""` {
			draft.fields[key] = value
		}
	}
	draft.text.WriteString(header.Text + header.Summary)
	draft.header.Signature = header.Signature
	if draft.header.ID == "" {
		draft.header.ID = header.ID
	}
	if draft.header.Format == "" {
		draft.header.Format = header.Format
	}
	return nil
}

func (draft *routerDraft) body() (json.RawMessage, error) {
	if draft.fields == nil {
		return draft.raw, nil
	}
	field := "text"
	if draft.header.Type == "reasoning.summary" {
		field = "summary"
	}
	text, err := json.Marshal(draft.text.String())
	if err != nil {
		return nil, err
	}
	draft.fields[field] = text
	return json.Marshal(draft.fields)
}

func (state *streamState) routerContinuation() ([]byte, error) {
	if err := state.flushRouterDraft(); err != nil {
		return nil, err
	}
	return json.Marshal(state.reasoningDetails)
}

func (state *streamState) flushRouterDraft() error {
	if state.reasoningDraft == nil {
		return nil
	}
	body, err := state.reasoningDraft.body()
	if err != nil {
		return err
	}
	state.reasoningDetails = append(state.reasoningDetails, body)
	state.reasoningDraft = nil
	return nil
}
