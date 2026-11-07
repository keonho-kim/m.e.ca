// Purpose: Release a completed large command while its socket waits for new input.
// Pattern: Loopback resource-lifetime regression test.
// Usage: go test ./internal/serverapi -run TestSocketReleasesCompletedCommand.
// Related: internal/serverapi/socket.go, internal/serverapi/wire_test.go.
package serverapi

import (
	"runtime"
	"strings"
	"testing"

	"github.com/coder/websocket"
)

func TestSocketReleasesCompletedCommand(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	w := tlsWire(t, "socket", f.api)
	ctx := deadline(t)
	conn, _, err := websocket.Dial(ctx, strings.Replace(w.url, "https:", "wss:", 1)+"/v1/socket", &websocket.DialOptions{HTTPClient: w.http})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.CloseNow()
	exchange := func(body string) {
		t.Helper()
		if err := conn.Write(ctx, websocket.MessageText, []byte(body)); err != nil {
			t.Fatal(err)
		}
		if _, _, err := conn.Read(ctx); err != nil {
			t.Fatal(err)
		}
	}
	heap := func() uint64 {
		runtime.GC()
		runtime.GC()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return m.HeapAlloc
	}
	exchange(`{"authenticate":"` + testToken + `"}`)
	exchange(`{"command":{"id":"large","start":{"sessionId":"session_AAAAAAAAAAAAAAAAAAAAAAAAAA","key":"audit-key","text":"` + strings.Repeat("x", 8<<20) + `"}}}`)
	large := heap()
	exchange(`{"command":{"id":"small","state":{}}}`)
	small := heap()
	if int64(large)-int64(small) > 2<<20 {
		t.Fatal("idle connection retained its completed command", int64(large)-int64(small))
	}
}
