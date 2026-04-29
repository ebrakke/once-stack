package ui

// App holds application-level configuration shown on every page.
type App struct {
	Name    string
	BaseURL string // optional, defaults to "/"
}

// Page holds per-page metadata passed to templates.
type Page struct {
	Title string
}
