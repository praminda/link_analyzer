package analyzer

import (
	"context"
	"net/url"
	"testing"
)

func TestExtractStructured(t *testing.T) {
	htmlDoc := `<!DOCTYPE html>
<html>
  <head><title>Example Title</title></head>
  <body>
    <h1>One</h1>
    <h2>Two</h2>
    <h2>Two Again</h2>
    <h4>Four</h4>
    <a href="/internal">Internal</a>
    <a href="https://other.com/a#frag">External</a>
    <a href="mailto:test@example.com">Mail</a>
    <a href="">Empty</a>
  </body>
</html>`

	base, err := url.Parse("https://example.com/root/page")
	if err != nil {
		t.Fatal(err)
	}

	out, links, err := extractStructured(context.Background(), nil, []byte(htmlDoc), base)
	if err != nil {
		t.Fatal(err)
	}

	if out.HTMLVersion != "HTML5" {
		t.Fatalf("HTMLVersion = %q", out.HTMLVersion)
	}
	if out.PageTitle != "Example Title" {
		t.Fatalf("PageTitle = %q", out.PageTitle)
	}
	if out.HeadingCounts.Heading1 != 1 || out.HeadingCounts.Heading2 != 2 || out.HeadingCounts.Heading4 != 1 {
		t.Fatalf("HeadingCounts = %+v", out.HeadingCounts)
	}
	if len(links) != 2 {
		t.Fatalf("links len = %d, links=%v", len(links), links)
	}
	if links[0] != "https://example.com/internal" {
		t.Fatalf("first link = %q", links[0])
	}
	if links[1] != "https://other.com/a" {
		t.Fatalf("second link = %q", links[1])
	}
}

func TestExtractStructured_PreservesDuplicateHrefs(t *testing.T) {
	htmlDoc := `<!DOCTYPE html><html><head><title>T</title></head><body>
<a href="/same">A</a>
<a href="/same">B</a>
<a href="/same">C</a>
</body></html>`
	base, err := url.Parse("https://example.com/page")
	if err != nil {
		t.Fatal(err)
	}
	_, links, err := extractStructured(context.Background(), nil, []byte(htmlDoc), base)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"https://example.com/same",
		"https://example.com/same",
		"https://example.com/same",
	}
	if len(links) != len(want) {
		t.Fatalf("links len = %d want %d, links=%v", len(links), len(want), links)
	}
	for i, w := range want {
		if links[i] != w {
			t.Fatalf("links[%d] = %q want %q", i, links[i], w)
		}
	}
}

func TestDetectHTMLVersion_NoDoctype(t *testing.T) {
	base, _ := url.Parse("https://example.com")
	out, _, err := extractStructured(context.Background(), nil, []byte("<html><head><title>T</title></head><body></body></html>"), base)
	if err != nil {
		t.Fatal(err)
	}
	if out.HTMLVersion != "Unknown" {
		t.Fatalf("HTMLVersion = %q", out.HTMLVersion)
	}
}

func TestExtractStructured_LoginDetection_RequiresUsernameAndPassword(t *testing.T) {
	base, _ := url.Parse("https://example.com")

	loginHTML := `<html><body>
<form>
  <input type="text" name="username" />
  <input type="password" name="password" />
</form>
</body></html>`
	out, _, err := extractStructured(context.Background(), nil, []byte(loginHTML), base)
	if err != nil {
		t.Fatal(err)
	}
	if !out.IsLoginPage {
		t.Fatal("expected IsLoginPage=true when form has username and password")
	}

	passwordOnlyHTML := `<html><body>
<form>
  <input type="password" name="password" />
</form>
</body></html>`
	out, _, err = extractStructured(context.Background(), nil, []byte(passwordOnlyHTML), base)
	if err != nil {
		t.Fatal(err)
	}
	if out.IsLoginPage {
		t.Fatal("expected IsLoginPage=false for password-only form")
	}

	usernameOnlyHTML := `<html><body>
<form>
  <input type="text" name="username" />
</form>
</body></html>`
	out, _, err = extractStructured(context.Background(), nil, []byte(usernameOnlyHTML), base)
	if err != nil {
		t.Fatal(err)
	}
	if out.IsLoginPage {
		t.Fatal("expected IsLoginPage=false for username-only form")
	}

	noCredentialsHTML := `<html><body>
<form>
  <input type="hidden" name="csrf" />
</form>
</body></html>`
	out, _, err = extractStructured(context.Background(), nil, []byte(noCredentialsHTML), base)
	if err != nil {
		t.Fatal(err)
	}
	if out.IsLoginPage {
		t.Fatal("expected IsLoginPage=false without credential fields")
	}
}
