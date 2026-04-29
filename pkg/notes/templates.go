// Package notes provides shared types and utilities for the notes application.
package notes

import (
	"embed"
	"html/template"
	"io"

	"once-stack/pkg/ui"
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

var renderer *ui.Renderer

func init() {
	app := ui.App{Name: "Notes", BaseURL: "/"}
	r, err := ui.NewRenderer(app, templateFS, "templates/*.html")
	if err != nil {
		panic("notes: failed to create renderer: " + err.Error())
	}
	renderer = r
}

// RenderIndex renders the note listing page.
func RenderIndex(w io.Writer, data *TemplateData) error {
	page := ui.Page{Title: data.Title}
	return renderer.Render(w, "index.html", page, data)
}

// RenderView renders a single note view page with rendered markdown.
func RenderView(w io.Writer, data *TemplateData) error {
	page := ui.Page{Title: data.Title}
	return renderer.Render(w, "view.html", page, data)
}

// RenderForm renders the new/edit note form page.
func RenderForm(w io.Writer, data *TemplateData) error {
	page := ui.Page{Title: data.Title}
	return renderer.Render(w, "form.html", page, data)
}

// RenderError renders a generic error page.
func RenderError(w io.Writer, title, message string) error {
	page := ui.Page{Title: title}
	data := &TemplateData{
		Title: title,
		Error: message,
	}
	return renderer.Render(w, "app-error.html", page, data)
}
