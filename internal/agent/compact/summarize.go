// Purpose: Receive one bounded summary from a selected source packet without enforcing an output format.
// Pattern: Model Adapter use case.
// Usage: Compaction controller calls summarize before installing a memory revision.
// Related: internal/agent/compact/controller.go, internal/agent/prompt/defaults/compaction.md.
package compact

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"strings"
	"unicode/utf8"
)

func (c *Controller) summarize(ctx context.Context, generate Generate, previous string, source []json.RawMessage, omitted int) (string, error) {
	messages, err := c.summaryMessages(previous, source, omitted)
	if err != nil {
		return "", err
	}
	parameters, err := inference.Resolve(c.Inference, inference.Compaction)
	if err != nil {
		return "", err
	}
	var reply strings.Builder
	done := false
	err = generate(ctx, parameters.Apply(modelapi.Request{ModelID: c.Model, Messages: messages}), func(event modelapi.Event) error {
		if event.Retry != nil {
			reply.Reset()
			done = false
			return ctx.Err()
		}
		if len(event.Calls) > 0 {
			return fmt.Errorf("compaction cannot call tools")
		}
		reply.WriteString(event.Delta)
		if reply.Len() > c.Settings.MaxSummaryBytes {
			return fmt.Errorf("compaction output exceeds limit")
		}
		if event.Done {
			done = true
		}
		return ctx.Err()
	})
	if err != nil {
		return "", err
	}
	if !done {
		return "", fmt.Errorf("compaction model response incomplete")
	}
	text := c.Redact(reply.String())
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("compaction response is empty")
	}
	if len(text) > c.Settings.MaxSummaryBytes || !utf8.ValidString(text) {
		return "", fmt.Errorf("compaction response exceeds size limit or is not valid UTF-8")
	}

	return text, nil
}
func (c *Controller) summaryMessages(previous string, source []json.RawMessage, omitted int) ([]modelapi.Message, error) {
	quoted, err := json.Marshal(struct {
		ActiveRequest       string
		PreviousSummary     string
		NextSource          []json.RawMessage
		OmittedObservations int
	}{c.Redact(c.ActiveRequest), previous, source, omitted})
	if err != nil {
		return nil, err
	}
	messages := []modelapi.Message{{Role: "system", Content: c.Redact(c.Prompt)}, {Role: "user", Content: "Summarize the work below, preserving the current request and unresolved requirements.\n" + string(quoted)}}
	bytes := 0
	for _, m := range messages {
		bytes += len(m.Content)
	}
	if bytes > c.Settings.MaxInputBytes {
		return nil, fmt.Errorf("compaction input cannot fit without dropping information")
	}

	return messages, nil
}
