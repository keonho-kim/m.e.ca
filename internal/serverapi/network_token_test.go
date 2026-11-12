// Purpose: Keep listener authentication bound to the same prepared token used for secret masking.
// Pattern: Network credential lifetime regression test.
// Usage: go test ./internal/serverapi -run TestListenerKeepsPreparedCredential.
// Related: cmd/meca/server.go, internal/serverapi/network.go, internal/serverapi/token.go.
package serverapi

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestListenerKeepsPreparedCredential(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte(testToken), 0600); err != nil {
		t.Fatal(err)
	}
	token := NewToken()
	if err := PublishToken(path, token); err != nil {
		t.Fatal(err)
	}
	// The fixture remembers this token before networking starts, as runServer
	// does. A later file replacement must not split authentication and masking.
	if token == testToken {
		t.Fatal("startup reused the old credential")
	}
	if err := f.api.Remember(token); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("new-token", 8)), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(deadline(t))
	address := make(chan string, 1)
	finished := make(chan struct{})
	var serveErr error
	go func() {
		defer close(finished)
		serveErr = ListenAndServe(ctx, f.api, token, Options{Protocol: "sse", Listen: "127.0.0.1:0"}, func(url string) { address <- url })
	}()
	t.Cleanup(func() {
		cancel()
		<-finished
		if serveErr != nil {
			t.Error(serveErr)
		}
	})
	var url string
	select {
	case url = <-address:
	case <-finished:
		t.Fatal(serveErr)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	body := `{"id":` + strconv.Quote(token) + `,"state":{}}`
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url+"/v1/command", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	defer client.CloseIdleConnections()
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("prepared credential rejected after file replacement: %s", res.Status)
	}
	if strings.Contains(string(data), token) || !strings.Contains(string(data), "[redacted]") {
		t.Fatal("authenticated credential was not masked in the reply")
	}
}

// A failed second listener must not invalidate the running listener's disk credential.
func TestFailedListenerDoesNotPublishCredential(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	path := filepath.Join(t.TempDir(), "server.token")
	if err := PublishToken(path, testToken); err != nil {
		t.Fatal(err)
	}
	token := NewToken()
	err = ListenAndServe(deadline(t), f.api, token, Options{Protocol: "sse", Listen: listener.Addr().String()}, func(string) {
		if err := PublishToken(path, token); err != nil {
			t.Error(err)
		}
	})
	if err == nil {
		t.Fatal("occupied address accepted")
	}
	body, err := os.ReadFile(path)
	if err != nil || strings.TrimSpace(string(body)) != testToken {
		t.Fatal("failed startup replaced active token", err)
	}
}
