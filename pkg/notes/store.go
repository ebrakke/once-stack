package notes

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	metadataFile = "notes.json"
	bodiesDir    = "notes"
	tmpSuffix    = ".tmp"
)

// Store provides filesystem-backed persistence for notes.
// Metadata is stored as JSON in <dir>/notes.json and note bodies are stored
// as individual markdown files in <dir>/notes/<id>.md.
type Store struct {
	mu  sync.Mutex
	dir string

	// in-memory index of notes by ID for fast lookups
	notes []Note
	byID  map[string]int // ID -> index in notes slice
}

type storeData struct {
	Notes []Note `json:"notes"`
}

// NewStore creates a new Store rooted at dir, creating the required
// directories and loading any existing metadata.
func NewStore(dir string) (*Store, error) {
	if dir == "" {
		return nil, fmt.Errorf("store dir must not be empty")
	}

	// Ensure the store root and bodies directory exist.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create store dir %q: %w", dir, err)
	}
	bodiesDirPath := filepath.Join(dir, bodiesDir)
	if err := os.MkdirAll(bodiesDirPath, 0o755); err != nil {
		return nil, fmt.Errorf("create bodies dir %q: %w", bodiesDirPath, err)
	}

	s := &Store{
		dir:  dir,
		byID: make(map[string]int),
	}

	if err := s.load(); err != nil {
		return nil, fmt.Errorf("load metadata: %w", err)
	}

	return s, nil
}

// load reads metadata from disk into memory. If the file doesn't exist,
// it starts with an empty store.
func (s *Store) load() error {
	path := filepath.Join(s.dir, metadataFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			s.notes = nil
			s.byID = make(map[string]int)
			return nil
		}
		return fmt.Errorf("read metadata: %w", err)
	}

	var sd storeData
	if err := json.Unmarshal(data, &sd); err != nil {
		return fmt.Errorf("unmarshal metadata: %w", err)
	}

	s.notes = sd.Notes
	s.byID = make(map[string]int, len(sd.Notes))
	for i, n := range sd.Notes {
		s.byID[n.ID] = i
	}

	return nil
}

// save writes the current metadata to disk atomically using a temp file
// followed by rename.
func (s *Store) save() error {
	sd := storeData{Notes: s.notes}
	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	path := filepath.Join(s.dir, metadataFile)
	tmpPath := path + tmpSuffix

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write temp metadata: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		// Try to clean up the temp file on error.
		os.Remove(tmpPath)
		return fmt.Errorf("rename metadata: %w", err)
	}

	return nil
}

// bodyPath returns the filesystem path for a note's body file.
func (s *Store) bodyPath(id string) string {
	return filepath.Join(s.dir, bodiesDir, id+".md")
}

// List returns all notes sorted by UpdatedAt descending.
func (s *Store) List() ([]Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.listLocked(), nil
}

func (s *Store) listLocked() []Note {
	result := make([]Note, len(s.notes))
	copy(result, s.notes)
	sortNotesByUpdatedAt(result)
	return result
}

// Get returns a note and its body content by ID.
func (s *Store) Get(id string) (Note, string, error) {
	if err := ValidateID(id); err != nil {
		return Note{}, "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx, ok := s.byID[id]
	if !ok {
		return Note{}, "", fmt.Errorf("note not found: %s", id)
	}

	body, err := os.ReadFile(s.bodyPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return Note{}, "", fmt.Errorf("note body not found: %s", id)
		}
		return Note{}, "", fmt.Errorf("read note body %q: %w", id, err)
	}

	return s.notes[idx], string(body), nil
}

