# UI System — `pkg/ui`

The ONCE Stack provides a shared UI system in `pkg/ui` so every app gets a
consistent look without duplicating templates, stylesheets, or rendering code.

Each app keeps its **own identity** (name, URL, page content). There is no
cross-app navigation shell. The shared system provides the header, footer,
CSS asset pipeline, component CSS classes, and template rendering helpers.

---

## Architecture

```
pkg/ui/
├── styles/once.css      # Tailwind source: theme tokens + component classes
├── static/once.css      # Compiled output (committed, embedded in binary)
├── templates/
│   ├── layout.html      # App header + footer shell
│   ├── error.html       # Shared error page template (once/error)
│   └── components.html  # Reusable template partials (once/flash, once/empty-state)
├── renderer.go          # Renderer: merges shared + app templates, renders pages
├── renderer_test.go     # Tests for template parsing, rendering, escaping
├── data.go              # App and Page data types
├── static.go            # embed of compiled CSS + static assets
└── testdata/            # Fixture assets for unit tests
```

### How it works

1. **Tailwind** compiles `pkg/ui/styles/once.css` → `pkg/ui/static/once.css`.
2. **`pkg/ui/static.go`** embeds `static/once.css` into the Go binary via `//go:embed`.
3. **`pkg/ui/renderer.go`** provides `NewRenderer` which merges shared templates
   (layout, error, components) with app-specific templates, detecting name collisions.
4. **App handlers** call `renderer.Render(...)` passing view data.
5. **`ui.AssetsHandler()`** serves the compiled CSS (and any future static assets)
   at the reserved route `GET /assets/once/`.

---

## Adding UI to a new app

### 1. Create app templates

Place your HTML templates in `pkg/<app>/templates/` (or `cmd/<app>/templates/`).
Each template reuses the shared layout via `{{template "once/header" .}}` and
`{{template "once/footer" .}}`:

```html
{{define "index.html"}}
{{template "once/header" .}}
<div class="space-y-4">
  <h1 class="text-xl font-bold">My App</h1>
  <p class="text-once-text">Hello from your new app!</p>
</div>
{{template "once/footer" .}}
{{end}}
```

All app template names must be **unique** — they must not collide with shared
template names (which use the `once/` prefix). If a collision occurs,
`NewRenderer` returns an error at startup.

### 2. Wire up the renderer

In your `pkg/<app>/templates.go` (or similar setup file):

```go
package myapp

import (
    "embed"
    "io"

    "once-stack/pkg/ui"
)

//go:embed templates/*.html
var templateFS embed.FS

var renderer *ui.Renderer

func init() {
    app := ui.App{Name: "MyApp", BaseURL: "/"}
    r, err := ui.NewRenderer(app, templateFS, "templates/*.html")
    if err != nil {
        panic("myapp: failed to create renderer: " + err.Error())
    }
    renderer = r
}

func RenderIndex(w io.Writer, title string, data any) error {
    return renderer.Render(w, "index.html", ui.Page{Title: title}, data)
}
```

Key types:

| Type | Purpose |
|------|---------|
| `ui.App` | App-wide config: `Name` (shown in header/browser tab), `BaseURL` (defaults to `"/"`) |
| `ui.Page` | Per-page metadata: `Title` (shown in `<title>` tag) |
| `renderer.Render(w, name, page, data)` | Executes a named template with app+page+data available as `.App`, `.Page`, `.Data` |
| `renderer.RenderError(w, statusCode, message)` | Renders the shared `once/error` template |

### 3. Mount routes

In your app handler (or `Routes()` method), register the static assets handler
and your page routes:

```go
import "once-stack/pkg/ui"

func (a *App) Routes() *http.ServeMux {
    mux := http.NewServeMux()
    mux.Handle("GET /assets/once/", ui.AssetsHandler())
    mux.HandleFunc("GET /{$}", a.handleIndex)
    // ... more routes
    return mux
}
```

**Do not** register `GET /up` — `pkg/server.New()` does that automatically.

### 4. Add a `main.go`

