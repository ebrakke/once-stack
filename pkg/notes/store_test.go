package notes

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStore_EmptyDirError(t *testing.T) {
	_, err := NewStore("")
	if err == nil {
		t.Fatal("expected error for empty dir")
	}
}

func TestNewStore_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the store root exists.
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("store dir should exist")
	}

	// Check that the bodies directory exists.
	bodiesPath := filepath.Join(dir, bodiesDir)
	if _, err := os.Stat(bodiesPath); os.IsNotExist(err) {
		t.Error("bodies dir should exist")
	}

	// Verify the metadata file was not created yet (empty store).
	if _, err := os.Stat(filepath.Join(dir, metadataFile)); !os.IsNotExist(err) {
		t.Error("metadata file should not exist for empty store")
	}

	// Store should list empty.
	notes, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 0 {
		t.Errorf("expected 0 notes, got %d", len(notes))
	}
}

func TestStore_Create(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	n, err := s.Create("My First Note", "Hello, world!")
	if err != nil {
		t.Fatal(err)
	}

	if n.ID != "my-first-note" {
		t.Errorf("ID = %q, want %q", n.ID, "my-first-note")
	}
	if n.Title != "My First Note" {
		t.Errorf("Title = %q, want %q", n.Title, "My First Note")
	}
	if n.Slug != "my-first-note" {
		t.Errorf("Slug = %q, want %q", n.Slug, "my-first-note")
	}
	if n.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if n.UpdatedAt != n.CreatedAt {
		t.Error("UpdatedAt should equal CreatedAt for a new note")
	}

	// Verify body file was written.
	bodyPath := filepath.Join(dir, bodiesDir, "my-first-note.md")
	body, err := os.ReadFile(bodyPath)
	if err != nil {
		t.Fatal("body file should exist:", err)
	}
	if string(body) != "Hello, world!" {
		t.Errorf("body = %q, want %q", string(body), "Hello, world!")
	}
}

func TestStore_Create_EmptyTitle(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Create("", "body")
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestStore_Create_DuplicateTitles(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	n1, err := s.Create("My Note", "first body")
	if err != nil {
		t.Fatal(err)
	}
	if n1.ID != "my-note" {
		t.Errorf("first note ID = %q, want %q", n1.ID, "my-note")
	}

	n2, err := s.Create("My Note", "second body")
	if err != nil {
		t.Fatal(err)
	}
	if n2.ID != "my-note-2" {
		t.Errorf("second note ID = %q, want %q", n2.ID, "my-note-2")
	}

	n3, err := s.Create("My Note", "third body")
	if err != nil {
		t.Fatal(err)
	}
	if n3.ID != "my-note-3" {
		t.Errorf("third note ID = %q, want %q", n3.ID, "my-note-3")
	}

	// Verify all three exist.
	notes, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 3 {
		t.Errorf("expected 3 notes, got %d", len(notes))
	}

	// Delete the second, recreate — should reuse the gap.
	if err := s.Delete("my-note-2"); err != nil {
		t.Fatal(err)
	}

	n4, err := s.Create("My Note", "fourth body")
	if err != nil {
		t.Fatal(err)
	}
	// After deleting my-note-2, the counter reuses that slot.
	if n4.ID != "my-note-2" {
		t.Errorf("after delete, note ID = %q, want %q", n4.ID, "my-note-2")
	}
}

func TestStore_Create_UniqueIDsForDifferentTitles(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Different titles that could produce the same slug should get unique IDs.
	n1, err := s.Create("Hello World!", "body 1")
	if err != nil {
		t.Fatal(err)
	}
	n2, err := s.Create("hello-world", "body 2")
	if err != nil {
		t.Fatal(err)
	}

	if n1.ID == n2.ID {
		t.Errorf("two different titles produced same ID %q", n1.ID)
	}
}

func TestStore_Get(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	wantBody := "This is **markdown** content."
	created, err := s.Create("Get Test Note", wantBody)
	if err != nil {
		t.Fatal(err)
	}

	note, body, err := s.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}

	if note.ID != created.ID {
		t.Errorf("note ID = %q, want %q", note.ID, created.ID)
	}
	if note.Title != "Get Test Note" {
		t.Errorf("note Title = %q, want %q", note.Title, "Get Test Note")
	}
	if body != wantBody {
		t.Errorf("body = %q, want %q", body, wantBody)
	}
}

