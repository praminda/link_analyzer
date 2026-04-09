package analyzer

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidURL means the input is not a usable absolute http(s) URL.
	ErrInvalidURL = errors.New("invalid request URL")
	// ErrDisallowedHost means the host resolves to addresses we refuse to fetch (SSRF mitigation).
	ErrDisallowedHost = errors.New("host not allowed")
	// ErrNotHTML means the response Content-Type is not treated as HTML.
	ErrNotHTML = errors.New("response is not HTML")
	// ErrBodyTooLarge means the response body exceeded the configured limit.
	ErrBodyTooLarge = errors.New("response body too large")
	// ErrTooManyRedirects means the redirect chain exceeded the limit.
	ErrTooManyRedirects = errors.New("too many redirects")
	// ErrFetchStatus means the HTTP status was not successful (2xx).
	ErrFetchStatus = errors.New("unexpected HTTP status")
	// ErrNilHTTPClient means fetchHTML was called with a nil *http.Client.
	ErrNilHTTPClient = errors.New("nil http client")
)

// UpstreamHTTPStatusError captures non-2xx status from an upstream page request.
type UpstreamHTTPStatusError struct {
	StatusCode int
}

func (e *UpstreamHTTPStatusError) Error() string {
	return fmt.Sprintf("upstream status %d", e.StatusCode)
}

// AnalyzeError is returned by Process and carries HTTP semantics for handlers.
type AnalyzeError struct {
	HTTPStatus int
	Code       string
	Message    string
	Cause      error
}

func (e *AnalyzeError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *AnalyzeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}
