package analyzer

import "errors"

var (
	// ErrInvalidURL means the input is not a usable absolute http(s) URL.
	ErrInvalidURL = errors.New("invalid request URL")
	// ErrDisallowedHost means the host resolves to addresses we refuse to fetch (SSRF mitigation).
	ErrDisallowedHost = errors.New("host not allowed")
)
