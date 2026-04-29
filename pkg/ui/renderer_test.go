package ui

import (
	"bytes"
	"embed"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

//go:embed testdata/templates/*.html
var testAppFS embed.FS

func TestNewRenderer(t *testing.T) {
	app := App{Name: "TestApp", BaseURL: "/"}
	_, err := NewRenderer(app, testAppFS, "testdata/templates/test-page.html")
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}
}

func TestNewRenderer_DefaultBaseURL(t *testing.T) {
	app := App{Name: "TestApp"}
	r, err := NewRenderer(app, testAppFS, "testdata/templates/test-page.html")
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}
	if r.app.BaseURL != "/" {
		t.Errorf("BaseURL = %q, want %q", r.app.BaseURL, "/")
	}
}

func TestNewRenderer_NoPatterns(t *testing.T) {
	app := App{Name: "TestApp"}
	_, err := NewRenderer(app, testAppFS)
	if err == nil {
		t.Fatal("expected error for empty patterns, got nil")
	}
	if !strings.Contains(err.Error(), "no template patterns") {
		t.Errorf(`error = %q, want "no template patterns"`, err.Error())
	}
}

func TestNewRenderer_NameCollision(t *testing.T) {
	app := App{Name: "TestApp"}
	_, err := NewRenderer(app, testAppFS, "testdata/templates/collision.html")
	if err == nil {
		t.Fatal("expected error for template name collision, got nil")
	}
	if !strings.Contains(err.Error(), "collision") {
		t.Errorf(`error = %q, want "collision"`, err.Error())
	}
}

func TestRender(t *testing.T) {
	app := App{Name: "TestApp", BaseURL: "/"}
	r, err := NewRenderer(app, testAppFS, "testdata/templates/test-page.html")
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	var buf bytes.Buffer
	page := Page{Title: "Hello"}
	data := map[string]string{"message": "world"}
	if err := r.Render(&buf, "test-page", page, data); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, "Hello — TestApp") {
		t.Errorf("expected title 'Hello — TestApp', got:\n%s", html)
	}
	if !strings.Contains(html, "world") {
		t.Errorf("expected body to contain 'world', got:\n%s", html)
	}
	if !strings.Contains(html, `<link rel="stylesheet" href="/assets/once/once.css">`) {
		t.Errorf("expected link to once.css, got:\n%s", html)
	}
	if !strings.Contains(html, `<a href="/"`) {
		t.Errorf("expected base URL link, got:\n%s", html)
	}
}

func TestRender_NoTitle(t *testing.T) {
	app := App{Name: "TestApp", BaseURL: "/"}
	r, err := NewRenderer(app, testAppFS, "testdata/templates/test-page.html")
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	var buf bytes.Buffer
	page := Page{Title: ""}
	data := map[string]string{}
	if err := r.Render(&buf, "test-page", page, data); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "<title>TestApp</title>") {
		t.Errorf("expected title 'TestApp' without prefix, got:\n%s", html)
	}
}

func TestRenderEscapesUntrustedTitleAndData(t *testing.T) {
	app := App{Name: "TestApp", BaseURL: "/"}
	r, err := NewRenderer(app, testAppFS, "testdata/templates/test-page.html")
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	var buf bytes.Buffer
	page := Page{Title: `<script>alert("title")</script>`}
	data := map[string]string{"message": `<img src=x onerror=alert("body")>`}
	if err := r.Render(&buf, "test-page", page, data); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := buf.String()
	if strings.Contains(html, `<script>alert("title")</script>`) {
		t.Errorf("expected title to be escaped, got:\n%s", html)
	}
	if strings.Contains(html, `<img src=x onerror=alert("body")>`) {
		t.Errorf("expected data to be escaped, got:\n%s", html)
	}
	if !strings.Contains(html, `&lt;script&gt;`) || !strings.Contains(html, `&lt;img`) {
		t.Errorf("expected escaped title and body markers, got:\n%s", html)
	}
}

func TestRenderError(t *testing.T) {
	app := App{Name: "TestApp", BaseURL: "/"}
	r, err := NewRenderer(app, testAppFS, "testdata/templates/test-page.html")
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	var buf bytes.Buffer
	if err := r.RenderError(&buf, http.StatusNotFound, "The page you requested could not be found."); err != nil {
		t.Fatalf("RenderError failed: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, "<title>404 Not Found — TestApp</title>") {
		t.Errorf("expected title '404 Not Found — TestApp', got:\n%s", html)
	}
	if !strings.Contains(html, "404") {
		t.Errorf("expected body to contain status code 404, got:\n%s", html)
	}
	if !strings.Contains(html, "The page you requested could not be found.") {
		t.Errorf("expected body to contain error message, got:\n%s", html)
	}
	if !strings.Contains(html, "Back to TestApp") {
		t.Errorf("expected link back to app, got:\n%s", html)
	}
	if !strings.Contains(html, `<link rel="stylesheet" href="/assets/once/once.css">`) {
		t.Errorf("expected link to once.css, got:\n%s", html)
	}
}

func TestRenderError_InternalServerError(t *testing.T) {
	app := App{Name: "TestApp", BaseURL: "/"}
	r, err := NewRenderer(app, testAppFS, "testdata/templates/test-page.html")
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	var buf bytes.Buffer
	if err := r.RenderError(&buf, http.StatusInternalServerError, "Something went wrong."); err != nil {
		t.Fatalf("RenderError failed: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "<title>500 Internal Server Error — TestApp</title>") {
		t.Errorf("expected title '500 Internal Server Error — TestApp', got:\n%s", html)
	}
	if !strings.Contains(html, "Something went wrong.") {
		t.Errorf("expected body to contain error message, got:\n%s", html)
	}
}

func TestAssetsHandler(t *testing.T) {
	h := AssetsHandler()
	srv := httptest.NewServer(h)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/assets/once/once.css")
	if err != nil {
		t.Fatalf("GET /assets/once/once.css failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("GET /assets/once/once.css = %d, want %d", res.StatusCode, http.StatusOK)
	}
	if ct := res.Header.Get("Content-Type"); ct != "text/css; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/css", ct)
	}

	body := new(bytes.Buffer)
	body.ReadFrom(res.Body)
	if !strings.Contains(body.String(), "tailwindcss") {
		t.Errorf("expected compiled tailwindcss output, got:\n%s", body.String())
	}
}

func TestAssetsHandler_404(t *testing.T) {
	h := AssetsHandler()
	srv := httptest.NewServer(h)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/assets/once/nonexistent.css")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("GET /assets/once/nonexistent.css = %d, want %d", res.StatusCode, http.StatusNotFound)
	}
}
