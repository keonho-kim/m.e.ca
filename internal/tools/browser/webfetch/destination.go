// Purpose: Resolve document addresses once and classify actual TCP destinations.
// Pattern: DNS adapter with a pure address policy.
// Usage: The HTTP client creates a document connection from the validated address set.
// Related: internal/tools/browser/webfetch/client.go, internal/tools/browser/webfetch/url.go.
package webfetch

import (
	"context"
	"net/netip"
	"net/url"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type addressScope uint8

const (
	blockedAddress addressScope = iota
	publicAddress
	privateAddress
)

type destination struct {
	origin
	scope addressScope
	ips   []netip.Addr
}

var specialPurposeRanges = [...]netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001::/32"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("3fff::/20"),
}

var globalIPv6 = netip.MustParsePrefix("2000::/3")

var ipv6Metadata = [...]netip.Addr{
	netip.MustParseAddr("fd00:ec2::254"),
	netip.MustParseAddr("fd20:ce::254"),
}

func addressType(ip netip.Addr) addressScope {
	ip = ip.Unmap()
	if ip.Zone() != "" || !ip.IsGlobalUnicast() && !ip.IsLoopback() {
		return blockedAddress
	}
	for _, addr := range ipv6Metadata {
		if ip == addr {
			return blockedAddress
		}
	}
	if ip.IsLoopback() || ip.IsPrivate() {
		return privateAddress
	}
	if ip.Is6() && !globalIPv6.Contains(ip) {
		return blockedAddress
	}
	for _, prefix := range specialPurposeRanges {
		if prefix.Contains(ip) {
			return blockedAddress
		}
	}
	return publicAddress
}

func (c *Client) resolve(ctx context.Context, target *url.URL) (destination, error) {
	site := urlOrigin(target)
	value := destination{origin: site}
	var answers []netip.Addr
	if literal, err := netip.ParseAddr(site.host); err == nil {
		answers = []netip.Addr{literal}
	} else {
		answers, err = c.lookup(ctx, "ip", site.host)
		if err != nil {
			return destination{}, fault.Errorf(diag.WebFetchTransfer, err)
		}
	}
	for _, ip := range answers {
		kind := addressType(ip)
		if kind == blockedAddress {
			continue
		}
		if value.scope != blockedAddress && value.scope != kind {
			return destination{}, fault.New(diag.PermissionDenied, diag.WebFetchMixedAddresses)
		}
		value.scope = kind
		value.ips = append(value.ips, ip.Unmap())
	}
	if len(value.ips) == 0 {
		return destination{}, fault.New(diag.PermissionDenied, diag.WebFetchBlocked)
	}
	return value, nil
}
