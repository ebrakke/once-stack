package notes

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	return NewApp(s)
}

// noRedirectClient returns an HTTP client that does not follow redirects.
func noRedirectClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func TestApp_Routes_Index(t *testing.T) {
	app := newTestApp(t)
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()

	res, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("GET / = %d, want %d", res.StatusCode, http.StatusOK)
	}
}

func TestApp_Routes_NewForm(t *testing.T) {
	app := newTestApp(t)
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()

	res, err := http.Get(srv.URL + "/new")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("GET /new = %d, want %d", res.StatusCode, http.StatusOK)
	}
}

func TestApp_Routes_CreateAndView(t *testing.T) {
	app := newTestApp(t)
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()

	client := noRedirectClient()

	// Create a note via POST.
	res, err := client.PostForm(srv.URL+"/notes", url.Values{
		"title": {"Test Note"},
		"body":  {"Hello *world*"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusSeeOther {
		t.Errorf("POST /notes = %d, want %d", res.StatusCode, http.StatusSeeOther)
	}

	loc := res.Header.Get("Location")
	if !strings.HasPrefix(loc, "/notes/") {
		t.Errorf("Location header = %q, want /notes/...", loc)
	}

	// Follow redirect to view.
	res, err = http.Get(srv.URL + loc)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("GET %s = %d, want %d", loc, res.StatusCode, http.StatusOK)
	}
}

func TestApp_Routes_EditForm(t *testing.T) {
	app := newTestApp(t)
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()

	// Create a note first.
	store := app.Store
	n, err := store.Create("Edit Test", "some content")
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.Get(srv.URL + "/notes/" + n.ID + "/edit")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("GET /notes/%s/edit = %d, want %d", n.ID, res.StatusCode, http.StatusOK)
	}
}

func TestApp_Routes_Update(t *testing.T) {
	app := newTestApp(t)
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()

	store := app.Store
	n, err := store.Create("Update Me", "original")
	if err != nil {
		t.Fatal(err)
	}

	client := noRedirectClient()

	res, err := client.PostForm(srv.URL+"/notes/"+n.ID, url.Values{
		"title": {"Updated Title"},
		"body":  {"updated content"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusSeeOther {
		t.Errorf("POST /notes/%s = %d, want %d", n.ID, res.StatusCode, http.StatusSeeOther)
	}

	loc := res.Header.Get("Location")
	if loc != "/notes/"+n.ID {
		t.Errorf("Location = %q, want /notes/%s", loc, n.ID)
	}

	// Verify update persisted.
	note, body, err := store.Get(n.ID)
	if err != nil {
		t.Fatal(err)
	}
	if note.Title != "Updated Title" {
		t.Errorf("title = %q, want %q", note.Title, "Updated Title")
	}
	if body != "updated content" {
		t.Errorf("body = %q, want %q", body, "updated content")
	}
}

func TestApp_Routes_Delete(t *testing.T) {
	app := newTestApp(t)
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()

	store := app.Store
	n, err := store.Create("Delete Me", "bye")
	if err != nil {
		t.Fatal(err)
	}

	client := noRedirectClient()

	res, err := client.PostForm(srv.URL+"/notes/"+n.ID+"/delete", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusSeeOther {
		t.Errorf("POST /notes/%s/delete = %d, want %d", n.ID, res.StatusCode, http.StatusSeeOther)
	}

	loc := res.Header.Get("Location")
	if loc != "/" {
		t.Errorf("Location = %q, want /", loc)
	}

	// Verify deletion.
	_, _, err = store.Get(n.ID)
	if err == nil {
		t.Error("expected error getting deleted note")
	}
}

func TestApp_Routes_NotFound(t *testing.T) {
	app := newTestApp(t)
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()

	res, err := http.Get(srv.URL + "/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("GET /nonexistent = %d, want %d", res.StatusCode, http.StatusNotFound)
	}
}

func TestApp_Routes_InvalidID(t *testing.T) {
	app := newTestApp(t)
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()

	res, err := http.Get(srv.URL + "/notes/../secret")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("GET /notes/../secret = %d, want %d", res.StatusCode, http.StatusNotFound)
	}
}

func TestApp_Routes_BodyTooLarge(t *testing.T) {
	app := newTestApp(t)
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()

	client := noRedirectClient()

	largeBody := strings.Repeat("A", maxBodySize+1)
	res, err := client.PostForm(srv.URL+"/notes", url.Values{
		"title": {"Large"},
		"body":  {largeBody},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	// Should get an error page (not 500, not redirect).
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("POST /notes with oversized body = %d, want %d", res.StatusCode, http.StatusBadRequest)
	}
}

func TestApp_Routes_Search(t *testing.T) {
	app := newTestApp(t)
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()

	store := app.Store
	_, err := store.Create("Alpha Note", "first one")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Create("Beta Note", "second one")
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.Get(srv.URL + "/?q=beta")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("GET /?q=beta = %d, want %d", res.StatusCode, http.StatusOK)
	}
}

func TestApp_Routes_EmptyList(t *testing.T) {
	app := newTestApp(t)
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()

	res, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("GET / = %d, want %d", res.StatusCode, http.StatusOK)
	}
}

// TestEmbeddedTemplatesParsed verifies that the embedded templates render correctly.
func TestEmbeddedTemplatesParsed(t *testing.T) {
	if renderer == nil {
		t.Fatal("renderer not initialized")
	}

	cases := []struct {
		name  string
		fn    func(io.Writer, string, *TemplateData) error
		title string
		data  *TemplateData
	}{
		{"index.html", RenderIndex, "", &TemplateData{Notes: []Note{}}},
		{"view.html", RenderView, "View", &TemplateData{Note: &Note{ID: "a", Title: "T"}, BodyHTML: "<p>x</p>"}},
		{"form.html", RenderForm, "Form", &TemplateData{IsNew: true}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := c.fn(&buf, c.title, c.data); err != nil {
				t.Fatalf("render %s: %v", c.name, err)
			}
		})
	}

	t.Run("app-error.html", func(t *testing.T) {
		var buf bytes.Buffer
		if err := RenderError(&buf, "Error", "Something went wrong."); err != nil {
			t.Fatalf("RenderError: %v", err)
		}
	})
}

func TestNotesRenderRegression(t *testing.T) {
	note := Note{ID: "unsafe-note", Title: `<b>Unsafe</b>`}

	t.Run("index", func(t *testing.T) {
		var buf bytes.Buffer
		data := &TemplateData{Notes: []Note{note}}
		if err := RenderIndex(&buf, "", data); err != nil {
			t.Fatalf("RenderIndex: %v", err)
		}
		html := buf.String()
		assertContains(t, html, `<title>Notes</title>`)
		assertContains(t, html, `<link rel="stylesheet" href="/assets/once/once.css">`)
		assertContains(t, html, `once-card`)
		assertContains(t, html, `&lt;b&gt;Unsafe&lt;/b&gt;`)
		assertNotContains(t, html, `<b>Unsafe</b>`)
	})

	t.Run("view", func(t *testing.T) {
		var buf bytes.Buffer
		data := &TemplateData{
			Note:     &note,
			BodyHTML: `<p><strong>sanitized markdown</strong></p>`,
		}
		if err := RenderView(&buf, `Unsafe <Title>`, data); err != nil {
			t.Fatalf("RenderView: %v", err)
		}
		html := buf.String()
		assertContains(t, html, `<title>Unsafe &lt;Title&gt; — Notes</title>`)
		assertContains(t, html, `once-prose`)
		assertContains(t, html, `<strong>sanitized markdown</strong>`)
	})

	t.Run("form", func(t *testing.T) {
		var buf bytes.Buffer
		data := &TemplateData{IsNew: true}
		if err := RenderForm(&buf, "New Note", data); err != nil {
			t.Fatalf("RenderForm: %v", err)
		}
		html := buf.String()
		assertContains(t, html, `<title>New Note — Notes</title>`)
		assertContains(t, html, `action="/notes"`)
		assertContains(t, html, `once-input`)
		assertContains(t, html, `once-textarea`)
	})

	t.Run("error", func(t *testing.T) {
		var buf bytes.Buffer
		if err := RenderError(&buf, "Not Found", `Missing <thing>`); err != nil {
			t.Fatalf("RenderError: %v", err)
		}
		html := buf.String()
		assertContains(t, html, `<title>Not Found — Notes</title>`)
		assertContains(t, html, `Missing &lt;thing&gt;`)
		assertContains(t, html, `Back to Notes`)
	})
}

func assertContains(t *testing.T, s, want string) {
	t.Helper()
	if !strings.Contains(s, want) {
		t.Fatalf("expected output to contain %q, got:\n%s", want, s)
	}
}

func assertNotContains(t *testing.T, s, unwanted string) {
	t.Helper()
	if strings.Contains(s, unwanted) {
		t.Fatalf("expected output not to contain %q, got:\n%s", unwanted, s)
	}
}

// TestFileOps tests that file creation, reading, and listing work through the handlers.
func TestFileOps(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	app := NewApp(s)
	srv := httptest.NewServer(app.Routes())
	defer srv.Close()

	client := noRedirectClient()

	// Create.
	res, err := client.PostForm(srv.URL+"/notes", url.Values{"title": {"File Op"}, "body": {"test"}})
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	loc := res.Header.Get("Location")

	// View.
	res, err = http.Get(srv.URL + loc)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("view = %d, want %d", res.StatusCode, http.StatusOK)
	}

	// The body file should exist on disk.
	bodyPath := s.bodyPath("file-op")
	if _, err := os.Stat(bodyPath); os.IsNotExist(err) {
		t.Errorf("body file %q should exist", bodyPath)
	}
}
