// Purpose: Bind one selected transport with explicit TLS policy and joined network shutdown.
// Pattern: Network lifecycle adapter.
// Usage: cmd/meca calls ListenAndServe after composing the server application.
// Related: cmd/meca/server.go, internal/serverapi/security.go.
package serverapi

import (
	"context"
	"crypto/tls"
	"errors"

	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/keonho-kim/meca/internal/diagnostics"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type Options struct {
	Protocol, Listen, Certificate, Key string
	Hosts, Origins                     []string
	OpenAI                             bool
}

func validateNetwork(options Options) error {
	host, port, err := net.SplitHostPort(options.Listen)
	if err != nil {
		return faults.Errorf(diagcatalog.ServerapiListenAddressMustBeIPPort, err)
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 0 || n > 65535 {
		return faults.Errorf(diagcatalog.ServerapiInvalidListenPort)
	}
	ip := net.ParseIP(host)
	if host != "" && ip == nil {
		return faults.Errorf(diagcatalog.ServerapiListenHostMustBeAnExplicitIP)
	}
	if (options.Certificate == "") != (options.Key == "") {
		return faults.Errorf(diagcatalog.ServerapiTLSCertificateAndKeyAreBothRequired)
	}
	if !ip.IsLoopback() && options.Certificate == "" {
		return faults.Errorf(diagcatalog.ServerapiNonLoopbackListenersRequireTlsCertAnd)
	}
	if (ip == nil || ip.IsUnspecified()) && len(options.Hosts) == 0 {
		return faults.Errorf(diagcatalog.ServerapiWildcardListenersRequireAtLeastOneAllow)
	}
	return nil
}

func ListenAndServe(ctx context.Context, api *API, token string, options Options, ready func(string)) error {
	if err := validateNetwork(options); err != nil {
		return err
	}
	var certificate tls.Certificate
	if options.Certificate != "" {
		var err error
		certificate, err = tls.LoadX509KeyPair(options.Certificate, options.Key)
		if err != nil {
			return faults.Errorf(diagcatalog.ServerapiLoadTLSCertificate, err)
		}
	}
	listener, err := net.Listen("tcp", options.Listen)
	if err != nil {
		return err
	}
	listener = &connectionListener{Listener: listener, slots: make(chan struct{}, maxConnections)}
	defer listener.Close()
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	hosts := append([]string(nil), options.Hosts...)
	ip := net.ParseIP(host)
	if !ip.IsUnspecified() {
		hosts = append(hosts, listener.Addr().String())
	}
	if ip.IsLoopback() {
		hosts = append(hosts, net.JoinHostPort("localhost", port), net.JoinHostPort("127.0.0.1", port), net.JoinHostPort("::1", port))
	}
	transport, err := newTransport(ctx, api, options.Protocol, token, hosts, options.Origins)
	if err != nil {
		return err
	}
	defer transport.Close()
	if options.OpenAI {
		transport.enableOpenAI()
	}
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)
	protocols.SetUnencryptedHTTP2(options.Protocol == "grpc" && options.Certificate == "")
	srv := &http.Server{Handler: transport, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10, Protocols: protocols, ErrorLog: diagnostics.HTTPLogger(ctx), BaseContext: func(net.Listener) context.Context { return ctx }}
	srv.HTTP2 = &http.HTTP2Config{MaxConcurrentStreams: maxRequests}
	if options.Certificate != "" {
		srv.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{certificate}}
	}
	done := make(chan error, 1)
	go func() {
		if options.Certificate != "" {
			done <- srv.ServeTLS(listener, "", "")
		} else {
			done <- srv.Serve(listener)
		}
	}()
	scheme := "http"
	if options.Certificate != "" {
		scheme = "https"
	}
	if ready != nil {
		ready(scheme + "://" + listener.Addr().String())
	}
	var serveErr error
	select {
	case serveErr = <-done:
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = srv.Shutdown(shutdown)
		cancel()
		if err != nil {
			_ = srv.Close()
		}
		serveErr = <-done
	}
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	return errors.Join(serveErr, err)
}

type deadlineWriter struct {
	http.ResponseWriter
	ctx      context.Context
	reported sync.Once
}

func (w *deadlineWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *deadlineWriter) Write(body []byte) (int, error) {
	controller := http.NewResponseController(w.ResponseWriter)
	_ = controller.SetWriteDeadline(time.Now().Add(15 * time.Second))
	defer controller.SetWriteDeadline(time.Time{})
	n, err := w.ResponseWriter.Write(body)
	if err == nil && n != len(body) {
		err = io.ErrShortWrite
	}
	w.ReportError(err)
	if err != nil {
		err = faults.New(diagcatalog.DeliveryFailed, diagcatalog.WrappedCause, err)
	}
	return n, err
}
func (w *deadlineWriter) Flush() { _ = w.FlushError() }
func (w *deadlineWriter) FlushError() error {
	controller := http.NewResponseController(w.ResponseWriter)
	_ = controller.SetWriteDeadline(time.Now().Add(15 * time.Second))
	defer controller.SetWriteDeadline(time.Time{})
	err := controller.Flush()
	w.ReportError(err)
	if err != nil {
		return faults.New(diagcatalog.DeliveryFailed, diagcatalog.WrappedCause, err)
	}
	return err
}

func (w *deadlineWriter) ReportError(err error) {
	if err == nil {
		return
	}
	w.reported.Do(func() {
		diagnostics.Log(w.ctx, diagcatalog.ResponseDeliveryFailed, "error_code", diagcatalog.DeliveryFailed)
	})
}
