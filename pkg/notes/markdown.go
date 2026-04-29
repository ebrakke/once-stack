// Package notes provides shared utilities for the notes application.
package notes

import (
	"bytes"
	"html/template"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
)

// RenderMarkdown converts Markdown input to safe HTML.
// It renders the input using goldmark and then sanitizes the output
// with bluemonday's UGCPolicy to allow safe, user-generated content.
func RenderMarkdown(input string) (template.HTML, error) {
	md := goldmark.New()
	var buf bytes.Buffer
	if err := md.Convert([]byte(input), &buf); err != nil {
		return "", err
	}
	p := bluemonday.UGCPolicy()
	safe := p.SanitizeBytes(buf.Bytes())
	return template.HTML(safe), nil
}
