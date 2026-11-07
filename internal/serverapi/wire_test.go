// Purpose: Exercise actual TLS HTTP, WebSocket and RPC clients against the selected transport.
// Pattern: Network contract test adapters.
// Usage: transports_test.go compares the same engine scenario across wire protocols.
// Related: internal/serverapi/http.go, internal/serverapi/socket.go, internal/serverapi/rpc.go.
package serverapi

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/coder/websocket"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"github.com/keonho-kim/meca/internal/serverapi/gen/mecav1/mecav1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

type observer struct {
	next  func() (*v1.Notification, error)
	close func() error
}
type wireClient struct {
	call  func(context.Context, *v1.Command) (*v1.Reply, error)
	watch func(context.Context, string) (observer, error)
	url   string
	http  *http.Client
}

func tlsWire(t *testing.T, kind string, api *API) wireClient {
	t.Helper()
	protocol := kind
	if strings.HasPrefix(kind, "grpc") || kind == "connect" {
		protocol = "grpc"
	}
	srv := httptest.NewUnstartedServer(nil)
	transport, err := newTransport(t.Context(), api, protocol, testToken, []string{srv.Listener.Addr().String()}, []string{"https://frontend.example"})
	if err != nil {
		srv.Close()
		t.Fatal(err)
	}
	srv.Config.Handler = transport
	srv.EnableHTTP2 = true
	srv.StartTLS()
	t.Cleanup(srv.Close)
	t.Cleanup(transport.Close)
	w := wireClient{url: srv.URL, http: srv.Client()}
	switch kind {
	case "sse":
		w.call = func(ctx context.Context, cmd *v1.Command) (*v1.Reply, error) {
			body, err := protojson.Marshal(cmd)
			if err != nil {
				return nil, err
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/v1/command", bytes.NewReader(body))
			if err != nil {
				return nil, err
			}
			req.Header.Set("Authorization", "Bearer "+testToken)
			req.Header.Set("Content-Type", "application/json")
			res, err := w.http.Do(req)
			if err != nil {
				return nil, err
			}
			defer res.Body.Close()
			data, err := io.ReadAll(res.Body)
			if err != nil {
				return nil, err
			}
			var reply v1.Reply
			err = protojson.Unmarshal(data, &reply)
			return &reply, err
		}
		w.watch = func(ctx context.Context, cursor string) (observer, error) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/v1/events", nil)
			if err != nil {
				return observer{}, err
			}
			req.Header.Set("Authorization", "Bearer "+testToken)
			req.Header.Set("Last-Event-ID", cursor)
			res, err := w.http.Do(req)
			if err != nil {
				return observer{}, err
			}
			if res.StatusCode != 200 {
				_ = res.Body.Close()
				return observer{}, fmt.Errorf("SSE status %d", res.StatusCode)
			}
			scanner := bufio.NewScanner(res.Body)
			scanner.Buffer(make([]byte, 4096), 8<<20)
			return observer{close: res.Body.Close, next: func() (*v1.Notification, error) {
				for scanner.Scan() {
					line := scanner.Text()
					if strings.HasPrefix(line, "data: ") {
						var n v1.Notification
						err := protojson.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &n)
						return &n, err
					}
				}
				if err := scanner.Err(); err != nil {
					return nil, err
				}
				return nil, io.EOF
			}}, nil
		}
	case "socket":
		dial := func(ctx context.Context) (*websocket.Conn, error) {
			conn, _, err := websocket.Dial(ctx, strings.Replace(srv.URL, "https:", "wss:", 1)+"/v1/socket", &websocket.DialOptions{HTTPClient: w.http})
			if err != nil {
				return nil, err
			}
			conn.SetReadLimit(8 << 20)
			body, _ := protojson.Marshal(&v1.SocketFrame{Body: &v1.SocketFrame_Authenticate{Authenticate: testToken}})
			if err := conn.Write(ctx, websocket.MessageText, body); err != nil {
				_ = conn.CloseNow()
				return nil, err
			}
			if _, _, err := conn.Read(ctx); err != nil {
				_ = conn.CloseNow()
				return nil, err
			}
			return conn, nil
		}
		w.call = func(ctx context.Context, cmd *v1.Command) (*v1.Reply, error) {
			conn, err := dial(ctx)
			if err != nil {
				return nil, err
			}
			defer conn.CloseNow()
			cmd.Id = "control"
			body, _ := protojson.Marshal(&v1.SocketFrame{Body: &v1.SocketFrame_Command{Command: cmd}})
			if err := conn.Write(ctx, websocket.MessageText, body); err != nil {
				return nil, err
			}
			_, body, err = conn.Read(ctx)
			if err != nil {
				return nil, err
			}
			var n v1.Notification
			err = protojson.Unmarshal(body, &n)
			return n.GetReply(), err
		}
		w.watch = func(ctx context.Context, cursor string) (observer, error) {
			conn, err := dial(ctx)
			if err != nil {
				return observer{}, err
			}
			body, _ := protojson.Marshal(&v1.SocketFrame{Body: &v1.SocketFrame_Watch{Watch: &v1.WatchRequest{Cursor: cursor}}})
			if err := conn.Write(ctx, websocket.MessageText, body); err != nil {
				_ = conn.CloseNow()
				return observer{}, err
			}
			return observer{close: conn.CloseNow, next: func() (*v1.Notification, error) {
				_, body, err := conn.Read(ctx)
				if err != nil {
					return nil, err
				}
				var n v1.Notification
				err = protojson.Unmarshal(body, &n)
				return &n, err
			}}, nil
		}
	default:
		var options []connecthttp.Option
		if kind == "grpc" {
			options = append(options, connecthttp.WithGRPC())
		}
		if kind == "grpc-web" {
			options = append(options, connecthttp.WithGRPCWeb())
		}
		client := &http.Client{Transport: bearerTransport{base: w.http.Transport}}
		rpc := mecav1connect.NewMecaServiceClient(connect.NewClient(connecthttp.NewTransport(client, srv.URL, options...)))
		w.call = rpc.Call
		w.watch = func(ctx context.Context, cursor string) (observer, error) {
			stream, err := rpc.WatchEvents(ctx, &v1.WatchRequest{Cursor: cursor})
			if err != nil {
				return observer{}, err
			}
			return observer{next: stream.Receive, close: stream.Close}, nil
		}
	}
	return w
}

type bearerTransport struct{ base http.RoundTripper }

func (b bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	copy := req.Clone(req.Context())
	copy.Header.Set("Authorization", "Bearer "+testToken)
	return b.base.RoundTrip(copy)
}

func call(t *testing.T, ctx context.Context, w wireClient, cmd *v1.Command) *v1.Reply {
	t.Helper()
	reply, err := w.call(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if reply == nil || reply.GetError() != nil {
		t.Fatalf("reply=%v", reply)
	}
	return reply
}
func next(t *testing.T, o observer) *v1.Notification {
	t.Helper()
	n, err := o.next()
	if err != nil {
		t.Fatal(err)
	}
	return n
}
