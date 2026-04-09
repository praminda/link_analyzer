package analyzer

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestParseAndValidateURL(t *testing.T) {
	public := func(ctx context.Context, host string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, nil
	}
	loopback := func(ctx context.Context, host string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}, nil
	}
	private := func(ctx context.Context, host string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("10.0.0.1")}}, nil
	}

	tests := []struct {
		name    string
		rawURL  string
		lookup  ipLookup
		wantErr error
	}{
		{
			name:    "empty",
			rawURL:  "",
			lookup:  public,
			wantErr: ErrInvalidURL,
		},
		{
			name:    "relative",
			rawURL:  "/path",
			lookup:  public,
			wantErr: ErrInvalidURL,
		},
		{
			name:    "bad scheme",
			rawURL:  "ftp://example.com/",
			lookup:  public,
			wantErr: ErrInvalidURL,
		},
		{
			name:    "missing host",
			rawURL:  "https:///path",
			lookup:  public,
			wantErr: ErrInvalidURL,
		},
		{
			name:   "ok https",
			rawURL: "https://example.com/page?q=1#frag",
			lookup: public,
		},
		{
			name:    "disallowed resolved loopback",
			rawURL:  "https://example.com/",
			lookup:  loopback,
			wantErr: ErrDisallowedHost,
		},
		{
			name:    "disallowed resolved private",
			rawURL:  "https://example.com/",
			lookup:  private,
			wantErr: ErrDisallowedHost,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := parseAndValidateURL(context.Background(), tt.rawURL, tt.lookup)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if u.Fragment != "" {
				t.Fatalf("fragment should be cleared, got %q", u.Fragment)
			}
		})
	}
}

func TestDisallowedIP(t *testing.T) {
	if !disallowedIP(net.ParseIP("127.0.0.1")) {
		t.Fatal("loopback should be disallowed")
	}
	if !disallowedIP(net.ParseIP("10.0.0.1")) {
		t.Fatal("private should be disallowed")
	}
	if !disallowedIP(net.ParseIP("169.254.1.1")) {
		t.Fatal("link-local should be disallowed")
	}
	if disallowedIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("public should be allowed")
	}
}
