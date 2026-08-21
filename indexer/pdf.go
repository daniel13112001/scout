package indexer

import (
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

// extractPDFText reads path as a PDF and returns its text content, one
// page at a time, with a "--- page N ---" marker before each page so page
// numbers survive into chunk content and search results - the same way
// scout already gets line numbers for free from plain text files.
func extractPDFText(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening pdf: %w", err)
	}
	defer f.Close()

	var text strings.Builder

	for i := 1; i <= r.NumPage(); i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}

		pageText, err := page.GetPlainText(nil)
		if err != nil {
			return "", fmt.Errorf("extracting page %d: %w", i, err)
		}

		fmt.Fprintf(&text, "--- page %d ---\n%s\n", i, cleanExtractedText(pageText))
	}

	return text.String(), nil
}

// cleanExtractedText trims trailing whitespace from each line and collapses
// runs of blank lines to one. PDF text reconstruction commonly produces
// both (lines padded out to a fixed width, blank lines standing in for
// vertical whitespace on the page), which would otherwise bloat chunk
// count with mostly-empty chunks.
func cleanExtractedText(s string) string {
	lines := strings.Split(s, "\n")

	var cleaned []string
	blank := false

	for _, line := range lines {
		line = strings.TrimRight(line, " \t")

		if line == "" {
			if blank {
				continue
			}
			blank = true
		} else {
			blank = false
		}

		cleaned = append(cleaned, line)
	}

	return strings.Join(cleaned, "\n")
}
