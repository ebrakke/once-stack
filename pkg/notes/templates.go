// Package notes provides shared types and utilities for the notes application.
package notes

import (
	"embed"
	"html/template"
	"io"
)

// TemplateData holds data for rendering page templates.
type TemplateData struct {
	Title    string
	Note     *Note
	Notes    []Note
	BodyHTML template.HTML // rendered markdown for the view page
	Content  string        // raw markdown content for the form page
	Error    string        // error message for the error page
	Slug     string        // note ID for form action URL
	IsNew    bool          // true when rendering a new-note form
}

//go:embed templates/*.html
var templateFS embed.FS

// templates is the parsed template tree shared across all renders.
var templates *template.Template

func init() {
	t, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		panic("notes: failed to parse templates: " + err.Error())
	}
	templates = t
}

// RenderIndex renders the note listing page.
func RenderIndex(w io.Writer, data *TemplateData) error {
	return templates.ExecuteTemplate(w, "index.html", data)
}

// RenderView renders a single note view page with rendered markdown.
func RenderView(w io.Writer, data *TemplateData) error {
	return templates.ExecuteTemplate(w, "view.html", data)
}

// RenderForm renders the new/edit note form page.
func RenderForm(w io.Writer, data *TemplateData) error {
	return templates.ExecuteTemplate(w, "form.html", data)
}

// RenderError renders a generic error page.
func RenderError(w io.Writer, title, message string) error {
	data := &TemplateData{
		Title: title,
		Error: message,
	}
	return templates.ExecuteTemplate(w, "error.html", data)
}
