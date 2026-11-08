// Purpose: Encode OpenAI stream lifecycle events using connection-owned typed delta values.
// Pattern: Single-writer wire encoder.
// Usage: Handler.stream owns one encoder until completion or observer disconnect.
// Related: internal/serverapi/openai/stream.go, internal/serverapi/openai/stream_text.go.
package openai

import (
	"encoding/json"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/serverapi/eventstream"
)

type encoder struct {
	writer               *eventstream.Writer
	value                server.Response
	chat, usage, started bool
	sequence             int
	chatID, messageID    string
	chunk                chatStream
	text                 string
	textDelta            textDelta
}

func (e *encoder) send(kind string, value any) error {
	if fields, ok := value.(map[string]any); ok && !e.chat {
		fields["type"] = kind
		fields["sequence_number"] = e.sequence
		e.sequence++
	}
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if e.chat {
		kind = ""
	}
	return e.writer.Send(kind, "", body)
}
func (e *encoder) start() error {
	e.chatID, e.messageID = chatID(e.value), messageID(e.value)
	if e.chat {
		e.chunk = chatChunk(e.chatID, e.value.Created, chatDelta{Role: "assistant", Content: &e.text}, nil)
		return e.send("", &e.chunk)
	}
	e.textDelta = textDelta{Type: "response.output_text.delta", Item: e.messageID, Logprobs: []any{}}
	initial := e.value
	initial.Status, initial.Completed, initial.Usage, initial.Error = "in_progress", 0, nil, ""
	if err := e.send("response.created", map[string]any{"response": responseValue(initial, "")}); err != nil {
		return err
	}
	return e.send("response.in_progress", map[string]any{"response": responseValue(initial, "")})
}
func (e *encoder) delta(text string) error {
	if text == "" {
		return nil
	}
	if e.chat {
		first := !e.started
		e.started = true
		e.text = text
		e.chunk.Choices[0].Delta = chatDelta{Content: &e.text}
		if err := e.send("", &e.chunk); err != nil {
			return err
		}
		if first {
			return e.writer.Flush()
		}
		return nil
	}
	first := !e.started
	if !e.started {
		e.started = true
		if err := e.send("response.output_item.added", map[string]any{"output_index": 0, "item": outputMessage(e.value, "", false, "in_progress")}); err != nil {
			return err
		}
		if err := e.send("response.content_part.added", map[string]any{"item_id": messageID(e.value), "output_index": 0, "content_index": 0, "part": textPart("")}); err != nil {
			return err
		}
	}
	e.textDelta.Sequence, e.textDelta.Delta = e.sequence, text
	e.sequence++
	if err := e.send(e.textDelta.Type, &e.textDelta); err != nil {
		return err
	}
	if first {
		return e.writer.Flush()
	}
	return nil
}
func (e *encoder) finish(value server.Response, text string) error {
	if e.chat {
		reason := finishReason(value)
		if err := e.send("", chatChunk(e.chatID, value.Created, chatDelta{}, &reason)); err != nil {
			return err
		}
		if e.usage {
			chunk := chatChunk(e.chatID, value.Created, chatDelta{}, nil)
			chunk.Choices, chunk.Usage = []chatStreamChoice{}, chatUsage(value)
			if err := e.send("", chunk); err != nil {
				return err
			}
		}
		return e.writer.Send("", "", []byte("[DONE]"))
	}
	// All four terminal events contain the same text. Escape it once while
	// preserving every protocol event and its complete original content.
	quoted, err := json.Marshal(text)
	if err != nil {
		return err
	}
	if e.started {
		base := map[string]any{"item_id": e.messageID, "output_index": 0, "content_index": 0, "text": nil, "logprobs": []any{}}
		if err := e.sendText("response.output_text.done", base, quoted); err != nil {
			return err
		}
		if err := e.sendText("response.content_part.done", map[string]any{"item_id": e.messageID, "output_index": 0, "content_index": 0, "part": textPart(nil)}, quoted); err != nil {
			return err
		}
		status := "completed"
		if value.Status != "completed" {
			status = "incomplete"
		}
		if err := e.sendText("response.output_item.done", map[string]any{"output_index": 0, "item": outputMessage(value, nil, true, status)}, quoted); err != nil {
			return err
		}
	}
	if text == "" {
		return e.send("response."+value.Status, map[string]any{"response": responseValue(value, "")})
	}
	return e.sendText("response."+value.Status, map[string]any{"response": responseWithText(value, nil, true)}, quoted)
}
func (e *encoder) fail(code, message string) {
	if e.chat {
		_ = e.send("", map[string]any{"error": apiError{Message: message, Type: diagcatalog.ServerError, Code: code}})
	} else {
		_ = e.send("error", map[string]any{"code": code, "message": message, "param": nil})
	}
}
