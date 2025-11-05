package format

import (
	"testing"
)

func TestExtractHTMLTitleFromB64Html(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "valid HTML with title",
			input:    []byte("<html><head><title>Test Page</title></head><body></body></html>"),
			expected: "Test Page",
		},
		{
			name:     "base64 encoded HTML",
			input:    []byte("PGh0bWw+PGhlYWQ+PHRpdGxlPlRlc3QgUGFnZTwvdGl0bGU+PC9oZWFkPjxib2R5PjwvYm9keT48L2h0bWw+"),
			expected: "Test Page",
		},
		{
			name:     "HTML with uppercase tags",
			input:    []byte("<HTML><HEAD><TITLE>Uppercase Test</TITLE></HEAD><BODY></BODY></HTML>"),
			expected: "Uppercase Test",
		},
		{
			name:     "HTML without title tag",
			input:    []byte("<html><head></head><body>No title</body></html>"),
			expected: "",
		},
		{
			name:     "empty title tag",
			input:    []byte("<html><head><title></title></head><body></body></html>"),
			expected: "",
		},
		{
			name:     "not HTML content",
			input:    []byte("This is plain text, not HTML"),
			expected: "",
		},
		{
			name:     "empty input",
			input:    []byte(""),
			expected: "",
		},
		{
			name:     "HTML fragment without html tag",
			input:    []byte("<div>Fragment</div>"),
			expected: "",
		},
		{
			name:     "malformed HTML",
			input:    []byte("<html><title>Broken</head></body>"),
			expected: "Broken</head></body>",
		},
		{
			name:     "HTML with nested tags in title",
			input:    []byte("<html><head><title>Simple Title</title></head><body></body></html>"),
			expected: "Simple Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractHTMLTitleFromB64Html(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractHTMLTitleFromB64Html() = %q, want %q", result, tt.expected)
			}
		})
	}
}
