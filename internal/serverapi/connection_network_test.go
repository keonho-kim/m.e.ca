// Purpose: Verify the real listener rejects idle TCP connections beyond its process admission limit.
// Pattern: Loopback HTTP lifecycle integration test.
// Usage: go test -race ./internal/serverapi -run TestNetworkConnectionLimit.
// Related: internal/serverapi/network.go, internal/serverapi/connections.go.
package serverapi

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestNetworkConnectionLimitIncludesIdlePeers(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	ctx, cancel := context.WithCancel(deadline(t))
	ready, done := make(chan string, 1), make(chan error, 1)
	go func() {
		done <- ListenAndServe(ctx, f.api, testToken, Options{Protocol: "sse", Listen: "127.0.0.1:0"}, func(address string) { ready <- address })
	}()
	var peers []net.Conn
	defer func() {
		for _, peer := range peers {
			peer.Close()
		}
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(5 * time.Second):
			t.Error("listener did not shut down")
		}
	}()
	var address string
	select {
	case address = <-ready:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	parsed, err := url.Parse(address)
	if err != nil {
		t.Fatal(err)
	}
	for i := range maxConnections + 1 {
		peer, err := (&net.Dialer{}).DialContext(ctx, "tcp", parsed.Host)
		if err != nil {
			t.Fatal(err)
		}
		peers = append(peers, peer)
		peer.SetDeadline(time.Now().Add(2 * time.Second))
		_, err = fmt.Fprintf(peer, "GET /connection-limit HTTP/1.1\r\nHost: %s\r\nAuthorization: Bearer %s\r\n\r\n", parsed.Host, testToken)
		if err != nil {
			if i == maxConnections {
				return
			}
			t.Fatal(err)
		}
		response, err := http.ReadResponse(bufio.NewReader(peer), nil)
		if i == maxConnections {
			if err == nil {
				response.Body.Close()
				t.Fatal("idle TCP connections bypassed the listener limit")
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.Copy(io.Discard, response.Body)
		closeErr := response.Body.Close()
		if err != nil || closeErr != nil || response.Close {
			t.Fatal("connection did not remain idle", err, closeErr)
		}
		peer.SetDeadline(time.Time{})
	}
}
