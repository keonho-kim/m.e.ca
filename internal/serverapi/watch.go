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
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/serverapi/eventstream"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func (a *API) sendBatch(ctx context.Context, batch server.Batch, taskID string, send func(*v1.Notification) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if batch.Snapshot != nil {
		n := &v1.Notification{Body: &v1.Notification_Snapshot{Snapshot: snapshot(*batch.Snapshot)}}
		a.sanitize(n)
		return send(n)
	}
	for _, value := range batch.Events {
		if taskID != "" && value.Task != taskID {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		n := a.projectEvent(value)
		if err := send(n); err != nil {
			return err
		}
	}
	return nil
}

func (a *API) watch(ctx context.Context, cursor, taskID string, send func(*v1.Notification) error) (resultErr error) {
	stopCache := a.events.observe()
	defer stopCache()
	if len(taskID) > 128 {
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerInvalidTaskFilter)
	}
	delivered := false
	write := send
	send = taskSender(taskID, func(n *v1.Notification) error { delivered = true; return write(n) })
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
		batch, err := a.Service.NextTask(poll, cursor, taskID)
		cancel()
		if err != nil {
			if ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			return err
		}
		delivered = false
		if err := a.sendBatch(ctx, batch, taskID, send); err != nil {
			return err
		}
		if delivered {
			idle.Activity()
		}
		if batch.Snapshot != nil {
			cursor = batch.Snapshot.Cursor
		} else {
			cursor = batch.Events[len(batch.Events)-1].Cursor
		}
	}
}
