// Purpose: Verify address scope and mixed DNS handling independently of HTTP.
// Pattern: Pure policy and deterministic resolver tests.
// Usage: go test ./internal/tools/browser/webfetch -run 'TestAddressPolicy|TestMixedResolution'.
// Related: internal/tools/browser/webfetch/destination.go, internal/tools/browser/webfetch/client.go.
package webfetch

import (
	"context"
	"net"
	"net/netip"
	"testing"
)

func TestAddressPolicy(t *testing.T) {
	for _, tc := range []struct {
		ip    string
		scope addressScope
	}{
		{"93.184.215.10", publicAddress},
		{"2606:4700::1111", publicAddress},
		{"::ffff:127.0.0.1", privateAddress},
		{"10.20.0.5", privateAddress},
		{"fd10::1234", privateAddress},
		{"169.254.169.254", blockedAddress},
		{"100.100.100.200", blockedAddress},
		{"fd00:ec2::254", blockedAddress},
		{"64:ff9b::a00:1", blockedAddress},
		{"2002:7f00:1::", blockedAddress},
		{"224.0.0.1", blockedAddress},
	} {
		if got := addressType(netip.MustParseAddr(tc.ip)); got != tc.scope {
			t.Errorf("%s classified as %d, want %d", tc.ip, got, tc.scope)
		}
	}
}

func TestMixedResolutionDoesNotDial(t *testing.T) {
	client := NewClient()
	client.lookup = func(_ context.Context, _, _ string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("93.184.215.10"), netip.MustParseAddr("127.0.0.1")}, nil
	}
	client.dial = func(context.Context, string, string) (net.Conn, error) {
		t.Error("a mixed-scope hostname reached connection setup")
		return nil, context.Canceled
	}
	target, err := parseURL("http://mixed.invalid/doc")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.open(t.Context(), target, target.String()); err == nil {
		t.Fatal("accepted a mixed DNS response")
	}
}
