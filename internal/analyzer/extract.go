package analyzer

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func extractStructured(rawHTML []byte, baseURL *url.URL) (AnalyzeResponse, []string, error) {
	doc, err := html.Parse(bytes.NewReader(rawHTML))
	if err != nil {
		return AnalyzeResponse{}, nil, fmt.Errorf("parse html: %w", err)
	}

	collector := newExtractor(baseURL)
	collector.walk(doc)
	return collector.toResponse(), collector.links, nil
}

type extractor struct {
	baseURL *url.URL

	htmlVersion string
	title       string
	headings    HeadingCounts
	isLoginPage bool

	links []string
	seen  map[string]struct{}
}

func newExtractor(baseURL *url.URL) *extractor {
	return &extractor{
		baseURL:     baseURL,
		htmlVersion: "Unknown",
		seen:        make(map[string]struct{}),
	}
}

// walk traverses DOM iteratively to avoid recursion depth risks.
func (c *extractor) walk(root *html.Node) {
	if root == nil {
		return
	}
	queue := []*html.Node{root}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		c.consumeNode(node)
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			queue = append(queue, child)
		}
	}
}

func (c *extractor) consumeNode(n *html.Node) {
	if n.Type == html.DoctypeNode && c.htmlVersion == "Unknown" {
		c.captureDoctype(n)
		return
	}
	if n.Type != html.ElementNode {
		return
	}

	name := strings.ToLower(n.Data)
	c.captureTitle(name, n)
	c.captureHeading(name)
	c.captureLink(name, n)
	c.captureLoginForm(name, n)
}

func (c *extractor) captureDoctype(n *html.Node) {
	data := strings.TrimSpace(n.Data)
	if strings.EqualFold(data, "html") {
		c.htmlVersion = "HTML5"
		return
	}
	if data != "" {
		c.htmlVersion = data
	}
}

func (c *extractor) captureTitle(name string, n *html.Node) {
	if name != "title" || c.title != "" {
		return
	}
	c.title = strings.TrimSpace(textContentRecursive(n))
}

func (c *extractor) captureHeading(name string) {
	switch name {
	case "h1":
		c.headings.Heading1++
	case "h2":
		c.headings.Heading2++
	case "h3":
		c.headings.Heading3++
	case "h4":
		c.headings.Heading4++
	case "h5":
		c.headings.Heading5++
	case "h6":
		c.headings.Heading6++
	}
}

func (c *extractor) captureLink(name string, n *html.Node) {
	if name != "a" || c.baseURL == nil {
		return
	}
	href := getAttr(n, "href")
	if href == "" {
		return
	}
	abs := resolveHTTPLink(c.baseURL, href)
	if abs == "" {
		return
	}
	if _, ok := c.seen[abs]; ok {
		return
	}
	c.seen[abs] = struct{}{}
	c.links = append(c.links, abs)
}

func (c *extractor) toResponse() AnalyzeResponse {
	return AnalyzeResponse{
		HTMLVersion:   c.htmlVersion,
		PageTitle:     c.title,
		HeadingCounts: c.headings,
		IsLoginPage:   c.isLoginPage,
	}
}

func (c *extractor) captureLoginForm(name string, n *html.Node) {
	if c.isLoginPage || name != "form" {
		return
	}
	if formHasCredentials(n) {
		c.isLoginPage = true
	}
}

// formHasCredentials checks if a form contains a username or password input.
// if one of those are available, we consider the form a login form.
func formHasCredentials(formNode *html.Node) bool {
	queue := []*html.Node{formNode}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		if node.Type == html.ElementNode && strings.EqualFold(node.Data, "input") {
			if isPasswordInput(node) || isUsernameLikeInput(node) {
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
