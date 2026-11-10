// Purpose: Serve Protobuf JSON control requests and independently resumable SSE notifications.
// Pattern: HTTP and SSE transport adapters.
// Usage: Every server protocol exposes POST /v1/command and GET /v1/events.
// Related: internal/serverapi/commands.go, internal/serverapi/watch.go.
package serverapi

import (
	"context"

	"io"
	"mime"
	"net/http"
	"net/url"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/serverapi/eventstream"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/encoding/protojson"
)

const maxRequestBytes = 17 << 20

func (t *Transport) command(w http.ResponseWriter, r *http.Request) {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		http.Error(w, diagcatalog.Text(diagcatalog.ServerapiContentTypeMustBeApplicationJSON), http.StatusUnsupportedMediaType)
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
		if faults.Code(err) == diagcatalog.ResourceExhausted || r.Context().Err() != nil {
			reply = t.api.errorReply("", err)
		} else {
			reply = t.api.errorReply("", faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerapiInvalidOrOversizedProtobufJSONCommand))
		}
	} else {
		body = nil
		reply, _ = t.api.Call(r.Context(), &cmd)
	}
	t.writeControlReply(w, r, reply)
}

func errorStatus(code string) int {
	return diagcatalog.HTTPStatus(code)
}

func (t *Transport) sse(w http.ResponseWriter, r *http.Request) {
	select {
	case t.streams <- struct{}{}:
		defer func() { <-t.streams }()
	default:
		http.Error(w, diagcatalog.Text(diagcatalog.ServerapiTooManyObservers), http.StatusTooManyRequests)
		return
	}
	query, err := url.ParseQuery(r.URL.RawQuery)
	for key, values := range query {
		if (key != "cursor" && key != "task_id" && key != "workspace_id" && key != "session_id") || len(values) != 1 {
			err = faults.Errorf(diagcatalog.ServerapiEventQueryParametersInvalid)
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cursor := query.Get("cursor")
	taskID := query.Get("task_id")
	if len(taskID) > 128 {
		http.Error(w, diagcatalog.Text(diagcatalog.ServerInvalidTaskFilter), http.StatusBadRequest)
		return
	}
	if header := r.Header.Get("Last-Event-ID"); header != "" {
		cursor = header
	}
	service, scope, release, err := t.observerService(r.Context(), query)
	if err != nil {
		t.writeControlReply(w, r, t.api.errorReply("", err))
		return
	}
	defer release()
	stopCache := t.api.events.observe()
	defer stopCache()
	first, err := service.ReplayTask(cursor, taskID)
	if err == nil {
		first, err = scope.prepare(r.Context(), service, first, taskID)
	}
	if err != nil {
		http.Error(w, diagcatalog.Text(diagcatalog.ServerInvalidEventCursor), http.StatusBadRequest)
		return
	}
	writer, err := eventstream.New(w)
	if err != nil {
		return
	}
	firstDelta := true
	lastDelivered := ""
	send := taskSender(taskID, func(n *v1.Notification) error {
		body, err := t.api.notificationJSON(n)
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
		lastDelivered = notificationCursor(n)
		if n.GetEvent().GetKind() != "assistant_delta" {
			firstDelta = true
			return writer.Flush()
		}
		if firstDelta {
			firstDelta = false
			return writer.Flush()
		}
		return nil
	})
	if err := writer.Batch(func() error { return t.api.sendBatch(r.Context(), scope.filter(first), taskID, send) }); err != nil {
		return
	}
	if first.Snapshot != nil {
		cursor = first.Snapshot.Cursor
	} else if len(first.Events) > 0 {
		cursor = first.Events[len(first.Events)-1].Cursor
	}
	if scope.scoped && len(first.Events) > 0 && cursor != lastDelivered {
		if err := writer.Send("cursor", cursor, []byte("{}")); err != nil {
			return
		}
		lastDelivered = cursor
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
		batch, err := service.NextTask(poll, cursor, taskID)
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
		batch, err = scope.prepare(r.Context(), service, batch, taskID)
		if err != nil {
			return
		}
		if err := writer.Batch(func() error { return t.api.sendBatch(r.Context(), scope.filter(batch), taskID, send) }); err != nil {
			return
		}
		if batch.Snapshot != nil {
			cursor = batch.Snapshot.Cursor
		} else {
			cursor = batch.Events[len(batch.Events)-1].Cursor
		}
		if scope.scoped && cursor != lastDelivered {
			if err := writer.Send("cursor", cursor, []byte("{}")); err != nil {
				return
			}
			lastDelivered = cursor
		}
	}
}
