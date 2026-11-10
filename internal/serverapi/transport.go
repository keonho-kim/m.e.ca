// Purpose: Compose one HTTP transport and own handler admission, cancellation and shutdown joins.
// Pattern: Transport lifecycle adapter.
// Usage: network.go creates Transport; every protocol runs through its common HTTP lifetime boundary.
// Related: internal/serverapi/security.go, internal/serverapi/network.go.
package serverapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"

	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/keonho-kim/meca/internal/diagnostics"

	"github.com/keonho-kim/meca/internal/agent/media"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/serverapi/admission"
)

const (
	maxRequests        = 32
	maxControlRequests = 4
	maxHTTP2Streams    = maxRequests + maxControlRequests + 1
)

type Transport struct {
	ui             http.Handler
	uiSeed         string
	browser        *browserSessions
	inputs         *admission.Budget
	mediaInputs    *admission.Budget
	thumbnails     thumbnailCache
	outputs        *admission.Budget
	outputOnce     sync.Once
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
	controls       chan struct{}
	streams        chan struct{}
}

func newTransport(ctx context.Context, api *API, protocol, token string, hosts, origins []string) (*Transport, error) {
	if api == nil || api.Service == nil || !validToken(token) {
		return nil, faults.Errorf(diagcatalog.ServerapiServerServiceAndPrivateTokenAreRequired)
	}
	ctx, cancel := context.WithCancel(ctx)
	t := &Transport{api: api, protocol: protocol, token: sha256.Sum256([]byte(token)), hosts: map[string]bool{}, origins: map[string]bool{}, ctx: ctx, cancel: cancel, requests: make(chan struct{}, maxRequests), streams: make(chan struct{}, 16), inputs: admission.New(32 << 20), mediaInputs: admission.New(2 * media.MaxBytes)}
	for _, host := range hosts {
		if !validHost(host) {
			cancel()
			return nil, faults.Errorf(diagcatalog.ServerapiAllowHostRequiresAnExactHTTPHost)
		}
		t.hosts[strings.ToLower(host)] = true
	}
	for _, origin := range origins {
		if !validOrigin(origin) {
			cancel()
			return nil, faults.Errorf(diagcatalog.ServerapiAllowOriginRequiresAnExactHttpS)
		}
		t.origins[origin] = true
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/command", t.command)
	mux.HandleFunc("GET /v1/events", t.sse)
	mux.HandleFunc("GET /v1/files/events", t.filesWatch)
	mux.HandleFunc("/v1/tasks/", t.taskControl)
	mux.HandleFunc("/v1/sessions/", t.historyImage)
	mux.HandleFunc("/v1/workspaces/", t.workspaceHTTP)
	mux.HandleFunc("/v1/attachments", t.attachment)
	mux.HandleFunc("/v1/attachments/", t.attachment)
	switch protocol {
	case "sse":
	case "socket":
		mux.HandleFunc("GET /v1/socket", t.socket)
	case "grpc":
		mountRPC(mux, api)
	default:
		cancel()
		return nil, faults.Errorf(diagcatalog.StartupProtocolMustBeSseSocketOrGrpc)
	}
	t.handler = mux
	t.controls = make(chan struct{}, maxControlRequests)
	return t, nil
}

func (t *Transport) socketHandshake(r *http.Request) bool {
	return t.protocol == "socket" && r.Method == "GET" && r.URL.Path == "/v1/socket" && strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

func (t *Transport) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.mu.Lock()
	if t.closing {
		t.mu.Unlock()
		http.Error(w, diagcatalog.Text(diagcatalog.ServerServerIsShuttingDown), http.StatusServiceUnavailable)
		return
	}
	t.workers.Add(1)
	t.mu.Unlock()
	defer t.workers.Done()
	ctx, cancel := context.WithCancel(r.Context())
	ctx = diagnostics.WithFields(diagnostics.WithSinkContext(ctx, t.ctx), "component", "http", "request_id", rand.Text())
	ctx = diagnostics.WithRedactor(ctx, t.api.Redact)
	stop := context.AfterFunc(t.ctx, cancel)
	defer stop()
	defer cancel()
	r = r.WithContext(ctx)
	w = &deadlineWriter{ResponseWriter: w, ctx: ctx}
	if !t.authorize(w, r) {
		if r.Method != "OPTIONS" {
			diagnostics.Log(ctx, diagcatalog.AccessRejected, "error_code", diagcatalog.AccessRejectedCode)
		}
		return
	}
	if t.ui != nil && (r.URL.Path == "/" || r.URL.Path == "/ui" || strings.HasPrefix(r.URL.Path, "/ui/")) && (r.Method == "GET" || r.Method == "HEAD") {
		t.handler.ServeHTTP(w, r)
		return
	}
	admissionSlots := t.requests
	if r.Method == "POST" && (strings.HasPrefix(r.URL.Path, "/v1/tasks/") || strings.HasPrefix(r.URL.Path, "/v1/workspaces/") && strings.Contains(r.URL.Path, "/tasks/")) {
		admissionSlots = t.controls
	}
	select {
	case admissionSlots <- struct{}{}:
		defer func() { <-admissionSlots }()
	default:
		http.Error(w, diagcatalog.Text(diagcatalog.ServerapiTooManyConnections), http.StatusTooManyRequests)
		return
	}
	if t.protocol == "grpc" && (r.URL.Path == "/meca.v1.MecaService/WatchEvents" || r.URL.Path == "/meca.v1.MecaService/WatchFiles") {
		select {
		case t.streams <- struct{}{}:
			defer func() { <-t.streams }()
		default:
			http.Error(w, diagcatalog.Text(diagcatalog.ServerapiTooManyObservers), http.StatusTooManyRequests)
			return
		}
	}
	if !t.socketHandshake(r) && r.Body != nil && r.Body != http.NoBody {
		budget := t.inputs
		limit := int64(maxRequestBytes)
		if r.Method == "POST" && r.URL.Path == "/v1/attachments" {
			budget = t.mediaInputs
			limit = media.MaxBytes
		}
		weight := r.ContentLength
		if weight < 0 || weight > limit {
			weight = 0
		}
		if encoding := r.Header.Get("Grpc-Encoding"); encoding != "" && encoding != "identity" || r.Header.Get("Content-Encoding") != "" {
			// Bound expansion before the RPC adapter allocates a decoded message.
			// Call/Watch resize or release this reservation immediately after decoding.
			weight = maxRequestBytes
		}
		input, err := admission.NewInput(budget, weight)
		if err != nil {
			http.Error(w, diagcatalog.Text(diagcatalog.ServerapiServerInputMemoryBudgetIsOccupied), http.StatusTooManyRequests)
			return
		}
		defer input.Release()
		r = r.WithContext(admission.WithInput(r.Context(), input))
		r.Body = &inputBody{Reader: input.Reader(r.Body), Closer: r.Body}
		controller := http.NewResponseController(w)
		_ = controller.SetReadDeadline(time.Now().Add(30 * time.Second))
		r.Body = deadlineBody{ReadCloser: r.Body, clear: func() { _ = controller.SetReadDeadline(time.Time{}) }}
		defer controller.SetReadDeadline(time.Time{})
	}
	t.handler.ServeHTTP(w, r)
}

type inputBody struct {
	io.Reader
	io.Closer
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
