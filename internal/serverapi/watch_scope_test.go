// Purpose: Verify scoped SSE drops background bodies while preserving controls, reset state and replay progress.
// Pattern: Real TLS streaming contract test.
// Usage: go test -race ./internal/serverapi -run TestScopedSSE.
// Related: internal/serverapi/watch_scope.go, internal/serverapi/http.go.
package serverapi

import (
	"bufio"
	"context"
	"github.com/keonho-kim/meca/internal/agent/model"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/encoding/protojson"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
)

type scopedFrame struct{ kind, id, data string }

func readScopedFrame(t *testing.T, scanner *bufio.Scanner) scopedFrame {
	t.Helper()
	var frame scopedFrame
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" && frame.data != "" {
			return frame
		}
		if strings.HasPrefix(line, "event: ") {
			frame.kind = strings.TrimPrefix(line, "event: ")
		}
		if strings.HasPrefix(line, "id: ") {
			frame.id = strings.TrimPrefix(line, "id: ")
		}
		if strings.HasPrefix(line, "data: ") {
			frame.data = strings.TrimPrefix(line, "data: ")
		}
	}
	t.Fatal("stream ended before expected frame", scanner.Err())
	return frame
}
func TestScopedSSECursorAndSelectedSnapshot(t *testing.T) {
	release := make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	f := newFixture(t, generateFunc(func(ctx context.Context, _ model.Request, emit func(model.Event) error) error {
		if err := emit(model.Event{Delta: "background-private-body", Streaming: true}); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-release:
		}
		return emit(model.Event{Delta: " finished", Done: true})
	}), false, 2)
	selected, err := f.app.Select(t.Context(), "", true)
	if err != nil {
		t.Fatal(err)
	}
	background, err := f.app.Select(t.Context(), "", true)
	if err != nil {
		t.Fatal(err)
	}
	wire := tlsWire(t, "sse", f.api)
	open := func(cursor string) *http.Response {
		t.Helper()
		query := url.Values{"session_id": {selected.Session.ID}}
		request, err := http.NewRequestWithContext(deadline(t), "GET", wire.url+"/v1/events?"+query.Encode(), nil)
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Authorization", "Bearer "+testToken)
		request.Header.Set("Last-Event-ID", cursor)
		response, err := wire.http.Do(request)
		if err != nil || response.StatusCode != 200 {
			t.Fatal(response, err)
		}
		return response
	}
	initial := open("")
	defer initial.Body.Close()
	scanner := bufio.NewScanner(initial.Body)
	frame := readScopedFrame(t, scanner)
	var snapshot v1.Notification
	if err := protojson.Unmarshal([]byte(frame.data), &snapshot); err != nil || snapshot.GetSnapshot().GetSession().GetId() != selected.Session.ID {
		t.Fatal("wrong reset session", frame, err)
	}
	task, err := f.app.Start(t.Context(), background.Session.ID, "scoped-replay-key", "hold")
	if err != nil {
		t.Fatal(err)
	}
	cursor := ""
	for cursor == "" {
		frame = readScopedFrame(t, scanner)
		if strings.Contains(frame.data, "background-private-body") {
			t.Fatal("background text crossed scope")
		}
		if frame.kind == "cursor" {
			cursor = frame.id
		}
	}
	resumed := open(cursor)
	defer resumed.Body.Close()
	next := bufio.NewScanner(resumed.Body)
	once.Do(func() { close(release) })
	for {
		frame = readScopedFrame(t, next)
		if frame.kind == "snapshot" {
			t.Fatal("valid scoped cursor forced a reset")
		}
		if strings.Contains(frame.data, "background-private-body") {
			t.Fatal("resumed observer received background text")
		}
		if frame.kind != "event" {
			continue
		}
		var value v1.Notification
		if err := protojson.Unmarshal([]byte(frame.data), &value); err != nil {
			t.Fatal(err)
		}
		if value.GetEvent().GetKind() == "request_finished" {
			if value.GetEvent().GetTaskId() != task.ID {
				t.Fatal(&value)
			}
			break
		}
	}
}
