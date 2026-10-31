// Purpose: Resolve output budgets from serving limits and exact vLLM chat tokenization.
// Pattern: HTTP preflight adapter.
// Usage: Prepared vLLM generation calls contextOutput before encoding the generation request.
// Related: internal/agent/model/output.go, internal/provider/httpclient/request.go.
package httpclient

import (
	"context"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/protocol"
	"github.com/keonho-kim/meca/internal/provider/transport"
	"maps"
	"net/http"
	"strings"
)

func (c *Client) contextOutput(ctx context.Context, model string, messages []protocol.Message, controls protocol.Controls, output int, definitions []toolapi.Definition) (int, error) {
	info := c.effectiveModel()
	// Image preprocessing contributes model-specific tokens. A text-only tokenizer
	// count is not an exact multimodal count; use server generation limits instead.
	for _, message := range messages {
		if message.HasImages() {
			return modelapi.OutputBudget(output, info.ContextWindow, info.MaxOutput, 0)
		}
	}
	if c.connection.Kind != connection.VLLM || c.modelInfo == nil || c.modelInfo.ContextWindow == 0 {
		return modelapi.OutputBudget(output, info.ContextWindow, info.MaxOutput, 0)
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
	body, err := protocol.TokenizerBody(messages, struct {
		Tools      []protocol.Tool `json:"tools,omitempty"`
		Model      string          `json:"model"`
		Template   map[string]any  `json:"chat_template_kwargs"`
		Generation bool            `json:"add_generation_prompt"`
	}{protocol.Tools(definitions), model, kwargs, true})
	if err != nil {
		return 0, err
	}
	if body.Len() > modelapi.MaxResponseBytes {
		return 0, fmt.Errorf("tokenizer request exceeds context size limit")
	}
	var count struct {
		Count *int `json:"count"`
		Limit int  `json:"max_model_len"`
	}
	if err := transport.ReadMetadata(ctx, c.transport, http.MethodPost, strings.TrimSuffix(c.connection.APIBase(), "/v1")+"/tokenize", body.Reader(), 16*1024*1024, &count); err != nil {
		return 0, fmt.Errorf("model context tokenization: %w", err)
	}
	if count.Count == nil || *count.Count < 0 || count.Limit <= 0 || count.Limit > 1<<40 {
		return 0, fmt.Errorf("invalid tokenizer context limits")
	}
	limit := min(info.ContextWindow, count.Limit)
	return modelapi.OutputBudget(output, limit, info.MaxOutput, *count.Count)
}