Follow the minimal app skeleton in [`AGENTS.md`](../AGENTS.md#app-skeleton) —
call `storage.OpenDir()`, build a mux, pass it to `server.New()`.

---

## Shared component classes vs raw Tailwind utilities

The file `pkg/ui/styles/once.css` provides two layers:

### Theme tokens (always available)

Any Tailwind utility that references an `--color-once-*` or `--container-once-*`
or `--radius-once-*` CSS variable is fair game. Examples:

```html
<!-- Using theme tokens via Tailwind utilities -->
<div class="bg-once-bg text-once-text border border-once-border">
<p class="text-once-muted text-sm">
<a class="text-once-primary hover:text-once-primary-hover">
```

### Component classes (for repeated UI primitives)

Pre-composed classes in `@layer components` for standard interactions. These
are the **preferred** way to render buttons, cards, form controls, alerts, and
empty states:

| Class | Use case |
|-------|----------|
| `once-btn once-btn-primary` | Primary call-to-action button |
| `once-btn once-btn-secondary` | Cancel / alternative action |
| `once-btn once-btn-danger` | Destructive action (delete) |
| `once-btn once-btn-ghost` | Low-emphasis text button |
| `once-card` / `once-card-hover` | Cards, list items |
| `once-input` | Text input |
| `once-textarea` | Multi-line textarea |
| `once-label` / `once-field` | Form label / field wrapper |
| `once-empty` / `once-empty-title` / `once-empty-body` | Empty states |
| `once-alert once-alert-info` / `once-alert-danger` | Flash messages, errors |
| `once-prose` | Markdown-rendered content container |

**Rule of thumb**: Use component classes for common interactive elements
(buttons, form controls, cards). Use raw Tailwind utilities for one-off layout
(alignment, spacing, sizing, grids) and for text styling (font size, weight,
color).

### Template partials (Go template reuse)

The file `pkg/ui/templates/components.html` provides `{{template "once/flash" .}}`
and `{{template "once/empty-state" .}}` for common content patterns.

---

## Tailwind build command

The CSS pipeline uses [Bun](https://bun.sh) + [Tailwind CLI v4](https://tailwindcss.com):

```bash
# Build CSS (run after changing pkg/ui/styles/once.css or any @source files)
just css-build

# The full quality pipeline (checks CSS is up-to-date, then fmt, vet, test)
just quality
```

The build command is:

```
bun run css:build
```

which runs:

```
tailwindcss -i pkg/ui/styles/once.css -o pkg/ui/static/once.css --minify
```

Tailwind scans all `.html` and `.go` files under `pkg/` and `cmd/` for class
usage via the `@source` directives in `once.css`. After changing any template
or adding new Tailwind classes, re-run `just css-build` and commit the updated
`pkg/ui/static/once.css`.

### CI check

`just quality` runs `just css-check` which builds CSS and fails if the
committed output differs from the freshly-built version. This ensures the
compiled CSS is never stale.

---

## Available shared templates

Templates live under `pkg/ui/templates/` and are embedded into every binary.

| Template name | Purpose |
|---------------|---------|
| `once/header` | Opens `<html>`, `<head>` with CSS link, opens `<body>` with app header bar |
| `once/footer` | Closes `<main>` and `<body>`, `</html>` |
| `once/error` | Full error page using `once/header` + `once/footer` |
| `once/flash` | Flash/alert message partial (expects `.Data.Flash`) |
| `once/empty-state` | Empty-state partial (expects `.Data.Title`, `.Data.Body`, `.Data.ActionURL`) |
| `once/header-extra` | **Block** — override in app templates to add nav items to the header |

Inside app templates, use `{{template "once/header" .}}` and
`{{template "once/footer" .}}` to wrap page content. For error pages, use the
`renderer.RenderError()` method or invoke `{{template "once/error" .}}`
directly.

---

## Adding shared header items

To add links or buttons to the header bar (e.g., a search form), define the
`once/header-extra` block in any app template:

```html
{{define "once/header-extra"}}
<div class="flex items-center gap-2">
    <input type="search" class="once-input" placeholder="Search…">
</div>
{{end}}
```

This block is empty by default. Only apps that need extra header content
need to define it.
