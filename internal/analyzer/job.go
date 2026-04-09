package analyzer

import (
	"context"
	"errors"
	"net"
	"net/http"
)

// AnalyzeJob runs link analysis for a single URL. Use one instance per request
// or worker. Call Process to validate the URL, fetch HTML, and stage bytes for
// later parsing steps.
type AnalyzeJob struct {
	URL string

	// lookup and httpClient are optional overrides (e.g. tests). When nil,
	// the default resolver and a hardened fetch client are used.
	lookup     ipLookup
	httpClient *http.Client

	// rawHTML holds the fetched document after a successful Process.
	rawHTML []byte
	// response is progressively filled as analysis stages complete.
	response AnalyzeResponse
	// resolvedLinks holds absolute HTTP(S) links extracted from the document.
	resolvedLinks []string
}

// RawHTML returns the fetched document body after Process succeeds.
func (j *AnalyzeJob) RawHTML() []byte {
	if j == nil {
		return nil
	}
	return j.rawHTML
}

// Response returns the structured analysis accumulated by Process.
func (j *AnalyzeJob) Response() AnalyzeResponse {
	if j == nil {
		return AnalyzeResponse{}
	}
	return j.response
}

// ResolvedLinks returns absolute HTTP(S) links extracted during Process.
func (j *AnalyzeJob) ResolvedLinks() []string {
	if j == nil {
		return nil
	}
	return j.resolvedLinks
}

func (j *AnalyzeJob) Process(ctx context.Context) error {
	if j == nil {
		return errors.New("Analyzer: nil AnalyzeJob")
	}
	lookup := j.lookup
	if lookup == nil {
		lookup = net.DefaultResolver.LookupIPAddr
	}
	u, err := parseAndValidateURL(ctx, j.URL, lookup)
	if err != nil {
		return err
	}
	client := j.httpClient
	if client == nil {
		client = newFetchHTTPClient(lookup)
	}
	body, err := fetchHTML(ctx, client, u, defaultMaxBodyBytes)
	if err != nil {
		return err
	}
	j.rawHTML = body

	out, links, err := extractStructured(body, u)
	if err != nil {
		return err
	}
	j.response = out
	j.resolvedLinks = links
	return nil
}