// Create creates a new note with the given title and body. If the generated
// ID (slug) conflicts with an existing note, a numeric suffix is appended
// to ensure uniqueness.
func (s *Store) Create(title, body string) (Note, error) {
	title = CleanTitle(title)
	if title == "" {
		return Note{}, fmt.Errorf("note title must not be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := SlugifyTitle(title)
	id = s.uniqueIDLocked(id)

	now := time.Now().UTC()
	n := Note{
		ID:        id,
		Title:     title,
		Slug:      id,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Write body file first.
	if err := os.WriteFile(s.bodyPath(id), []byte(body), 0o644); err != nil {
		return Note{}, fmt.Errorf("write note body %q: %w", id, err)
	}

	// Append to metadata.
	s.notes = append(s.notes, n)
	s.byID[id] = len(s.notes) - 1

	if err := s.save(); err != nil {
		// Rollback: remove the body file and undo the in-memory changes.
		os.Remove(s.bodyPath(id))
		s.notes = s.notes[:len(s.notes)-1]
		delete(s.byID, id)
		return Note{}, fmt.Errorf("save metadata after create: %w", err)
	}

	return n, nil
}

// Update updates an existing note's title and body.
func (s *Store) Update(id, title, body string) (Note, error) {
	if err := ValidateID(id); err != nil {
		return Note{}, err
	}

	title = CleanTitle(title)
	if title == "" {
		return Note{}, fmt.Errorf("note title must not be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx, ok := s.byID[id]
	if !ok {
		return Note{}, fmt.Errorf("note not found: %s", id)
	}

	now := time.Now().UTC()
	n := Note{
		ID:        id,
		Title:     title,
		Slug:      id,
		CreatedAt: s.notes[idx].CreatedAt,
		UpdatedAt: now,
	}

	// Write body file first.
	if err := os.WriteFile(s.bodyPath(id), []byte(body), 0o644); err != nil {
		return Note{}, fmt.Errorf("write note body %q: %w", id, err)
	}

	// Update in-memory metadata.
	s.notes[idx] = n

	if err := s.save(); err != nil {
		// Note: if save fails, the body file has already been written but the
		// in-memory metadata is unchanged so the store remains consistent.
		return Note{}, fmt.Errorf("save metadata after update: %w", err)
	}

	return n, nil
}

// Delete removes a note and its body file from storage.
func (s *Store) Delete(id string) error {
	if err := ValidateID(id); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx, ok := s.byID[id]
	if !ok {
		return fmt.Errorf("note not found: %s", id)
	}

	// Remove body file.
	if err := os.Remove(s.bodyPath(id)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove note body %q: %w", id, err)
	}

	// Remove from in-memory metadata.
	s.notes = append(s.notes[:idx], s.notes[idx+1:]...)
	delete(s.byID, id)

	// Rebuild byID indices after removal.
	for i := idx; i < len(s.notes); i++ {
		s.byID[s.notes[i].ID] = i
	}

	if err := s.save(); err != nil {
		return fmt.Errorf("save metadata after delete: %w", err)
	}

	return nil
}

// Search returns notes whose title or body contain the query string
// (case-insensitive). Results are sorted by UpdatedAt descending.
func (s *Store) Search(query string) ([]Note, error) {
	if query == "" {
		return s.List()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	q := strings.ToLower(query)
	var results []Note

	for _, n := range s.notes {
		if strings.Contains(strings.ToLower(n.Title), q) {
			results = append(results, n)
			continue
		}
		// Read body and check.
		body, err := os.ReadFile(s.bodyPath(n.ID))
		if err != nil {
			// Skip notes with unreadable bodies.
			continue
		}
		if strings.Contains(strings.ToLower(string(body)), q) {
			results = append(results, n)
		}
	}

	sortNotesByUpdatedAt(results)
	return results, nil
}

// uniqueIDLocked generates a unique ID by appending numeric suffixes until
// no collision is found. Must be called with s.mu held.
func (s *Store) uniqueIDLocked(base string) string {
	if _, exists := s.byID[base]; !exists {
		return base
	}
	counter := 2
	for {
		candidate := fmt.Sprintf("%s-%d", base, counter)
		if _, exists := s.byID[candidate]; !exists {
			return candidate
		}
		counter++
	}
}

// sortNotesByUpdatedAt sorts notes in place by UpdatedAt descending.
func sortNotesByUpdatedAt(notes []Note) {
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].UpdatedAt.After(notes[j].UpdatedAt)
	})
}
