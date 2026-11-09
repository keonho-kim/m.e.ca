// Purpose: Verify replies, live task events, resubscription and unwatch share one authenticated WebSocket.
// Pattern: Bidirectional wire integration test with deterministic model cancellation.
// Usage: go test -race ./internal/serverapi -run TestSocketDuplex.
// Related: internal/serverapi/socket.go, internal/serverapi/wire_test.go.
package serverapi

import (
	"context"
	"strings"
	"testing"

	"github.com/coder/websocket"
	"github.com/keonho-kim/meca/internal/agent/model"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/encoding/protojson"
)

type socketPeer struct {
	t       *testing.T
	ctx     context.Context
	conn    *websocket.Conn
	pending []*v1.Notification
}

func (p *socketPeer) send(frame *v1.SocketFrame) {
	p.t.Helper()
	body, err := protojson.Marshal(frame)
	if err != nil {
		p.t.Fatal(err)
	}
	if err := p.conn.Write(p.ctx, websocket.MessageText, body); err != nil {
		p.t.Fatal(err)
	}
}

func (p *socketPeer) receive(matches func(*v1.Notification) bool) *v1.Notification {
	p.t.Helper()
	for i, n := range p.pending {
		if matches(n) {
			p.pending = append(p.pending[:i], p.pending[i+1:]...)
			return n
		}
	}
	for {
		_, body, err := p.conn.Read(p.ctx)
		if err != nil {
			p.t.Fatal(err)
		}
		var n v1.Notification
		if err := protojson.Unmarshal(body, &n); err != nil {
			p.t.Fatal(err)
		}
		if matches(&n) {
			return &n
		}
		p.pending = append(p.pending, &n)
	}
}

func (p *socketPeer) command(command *v1.Command) *v1.Reply {
	p.t.Helper()
	p.send(&v1.SocketFrame{Body: &v1.SocketFrame_Command{Command: command}})
	r := p.receive(func(n *v1.Notification) bool { return n.GetReply().GetId() == command.Id }).GetReply()
	if r.GetError() != nil {
		p.t.Fatal(r)
	}
	return r
}

func TestSocketDuplexControlAndUnwatch(t *testing.T) {
	f := newFixture(t, generateFunc(func(ctx context.Context, _ model.Request, emit func(model.Event) error) error {
		if err := emit(model.Event{Delta: "live answer", Streaming: true}); err != nil {
			return err
		}
		<-ctx.Done()
		return ctx.Err()
	}), false)
	w := tlsWire(t, "socket", f.api)
	ctx := deadline(t)
	conn, _, err := websocket.Dial(ctx, strings.Replace(w.url, "https:", "wss:", 1)+"/v1/socket", &websocket.DialOptions{HTTPClient: w.http})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	conn.SetReadLimit(8 << 20)
	p := &socketPeer{t: t, ctx: ctx, conn: conn}
	p.send(&v1.SocketFrame{Body: &v1.SocketFrame_Authenticate{Authenticate: testToken}})
	p.receive(func(n *v1.Notification) bool { return n.GetReply().GetId() == "authenticate" })
	state := p.command(&v1.Command{Id: "select", Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{Create: true}}}).GetState()
	p.send(&v1.SocketFrame{Body: &v1.SocketFrame_Watch{Watch: &v1.WatchRequest{}}})
	p.receive(func(n *v1.Notification) bool { return n.GetSnapshot() != nil })
	task := p.command(&v1.Command{Id: "start", Operation: &v1.Command_Start{Start: &v1.StartRequest{SessionId: state.Session.Id, Key: "duplex-request", Text: "wait for cancellation"}}}).GetTask()
	p.receive(func(n *v1.Notification) bool { return n.GetEvent().GetKind() == "assistant_delta" })
	p.command(&v1.Command{Id: "cancel", Operation: &v1.Command_Cancel{Cancel: &v1.TaskRequest{TaskId: task.Id}}})
	p.receive(func(n *v1.Notification) bool { return n.GetEvent().GetKind() == "request_finished" })
	// A control reply after unwatch is a barrier: prior watch writes have joined.
	p.send(&v1.SocketFrame{Body: &v1.SocketFrame_Unwatch{Unwatch: &v1.Empty{}}})
	before := p.command(&v1.Command{Id: "unwatched", Operation: &v1.Command_State{State: &v1.StateRequest{}}}).GetState()
	p.pending = nil
	permission := "all_mighty"
	if _, err := f.app.Configure(ctx, f.app.State().Session.ID, &permission, nil); err != nil {
		t.Fatal(err)
	}
	p.send(&v1.SocketFrame{Body: &v1.SocketFrame_Command{Command: &v1.Command{Id: "quiet", Operation: &v1.Command_State{State: &v1.StateRequest{}}}}})
	n := p.receive(func(*v1.Notification) bool { return true })
	if n.GetReply().GetId() != "quiet" {
		t.Fatal("unwatch delivered another event", n)
	}
	p.send(&v1.SocketFrame{Body: &v1.SocketFrame_Watch{Watch: &v1.WatchRequest{Cursor: before.Cursor}}})
	p.command(&v1.Command{Id: "resumed", Operation: &v1.Command_State{State: &v1.StateRequest{}}})
	p.receive(func(n *v1.Notification) bool { return n.GetEvent().GetKind() == "settings_changed" })
	p.send(&v1.SocketFrame{Body: &v1.SocketFrame_Watch{Watch: &v1.WatchRequest{}}})
	p.receive(func(n *v1.Notification) bool { return n.GetSnapshot().GetPermission() == permission })
}
