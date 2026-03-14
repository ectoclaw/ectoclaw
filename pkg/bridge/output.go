package bridge

import (
	"regexp"
	"strings"
)

var (
	reFile  = regexp.MustCompile(`(?i)\[FILE:\s*([^\]]+)\]`)
	reImage = regexp.MustCompile(`(?i)\[IMAGE:\s*([^\]]+)\]`)
	// Combined pattern for document-order extraction.
	reMarker = regexp.MustCompile(`(?i)\[(?:FILE|IMAGE):\s*([^\]]+)\]`)
)

// ParseOutput strips [FILE: ...] and [IMAGE: ...] markers from raw claude output.
// Returns the cleaned text and extracted file paths in document order, trimmed of whitespace.
func ParseOutput(raw string) (text string, filePaths []string) {
	var paths []string
	for _, match := range reMarker.FindAllStringSubmatch(raw, -1) {
		if p := strings.TrimSpace(match[1]); p != "" {
			paths = append(paths, p)
		}
	}

	// Strip markers from text.
	text = reFile.ReplaceAllString(raw, "")
	text = reImage.ReplaceAllString(text, "")
	text = strings.TrimSpace(text)

	return text, paths
}
