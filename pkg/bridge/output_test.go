package bridge

import (
	"testing"
)

func TestParseOutput(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantText  string
		wantPaths []string
	}{
		{
			name:      "no markers",
			raw:       "Hello, world!",
			wantText:  "Hello, world!",
			wantPaths: nil,
		},
		{
			name:      "FILE only",
			raw:       "Here is [FILE: /tmp/report.pdf] your report.",
			wantText:  "Here is  your report.",
			wantPaths: []string{"/tmp/report.pdf"},
		},
		{
			name:      "IMAGE only",
			raw:       "See [IMAGE: /tmp/photo.png] for details.",
			wantText:  "See  for details.",
			wantPaths: []string{"/tmp/photo.png"},
		},
		{
			name:      "mixed document order",
			raw:       "[FILE: a.txt] then [IMAGE: b.png] then [FILE: c.txt]",
			wantText:  "then  then",
			wantPaths: []string{"a.txt", "b.png", "c.txt"},
		},
		{
			name:      "case insensitive",
			raw:       "[file: /tmp/doc.pdf] and [IMAGE: /tmp/img.jpg]",
			wantText:  "and",
			wantPaths: []string{"/tmp/doc.pdf", "/tmp/img.jpg"},
		},
		{
			name:      "whitespace in path",
			raw:       "[FILE:  /tmp/my file.pdf ]",
			wantText:  "",
			wantPaths: []string{"/tmp/my file.pdf"},
		},
		{
			name:      "multiple images",
			raw:       "[IMAGE: a.jpg][IMAGE: b.jpg]",
			wantText:  "",
			wantPaths: []string{"a.jpg", "b.jpg"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotText, gotPaths := ParseOutput(tc.raw)
			if gotText != tc.wantText {
				t.Errorf("text: got %q, want %q", gotText, tc.wantText)
			}
			if len(gotPaths) != len(tc.wantPaths) {
				t.Errorf("paths len: got %d (%v), want %d (%v)", len(gotPaths), gotPaths, len(tc.wantPaths), tc.wantPaths)
				return
			}
			for i, p := range gotPaths {
				if p != tc.wantPaths[i] {
					t.Errorf("paths[%d]: got %q, want %q", i, p, tc.wantPaths[i])
				}
			}
		})
	}
}
