// Purpose: Encode compatible SSE events and send idle heartbeats while observing process-owned tasks.
// Pattern: Single-writer streaming HTTP adapter.
// Usage: Handler.submit owns stream admission and calls stream until finish or observer disconnect.
// Related: internal/serverapi/eventstream/writer.go, internal/serverapi/openai/output.go.
package openai

import (
	"context"
	"errors"
	"net/http"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/serverapi/eventstream"
	"github.com/keonho-kim/meca/internal/storage/history"
)

func (h *Handler) stream(w http.ResponseWriter, r *http.Request, p parsed, accepted server.Submitted) {
	writer, err := eventstream.New(w)
	if err != nil {
		return
	}
	encoding := encoder{writer: writer, value: accepted.Response, chat: p.input.Kind == "chat", usage: p.includeUsage}
	if err := encoding.start(); err != nil {
		return
	}
	projection := newOutput(accepted.Response.Team || accepted.Duplicate, encoding.delta)
	cursor := accepted.Cursor
	if cursor == "" {
		cursor = h.Service.State().Cursor
	}
	for {
		if r.Context().Err() != nil {
			return
		}
		if writer.Due() {
			if err := writer.Heartbeat(); err != nil {
				return
			}
		}
		value, err := eventstream.Await(r.Context(), writer, func(ctx context.Context) (server.Response, error) {
			return h.Service.ResponseState(ctx, accepted.Response.ID, false)
		})
		if err != nil {
			encoding.fail("server_error", "could not read task status")
			return
		}
		value.Error = h.redact(value.Error)
		if terminal(value) {
			if value.Status == "failed" || value.Status == "cancelled" {
				if value.Status == "failed" && !encoding.chat {
					_ = encoding.send("response.failed", map[string]any{"response": responseValue(value, "")})
				} else {
					encoding.fail(value.Code, "the engine task did not complete")
				}
				return
			}
			err := h.withResult(r.Context(), writer, value, func(text string) error {
				if err := projection.finish(text); err != nil {
					encoding.fail("stream_rewrite", "public output changed; retrieve the response result")
					return nil
				}
				return encoding.finish(value, text)
			})
			if err != nil {
				if errors.Is(err, history.ErrReadLimit) {
					encoding.fail("resource_exhausted", err.Error())
				} else {
					encoding.fail("server_error", "could not prepare or send original task output")
				}
			}
			return
		}
		poll, cancel := writer.PollContext(r.Context())
		if projection.buffered {
			_, err := h.Service.WaitResponse(poll, accepted.Response.ID)
			cancel()
			if err == nil {
				continue
			}
			if errors.Is(err, context.DeadlineExceeded) && r.Context().Err() == nil {
				if writer.Heartbeat() != nil {
					return
				}
				continue
			}
			return
		}
		batch, err := h.Service.Next(poll, cursor)
		cancel()
		if err != nil {
			if r.Context().Err() != nil {
				return
			}
			if errors.Is(err, context.DeadlineExceeded) {
				if writer.Heartbeat() != nil {
					return
				}
				continue
			}
			encoding.fail("server_error", "event observation ended; retrieve the response result")
			return
		}
		cursor = batchCursor(batch, cursor)
		if batch.Snapshot != nil && !projection.buffered {
			encoding.fail("stream_gap", "observer fell behind; retrieve the response result")
			return
		}
		err = writer.Batch(func() error {
			for _, event := range batch.Events {
				if err := r.Context().Err(); err != nil {
					return err
				}
				if event.Task != accepted.Response.Task || !projection.accepts(event) {
					continue
				}
				event.Record.Text = h.redact(event.Record.Text)
				if err := projection.event(event); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			encoding.fail("stream_rewrite", "generation rewrote a draft; the task continues, retrieve its result")
			return
		}
	}
}