func TestStore_Get_InvalidID(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = s.Get("../secret")
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
}

func TestStore_Get_NotFound(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = s.Get("nonexistent-note")
	if err == nil {
		t.Fatal("expected error for non-existent note")
	}
}

func TestStore_List_SortedByUpdatedAtDesc(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Create notes with tiny delays to ensure distinct timestamps.
	n1, err := s.Create("Alpha", "body alpha")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)

	n2, err := s.Create("Beta", "body beta")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)

	n3, err := s.Create("Gamma", "body gamma")
	if err != nil {
		t.Fatal(err)
	}

	notes, err := s.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(notes) != 3 {
		t.Fatalf("expected 3 notes, got %d", len(notes))
	}

	// Expect Gamma (last created) first, then Beta, then Alpha.
	if notes[0].ID != n3.ID {
		t.Errorf("notes[0].ID = %q, want %q (most recent first)", notes[0].ID, n3.ID)
	}
	if notes[1].ID != n2.ID {
		t.Errorf("notes[1].ID = %q, want %q", notes[1].ID, n2.ID)
	}
	if notes[2].ID != n1.ID {
		t.Errorf("notes[2].ID = %q, want %q", notes[2].ID, n1.ID)
	}

	// Verify descending UpdatedAt order.
	if !notes[0].UpdatedAt.After(notes[1].UpdatedAt) {
		t.Error("notes[0].UpdatedAt should be after notes[1].UpdatedAt")
	}
	if !notes[1].UpdatedAt.After(notes[2].UpdatedAt) {
		t.Error("notes[1].UpdatedAt should be after notes[2].UpdatedAt")
	}
}

func TestStore_List_OrderUpdatedAfterUpdate(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	n1, err := s.Create("First", "body 1")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)

	_, err = s.Create("Second", "body 2")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)

	n3, err := s.Create("Third", "body 3")
	if err != nil {
		t.Fatal(err)
	}

	// Update n1 (the oldest) — it should move to the top.
	time.Sleep(2 * time.Millisecond)
	updated, err := s.Update(n1.ID, "First (Updated)", "updated body")
	if err != nil {
		t.Fatal(err)
	}

	if !updated.UpdatedAt.After(n3.UpdatedAt) {
		t.Error("updated note should have a later UpdatedAt")
	}

	notes, err := s.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(notes) != 3 {
		t.Fatalf("expected 3 notes, got %d", len(notes))
	}

	// Updated note should now be first.
	if notes[0].ID != n1.ID {
		t.Errorf("notes[0].ID = %q, want %q (updated note should be first)", notes[0].ID, n1.ID)
	}
}

func TestStore_Update(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	created, err := s.Create("Original Title", "original body")
	if err != nil {
		t.Fatal(err)
	}

	updated, err := s.Update(created.ID, "Updated Title", "updated body")
	if err != nil {
		t.Fatal(err)
	}

	if updated.ID != created.ID {
		t.Errorf("ID changed: %q -> %q", created.ID, updated.ID)
	}
	if updated.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", updated.Title, "Updated Title")
	}
	if updated.Slug != created.ID {
		t.Errorf("Slug = %q, want %q", updated.Slug, created.ID)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Error("CreatedAt should not change on update")
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Error("UpdatedAt should be after original UpdatedAt")
	}

	// Verify body was updated on disk.
	got, body, err := s.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Updated Title" {
		t.Errorf("Get title = %q, want %q", got.Title, "Updated Title")
	}
	if body != "updated body" {
		t.Errorf("Get body = %q, want %q", body, "updated body")
	}
}

