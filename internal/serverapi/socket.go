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
	"github.com/keonho-kim/meca/internal/commands/server"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/encoding/protojson"
)

func (t *Transport) socket(w http.ResponseWriter, r *http.Request) {
	select {
	case t.streams <- struct{}{}:
		defer func() { <-t.streams }()
	default:
		http.Error(w, "too many sockets", http.StatusTooManyRequests)
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
		_ = conn.Close(websocket.StatusPolicyViolation, "authentication required")
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
		return conn.Write(write, websocket.MessageText, body)
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
		release, ok := t.inputs.Try(maxRequestBytes)
		if !ok {
			readCancel()
			_ = conn.Close(websocket.StatusTryAgainLater, "server input memory budget is occupied")
			return
		}
		keep := func() bool {
			defer release()
			defer frame.Reset()
			defer readCancel()
			timer := time.AfterFunc(30*time.Second, readCancel)
			defer timer.Stop()
			body, err := io.ReadAll(reader)
			timer.Stop()
			if err != nil {
				return false
			}
			if kind != websocket.MessageText || protojson.Unmarshal(body, &frame) != nil {
				_ = conn.Close(websocket.StatusInvalidFramePayloadData, "expected Protobuf JSON frame")
				return false
			}
			switch op := frame.GetBody().(type) {
			case *v1.SocketFrame_Command:
				if !validCorrelation(op.Command.GetId()) {
					_ = conn.Close(websocket.StatusPolicyViolation, "command ID is required")
					return false
				}
				reply, _ := t.api.Call(ctx, op.Command)
				if err := send(&v1.Notification{Body: &v1.Notification_Reply{Reply: reply}}); err != nil {
					return false
				}
			case *v1.SocketFrame_Watch:
				stopWatch()
				watchCtx, done := context.WithCancel(ctx)
				watchCancel = done
				watchDone = make(chan struct{})
				cursor := op.Watch.GetCursor()
				go func() {
					defer close(watchDone)
					_ = t.api.watch(watchCtx, cursor, send)
					if watchCtx.Err() == nil {
						_ = send(&v1.Notification{Body: &v1.Notification_Reply{Reply: t.api.errorReply("watch", &server.Error{Code: "invalid_argument", Message: "event subscription closed; check cursor"})}})
					}
				}()
			case *v1.SocketFrame_Unwatch:
				stopWatch()
			default:
				_ = conn.Close(websocket.StatusPolicyViolation, "authenticate once, then send command or watch")
				return false
			}
			return true
		}()
		if !keep {
			return
		}

	}
}
