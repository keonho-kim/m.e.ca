// Purpose: Encode compatible SSE events and send idle heartbeats while observing process-owned tasks.
// Pattern: Single-writer streaming HTTP adapter.
// Usage: Handler.submit owns stream admission and calls stream until finish or observer disconnect.
// Related: internal/serverapi/eventstream/writer.go, internal/serverapi/openai/output.go.
package openai

import (
	"context"
	"errors"
	"net/http"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

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
			encoding.fail(diagcatalog.ServerError, diagcatalog.Text(diagcatalog.OpenaiCouldNotReadTaskStatus))
			return
		}
		value.Error = h.redact(value.Error)
		if terminal(value) {
			if value.Status == "failed" || value.Status == "cancelled" {
				if value.Status == "failed" && !encoding.chat {
					_ = encoding.send("response.failed", map[string]any{"response": responseValue(value, "")})
				} else {
					encoding.fail(value.Code, diagcatalog.Text(diagcatalog.OpenaiTheEngineTaskDidNotComplete))
				}
				return
			}
			err := h.withResult(r.Context(), writer, value, func(text string) error {
				if err := projection.finish(text); err != nil {
					if faults.Code(err) != diagcatalog.StreamRewrite {
						return err
					}
					encoding.fail(diagcatalog.StreamRewrite, diagcatalog.Text(diagcatalog.OpenaiPublicOutputChangedRetrieveTheResponse))
					return nil
				}
				return encoding.finish(value, text)
			})
			if err != nil {
				if r.Context().Err() != nil || faults.Code(err) == diagcatalog.DeliveryFailed {
					return
				}
				if errors.Is(err, history.ErrReadLimit) {
					encoding.fail(diagcatalog.ResourceExhausted, err.Error())
				} else {
					encoding.fail(diagcatalog.ServerError, diagcatalog.Text(diagcatalog.OpenaiCouldNotPrepareOrSendOriginal))
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
			encoding.fail(diagcatalog.ServerError, diagcatalog.Text(diagcatalog.OpenaiEventObservationEndedRetrieveTheResponse))
			return
		}
		cursor = batchCursor(batch, cursor)
		if batch.Snapshot != nil && !projection.buffered {
			encoding.fail(diagcatalog.StreamGap, diagcatalog.Text(diagcatalog.OpenaiObserverFellBehindRetrieveTheResponse))
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
			if faults.Code(err) == diagcatalog.StreamRewrite {
				encoding.fail(diagcatalog.StreamRewrite, diagcatalog.Text(diagcatalog.StreamRewritten))
			}
			return
		}
	}
}
