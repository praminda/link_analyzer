package analyzer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultFetchTimeout = 30 * time.Second
	// maxRedirectFollows is how many Location hops we allow after the initial GET.
	maxRedirectFollows  = 2
	defaultMaxBodyBytes = 2 << 20 // 2 MiB
	defaultUserAgent    = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.14; rv:145.0) Gecko/20090101 Firefox/145.0"
)

func newFetchHTTPClient(lookup ipLookup) *http.Client {
	return &http.Client{
		Timeout: defaultFetchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > maxRedirectFollows {
				return ErrTooManyRedirects
			}
			if _, err := parseAndValidateURL(req.Context(), req.URL.String(), lookup); err != nil {
				return err
			}
			return nil
		},
	}
}

func fetchHTML(ctx context.Context, client *http.Client, u *url.URL, maxBody int64) ([]byte, error) {
	if client == nil {
		return nil, fmt.Errorf("fetch: %w", ErrNilHTTPClient)
	}
	if maxBody <= 0 {
		maxBody = defaultMaxBodyBytes
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("fetch: build request: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status %d", ErrFetchStatus, resp.StatusCode)
	}
	if !isHTMLContentType(resp.Header.Get("Content-Type")) {
		return nil, ErrNotHTML
	}

	lr := io.LimitReader(resp.Body, maxBody+1)
	body, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("fetch: read body: %w", err)
	}
	if int64(len(body)) > maxBody {
		return nil, ErrBodyTooLarge
	}
	return body, nil
}

func isHTMLContentType(ct string) bool {
	ct = strings.TrimSpace(ct)
	if ct == "" {
		return true
	}
	base := strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
	if base == "text/html" || base == "application/xhtml+xml" {
		return true
	}
	return false
}
