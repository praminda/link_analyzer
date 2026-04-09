package analyzer

// HeadingCounts holds per-level heading counts from a page.
type HeadingCounts struct {
	Heading1 int
	Heading2 int
	Heading3 int
	Heading4 int
	Heading5 int
	Heading6 int
}

// AnalyzeResponse is the API output shape for web page analysis.
type AnalyzeResponse struct {
	HTMLVersion       string
	PageTitle         string
	HeadingCounts     HeadingCounts
	ExternalLinks     int
	InternalLinks     int
	InaccessibleLinks int
	IsLoginPage       bool
}
