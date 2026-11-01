// Purpose: Combine fresh controls and immutable schema fragments without copying schema bodies.
// Pattern: Provider wire adapter with replayable body segments.
// Usage: Encoder uses these helpers for small requests and cached conversation prefixes.
// Related: internal/provider/protocol/encoder.go, internal/provider/protocol/tool_snapshot.go.
package protocol

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

type requestHeader struct{ controls, tools []byte }

func (h requestHeader) appendTo(parts [][]byte, field string) ([][]byte, int) {
	parts = append(parts, h.controls[:len(h.controls)-1])
	size := len(h.controls) - 1
	if len(h.tools) > 0 {
		parts = append(parts, []byte(`,"tools":`), h.tools)
		size += len(`,"tools":`) + len(h.tools)
	}
	return append(parts, []byte(field)), size + len(field)
}

func (e *Encoder) header(kind connection.ProviderKind, baseURL string, info *connection.ModelInfo, request modelapi.Request, controls Controls) (string, requestHeader, error) {
	tools, err := e.EncodedTools(kind, request.Tools)
	if err != nil {
		return "", requestHeader{}, err
	}
	request.Tools = nil
	if kind == connection.LMStudio {
		base := lmHeader(request.ModelID, request.MaxOutput, controls.ReasoningEffort, request.Temperature, nil)
		if len(tools) > 0 {
			parallel := true
			base.Parallel, base.ToolChoice = &parallel, "auto"
		}
		body, err := wireJSON.Marshal(base)
		return baseURL + "/responses", requestHeader{controls: body, tools: tools}, err
	}
	endpoint, base := chatPayload(kind, baseURL, info, request, controls)
	if len(tools) > 0 && kind != connection.Ollama {
		parallel := true
		base.ToolChoice, base.ParallelCalls = "auto", &parallel
	}
	body, err := wireJSON.Marshal(base)
	return endpoint, requestHeader{controls: body, tools: tools}, err
}

func (e *Encoder) encodeWhole(kind connection.ProviderKind, baseURL string, info *connection.ModelInfo, request modelapi.Request, messages []Message, controls Controls) (string, Body, error) {
	endpoint, header, err := e.header(kind, baseURL, info, request, controls)
	if err != nil {
		e.Clear()
		return "", Body{}, err
	}
	var content []byte
	field := `,"messages":`
	if kind == connection.LMStudio {
		field = `,"input":`
		values, convertErr := lmMessages(messages)
		if convertErr != nil {
			e.Clear()
			return "", Body{}, convertErr
		}
		content, err = wireJSON.Marshal(values)
	} else {
		content, err = wireJSON.Marshal(messages)
	}
	if err != nil {
		e.Clear()
		return "", Body{}, err
	}
	parts, size := header.appendTo(make([][]byte, 0, 6), field)
	parts = append(parts, content, []byte("}"))
	return endpoint, Body{parts: parts, size: size + len(content) + 1}, nil
}
