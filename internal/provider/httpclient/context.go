// Purpose: Resolve output budgets from serving limits and exact vLLM chat tokenization.
// Pattern: HTTP preflight adapter.
// Usage: HTTP generation resolves effective limits before encoding the request.
// Related: internal/agent/model/output.go, internal/provider/httpclient/request.go.
package httpclient

import (
	"context"
	"encoding/json"

	"maps"
	"net/http"
	"strings"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/protocol"
	"github.com/keonho-kim/meca/internal/provider/transport"
)

func (c *Client) contextOutput(ctx context.Context, request modelapi.Request, messages []protocol.Message, controls protocol.Controls, encoder *protocol.Encoder) (modelapi.InputBudget, error) {
	info := c.effectiveModel()
	budget := modelapi.InputBudget{Window: info.ContextWindow}
	resolve := func() (modelapi.InputBudget, error) {
		value, err := request.OutputBudget(budget.Window, info.MaxOutput, budget.InputTokens)
		budget.OutputTokens = value
		return budget, err
	}
	// Image preprocessing contributes model-specific tokens. A text-only tokenizer
	// count is not an exact multimodal count; use server generation limits instead.
	for _, message := range messages {
		if message.HasImages() {
			return resolve()
		}
	}
	if c.connection.Kind != connection.VLLM || c.modelInfo == nil || c.modelInfo.ContextWindow == 0 {
		return resolve()
	}
	kwargs := maps.Clone(controls.Template)
	if kwargs == nil {
		kwargs = map[string]any{}
	}
	// Match ChatCompletionRequest.build_chat_params, including its explicit None default.
	kwargs["reasoning_effort"] = nil
	if controls.ReasoningEffort != "" {
		kwargs["reasoning_effort"] = controls.ReasoningEffort
	}
	tools, err := encoder.EncodedTools(connection.VLLM, request.Tools)
	if err != nil {
		return budget, err
	}
	body, err := protocol.TokenizerBody(messages, struct {
		Tools      json.RawMessage `json:"tools,omitempty"`
		Model      string          `json:"model"`
		Template   map[string]any  `json:"chat_template_kwargs"`
		Generation bool            `json:"add_generation_prompt"`
	}{tools, request.ModelID, kwargs, true})
	if err != nil {
		return budget, err
	}
	if body.Len() > modelapi.MaxResponseBytes {
		return budget, &modelapi.ContextLimitError{Cause: faults.Errorf(diagcatalog.HttpclientTokenizerRequestExceedsContextSizeLimit)}
	}
	var count struct {
		Count *int `json:"count"`
		Limit int  `json:"max_model_len"`
	}
	if err := transport.ReadStreamingMetadata(ctx, c.transport, http.MethodPost, strings.TrimSuffix(c.connection.APIBase(), "/v1")+"/tokenize", body.Reader(), 16*1024*1024, &count); err != nil {
		return budget, faults.Errorf(diagcatalog.HttpclientModelContextTokenization, err)
	}
	if count.Count == nil || *count.Count < 0 || count.Limit <= 0 || count.Limit > 1<<40 {
		return budget, faults.Errorf(diagcatalog.HttpclientInvalidTokenizerContextLimits)
	}
	budget.Window, budget.InputTokens = min(info.ContextWindow, count.Limit), *count.Count
	return resolve()
}
