package http

// AnalyzeRequest is the API input shape for web page analysis.
type AnalyzeRequest struct {
	URL string `json:"url"`
}

// AnalyzeResponse is the API output shape for web page analysis.
type AnalyzeResponse struct {
	HTMLVersion       string        `json:"htmlVersion"`
	PageTitle         string        `json:"pageTitle"`
	HeadingCounts     HeadingCounts `json:"headingCounts"`
	ExternalLinks     int           `json:"externalLinks"`
	InternalLinks     int           `json:"internalLinks"`
	InaccessibleLinks int           `json:"inaccessibleLinks"`
	IsLoginPage       bool          `json:"containsLogin"`
}

// HeadingCounts holds per-level heading counts from a page.
type HeadingCounts struct {
	Heading1 int `json:"heading1"`
	Heading2 int `json:"heading2"`
	Heading3 int `json:"heading3"`
	Heading4 int `json:"heading4"`
	Heading5 int `json:"heading5"`
	Heading6 int `json:"heading6"`
}
