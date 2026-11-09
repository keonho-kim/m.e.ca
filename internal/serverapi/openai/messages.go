// Purpose: Preserve ordered text input with bounded message, part and text budgets.
// Pattern: Typed input parser over validated JSON slices.
// Usage: input.go parses Chat messages and Responses input before any engine side effects.
// Related: internal/serverapi/openai/json.go, internal/commands/server/submission_session.go.
package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	"github.com/keonho-kim/meca/internal/commands/server"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

func parseMessages(ctx context.Context, raw json.RawMessage, param string, responses bool) ([]server.InputMessage, error) {
	var text string
	if responses && jsonString(raw, &text) {
		return []server.InputMessage{{Role: "user", Text: text}}, nil
	}
	list, ok := jsonArray(raw)
	if !ok {
		return nil, faults.Invalid(param, diagcatalog.OpenaiExpectedToTextMessages)
	}
	messages := make([]server.InputMessage, 0, 8)
	parts, size := 0, 0
	for item := list.next(); item != nil; item = list.next() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(messages) == 4096 {
			return nil, faults.Invalid(param, diagcatalog.OpenaiExpectedToTextMessages)
		}
		message, err := parseMessage(ctx, item, responses, &parts)
		if err != nil {
			return nil, prefixError(err, fmt.Sprintf("%s[%d]", param, len(messages)))
		}
		size += len(message.Text)
		if size > 16<<20 {
			return nil, faults.Invalid(param, diagcatalog.ServerInputTextExceedsMiB)
		}
		messages = append(messages, message)
	}
	if len(messages) == 0 || messages[len(messages)-1].Role != "user" {
		return nil, faults.Invalid(param, diagcatalog.OpenaiTheLastInputMessageMustHaveRole)
	}
	return messages, nil
}

func parseMessage(ctx context.Context, item []byte, responses bool, parts *int) (server.InputMessage, error) {
	fields, err := messageFields(item, responses)
	if err != nil {
		return server.InputMessage{}, err
	}
	if fields.kind != nil {
		kind, err := requiredString(fields.kind, "type")
		if err != nil || kind != "message" {
			return server.InputMessage{}, faults.Invalid("type", diagcatalog.OpenaiOnlyMessageInputItemsAreSupported)
		}
	}
	role, err := requiredString(fields.role, "role")
	if err != nil {
		return server.InputMessage{}, err
	}
	switch role {
	case "system", "developer", "user", "assistant":
	default:
		return server.InputMessage{}, faults.Invalid("role", diagcatalog.ServerUnsupportedInputRole)
	}
	text, err := parseContent(ctx, fields.content, responses, role, parts)
	return server.InputMessage{Role: role, Text: text}, prefixError(err, "content")
}

func parseContent(ctx context.Context, raw []byte, responses bool, role string, count *int) (string, error) {
	var text string
	if jsonString(raw, &text) {
		return text, nil
	}
	parts, ok := jsonArray(raw)
	if !ok {
		return "", faults.Invalid("", diagcatalog.OpenaiExpectedTextOrTextContentParts)
	}
	var content strings.Builder
	i := 0
	for part := parts.next(); part != nil; part = parts.next() {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if *count == 16384 {
			return "", faults.Invalid("", diagcatalog.OpenaiInputExceedsTextContentParts)
		}
		(*count)++
		value, err := parsePart(part, responses, role)
		if err != nil {
			return "", prefixError(err, fmt.Sprintf("[%d]", i))
		}
		if content.Len()+len(value) > 16<<20 {
			return "", faults.Invalid("", diagcatalog.ServerInputTextExceedsMiB)
		}
		content.WriteString(value)
		i++
	}
	if i == 0 {
		return "", faults.Invalid("", diagcatalog.OpenaiExpectedTextContentParts)
	}
	return content.String(), nil
}

func parsePart(part []byte, responses bool, role string) (string, error) {
	fields, err := partFields(part)
	if err != nil {
		return "", err
	}
	kind, err := requiredString(fields.kind, "type")
	if err != nil {
		return "", err
	}
	if !responses && kind != "text" || responses && kind != "input_text" && !(role == "assistant" && kind == "output_text") {
		return "", faults.Invalid("type", diagcatalog.OpenaiOnlyTextContentIsSupported)
	}
	return requiredString(fields.text, "text")
}
