package http

import (
	"encoding/json"
	stdhttp "net/http"
)

func AnalyzeHandler(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if r.Method != stdhttp.MethodPost {
		stdhttp.Error(w, "method not allowed", stdhttp.StatusMethodNotAllowed)
		return
	}

	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		stdhttp.Error(w, "invalid json body", stdhttp.StatusBadRequest)
		return
	}

	resp := AnalyzeResponse{
		HTMLVersion: "HTML5",
		PageTitle:   "Dummy Page Title",
		HeadingCounts: HeadingCounts{
			Heading1: 4,
			Heading2: 2,
			Heading3: 0,
			Heading4: 0,
			Heading5: 0,
			Heading6: 0,
		},
		ExternalLinks:     12,
		InternalLinks:     8,
		InaccessibleLinks: 1,
		IsLoginPage:       false,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
