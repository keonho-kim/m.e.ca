// Purpose: Multiplex browser-only read subscriptions without consuming HTTP/1 request slots.
// Pattern: Cookie-authenticated WebSocket lifecycle with bounded, joined subscriptions.
// Usage: enableUI mounts this independently of the external API's selected protocol.
// Related: internal/serverapi/ui_observation.go, internal/serverapi/browser_sessions.go.
package serverapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/coder/websocket"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"google.golang.org/protobuf/encoding/protojson"
)

type uiSubscription struct {
	cancel context.CancelFunc
	done   chan struct{}
}

type uiWatchFrame struct {
	ID  string `json:"id"`
	URL string `json:"url,omitempty"`
	Ack bool   `json:"ack,omitempty"`
}

func (t *Transport) uiObserver(w http.ResponseWriter, r *http.Request) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if r.Header.Get("Origin") != scheme+"://"+r.Host || !t.browserAuthenticated(r) {
		writeAuthError(w, r)
		return
	}
	select {
	case t.streams <- struct{}{}:
		defer func() { <-t.streams }()
	default:
		http.Error(w, diag.Text(diag.ServerapiTooManyObservers), http.StatusTooManyRequests)
		return
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true, CompressionMode: websocket.CompressionDisabled})
	if err != nil {
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	active := make(map[string]uiSubscription)
	// TCP can finish writes while the browser's main thread is stalled. Limit
	// unacknowledged frames instead of moving an unbounded queue into the browser.
	window := make(chan struct{}, 4)
	stop := context.AfterFunc(ctx, func() { _ = conn.CloseNow() })
	defer func() {
		cancel()
		_ = conn.CloseNow()
		for _, subscription := range active {
			subscription.cancel()
			<-subscription.done
		}
		stop()
	}()
	conn.SetReadLimit(8192)
	for {
		kind, body, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var frame uiWatchFrame
		if kind != websocket.MessageText || decodeExactJSON(body, &frame) != nil || !validCorrelation(frame.ID) || len(frame.ID) > 64 {
			_ = conn.Close(websocket.StatusPolicyViolation, diag.Text(diag.ServerPageInvalid))
			return
		}
		if frame.Ack {
			if frame.URL != "" {
				return
			}
			select {
			case <-window:
			default:
				return
			}
			continue
		}
		if previous, ok := active[frame.ID]; ok {
			previous.cancel()
			<-previous.done
			delete(active, frame.ID)
		}
		if frame.URL == "" {
			continue
		}
		target, err := url.Parse(frame.URL)
		if err != nil || target.IsAbs() || target.Host != "" || target.Fragment != "" || len(frame.URL) > 6144 || (target.Path != "/v1/events" && target.Path != "/v1/files/events") || len(active) >= 2 {
			_ = conn.Close(websocket.StatusPolicyViolation, diag.Text(diag.ServerPageInvalid))
			return
		}
		watchCtx, stopWatch := context.WithCancel(ctx)
		done := make(chan struct{})
		active[frame.ID] = uiSubscription{stopWatch, done}
		go func() {
			defer close(done)
			send := func(messages []uiMessage) error {
				select {
				case window <- struct{}{}:
				case <-watchCtx.Done():
					return watchCtx.Err()
				}
				encoded, err := json.Marshal(struct {
					ID       string      `json:"id"`
					Messages []uiMessage `json:"messages"`
				}{frame.ID, messages})
				if err != nil {
					<-window
					return err
				}
				write, release := context.WithTimeout(watchCtx, 15*time.Second)
				defer release()
				err = conn.Write(write, websocket.MessageText, encoded)
				if err != nil {
					cancel()
				}
				return err
			}
			err := t.observeUI(watchCtx, target, send)
			if err != nil && watchCtx.Err() == nil {
				body, _ := protojson.Marshal(t.api.errorReply("", err).GetError())
				if send([]uiMessage{{Event: "error", Payload: body}}) != nil {
					cancel()
				}
			}
		}()
	}
}
