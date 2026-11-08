// Purpose: Encode Gemini contents and validate original thought signatures and native call pairing.
// Pattern: Concrete GenerateContent request adapter.
// Usage: Parent encoders call Base and EncodeMessage for whole or cached body construction.
// Related: internal/provider/protocol/gemini/types.go, internal/provider/protocol/gemini/stream.go.
package gemini

import (
	"encoding/json"

	"strings"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/protocol/native"
)

func Base(request modelapi.Request, system []string, thinking any, omitSummary bool) Request {
	body := Request{Config: Generation{MaxOutput: request.MaxOutput, Temperature: request.Temperature, Thinking: thinking}}
	if !omitSummary && body.Config.Thinking == nil {
		body.Config.Thinking = map[string]bool{"includeThoughts": true}
	}
	if len(request.Tools) > 0 {
		tool := Tool{}
		for _, d := range request.Tools {
			tool.Functions = append(tool.Functions, Declaration{Name: d.Name, Description: d.Description, Schema: d.JSONSchema()})
		}
		body.Tools = []Tool{tool}
	}
	for _, text := range system {
		if body.System == nil {
			body.System = &Content{}
		}
		body.System.Parts = append(body.System.Parts, Part{Text: new(text)})
	}

	return body
}

func EncodeMessage(message native.Input, wireIDs map[string]string) (string, []Part, error) {
	if message.Role == "system" {
		return "", nil, nil
	}
	text := message.Text
	if message.Role == "tool" {
		return "user", []Part{{Response: &FunctionResponse{ID: wireIDs[message.ToolCallID], Name: message.ToolName, Response: map[string]any{"output": text}}}}, nil
	}
	role := message.Role
	if role == "assistant" {
		role = "model"
	}
	parts, err := messageParts(message, wireIDs)
	return role, parts, err
}

func messageParts(message native.Input, wireIDs map[string]string) ([]Part, error) {
	calls := message.Calls
	text := message.Text
	var parts []Part
	if len(message.State) > 0 {
		var prior Continuation
		if err := json.Unmarshal(message.State, &prior); err != nil || prior.Parts == nil {
			return nil, faults.Errorf(diagcatalog.GeminiInvalidGeminiContinuation)
		}
		at := 0
		var original strings.Builder
		for _, part := range prior.Parts {
			if part.Text != nil && !part.Thought {
				original.WriteString(*part.Text)
			}
			if part.Call == nil {
				continue
			}
			if at >= len(calls) || at >= len(prior.CallIDs) || prior.CallIDs[at] != calls[at].ID || part.Call.Name != calls[at].Name {
				return nil, faults.Errorf(diagcatalog.GeminiGeminiContinuationDiffersFromToolHistory)
			}
			if !native.SameArguments(string(part.Call.Arguments()), string(calls[at].Arguments)) {
				return nil, faults.Errorf(diagcatalog.GeminiGeminiSignedArgumentsDifferFromToolHistory)
			}
			wireIDs[calls[at].ID] = part.Call.ID
			at++
		}
		if at != len(calls) || at != len(prior.CallIDs) {
			return nil, faults.Errorf(diagcatalog.GeminiGeminiContinuationOmittedAToolCall)
		}
		if original.String() != text && strings.TrimSpace(original.String()) != text {
			return nil, faults.Errorf(diagcatalog.GeminiGeminiContinuationTextDiffersFromMessageHistory)
		}
		// Signed parts must keep their exact text, argument and signature pairing.
		return prior.Parts, nil
	}
	if text != "" {
		parts = append(parts, Part{Text: new(text)})
	}
	if len(message.Parts) > 0 {
		for _, part := range message.Parts {
			if part.ImageURL == nil {
				parts = append(parts, Part{Text: new(part.Text)})
				continue
			}
			mime, data, ok := native.DataImage(part.ImageURL.URL)
			if !ok {
				return nil, faults.Errorf(diagcatalog.GeminiGeminiRequiresEmbeddedImages)
			}
			parts = append(parts, Part{Inline: &Blob{MIME: mime, Data: data}})
		}
	}
	for _, call := range calls {
		wireIDs[call.ID] = call.ID
		parts = append(parts, Part{Call: &FunctionCall{ID: call.ID, Name: call.Name, Args: call.Arguments}})
	}
	return parts, nil
}
