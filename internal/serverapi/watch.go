// Purpose: Deliver the same ordered event batches and snapshots to every streaming adapter.
// Pattern: External event projection boundary.
// Usage: SSE, WebSocket and RPC send notifications through these functions.
// Related: internal/commands/server/events.go, internal/serverapi/projection.go.
package serverapi

import (
	"context"
	"errors"
	"time"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/diagnostics"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/serverapi/eventstream"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func (a *API) sendBatch(ctx context.Context, batch server.Batch, send func(*v1.Notification) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if batch.Snapshot != nil {
		n := &v1.Notification{Body: &v1.Notification_Snapshot{Snapshot: snapshot(*batch.Snapshot)}}
		a.sanitize(n)
		return send(n)
	}
	for _, value := range batch.Events {
		if err := ctx.Err(); err != nil {
			return err
		}
		n := &v1.Notification{Body: &v1.Notification_Event{Event: event(value)}}
		a.sanitize(n)
		if err := send(n); err != nil {
			return err
		}
	}
	return nil
}

func (a *API) watch(ctx context.Context, cursor string, send func(*v1.Notification) error) (resultErr error) {
	started := time.Now()
	defer func() {
		diagnostics.Log(ctx, diagcatalog.StreamClosed, "component", "serverapi", "duration_ms", time.Since(started).Milliseconds(), "error_code", diagnostics.Code(resultErr))
	}()
	idle := eventstream.NewIdle()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if idle.Due() {
			if err := send(&v1.Notification{Body: &v1.Notification_Event{Event: &v1.Event{Kind: "heartbeat"}}}); err != nil {
				return err
			}
			idle.Heartbeat()
		}
		poll, cancel := idle.PollContext(ctx)
		batch, err := a.Service.Next(poll, cursor)
		cancel()
		if err != nil {
			if ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			return err
		}
		if err := a.sendBatch(ctx, batch, send); err != nil {
			return err
		}
		idle.Activity()
		if batch.Snapshot != nil {
			cursor = batch.Snapshot.Cursor
		} else {
			cursor = batch.Events[len(batch.Events)-1].Cursor
		}
	}
}
