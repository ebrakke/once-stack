package notes

import (
	"testing"
)

func TestSlugifyTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"  Hello   World  ", "hello-world"},
		{"Hello-World!", "hello-world"},
		{"Special Characters: @#$%", "special-characters"},
		{"Already-slugged", "already-slugged"},
		{"", "untitled"},
		{"   ", "untitled"},
		{"---", "untitled"},
		{"Café", "caf"},
		{"A", "a"},
		{"UPPERCASE", "uppercase"},
		{"Leading-Trailing---", "leading-trailing"},
		{"Multiple   Spaces   Between", "multiple-spaces-between"},
		{"dots.and.things", "dotsandthings"},
		{"hyphen-at-end-", "hyphen-at-end"},
		{"-hyphen-at-start", "hyphen-at-start"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SlugifyTitle(tt.input)
			if got != tt.want {
				t.Errorf("SlugifyTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateID(t *testing.T) {
	tests := []struct {
		id      string
		wantErr bool
	}{
		{"hello-world", false},
		{"a", false},
		{"abc123", false},
		{"my-note-42", false},
		{"z", false},
		{"0", false},
		{"a-b-c", false},
		{"", true},
		{"Hello", true},
		{"has space", true},
		{"contains/slash", true},
		{"contains\\backslash", true},
		{"contains..dots", true},
		{"UPPERCASE", true},
		{"trailing-hyphen-", true},
		{"-leading-hyphen", true},
		{"double--hyphen", false},
		{"special@char", true},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			err := ValidateID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateID(%q) error = %v, wantErr = %v", tt.id, err, tt.wantErr)
			}
		})
	}
}

func TestCleanTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "Hello World"},
		{"  Hello   World  ", "Hello World"},
		{"", ""},
		{"   ", ""},
		{"Single", "Single"},
		{"Leading and trailing   ", "Leading and trailing"},
		{"\tTab\tand\tnewline\n", "Tabandnewline"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := CleanTitle(tt.input)
			if got != tt.want {
				t.Errorf("CleanTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
