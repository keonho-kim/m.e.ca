// Purpose: Project bounded conversation and directory batches for a browser subscription.
// Pattern: Read-only adapter sharing native scope, cursor and redaction contracts.
// Usage: uiObserver owns cancellation and delivery of each subscription.
// Related: internal/serverapi/watch_scope.go, internal/serverapi/eventstream/idle.go.
package serverapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"

	"github.com/keonho-kim/meca/internal/commands/server"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/serverapi/eventstream"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/encoding/protojson"
)

type uiMessage struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

func (t *Transport) observeUI(ctx context.Context, target *url.URL, send func([]uiMessage) error) error {
	query, err := url.ParseQuery(target.RawQuery)
	for key, values := range query {
		allowed := key == "workspace_id" || target.Path == "/v1/events" && (key == "cursor" || key == "session_id") || target.Path == "/v1/files/events" && key == "path"
		if !allowed || len(values) != 1 {
			err = fault.New(diag.InvalidArgument, diag.ServerPageInvalid)
		}
	}
	if err != nil || len(query.Get("path")) > 4096 {
		return fault.New(diag.InvalidArgument, diag.ServerPageInvalid)
	}
	service, scope, release, err := t.observerService(ctx, query)
	if err != nil {
		return err
	}
	defer release()
	if target.Path == "/v1/files/events" {
		return service.WatchFiles(ctx, query.Get("path"), func(change fileview.FileChange) error {
			body, err := protojson.Marshal(&v1.FileChange{Path: query.Get("path"), Refresh: change.Refresh, ChangedNames: change.Names, Rescan: change.Rescan})
			if err != nil {
				return err
			}
			return send([]uiMessage{{"change", body}})
		})
	}
	stopCache := t.api.events.observe()
	defer stopCache()
	cursor := query.Get("cursor")
	batch, err := service.ReplayTask(cursor, "")
	idle := eventstream.NewIdle()
	for {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		batch, err = scope.prepare(ctx, service, batch, "")
		if err != nil {
			return err
		}
		delivered := false
		if err := t.sendUIBatch(ctx, scope.filter(batch), func(messages []uiMessage) error { delivered = true; return send(messages) }); err != nil {
			return err
		}
		if batch.Snapshot != nil {
			cursor = batch.Snapshot.Cursor
		} else if len(batch.Events) > 0 {
			cursor = batch.Events[len(batch.Events)-1].Cursor
		}
		if delivered {
			idle.Activity()
		}
		for {
			if idle.Due() {
				if err := send([]uiMessage{{"heartbeat", json.RawMessage(`{}`)}}); err != nil {
					return err
				}
				idle.Heartbeat()
			}
			poll, cancel := idle.PollContext(ctx)
			batch, err = service.NextTask(poll, cursor, "")
			cancel()
			if !errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
				break
			}
			if err := send([]uiMessage{{"heartbeat", json.RawMessage(`{}`)}}); err != nil {
				return err
			}
			idle.Heartbeat()
		}
	}
}

func (t *Transport) sendUIBatch(ctx context.Context, batch server.Batch, send func([]uiMessage) error) error {
	messages := make([]uiMessage, 0, min(128, len(batch.Events)+1))
	bytes := 0
	flush := func() error {
		if len(messages) == 0 {
			return nil
		}
		err := send(messages)
		messages = messages[:0]
		bytes = 0
		return err
	}
	err := t.api.sendBatch(ctx, batch, "", func(n *v1.Notification) error {
		body, err := t.api.notificationJSON(n)
		if err != nil {
			return err
		}
		kind := "event"
		if n.GetSnapshot() != nil {
			kind = "snapshot"
		}
		messages = append(messages, uiMessage{kind, body})
		bytes += len(body)
		if len(messages) >= 128 || bytes >= 256<<10 || n.GetEvent().GetKind() != "assistant_delta" {
			return flush()
		}
		return nil
	})
	if err != nil {
		return err
	}
	return flush()
}
