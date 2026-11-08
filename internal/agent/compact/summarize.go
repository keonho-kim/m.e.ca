// Purpose: Receive one bounded summary from a selected source packet without enforcing an output format.
// Pattern: Model Adapter use case.
// Usage: Compaction controller calls summarize before installing a memory revision.
// Related: internal/agent/compact/controller.go, internal/agent/prompt/defaults/compaction.md.
package compact

import (
	"context"
	"encoding/json"

	"strings"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (c *Controller) summarize(ctx context.Context, generate Generate, messages []modelapi.Message, budget summaryBudget) (string, error) {
	size := 0
	for _, message := range messages {
		size += len(message.Content)
	}
	if size > budget.inputBytes {
		return "", faults.Errorf(diagcatalog.CompactCompactionPacketExceedsModelContextInputBudget)
	}
	parameters, err := inference.Resolve(c.Inference, inference.Compaction)
	if err != nil {
		return "", err
	}
	var reply strings.Builder
	done := false
	request := parameters.Apply(modelapi.Request{ModelID: c.Model, Messages: messages})
	request.MaxOutput = budget.outputTokens
	err = generate(ctx, request, func(event modelapi.Event) error {
		if event.Retry != nil {
			reply.Reset()
			done = false
			return ctx.Err()
		}
		if len(event.Calls) > 0 {
			return faults.Errorf(diagcatalog.CompactCompactionCannotCallTools)
		}
		reply.WriteString(event.Delta)
		if reply.Len() > c.Settings.MaxSummaryBytes {
			return faults.Errorf(diagcatalog.CompactCompactionOutputExceedsLimit)
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
		return "", faults.Errorf(diagcatalog.CompactCompactionModelResponseIncomplete)
	}
	text := c.Redact(reply.String())
	if strings.TrimSpace(text) == "" {
		return "", faults.Errorf(diagcatalog.CompactCompactionResponseIsEmpty)
	}
	if len(text) > c.Settings.MaxSummaryBytes || !utf8.ValidString(text) {
		return "", faults.Errorf(diagcatalog.CompactCompactionResponseExceedsSizeLimitOrIs)
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
	messages := []modelapi.Message{{Role: "system", Content: c.Redact(c.Prompt)}, {Role: "user", Content: string(quoted)}}
	bytes := 0
	for _, m := range messages {
		bytes += len(m.Content)
	}
	if bytes > c.Settings.MaxInputBytes {
		return nil, faults.Errorf(diagcatalog.CompactCompactionInputCannotFitWithoutDroppingInformation)
	}

	return messages, nil
}
