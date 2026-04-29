package ui

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed templates/*.html
var sharedTemplateFS embed.FS

// Renderer handles template parsing and rendering for an ONCE app.
type Renderer struct {
	app App
	tpl *template.Template
}

// NewRenderer creates a Renderer that merges shared UI templates with app-specific templates.
// appFS is the app's embedded template filesystem.
// patterns are the glob patterns for app templates (e.g., "templates/*.html").
func NewRenderer(app App, appFS embed.FS, patterns ...string) (*Renderer, error) {
	if app.BaseURL == "" {
		app.BaseURL = "/"
	}

	shared, err := parseSharedTemplates()
	if err != nil {
		return nil, fmt.Errorf("ui: parse shared templates: %w", err)
	}

	if len(patterns) == 0 {
		return nil, fmt.Errorf("ui: no template patterns provided")
	}

	appTmpl, err := template.ParseFS(appFS, patterns...)
	if err != nil {
		return nil, fmt.Errorf("ui: parse app templates: %w", err)
	}

	// Merge app templates into shared tree, detecting collisions.
	for _, t := range appTmpl.Templates() {
		name := t.Name()
		if name == "" {
			continue
		}
		if shared.Lookup(name) != nil {
			return nil, fmt.Errorf("ui: template name collision: %q exists in shared templates", name)
		}
		if _, err := shared.AddParseTree(name, t.Tree); err != nil {
			return nil, fmt.Errorf("ui: add app template %q: %w", name, err)
		}
	}

	return &Renderer{app: app, tpl: shared}, nil
}

func parseSharedTemplates() (*template.Template, error) {
	tpl := template.New("")
	entries, err := sharedTemplateFS.ReadDir("templates")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		path := "templates/" + entry.Name()
		content, err := fs.ReadFile(sharedTemplateFS, path)
		if err != nil {
			return nil, err
		}
		if _, err := tpl.New(path).Parse(string(content)); err != nil {
			return nil, err
		}
	}
	return tpl, nil
}

// Render executes the named template to w.
// name is the template to execute (e.g., "index" or "index.html").
// page provides common page metadata; data is app-specific page data.
func (r *Renderer) Render(w io.Writer, name string, page Page, data any) error {
	type viewModel struct {
		App  App
		Page Page
		Data any
	}
	vm := viewModel{App: r.app, Page: page, Data: data}
	return r.tpl.ExecuteTemplate(w, name, vm)
}

// RenderError renders the shared once/error template.
func (r *Renderer) RenderError(w io.Writer, statusCode int, message string) error {
	page := Page{Title: fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode))}
	data := map[string]any{
		"StatusCode": statusCode,
		"Message":    message,
	}
	return r.Render(w, "once/error", page, data)
}

// AssetsHandler returns an http.Handler that serves compiled shared UI assets
// from the reserved route /assets/once/. Mount it on your mux with:
//
//	mux.Handle("GET /assets/once/", ui.AssetsHandler())
func AssetsHandler() http.Handler {
	sub, err := fs.Sub(StaticFS, "static")
	if err != nil {
		panic("ui: failed to sub static FS: " + err.Error())
	}
	return http.StripPrefix("/assets/once/", http.FileServer(http.FS(sub)))
}
