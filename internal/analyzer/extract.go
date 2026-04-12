package analyzer

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func extractStructured(ctx context.Context, log *slog.Logger, rawHTML []byte, baseURL *url.URL) (AnalyzeResponse, []string, error) {
	doc, err := html.Parse(bytes.NewReader(rawHTML))
	if err != nil {
		return AnalyzeResponse{}, nil, fmt.Errorf("parse html: %w", err)
	}

	collector := newExtractor(baseURL)
	collector.walk(doc)
	out := collector.toResponse()
	if log != nil {
		log.InfoContext(ctx, "html extracted",
			"link_count", len(collector.links),
			"is_login_page", out.IsLoginPage,
			"html_version", out.HTMLVersion,
		)
	}
	return out, collector.links, nil
}

type extractor struct {
	baseURL *url.URL

	htmlVersion string
	title       string
	headings    HeadingCounts
	isLoginPage bool

	links []string
}

func newExtractor(baseURL *url.URL) *extractor {
	return &extractor{
		baseURL:     baseURL,
		htmlVersion: "Unknown",
	}
}

// walk traverses DOM iteratively to avoid recursion depth risks.
func (ext *extractor) walk(root *html.Node) {
	if root == nil {
		return
	}
	queue := []*html.Node{root}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		ext.consumeNode(node)
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			queue = append(queue, child)
		}
	}
}

func (ext *extractor) consumeNode(node *html.Node) {
	if node.Type == html.DoctypeNode && ext.htmlVersion == "Unknown" {
		ext.captureDoctype(node)
		return
	}
	if node.Type != html.ElementNode {
		return
	}

	name := strings.ToLower(node.Data)
	ext.captureTitle(name, node)
	ext.captureHeading(name)
	ext.captureLink(name, node)
	ext.captureLoginForm(name, node)
}

func (ext *extractor) captureDoctype(node *html.Node) {
	data := strings.TrimSpace(node.Data)
	if strings.EqualFold(data, "html") {
		ext.htmlVersion = "HTML5"
		return
	}
	if data != "" {
		ext.htmlVersion = data
	}
}

func (ext *extractor) captureTitle(name string, node *html.Node) {
	if name != "title" || ext.title != "" {
		return
	}
	ext.title = strings.TrimSpace(textContentRecursive(node))
}

func (ext *extractor) captureHeading(name string) {
	switch name {
	case "h1":
		ext.headings.Heading1++
	case "h2":
		ext.headings.Heading2++
	case "h3":
		ext.headings.Heading3++
	case "h4":
		ext.headings.Heading4++
	case "h5":
		ext.headings.Heading5++
	case "h6":
		ext.headings.Heading6++
	}
}

func (ext *extractor) captureLink(name string, node *html.Node) {
	if name != "a" || ext.baseURL == nil {
		return
	}
	href := getAttr(node, "href")
	if href == "" {
		return
	}
	abs := resolveHTTPLink(ext.baseURL, href)
	if abs == "" {
		return
	}
	ext.links = append(ext.links, abs)
}

func (ext *extractor) toResponse() AnalyzeResponse {
	return AnalyzeResponse{
		HTMLVersion:   ext.htmlVersion,
		PageTitle:     ext.title,
		HeadingCounts: ext.headings,
		IsLoginPage:   ext.isLoginPage,
	}
}

func (ext *extractor) captureLoginForm(name string, n *html.Node) {
	if ext.isLoginPage || name != "form" {
		return
	}
	if formHasCredentials(n) {
		ext.isLoginPage = true
	}
}

// formHasCredentials reports whether the form has both a username-like field
// and a password field.
func formHasCredentials(formNode *html.Node) bool {
	var hasUsername, hasPassword bool
	queue := []*html.Node{formNode}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		if node.Type == html.ElementNode && strings.EqualFold(node.Data, "input") {
			if isPasswordInput(node) {
				hasPassword = true
			} else if isUsernameLikeInput(node) {
				hasUsername = true
			}
			if hasUsername && hasPassword {
				return true
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			queue = append(queue, child)
		}
	}
	return false
}

// isPasswordInput checks if an input node is a password input.
func isPasswordInput(n *html.Node) bool {
	return strings.EqualFold(strings.TrimSpace(getAttr(n, "type")), "password")
}

// isUsernameLikeInput checks if an input node is a username input.
func isUsernameLikeInput(n *html.Node) bool {
	inputType := strings.ToLower(strings.TrimSpace(getAttr(n, "type")))
	if inputType == "" {
		inputType = "text"
	}
	if inputType != "text" && inputType != "email" {
		return false
	}

	if strings.EqualFold(strings.TrimSpace(getAttr(n, "autocomplete")), "username") {
		return true
	}

	candidates := []string{
		strings.ToLower(getAttr(n, "name")),
		strings.ToLower(getAttr(n, "id")),
	}
	for _, v := range candidates {
		if strings.Contains(v, "user") || strings.Contains(v, "email") || strings.Contains(v, "login") {
			return true
		}
	}
	return false
}

func resolveHTTPLink(baseURL *url.URL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}
	abs := baseURL.ResolveReference(ref)
	scheme := strings.ToLower(abs.Scheme)
	if scheme != "http" && scheme != "https" {
		return ""
	}
	abs.Fragment = ""
	abs.RawFragment = ""
	return abs.String()
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

// textContentRecursive extracts text content from a node and all its children.
// Using recursion here because title nodes are typically small.
func textContentRecursive(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node == nil {
			return
		}
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
	return b.String()
}