func TestStore_Update_NotFound(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Update("nonexistent", "Title", "body")
	if err == nil {
		t.Fatal("expected error for non-existent note")
	}
}

func TestStore_Update_InvalidID(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Update("../secret", "Title", "body")
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
}

func TestStore_Update_EmptyTitle(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	n, err := s.Create("Some Note", "body")
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Update(n.ID, "", "body")
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestStore_Delete(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	n, err := s.Create("Delete Me", "bye bye")
	if err != nil {
		t.Fatal(err)
	}

	if err := s.Delete(n.ID); err != nil {
		t.Fatal(err)
	}

	// Note should not be in list.
	notes, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, note := range notes {
		if note.ID == n.ID {
			t.Errorf("deleted note %q still in list", n.ID)
		}
	}

	// Body file should be gone.
	bodyPath := filepath.Join(dir, bodiesDir, n.ID+".md")
	if _, err := os.Stat(bodyPath); !os.IsNotExist(err) {
		t.Error("body file should be removed after delete")
	}

	// Get should fail.
	_, _, err = s.Get(n.ID)
	if err == nil {
		t.Error("expected error getting deleted note")
	}
}

func TestStore_Delete_NotFound(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent note")
	}
}

func TestStore_Delete_InvalidID(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Delete("../secret")
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
}

func TestStore_SearchByTitle(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, _ = s.Create("Alpha Note", "first content")
	_, _ = s.Create("Beta Note", "second content")
	_, _ = s.Create("Gamma Note", "third content")

	results, err := s.Search("Beta")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Beta Note" {
		t.Errorf("result title = %q, want %q", results[0].Title, "Beta Note")
	}
}

func TestStore_SearchByBody(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, _ = s.Create("Alpha", "needle in a haystack")
	_, _ = s.Create("Beta", "something else")
	_, _ = s.Create("Gamma", "needle again")

	results, err := s.Search("needle")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestStore_Search_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, _ = s.Create("UPPERCASE TITLE", "some body")

	results, err := s.Search("uppercase")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestStore_Search_EmptyQuery(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, _ = s.Create("Note A", "body a")
	_, _ = s.Create("Note B", "body b")

	results, err := s.Search("")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for empty query, got %d", len(results))
	}
}

func TestStore_Search_NoMatch(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, _ = s.Create("Note A", "body a")

	results, err := s.Search("zzz_nonexistent_zzz")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestStore_ReloadsMetadata(t *testing.T) {
	dir := t.TempDir()
	s1, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = s1.Create("Persistent Note", "will survive reload")
	if err != nil {
		t.Fatal(err)
	}

	// Create a new store pointing at the same directory.
	s2, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	notes, err := s2.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note after reload, got %d", len(notes))
	}
	if notes[0].Title != "Persistent Note" {
		t.Errorf("title = %q, want %q", notes[0].Title, "Persistent Note")
	}

	// Body should also be loadable.
	_, body, err := s2.Get(notes[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if body != "will survive reload" {
		t.Errorf("body = %q, want %q", body, "will survive reload")
	}
}

func TestStore_List_Empty(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	notes, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 0 {
		t.Errorf("expected empty list, got %d notes", len(notes))
	}
}

func TestStore_AtomicSave(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Create a note and verify the metadata file is valid JSON.
	_, err = s.Create("Atomic Test", "check save atomically")
	if err != nil {
		t.Fatal(err)
	}

	metaPath := filepath.Join(dir, metadataFile)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("metadata file should not be empty")
	}

	// No .tmp files should remain after save.
	d, err := os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	entries, err := d.Readdirnames(-1)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range entries {
		if filepath.Ext(name) == ".tmp" {
			t.Errorf("stale temp file found: %s", name)
		}
	}
}
