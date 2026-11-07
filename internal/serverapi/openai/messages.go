// Purpose: Preserve ordered text input with bounded message, part and text budgets.
// Pattern: Typed input parser over validated JSON slices.
// Usage: input.go parses Chat messages and Responses input before any engine side effects.
// Related: internal/serverapi/openai/json.go, internal/commands/server/submission_session.go.
package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/meca/internal/commands/server"
	"strings"
)

func parseMessages(ctx context.Context, raw json.RawMessage, param string, responses bool) ([]server.InputMessage, error) {
	var text string
	if responses && jsonString(raw, &text) {
		return []server.InputMessage{{Role: "user", Text: text}}, nil
	}
	list, ok := jsonArray(raw)
	if !ok {
		return nil, invalid(param, "expected 1 to 4096 text messages")
	}
	messages := make([]server.InputMessage, 0, 8)
	parts, size := 0, 0
	for item := list.next(); item != nil; item = list.next() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(messages) == 4096 {
			return nil, invalid(param, "expected 1 to 4096 text messages")
		}
		message, err := parseMessage(ctx, item, responses, &parts)
		if err != nil {
			return nil, prefixError(err, fmt.Sprintf("%s[%d]", param, len(messages)))
		}
		size += len(message.Text)
		if size > 16<<20 {
			return nil, invalid(param, "input text exceeds 16 MiB")
		}
		messages = append(messages, message)
	}
	if len(messages) == 0 || messages[len(messages)-1].Role != "user" {
		return nil, invalid(param, "the last input message must have role user")
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
			return server.InputMessage{}, invalid("type", "only message input items are supported")
		}
	}
	role, err := requiredString(fields.role, "role")
	if err != nil {
		return server.InputMessage{}, err
	}
	switch role {
	case "system", "developer", "user", "assistant":
	default:
		return server.InputMessage{}, invalid("role", "unsupported input role")
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
		return "", invalid("", "expected text or text content parts")
	}
	var content strings.Builder
	i := 0
	for part := parts.next(); part != nil; part = parts.next() {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if *count == 16384 {
			return "", invalid("", "input exceeds 16384 text content parts")
		}
		(*count)++
		value, err := parsePart(part, responses, role)
		if err != nil {
			return "", prefixError(err, fmt.Sprintf("[%d]", i))
		}
		if content.Len()+len(value) > 16<<20 {
			return "", invalid("", "input text exceeds 16 MiB")
		}
		content.WriteString(value)
		i++
	}
	if i == 0 {
		return "", invalid("", "expected text content parts")
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
		return "", invalid("type", "only text content is supported")
	}
	return requiredString(fields.text, "text")
}
