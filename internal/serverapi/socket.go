// Purpose: Authenticate browser WebSockets and multiplex controls with an independent event subscription.
// Pattern: Bidirectional WebSocket transport adapter.
// Usage: --protocol socket exposes /v1/socket; clients authenticate in the first frame.
// Related: api/meca/v1/control.proto, internal/serverapi/security.go.
package serverapi

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/coder/websocket"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/serverapi/admission"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func (t *Transport) socket(w http.ResponseWriter, r *http.Request) {
	select {
	case t.streams <- struct{}{}:
		defer func() { <-t.streams }()
	default:
		http.Error(w, diagcatalog.Text(diagcatalog.ServerapiTooManySockets), http.StatusTooManyRequests)
		return
	}
	// The middleware already enforces exact origin scheme and authority.
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true, CompressionMode: websocket.CompressionDisabled})
	if err != nil {
		return
	}
	defer conn.CloseNow()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	stop := context.AfterFunc(ctx, func() { _ = conn.CloseNow() })
	defer stop()
	conn.SetReadLimit(4096)
	auth, authCancel := context.WithTimeout(ctx, 5*time.Second)
	kind, body, err := conn.Read(auth)
	authCancel()
	var frame v1.SocketFrame
	if err != nil || kind != websocket.MessageText || protojson.Unmarshal(body, &frame) != nil || !t.authenticates(frame.GetAuthenticate()) {
		_ = conn.Close(websocket.StatusPolicyViolation, diagcatalog.Text(diagcatalog.ServerapiAuthenticationRequired))
		return
	}
	frame.Reset()
	conn.SetReadLimit(maxRequestBytes)
	send := func(n *v1.Notification) error {
		body, err := protojson.Marshal(n)
		if err != nil {
			return err
		}
		write, done := context.WithTimeout(ctx, 15*time.Second)
		defer done()
		if err := conn.Write(write, websocket.MessageText, body); err != nil {
			return faults.New(diagcatalog.DeliveryFailed, diagcatalog.WrappedCause, err)
		}
		return nil
	}
	if err := send(&v1.Notification{Body: &v1.Notification_Reply{Reply: &v1.Reply{Id: "authenticate", Result: &v1.Reply_Ok{Ok: &v1.EmptyResult{}}}}}); err != nil {
		return
	}
	var watchCancel context.CancelFunc
	var watchDone chan struct{}
	stopWatch := func() {
		if watchCancel != nil {
			watchCancel()
			<-watchDone
			watchCancel = nil
		}
	}
	defer func() {
		cancel()
		_ = conn.CloseNow()
		stopWatch()
	}()
	for {
		frame.Reset()
		readCtx, readCancel := context.WithCancel(ctx)
		kind, reader, err := conn.Reader(readCtx)
		if err != nil {
			readCancel()
			return
		}
		input, err := admission.NewInput(t.inputs, 0)
		if err != nil {
			readCancel()
			_ = conn.Close(websocket.StatusTryAgainLater, diagcatalog.Text(diagcatalog.ServerapiServerInputMemoryBudgetIsOccupied))
			return
		}
		keep := func() bool {
			defer input.Release()
			defer frame.Reset()
			defer readCancel()
			timer := time.AfterFunc(30*time.Second, readCancel)
			defer timer.Stop()
			body, err := io.ReadAll(input.Reader(reader))
			timer.Stop()
			if err != nil {
				return false
			}
			if kind != websocket.MessageText || protojson.Unmarshal(body, &frame) != nil {
				_ = conn.Close(websocket.StatusInvalidFramePayloadData, diagcatalog.Text(diagcatalog.ServerapiExpectedProtobufJSONFrame))
				return false
			}
			body = nil
			commandCtx := admission.WithInput(ctx, input)
			if err := input.Resize(int64(proto.Size(&frame)) + 4096); err != nil {
				return false
			}
			switch op := frame.GetBody().(type) {
			case *v1.SocketFrame_Command:
				if !validCorrelation(op.Command.GetId()) {
					_ = conn.Close(websocket.StatusPolicyViolation, diagcatalog.Text(diagcatalog.ServerapiCommandIDIsRequired))
					return false
				}
				reply, _ := t.api.Call(commandCtx, op.Command)
				if err := send(&v1.Notification{Body: &v1.Notification_Reply{Reply: reply}}); err != nil {
					return false
				}
			case *v1.SocketFrame_Watch:
				if proto.Size(op.Watch) > 4096 {
					_ = conn.Close(websocket.StatusPolicyViolation, diagcatalog.Text(diagcatalog.ServerapiWatchRequestExceedsBytes))
					return false
				}
				input.Release()
				stopWatch()
				watchCtx, done := context.WithCancel(ctx)
				watchCancel = done
				watchDone = make(chan struct{})
				cursor := op.Watch.GetCursor()
				taskID := op.Watch.GetTaskId()
				go func() {
					defer close(watchDone)
					watchErr := t.api.watch(watchCtx, cursor, taskID, send)
					if watchCtx.Err() == nil {
						if code := faults.Code(watchErr); code != diagcatalog.DeliveryFailed && code != diagcatalog.Canceled {
							_ = send(&v1.Notification{Body: &v1.Notification_Reply{Reply: t.api.errorReply("watch", watchErr)}})
						}
					}
				}()
			case *v1.SocketFrame_Unwatch:
				stopWatch()
			default:
				_ = conn.Close(websocket.StatusPolicyViolation, diagcatalog.Text(diagcatalog.ServerapiAuthenticateOnceThenSendCommandOr))
				return false
			}
			return true
		}()
		if !keep {
			return
		}

	}
}
