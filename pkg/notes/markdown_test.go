package notes

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string // substring expected in the output
	}{
		{"plain text", "hello", "hello"},
		{"bold", "**bold**", "<strong>bold</strong>"},
		{"italic", "*italic*", "<em>italic</em>"},
		{"code", "`code`", "<code>code</code>"},
		{"link", "[link](https://example.com)", "<a href=\"https://example.com\" rel=\"nofollow\">link</a>"},
		{"header", "# Title", "<h1>Title</h1>"},
		{"paragraph", "Paragraph text.", "<p>Paragraph text.</p>"},
		{"empty", "", ""},
		{"list", "- item", "<li>item</li>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderMarkdown(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(got), tt.want) {
				t.Errorf("RenderMarkdown(%q) = %q, want substring %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRenderMarkdown_NoUnsafeHTML(t *testing.T) {
	inputs := []string{
		`<script>alert("xss")</script>`,
		`<img onerror="alert(1)" src=x>`,
		`<iframe src="https://evil.com"></iframe>`,
	}

	for _, input := range inputs {
		name := input
		if len(name) > 20 {
			name = name[:20]
		}
		t.Run(name, func(t *testing.T) {
			got, err := RenderMarkdown(input)
			if err != nil {
				t.Fatal(err)
			}
			html := string(got)
			if strings.Contains(html, "<script>") {
				t.Errorf("output contains raw <script> tag: %s", html)
			}
			if strings.Contains(html, "onerror") {
				t.Errorf("output contains event handler: %s", html)
			}
			if strings.Contains(html, "<iframe") {
				t.Errorf("output contains <iframe> tag: %s", html)
			}
		})
	}
}
