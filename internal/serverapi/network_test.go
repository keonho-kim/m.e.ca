// Purpose: Verify actual TLS and loopback HTTP/2 startup and complete network shutdown.
// Pattern: Listener lifecycle integration tests with a local certificate.
// Usage: go test ./internal/serverapi -run 'TestNetwork|TestListener'.
// Related: internal/serverapi/network.go, internal/serverapi/security.go.
package serverapi

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"github.com/keonho-kim/meca/internal/serverapi/gen/mecav1/mecav1connect"
)

func TestNetworkRejectsRemoteCleartext(t *testing.T) {
	for _, address := range []string{"0.0.0.0:8787", "[::]:8787", "192.0.2.1:8787", ":8787", "localhost:8787"} {
		if err := validateNetwork(Options{Listen: address}); err == nil {
			t.Fatal("insecure bind accepted", address)
		}
	}
	for _, address := range []string{"127.0.0.1:0", "[::1]:8787"} {
		if err := validateNetwork(Options{Listen: address}); err != nil {
			t.Fatal(address, err)
		}
	}
}

func TestListenerShutdownClosesUnstartedConnectionAndDrainsResponse(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	ctx, stop := context.WithCancel(deadline(t))
	ready, done := make(chan string, 1), make(chan error, 1)
	started, finish := make(chan struct{}), make(chan struct{})
	finishResponse := sync.OnceFunc(func() { close(finish) })
	options := Options{Protocol: "sse", Listen: "127.0.0.1:0", UI: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-finish
		_, _ = io.WriteString(w, "finished")
	})}
	go func() {
		done <- ListenAndServe(ctx, f.api, testToken, options, func(address string) { ready <- address })
	}()
	defer func() {
		finishResponse()
		stop()
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	var address string
	select {
	case address = <-ready:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	dial := net.Dialer{}
	host := strings.TrimPrefix(address, "http://")
	unused, err := dial.DialContext(ctx, "tcp", host)
	if err != nil {
		t.Fatal(err)
	}
	defer unused.Close()
	active, err := dial.DialContext(ctx, "tcp", host)
	if err != nil {
		t.Fatal(err)
	}
	defer active.Close()
	if err := active.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprintf(active, "GET /ui/ HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", host); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	stop()
	if err := unused.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := unused.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Fatalf("unstarted HTTP connection did not close during shutdown: %v", err)
	}
	finishResponse()
	response, err := http.ReadResponse(bufio.NewReader(active), nil)
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(response.Body)
	if err := errors.Join(readErr, response.Body.Close()); err != nil || string(body) != "finished" {
		t.Fatalf("active response was not drained: %q %v", body, err)
	}
}

func TestListenerTLSAndLoopbackHTTP2(t *testing.T) {
	for _, encrypted := range []bool{false, true} {
		t.Run(map[bool]string{false: "h2c", true: "tls"}[encrypted], func(t *testing.T) {
			f := newFixture(t, generateFunc(idleGenerator), false)
			ctx, cancel := context.WithCancel(deadline(t))
			defer cancel()
			directory := t.TempDir()
			options := Options{Protocol: "grpc", Listen: "127.0.0.1:0"}
			token := NewToken()
			transport := &http.Transport{Protocols: new(http.Protocols)}
			transport.Protocols.SetHTTP2(true)
			if encrypted {
				certServer := httptest.NewTLSServer(http.NotFoundHandler())
				cert := certServer.TLS.Certificates[0]
				certServer.Close()
				options.Certificate, options.Key = filepath.Join(directory, "cert.pem"), filepath.Join(directory, "key.pem")
				if err := os.WriteFile(options.Certificate, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}), 0600); err != nil {
					t.Fatal(err)
				}
				key, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(options.Key, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}), 0600); err != nil {
					t.Fatal(err)
				}
				pool := x509.NewCertPool()
				leaf, err := x509.ParseCertificate(cert.Certificate[0])
				if err != nil {
					t.Fatal(err)
				}
				pool.AddCert(leaf)
				transport.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
			} else {
				transport.Protocols.SetUnencryptedHTTP2(true)
			}
			defer transport.CloseIdleConnections()
			address := make(chan string, 1)
			done := make(chan error, 1)
			go func() { done <- ListenAndServe(ctx, f.api, token, options, func(url string) { address <- url }) }()
			var url string
			select {
			case url = <-address:
			case err := <-done:
				t.Fatal(err)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			client := &http.Client{Transport: tokenTransport{base: transport, token: token}}
			rpc := mecav1connect.NewMecaServiceClient(connect.NewClient(connecthttp.NewTransport(client, url, connecthttp.WithGRPC())))
			if _, err := rpc.Call(ctx, &v1.Command{Operation: &v1.Command_State{State: &v1.StateRequest{}}}); err != nil {
				t.Fatal(err)
			}
			watch, err := rpc.WatchEvents(ctx, &v1.WatchRequest{})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := watch.Receive(); err != nil {
				t.Fatal(err)
			}
			cancel()
			if _, err := watch.Receive(); err == nil {
				t.Fatal("observer survived shutdown")
			}
			_ = watch.Close()
			if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}

type tokenTransport struct {
	base  http.RoundTripper
	token string
}

func (b tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	copy := req.Clone(req.Context())
	copy.Header.Set("Authorization", "Bearer "+b.token)
	return b.base.RoundTrip(copy)
}

type timedWriter struct {
	*httptest.ResponseRecorder
	deadline time.Time
	bounded  bool
}

func (w *timedWriter) SetWriteDeadline(value time.Time) error { w.deadline = value; return nil }
func (w *timedWriter) Write(body []byte) (int, error) {
	w.bounded = !w.deadline.IsZero()
	return w.ResponseRecorder.Write(body)
}
func (w *timedWriter) Flush() {
	w.bounded = w.bounded && !w.deadline.IsZero()
	w.ResponseRecorder.Flush()
}

func TestNetworkWriteDeadlineDoesNotExpireIdleObservers(t *testing.T) {
	underlying := &timedWriter{ResponseRecorder: httptest.NewRecorder()}
	writer := deadlineWriter{ResponseWriter: underlying, ctx: t.Context()}
	if _, err := writer.Write([]byte("event")); err != nil {
		t.Fatal(err)
	}
	if !underlying.bounded || !underlying.deadline.IsZero() {
		t.Fatal("deadline survived a completed write")
	}
	if err := writer.FlushError(); err != nil {
		t.Fatal(err)
	}
	if !underlying.bounded || !underlying.deadline.IsZero() {
		t.Fatal("idle observer retained a write deadline")
	}
}
