// Purpose: Stream requested tool arguments, results or narrative pages without unrelated detail envelopes.
// Pattern: Explicit public projection over the shared masked page writer.
// Usage: The detail cache keys each view separately and keeps its existing byte and storage budgets.
// Related: internal/serverapi/detail/cache.go, internal/serverapi/detail/write.go, internal/commands/server/detail.go.
package detail

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"reflect"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/commands/server"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func viewEvent(original server.DetailOriginal) *record.Event {
	if original.Tool != nil {
		return original.Tool
	}
	if original.History != nil && original.History.Kind == "event" {
		return &original.History.Event
	}
	return nil
}
func writeView(ctx context.Context, out io.Writer, original server.DetailOriginal, source server.DetailSource, mask func() func(string, bool) string) error {
	view := source.View
	if mask == nil {
		return fault.Errorf(diag.DetailMaskFactoryRequired)
	}
	if view == server.DetailDocument {
		return writeDocument(ctx, out, original, mask)
	}
	event := viewEvent(original)
	if view == server.DetailText || view == server.DetailThinking {
		text := ""
		if event != nil {
			if view == server.DetailThinking {
				text = event.Thinking
			} else {
				text = event.Text
			}
		}
		if view == server.DetailText && original.History != nil && original.History.Kind == "message" {
			text = original.History.Message.Content
		}
		return writePlainText(ctx, out, text, mask)
	}
	if event == nil {
		return fault.New(diag.InvalidArgument, diag.ServerChooseExactlyOneValidDetailReference)
	}
	var value any
	switch view {
	case server.DetailToolRequest:
		if event.Call == nil {
			return fault.New(diag.NotFound, diag.ServerToolDetailWasNotFound)
		}
		value = event.Call.Arguments
	case server.DetailToolResult:
		if len(event.Output) > 0 {
			value = event.Output
		} else {
			value = event.Text
		}
	default:
		return fault.New(diag.InvalidArgument, diag.ServerChooseExactlyOneValidDetailReference)
	}
	if source.Field != "" || source.ListFields {
		var err error
		value, err = selectToolField(ctx, value, source.Field, source.ListFields)
		if err != nil {
			return err
		}
		if source.ListFields {
			for _, key := range value.([]string) {
				if mask()(key, true) != key {
					value = []string{}
					break
				}
			}
		}
		if raw, ok := value.(json.RawMessage); ok && len(raw) > 0 && raw[0] == '"' {
			return writeJSONText(ctx, out, raw, mask)
		}
	}
	buffer := bufio.NewWriterSize(out, 32<<10)
	writer := documentWriter{ctx: ctx, output: buffer, newMask: mask}
	if err := writer.value(reflect.ValueOf(value)); err != nil {
		return err
	}
	return buffer.Flush()
}
