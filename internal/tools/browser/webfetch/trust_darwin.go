//go:build darwin

// Purpose: Verify WebFetch TLS with macOS trust settings and offline Security.framework policy.
// Pattern: Build-selected pinned TLS dialer and native trust adapter.
// Usage: The private transport validates each new HTTPS connection before sending HTTP.
// Related: internal/tools/browser/webfetch/client.go, internal/diagnostics/catalog/webfetch.go.
package webfetch

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"runtime"
	"sync"

	"github.com/ebitengine/purego"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func configureOfflineTLS(transport *http.Transport) {
	dial := transport.DialContext
	timeout := transport.TLSHandshakeTimeout
	transport.DialTLSContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		raw, err := dial(ctx, network, address)
		if err != nil {
			return nil, err
		}
		connection := tls.Client(raw, &tls.Config{
			ServerName: host,
			MinVersion: tls.VersionTLS12,
			// The default macOS verifier can perform network lookups.
			// The callback below checks both the host and the local trust chain.
			InsecureSkipVerify: true,
			VerifyConnection: func(state tls.ConnectionState) error {
				return verifySystemTrust(host, state.PeerCertificates)
			},
		})
		handshake, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		if err := connection.HandshakeContext(handshake); err != nil {
			return nil, errors.Join(err, raw.Close())
		}
		return connection, nil
	}
}

func verifySystemTrust(host string, certificates []*x509.Certificate) error {
	if len(certificates) == 0 {
		return fault.New(diag.PermissionDenied, diag.WebFetchTLSUntrusted)
	}
	// The dial authority includes IP literals, which TLS omits from SNI.
	if err := certificates[0].VerifyHostname(host); err != nil {
		return err
	}
	api, err := systemTrust()
	if err != nil {
		return err
	}
	trust, err := api.offlineTrust(host, certificates)
	if err != nil {
		return err
	}
	defer api.release(trust)
	return api.evaluate(trust)
}

type trustAPI struct {
	release            func(uintptr)
	createArray        func(uintptr, int, uintptr) uintptr
	appendArray        func(uintptr, uintptr)
	createData         func(uintptr, *byte, int) uintptr
	createString       func(uintptr, string, uint32) uintptr
	createCertificate  func(uintptr, uintptr) uintptr
	createSSLPolicy    func(uint8, uintptr) uintptr
	createRevocation   func(uint) uintptr
	createTrust        func(uintptr, uintptr, *uintptr) int32
	setNetworkFetch    func(uintptr, uint8) int32
	evaluateTrust      func(uintptr, *uintptr) bool
	arrayTypeCallbacks uintptr
}

const coreFoundation = "/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation"
const securityFramework = "/System/Library/Frameworks/Security.framework/Security"

var systemTrust = sync.OnceValues(loadSystemTrust)

func loadSystemTrust() (_ *trustAPI, err error) {
	cf, err := purego.Dlopen(coreFoundation, purego.RTLD_LOCAL|purego.RTLD_NOW)
	if err != nil {
		return nil, fault.New(diag.Unavailable, diag.WebFetchTLSUnavailable)
	}
	defer func() {
		if err != nil {
			purego.Dlclose(cf)
		}
	}()
	security, err := purego.Dlopen(securityFramework, purego.RTLD_LOCAL|purego.RTLD_NOW)
	if err != nil {
		return nil, fault.New(diag.Unavailable, diag.WebFetchTLSUnavailable)
	}
	defer func() {
		if err != nil {
			purego.Dlclose(security)
		}
	}()
	api := &trustAPI{}
	for _, symbol := range []struct {
		library uintptr
		name    string
		target  any
	}{
		{cf, "CFRelease", &api.release},
		{cf, "CFArrayCreateMutable", &api.createArray},
		{cf, "CFArrayAppendValue", &api.appendArray},
		{cf, "CFDataCreate", &api.createData},
		{cf, "CFStringCreateWithCString", &api.createString},
		{security, "SecCertificateCreateWithData", &api.createCertificate},
		{security, "SecPolicyCreateSSL", &api.createSSLPolicy},
		{security, "SecPolicyCreateRevocation", &api.createRevocation},
		{security, "SecTrustCreateWithCertificates", &api.createTrust},
		{security, "SecTrustSetNetworkFetchAllowed", &api.setNetworkFetch},
		{security, "SecTrustEvaluateWithError", &api.evaluateTrust},
	} {
		address, err := purego.Dlsym(symbol.library, symbol.name)
		if err != nil {
			return nil, fault.New(diag.Unavailable, diag.WebFetchTLSUnavailable)
		}
		purego.RegisterFunc(symbol.target, address)
	}
	api.arrayTypeCallbacks, err = purego.Dlsym(cf, "kCFTypeArrayCallBacks")
	if err != nil {
		return nil, fault.New(diag.Unavailable, diag.WebFetchTLSUnavailable)
	}
	// Keep both frameworks loaded for the process lifetime of these bindings.
	return api, nil
}

func (api *trustAPI) certificate(der []byte) uintptr {
	if len(der) == 0 {
		return 0
	}
	data := api.createData(0, &der[0], len(der))
	runtime.KeepAlive(der)
	if data == 0 {
		return 0
	}
	defer api.release(data)
	return api.createCertificate(0, data)
}

func (api *trustAPI) offlineTrust(host string, certificates []*x509.Certificate) (uintptr, error) {
	unavailable := fault.New(diag.Unavailable, diag.WebFetchTLSUnavailable)
	certs := api.createArray(0, len(certificates), api.arrayTypeCallbacks)
	if certs == 0 {
		return 0, unavailable
	}
	defer api.release(certs)
	for _, certificate := range certificates {
		value := api.certificate(certificate.Raw)
		if value == 0 {
			return 0, fault.New(diag.PermissionDenied, diag.WebFetchTLSUntrusted)
		}
		api.appendArray(certs, value)
		api.release(value)
	}
	const utf8Encoding = 0x08000100 // kCFStringEncodingUTF8
	name := api.createString(0, host, utf8Encoding)
	if name == 0 {
		return 0, unavailable
	}
	defer api.release(name)
	ssl := api.createSSLPolicy(1, name)
	if ssl == 0 {
		return 0, unavailable
	}
	defer api.release(ssl)
	const revocationAnyMethod = (1 << 0) | (1 << 1) // kSecRevocationUseAnyAvailableMethod
	const noRevocationNetwork = 1 << 4              // kSecRevocationNetworkAccessDisabled
	revocation := api.createRevocation(revocationAnyMethod | noRevocationNetwork)
	if revocation == 0 {
		return 0, unavailable
	}
	defer api.release(revocation)
	policies := api.createArray(0, 2, api.arrayTypeCallbacks)
	if policies == 0 {
		return 0, unavailable
	}
	defer api.release(policies)
	api.appendArray(policies, ssl)
	api.appendArray(policies, revocation)
	var trust uintptr
	status := api.createTrust(certs, policies, &trust)
	if status != 0 || trust == 0 {
		if trust != 0 {
			api.release(trust)
		}
		return 0, unavailable
	}
	if api.setNetworkFetch(trust, 0) != 0 {
		api.release(trust)
		return 0, unavailable
	}
	return trust, nil
}

func (api *trustAPI) evaluate(trust uintptr) error {
	var detail uintptr
	trusted := api.evaluateTrust(trust, &detail)
	if detail != 0 {
		api.release(detail)
	}
	if !trusted {
		return fault.New(diag.PermissionDenied, diag.WebFetchTLSUntrusted)
	}
	return nil
}
