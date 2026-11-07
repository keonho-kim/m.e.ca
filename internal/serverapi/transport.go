// Purpose: Compose one HTTP transport and own handler admission, cancellation and shutdown joins.
// Pattern: Transport lifecycle adapter.
// Usage: network.go creates Transport; every protocol runs through its common HTTP lifetime boundary.
// Related: internal/serverapi/security.go, internal/serverapi/network.go.
package serverapi

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/keonho-kim/meca/internal/serverapi/admission"
	"github.com/keonho-kim/meca/internal/serverapi/openai"
)

const maxRequests = 32

type Transport struct {
	inputs         *admission.Budget
	api            *API
	protocol       string
	handler        http.Handler
	token          [32]byte
	hosts, origins map[string]bool
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.Mutex
	closing        bool
	workers        sync.WaitGroup
	requests       chan struct{}
	streams        chan struct{}
	openAI         bool
}

func newTransport(ctx context.Context, api *API, protocol, token string, hosts, origins []string) (*Transport, error) {
	if api == nil || api.Service == nil || !validToken(token) {
		return nil, fmt.Errorf("server service and private token are required")
	}
	ctx, cancel := context.WithCancel(ctx)
	t := &Transport{api: api, protocol: protocol, token: sha256.Sum256([]byte(token)), hosts: map[string]bool{}, origins: map[string]bool{}, ctx: ctx, cancel: cancel, requests: make(chan struct{}, maxRequests), streams: make(chan struct{}, 16), inputs: admission.New(32 << 20)}
	for _, host := range hosts {
		if !validHost(host) {
			cancel()
			return nil, fmt.Errorf("allow-host requires an exact HTTP Host authority")
		}
		t.hosts[strings.ToLower(host)] = true
	}
	for _, origin := range origins {
		if !validOrigin(origin) {
			cancel()
			return nil, fmt.Errorf("allow-origin requires an exact http(s) origin without a path")
		}
		t.origins[origin] = true
	}
	mux := http.NewServeMux()
	switch protocol {
	case "sse":
		mux.HandleFunc("POST /v1/command", t.command)
		mux.HandleFunc("GET /v1/events", t.sse)
	case "socket":
		mux.HandleFunc("GET /v1/socket", t.socket)
	case "grpc":
		mountRPC(mux, api)
	default:
		cancel()
		return nil, fmt.Errorf("protocol must be sse, socket or grpc")
	}
	t.handler = mux
	return t, nil
}

func (t *Transport) enableOpenAI() {
	t.openAI = true
	(&openai.Handler{Service: t.api.Service, Redact: t.api.Redact, Streams: t.streams, Reads: t.api.readLimit()}).Mount(t.handler.(*http.ServeMux))
}

func (t *Transport) reject(w http.ResponseWriter, r *http.Request, message string, status int) {
	if t.openAI && openai.IsPath(r.URL.Path) {
		code := map[int]string{400: "invalid_request", 401: "invalid_api_key", 403: "forbidden", 429: "resource_exhausted", 503: "unavailable"}[status]
		openai.WriteError(w, status, message, code, "")
		return
	}
	http.Error(w, message, status)
}

func (t *Transport) socketHandshake(r *http.Request) bool {
	return t.protocol == "socket" && r.Method == "GET" && r.URL.Path == "/v1/socket" && strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

func (t *Transport) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w = deadlineWriter{w}
	t.mu.Lock()
	if t.closing {
		t.mu.Unlock()
		t.reject(w, r, "server is shutting down", http.StatusServiceUnavailable)
		return
	}
	t.workers.Add(1)
	t.mu.Unlock()
	defer t.workers.Done()
	ctx, cancel := context.WithCancel(r.Context())
	stop := context.AfterFunc(t.ctx, cancel)
	defer stop()
	defer cancel()
	r = r.WithContext(ctx)
	if !t.authorize(w, r) {
		return
	}
	select {
	case t.requests <- struct{}{}:
		defer func() { <-t.requests }()
	default:
		t.reject(w, r, "too many connections", http.StatusTooManyRequests)
		return
	}
	if t.protocol == "grpc" && r.URL.Path == "/meca.v1.MecaService/WatchEvents" {
		select {
		case t.streams <- struct{}{}:
			defer func() { <-t.streams }()
		default:
			http.Error(w, "too many observers", http.StatusTooManyRequests)
			return
		}
	}
	if !t.socketHandshake(r) && r.Body != nil && r.Body != http.NoBody {
		weight := r.ContentLength
		if weight <= 0 || weight > maxRequestBytes || r.Header.Get("Content-Encoding") != "" || r.Header.Get("Grpc-Encoding") != "" || strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			weight = maxRequestBytes
		}
		release, ok := t.inputs.Try(weight)
		if !ok {
			t.reject(w, r, "server input memory budget is occupied", http.StatusTooManyRequests)
			return
		}
		defer release()
		r = r.WithContext(admission.WithInput(r.Context(), release))
		controller := http.NewResponseController(w)
		_ = controller.SetReadDeadline(time.Now().Add(30 * time.Second))
		r.Body = deadlineBody{ReadCloser: r.Body, clear: func() { _ = controller.SetReadDeadline(time.Time{}) }}
		defer controller.SetReadDeadline(time.Time{})
	}
	t.handler.ServeHTTP(w, r)
}

// The deadline bounds input reading, not the lifetime of a streaming reply.
type deadlineBody struct {
	io.ReadCloser
	clear func()
}

func (b deadlineBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err != nil {
		b.clear()
	}
	return n, err
}
func (b deadlineBody) Close() error { b.clear(); return b.ReadCloser.Close() }

func (t *Transport) Close() {
	t.mu.Lock()
	t.closing = true
	t.cancel()
	t.mu.Unlock()
	t.workers.Wait()
}
