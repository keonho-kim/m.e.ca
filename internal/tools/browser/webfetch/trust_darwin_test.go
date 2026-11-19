//go:build darwin

// Purpose: Exercise macOS offline trust evaluation with a private, per-object test anchor.
// Pattern: Native integration using an generated chain and a loopback lookup trap.
// Usage: go test ./internal/tools/browser/webfetch -run TestDarwinOfflineTrust.
// Related: internal/tools/browser/webfetch/trust_darwin.go.
package webfetch

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ebitengine/purego"
)

func TestDarwinOfflineTrust(t *testing.T) {
	var lookups atomic.Int32
	trap := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		w.WriteHeader(http.StatusForbidden)
	}))
	defer trap.Close()

	now := time.Now()
	template := func(serial int64, subject string) *x509.Certificate {
		return &x509.Certificate{
			SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: subject},
			NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
			BasicConstraintsValid: true,
		}
	}
	rootTemplate := template(1, "MECA WebFetch test root")
	rootTemplate.IsCA, rootTemplate.KeyUsage = true, x509.KeyUsageCertSign|x509.KeyUsageCRLSign
	root, rootKey := issueTestCertificate(t, rootTemplate, nil, nil)
	intermediateTemplate := template(2, "MECA WebFetch test issuer")
	intermediateTemplate.IsCA, intermediateTemplate.KeyUsage = true, x509.KeyUsageCertSign|x509.KeyUsageCRLSign
	issuer, issuerKey := issueTestCertificate(t, intermediateTemplate, root, rootKey)
	leafTemplate := template(3, "manual.example")
	leafTemplate.DNSNames = []string{"manual.example"}
	leafTemplate.KeyUsage = x509.KeyUsageDigitalSignature
	leafTemplate.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	leafTemplate.IssuingCertificateURL = []string{trap.URL + "/issuer.crt"}
	leafTemplate.OCSPServer = []string{trap.URL + "/ocsp"}
	leaf, _ := issueTestCertificate(t, leafTemplate, issuer, issuerKey)

	api, err := systemTrust()
	if err != nil {
		t.Fatal(err)
	}
	library, err := purego.Dlopen(securityFramework, purego.RTLD_LOCAL|purego.RTLD_NOW)
	if err != nil {
		t.Fatal(err)
	}
	defer purego.Dlclose(library)
	var setAnchors func(uintptr, uintptr) int32
	var networkFetchAllowed func(uintptr, *uint8) int32
	purego.RegisterLibFunc(&setAnchors, library, "SecTrustSetAnchorCertificates")
	purego.RegisterLibFunc(&networkFetchAllowed, library, "SecTrustGetNetworkFetchAllowed")
	anchors := api.createArray(0, 1, api.arrayTypeCallbacks)
	if anchors == 0 {
		t.Fatal("could not create test anchors")
	}
	defer api.release(anchors)
	ref := api.certificate(root.Raw)
	if ref == 0 {
		t.Fatal("could not create test root")
	}
	api.appendArray(anchors, ref)
	api.release(ref)

	evaluate := func(chain []*x509.Certificate) error {
		t.Helper()
		trust, err := api.offlineTrust("manual.example", chain)
		if err != nil {
			t.Fatal(err)
		}
		defer api.release(trust)
		var network uint8
		if status := networkFetchAllowed(trust, &network); status != 0 || network != 0 {
			t.Fatalf("certificate downloads enabled: status=%d allowed=%d", status, network)
		}
		if status := setAnchors(trust, anchors); status != 0 {
			t.Fatalf("could not apply test anchors: %d", status)
		}
		return api.evaluate(trust)
	}
	if err := evaluate([]*x509.Certificate{leaf}); err == nil {
		t.Fatal("trusted a server with a missing issuer")
	}
	if err := evaluate([]*x509.Certificate{leaf, issuer}); err != nil {
		t.Fatalf("rejected a complete local trust chain: %v", err)
	}
	var wrongHost x509.HostnameError
	if err := verifySystemTrust("different.example", []*x509.Certificate{leaf, issuer}); !errors.As(err, &wrongHost) {
		t.Fatalf("wrong hostname was not rejected: %v", err)
	}
	if hits := lookups.Load(); hits != 0 {
		t.Fatalf("TLS verification attempted %d external certificate lookups", hits)
	}
}

func issueTestCertificate(t *testing.T, template, issuer *x509.Certificate, issuerKey *ecdsa.PrivateKey) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if issuer == nil {
		issuer, issuerKey = template, key
	}
	der, err := x509.CreateCertificate(rand.Reader, template, issuer, &key.PublicKey, issuerKey)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return certificate, key
}
