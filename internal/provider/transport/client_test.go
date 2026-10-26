// Purpose: Verify HTTP/2 negotiation, HTTP/1.1 fallback and streaming cancellation.
// Pattern: Local TLS transport contract tests.
// Usage: Run TestProviderNegotiatedStream without external endpoints or credentials.
// Related: internal/provider/transport/client.go, internal/provider/transport/idle.go.
package transport

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProviderNegotiatedStream(t *testing.T) {
	for _, version := range []int{1, 2} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			finished := make(chan struct{})
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				defer close(finished)
				if request.Header.Get("Authorization") != "Bearer fixture" {
					http.Error(w, "missing authentication", http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, "data: first\n\n")
				w.(http.Flusher).Flush()
				<-request.Context().Done()
			}))
			server.EnableHTTP2 = version == 2
			server.StartTLS()
			defer server.Close()

			client := New("fixture")
			defer client.Close()
			roots := x509.NewCertPool()
			roots.AddCert(server.Certificate())
			client.HTTP.Transport.(*http.Transport).TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			request, err := client.NewRequest(ctx, http.MethodGet, server.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			response, err := client.SendOnce(request, time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.ProtoMajor != version {
				t.Fatalf("negotiated %s, want HTTP/%d", response.Proto, version)
			}
			reader := bufio.NewReader(response.Body)
			if line, err := reader.ReadString('\n'); err != nil || line != "data: first\n" {
				t.Fatalf("first frame %q: %v", line, err)
			}
			cancel()
			if _, err := io.ReadAll(reader); err == nil {
				t.Fatal("cancelled stream completed without an error")
			}
			select {
			case <-finished:
			case <-time.After(5 * time.Second):
				t.Fatal("server stream did not release after cancellation")
			}
		})
	}
}
