package notes

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
)

const maxBodySize = 1 << 20 // 1 MB

// App wires a Store to HTTP handlers.
type App struct {
	Store *Store
}

// NewApp creates an App with the given Store.
func NewApp(store *Store) *App {
	return &App{Store: store}
}

// Routes returns a *http.ServeMux that serves the notes web interface.
// Routes are registered on a new ServeMux; /up is NOT registered here.
func (a *App) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", a.handleIndex)
	mux.HandleFunc("GET /new", a.handleNewForm)
	mux.HandleFunc("POST /notes", a.handleCreate)
	mux.HandleFunc("GET /notes/{id}", a.handleView)
	mux.HandleFunc("GET /notes/{id}/edit", a.handleEditForm)
	mux.HandleFunc("POST /notes/{id}", a.handleUpdate)
	mux.HandleFunc("POST /notes/{id}/delete", a.handleDelete)
	return mux
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	var notes []Note
	var err error
	if q != "" {
		notes, err = a.Store.Search(q)
	} else {
		notes, err = a.Store.List()
	}
	if err != nil {
		slog.Error("list notes", "err", err)
		renderError(w, http.StatusInternalServerError, "List Failed", "Could not load notes.")
		return
	}

	data := &TemplateData{
		Title: "Notes",
		Notes: notes,
	}
	if err := RenderIndex(w, data); err != nil {
		slog.Error("render index", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (a *App) handleNewForm(w http.ResponseWriter, r *http.Request) {
	data := &TemplateData{
		Title: "New Note",
		IsNew: true,
	}
	if err := RenderForm(w, data); err != nil {
		slog.Error("render new form", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (a *App) handleCreate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	if err := r.ParseForm(); err != nil {
		slog.Error("parse create form", "err", err)
		renderError(w, http.StatusBadRequest, "Bad Request", "Request too large or malformed.")
		return
	}

	title := r.PostFormValue("title")
	body := r.PostFormValue("body")

	note, err := a.Store.Create(title, body)
	if err != nil {
		slog.Error("create note", "err", err)
		renderError(w, http.StatusBadRequest, "Create Failed", "Could not create note: "+err.Error())
		return
	}

	http.Redirect(w, r, "/notes/"+note.ID, http.StatusSeeOther)
}

func (a *App) handleView(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := ValidateID(id); err != nil {
		renderError(w, http.StatusNotFound, "Not Found", "Note not found.")
		return
	}

	note, body, err := a.Store.Get(id)
	if err != nil {
		slog.Error("get note", "id", id, "err", err)
		renderError(w, http.StatusNotFound, "Not Found", "Note not found.")
		return
	}

	bodyHTML, err := RenderMarkdown(body)
	if err != nil {
		slog.Error("render markdown", "id", id, "err", err)
		renderError(w, http.StatusInternalServerError, "Render Error", "Could not render note content.")
		return
	}

	data := &TemplateData{
		Title:    note.Title + " — Notes",
		Note:     &note,
		BodyHTML: bodyHTML,
	}
	if err := RenderView(w, data); err != nil {
		slog.Error("render view", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (a *App) handleEditForm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := ValidateID(id); err != nil {
		renderError(w, http.StatusNotFound, "Not Found", "Note not found.")
		return
	}

	note, body, err := a.Store.Get(id)
	if err != nil {
		slog.Error("get note for edit", "id", id, "err", err)
		renderError(w, http.StatusNotFound, "Not Found", "Note not found.")
		return
	}

	data := &TemplateData{
		Title:   "Edit " + note.Title + " — Notes",
		Note:    &note,
		Content: body,
		Slug:    note.ID,
		IsNew:   false,
	}
	if err := RenderForm(w, data); err != nil {
		slog.Error("render edit form", "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (a *App) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := ValidateID(id); err != nil {
		renderError(w, http.StatusNotFound, "Not Found", "Note not found.")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	if err := r.ParseForm(); err != nil {
		slog.Error("parse update form", "err", err)
		renderError(w, http.StatusBadRequest, "Bad Request", "Request too large or malformed.")
		return
	}

	title := r.PostFormValue("title")
	body := r.PostFormValue("body")

	note, err := a.Store.Update(id, title, body)
	if err != nil {
		slog.Error("update note", "id", id, "err", err)
		renderError(w, http.StatusBadRequest, "Update Failed", "Could not update note: "+err.Error())
		return
	}

	http.Redirect(w, r, "/notes/"+note.ID, http.StatusSeeOther)
}

func (a *App) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := ValidateID(id); err != nil {
		renderError(w, http.StatusNotFound, "Not Found", "Note not found.")
		return
	}

	if err := a.Store.Delete(id); err != nil {
		slog.Error("delete note", "id", id, "err", err)
		renderError(w, http.StatusInternalServerError, "Delete Failed", "Could not delete note: "+err.Error())
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// renderError renders a formatted error page with the given HTTP status.
func renderError(w http.ResponseWriter, status int, title, message string) {
	data := &TemplateData{
		Title: title,
		Error: message,
	}
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, "error.html", data); err != nil {
		slog.Error("render error", "err", err)
		http.Error(w, title+": "+message, status)
		return
	}
	w.WriteHeader(status)
	w.Write(buf.Bytes())
}
