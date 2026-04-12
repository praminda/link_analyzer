package analyzer

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	// TODO: make configurable
	defaultLinkCheckConcurrency = 10
)

type linkMetrics struct {
	internal     int
	external     int
	inaccessible int
}

func generateLinkMetrics(ctx context.Context, log *slog.Logger, client *http.Client, lookup ipLookup, baseURL *url.URL, links []string) (linkMetrics, error) {
	if client == nil {
		return linkMetrics{}, fmt.Errorf("link metrics: %w", ErrNilHTTPClient)
	}
	if baseURL == nil {
		return linkMetrics{}, nil
	}
	if len(links) == 0 {
		return linkMetrics{}, nil
	}

	if log != nil {
		log.DebugContext(ctx, "link metrics started", "link_count", len(links))
	}

	metrics := linkMetrics{}
	unique := make([]string, 0, len(links))
	seen := make(map[string]struct{}, len(links))
	for _, link := range links {
		if isInternalLink(baseURL, link) {
			metrics.internal++
		} else {
			metrics.external++
		}
		if _, dup := seen[link]; dup {
			continue
		}
		seen[link] = struct{}{}
		unique = append(unique, link)
	}

	sem := make(chan struct{}, defaultLinkCheckConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	accessibleByURL := make(map[string]bool, len(unique))

	for _, link := range unique {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()

			// Validate extracted links before requesting them.
			u, err := parseAndValidateURL(ctx, link, lookup)
			if err != nil {
				mu.Lock()
				accessibleByURL[link] = false
				mu.Unlock()
				return
			}

			accessible, _ := probeLink(ctx, client, u.String())
			mu.Lock()
			accessibleByURL[link] = accessible
			mu.Unlock()
		})
	}
	wg.Wait()

	// Second pass: each anchor inherits its URL's probe result (duplicates share one probe).
	for _, link := range links {
		if ok, exists := accessibleByURL[link]; !exists || !ok {
			metrics.inaccessible++
		}
	}

	if log != nil {
		log.InfoContext(ctx, "link metrics completed",
			"internal_links", metrics.internal,
			"external_links", metrics.external,
			"inaccessible_links", metrics.inaccessible,
		)
	}
	return metrics, nil
}

func isInternalLink(baseURL *url.URL, link string) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}
	return strings.EqualFold(baseURL.Hostname(), u.Hostname())
}

func probeLink(ctx context.Context, client *http.Client, target string) (bool, int) {
	status, err := doRequestStatus(ctx, client, http.MethodHead, target)
	if err == nil {
		// Fall back to GET for servers not supporting HEAD.
		if status != http.StatusMethodNotAllowed && status != http.StatusNotImplemented {
			return status >= 200 && status < 400, status
		}
	}

	status, err = doRequestStatus(ctx, client, http.MethodGet, target)
	if err != nil {
		return false, 0
	}
	return status >= 200 && status < 400, status
}

func doRequestStatus(ctx context.Context, client *http.Client, method, target string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, target, nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}
