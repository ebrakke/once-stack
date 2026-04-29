// Package notes provides shared types and utilities for the notes application.
package notes

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// Note represents a single note's metadata. The body content is stored
// separately on disk and retrieved via Store.Get.
type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	// safeIDPattern matches valid note IDs: lowercase alphanumeric and hyphens.
	safeIDPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

	// multipleHyphens matches two or more consecutive hyphens.
	multipleHyphens = regexp.MustCompile(`-{2,}`)

	// nonSlugChars matches anything that isn't a letter, digit, space, or hyphen.
	nonSlugChars = regexp.MustCompile(`[^a-zA-Z0-9 -]`)
)

// ValidateID checks that id is a safe, non-empty note identifier.
// It returns an error if the id is empty, contains path separators, "..",
// or any character outside the allowed set [a-z0-9-].
func ValidateID(id string) error {
	if id == "" {
		return fmt.Errorf("note id must not be empty")
	}
	if strings.Contains(id, "/") || strings.Contains(id, "\\") {
		return fmt.Errorf("note id %q must not contain path separators", id)
	}
	if strings.Contains(id, "..") {
		return fmt.Errorf("note id %q must not contain '..'", id)
	}
	if !safeIDPattern.MatchString(id) {
		return fmt.Errorf("note id %q must match %s", id, safeIDPattern.String())
	}
	return nil
}

// SlugifyTitle converts a title into a URL-safe slug suitable for use as a
// note ID. It lowercases the input, removes non-alphanumeric/non-hyphen/non-space
// characters, replaces spaces with hyphens, collapses multiple hyphens, and
// trims leading/trailing hyphens.
func SlugifyTitle(title string) string {
	// Lowercase
	slug := strings.ToLower(title)

	// Collapse whitespace and strip leading/trailing whitespace
	slug = strings.Join(strings.Fields(slug), " ")

	// Remove characters that are not letters, digits, spaces, or hyphens
	slug = nonSlugChars.ReplaceAllString(slug, "")

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Collapse multiple hyphens
	slug = multipleHyphens.ReplaceAllString(slug, "-")

	// Trim leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	if slug == "" {
		slug = "untitled"
	}

	return slug
}

// CleanTitle trims leading/trailing whitespace, collapses internal whitespace
// to single spaces, and removes non-printable characters.
func CleanTitle(title string) string {
	// Remove non-printable characters
	clean := strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, title)

	// Collapse whitespace and trim
	clean = strings.Join(strings.Fields(clean), " ")

	return clean
}
