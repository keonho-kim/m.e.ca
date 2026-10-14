// Purpose: Query optional model identity metadata with bounded resources and safe failure behavior.
// Pattern: HTTP Adapter.
// Usage: Setup calls Client.Identify before generation diagnostics.
// Related: internal/provider/identity/report.go, internal/provider/discovery/identify_models.go.
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/identity"
	"strings"
	"unicode"
	"unicode/utf8"
)

func (c *Client) Identify(ctx context.Context, model string) (identity.Report, error) {
	if err := ctx.Err(); err != nil {
		return identity.Report{}, err
	}
	if model == "" || len(model) > 512 || !utf8.ValidString(model) || strings.ContainsFunc(model, func(r rune) bool { return unicode.IsControl(r) || unicode.Is(unicode.Cf, r) }) {
		return identity.Report{}, fmt.Errorf("invalid Model ID")
	}
	var report identity.Report
	var err error
	switch c.Connection.Kind {
	case connection.Ollama:
		report, err = c.identifyOllama(ctx, model)
	case connection.SGLang:
		report, err = c.identifySGLang(ctx, model)
	case connection.LMStudio:
		report, err = c.identifyLMStudio(ctx, model)
	case connection.LiteLLM:
		report, err = c.identifyLiteLLM(ctx, model)
	default:
		report, err = c.identifyModels(ctx, model)
	}
	if ctx.Err() != nil {
		return identity.Report{}, ctx.Err()
	}
	if err != nil {
		report = identity.Report{Unavailable: true, Failure: c.HTTP.FailureReason(err)}
	}
	report.Redact(c.HTTP.Redact)
	return report, nil
}

func (c *Client) serverRoot() string { return strings.TrimSuffix(c.Connection.APIBase(), "/v1") }

// Retain only scalar advertised Thinking options, without treating them as verified controls.
func thinkingFacts(report *identity.Report, source string, values []json.RawMessage) {
	for _, raw := range values {
		var value string
		if json.Unmarshal(raw, &value) == nil {
			report.Add(source, "thinking", value)
			continue
		}
		if string(raw) == "true" || string(raw) == "false" {
			report.Add(source, "thinking", string(raw))
		}
	}
}
