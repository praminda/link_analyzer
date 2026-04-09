package analyzer

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ipLookup resolves a hostname to IP addresses.
type ipLookup func(ctx context.Context, host string) ([]net.IPAddr, error)

func parseAndValidateURL(ctx context.Context, rawURL string, lookup ipLookup) (*url.URL, error) {
	if strings.TrimSpace(rawURL) == "" {
		return nil, fmt.Errorf("%w: empty", ErrInvalidURL)
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidURL, err)
	}
	if !u.IsAbs() {
		return nil, fmt.Errorf("%w: not absolute", ErrInvalidURL)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("%w: scheme %q", ErrInvalidURL, u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("%w: missing host", ErrInvalidURL)
	}
	if err := checkHostIPs(ctx, host, lookup); err != nil {
		return nil, err
	}
	u.Fragment = ""
	u.RawFragment = ""
	return u, nil
}

func checkHostIPs(ctx context.Context, host string, lookup ipLookup) error {
	if lookup == nil {
		lookup = net.DefaultResolver.LookupIPAddr
	}
	ips, err := lookup(ctx, host)
	if err != nil {
		return fmt.Errorf("%w: resolve %q: %w", ErrDisallowedHost, host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("%w: no addresses for %q", ErrDisallowedHost, host)
	}
	for _, addr := range ips {
		if disallowedIP(addr.IP) {
			return fmt.Errorf("%w: %v", ErrDisallowedHost, addr.IP)
		}
	}
	return nil
}

func disallowedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsUnspecified() {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip.IsMulticast() {
		return true
	}
	return false
}
