// Purpose: Serve Protobuf JSON control requests and independently resumable SSE notifications.
// Pattern: HTTP and SSE transport adapters.
// Usage: --protocol sse exposes POST /v1/command and GET /v1/events.
// Related: internal/serverapi/commands.go, internal/serverapi/watch.go.
package serverapi

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/serverapi/eventstream"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/encoding/protojson"
)

const maxRequestBytes = 17 << 20

func (t *Transport) command(w http.ResponseWriter, r *http.Request) {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	var cmd v1.Command
	if err == nil {
		err = protojson.Unmarshal(body, &cmd)
	}
	var reply *v1.Reply
	if err != nil {
		reply = t.api.errorReply("", &server.Error{Code: "invalid_argument", Message: "invalid or oversized Protobuf JSON command"})
	} else {
		reply, _ = t.api.Call(r.Context(), &cmd)
	}
	data, encodeErr := protojson.Marshal(reply)
	if encodeErr != nil {
		http.Error(w, "could not encode reply", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if reply.GetError() != nil {
		w.WriteHeader(errorStatus(reply.GetError().Code))
	}
	_, _ = w.Write(data)
}

func errorStatus(code string) int {
	switch code {
	case "invalid_argument":
		return http.StatusBadRequest
	case "not_found":
		return http.StatusNotFound
	case "conflict", "failed_precondition", "canceled":
		return http.StatusConflict
	case "resource_exhausted":
		return http.StatusTooManyRequests
	case "unavailable":
		return http.StatusServiceUnavailable
	case "deadline_exceeded":
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

func (t *Transport) sse(w http.ResponseWriter, r *http.Request) {
	select {
	case t.streams <- struct{}{}:
		defer func() { <-t.streams }()
	default:
		http.Error(w, "too many observers", http.StatusTooManyRequests)
		return
	}
	query, err := r.URL.Query(), error(nil)
	for key, values := range query {
		if key != "cursor" || len(values) != 1 {
			err = fmt.Errorf("only one cursor query parameter is supported")
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cursor := query.Get("cursor")
	if header := r.Header.Get("Last-Event-ID"); header != "" {
		cursor = header
	}
	first, err := t.api.Service.Replay(cursor)
	if err != nil {
		http.Error(w, "invalid event cursor", http.StatusBadRequest)
		return
	}
	writer, err := eventstream.New(w)
	if err != nil {
		return
	}
	firstDelta := true
	send := func(n *v1.Notification) error {
		body, err := protojson.Marshal(n)
		if err != nil {
			return err
		}
		kind := "event"
		if n.GetSnapshot() != nil {
			kind = "snapshot"
		}
		if err := writer.Send(kind, notificationCursor(n), body); err != nil {
			return err
		}
		if n.GetEvent().GetKind() != "assistant_delta" {
			firstDelta = true
			return writer.Flush()
		}
		if firstDelta {
			firstDelta = false
			return writer.Flush()
		}
		return nil
	}
	if err := writer.Batch(func() error { return t.api.sendBatch(r.Context(), first, send) }); err != nil {
		return
	}
	if first.Snapshot != nil {
		cursor = first.Snapshot.Cursor
	} else if len(first.Events) > 0 {
		cursor = first.Events[len(first.Events)-1].Cursor
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
		poll, cancel := writer.PollContext(r.Context())
		batch, err := t.api.Service.Next(poll, cursor)
		cancel()
		if err != nil {
			if r.Context().Err() != nil {
				return
			}
			if err == context.DeadlineExceeded {
				err = writer.Heartbeat()
				if err != nil {
					return
				}
				continue
			}
			return
		}
		if err := writer.Batch(func() error { return t.api.sendBatch(r.Context(), batch, send) }); err != nil {
			return
		}
		if batch.Snapshot != nil {
			cursor = batch.Snapshot.Cursor
		} else {
			cursor = batch.Events[len(batch.Events)-1].Cursor
		}
	}
}
