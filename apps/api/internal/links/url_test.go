package links

import (
	"strings"
	"testing"
)

// A table test: one row per input, so a new case is one line, and
// `go test -run 'TestCheckURL/ftp'` runs a single row.
func TestCheckURL(t *testing.T) {
	longest := "https://" + strings.Repeat("a", maxURLLength-len("https://.com")) + ".com"
	tests := []struct {
		name string
		url  string
		ok   bool
	}{
		{"https", "https://go.dev/doc/", true},
		{"http with port and query", "http://localhost:8080/a?b=c", true},
		{"at the length limit", longest, true},
		{"over the length limit", longest + "/", false},
		{"empty", "", false},
		{"no scheme", "go.dev", false},
		{"no host", "https://", false},
		{"javascript", "javascript:alert(1)", false},
		{"ftp", "ftp://example.com/file", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			problem := checkURL(tt.url)
			if ok := problem == ""; ok != tt.ok {
				t.Errorf("checkURL(%.40q) = %q, want ok = %v", tt.url, problem, tt.ok)
			}
		})
	}
}
