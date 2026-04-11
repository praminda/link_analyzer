package analyzer

import (
	"context"
	"errors"
	"testing"
)

func TestValidateAnalyzeURL_Table(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name       string
		raw        string
		wantErr    error // errors.Is match; nil means expect success
		wantErrNil bool
	}{
		{name: "empty string", raw: "", wantErr: ErrInvalidURL},
		{name: "whitespace only", raw: "   \t", wantErr: ErrInvalidURL},
		{name: "relative path not absolute", raw: "/articles", wantErr: ErrInvalidURL},
		{name: "scheme ftp", raw: "ftp://example.com/", wantErr: ErrInvalidURL},
		{name: "scheme file", raw: "file:///etc/passwd", wantErr: ErrInvalidURL},
		{name: "missing host", raw: "https://", wantErr: ErrInvalidURL},
		{name: "http uppercase scheme ok", raw: "HTTP://example.com/", wantErrNil: true},
		{name: "loopback IPv4 literal", raw: "http://127.0.0.1/", wantErr: ErrDisallowedHost},
		{name: "loopback IPv6 literal", raw: "http://[::1]/", wantErr: ErrDisallowedHost},
		{name: "private IPv4 literal", raw: "http://10.0.0.1/", wantErr: ErrDisallowedHost},
		{name: "link local IPv4 literal", raw: "http://169.254.1.1/", wantErr: ErrDisallowedHost},
		{name: "unspecified IPv4 literal", raw: "http://0.0.0.0/", wantErr: ErrDisallowedHost},
		{name: "public host ok", raw: "https://example.com/", wantErrNil: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAnalyzeURL(ctx, tt.raw)
			if tt.wantErrNil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("errors.Is(..., %v) = false, err = %v", tt.wantErr, err)
			}
		})
	}
}
