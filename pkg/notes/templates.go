// Package notes provides shared types and utilities for the notes application.
package notes

import (
	"embed"
	"html/template"
	"io"

	"once-stack/pkg/ui"
)

// TemplateData holds data for rendering page templates.
// The page title is passed as a separate argument to render functions;
// only page-specific structured data lives here.
type TemplateData struct {
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
// title is used for the browser <title> tag (App.Name is appended automatically).
func RenderIndex(w io.Writer, title string, data *TemplateData) error {
	page := ui.Page{Title: title}
	return renderer.Render(w, "index.html", page, data)
}

// RenderView renders a single note view page with rendered markdown.
// title is used for the browser <title> tag (App.Name is appended automatically).
func RenderView(w io.Writer, title string, data *TemplateData) error {
	page := ui.Page{Title: title}
	return renderer.Render(w, "view.html", page, data)
}

// RenderForm renders the new/edit note form page.
// title is used for the browser <title> tag (App.Name is appended automatically).
func RenderForm(w io.Writer, title string, data *TemplateData) error {
	page := ui.Page{Title: title}
	return renderer.Render(w, "form.html", page, data)
}

// RenderError renders a generic error page.
func RenderError(w io.Writer, title, message string) error {
	page := ui.Page{Title: title}
	data := &TemplateData{
		Error: message,
	}
	return renderer.Render(w, "app-error.html", page, data)
}
