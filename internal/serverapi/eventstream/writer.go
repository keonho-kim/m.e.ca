// Purpose: Write SSE records and schedule idle heartbeats without altering application payloads.
// Pattern: Single-writer HTTP streaming adapter.
// Usage: Native SSE handlers own one Writer per connection.
// Related: internal/serverapi/http.go.
package eventstream

import (
	"context"
	"net/http"
)

type Writer struct {
	writer               http.ResponseWriter
	controller           *http.ResponseController
	idle                 Idle
	frame                []byte
	batch, pending, sent bool
	queued               int
}

func New(w http.ResponseWriter) (*Writer, error) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	writer := &Writer{writer: w, controller: http.NewResponseController(w), idle: NewIdle()}
	return writer, writer.controller.Flush()
}

// Send accepts compact JSON, from its caller.
func (w *Writer) Send(event, id string, body []byte) error {
	return w.SendParts(event, id, body)
}

// SendParts writes already encoded JSON fragments as one SSE record. Large
// repeated text can be shared across terminal records without copying it.
func (w *Writer) SendParts(event, id string, parts ...[]byte) error {
	w.frame = w.frame[:0]
	if event != "" {
		w.frame = append(w.frame, "event: "...)
		w.frame = append(w.frame, event...)
		w.frame = append(w.frame, '\n')
	}
	if id != "" {
		w.frame = append(w.frame, "id: "...)
		w.frame = append(w.frame, id...)
		w.frame = append(w.frame, '\n')
	}
	w.frame = append(w.frame, "data: "...)
	size := 0
	for _, part := range parts {
		size += len(part)
	}
	if size <= 64<<10 {
		for _, part := range parts {
			w.frame = append(w.frame, part...)
		}
		w.frame = append(w.frame, '\n', '\n')
		if _, err := w.writer.Write(w.frame); err != nil {
			return err
		}
	} else {
		// Large terminal bodies are already encoded. Do not retain another
		// full-sized copy in every observer's framing buffer.
		if _, err := w.writer.Write(w.frame); err != nil {
			return err
		}
		for _, part := range parts {
			if _, err := w.writer.Write(part); err != nil {
				return err
			}
		}
		if _, err := w.writer.Write([]byte("\n\n")); err != nil {
			return err
		}
	}
	w.pending = true
	w.queued += len(w.frame)
	if size > 64<<10 {
		w.queued += size + 2
	}
	if w.batch && w.sent && w.queued < 64<<10 {
		return nil
	}
	return w.Flush()
}

func (w *Writer) PollContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return w.idle.PollContext(ctx)
}

func (w *Writer) Due() bool { return w.idle.Due() }

func (w *Writer) Heartbeat() error {
	if _, err := w.writer.Write([]byte(": heartbeat\n\n")); err != nil {
		return err
	}
	if err := w.controller.Flush(); err != nil {
		return err
	}
	application := w.pending
	if application {
		w.sent = true
		w.pending = false
		w.queued = 0
	}
	// A heartbeat keeps the connection alive; it is not new application output.
	if application {
		w.idle.Activity()
	} else {
		w.idle.Heartbeat()
	}
	return nil
}
