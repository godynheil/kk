package ignore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/godynheil/kk/internal/core"
)

func TestLoad(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Loading with no ignore file
	matcher := Load(tempDir)
	if len(matcher.patterns) != 0 {
		t.Fatalf("expected 0 patterns, got %d", len(matcher.patterns))
	}

	// 2. Loading with empty or commented ignore file
	ignoreContent := `
# This is a comment
   
   # Another comment
`
	err := os.WriteFile(filepath.Join(tempDir, core.KKIgnoreFile), []byte(ignoreContent), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	matcher = Load(tempDir)
	if len(matcher.patterns) != 0 {
		t.Fatalf("expected 0 patterns for comments/empty lines, got %d", len(matcher.patterns))
	}

	// 3. Loading with active patterns
	ignoreContent2 := `
*.bin
# a comment in between
temp/
docs/notes.txt
`
	err = os.WriteFile(filepath.Join(tempDir, core.KKIgnoreFile), []byte(ignoreContent2), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	matcher = Load(tempDir)
	expected := []string{"*.bin", "temp/", "docs/notes.txt"}
	if len(matcher.patterns) != len(expected) {
		t.Fatalf("expected %d patterns, got %d", len(expected), len(matcher.patterns))
	}

	for i, pattern := range expected {
		if matcher.patterns[i] != pattern {
			t.Errorf("expected pattern[%d] = %q, got %q", i, pattern, matcher.patterns[i])
		}
	}
}

func TestIgnored(t *testing.T) {
	patterns := []string{
		"*.bin",
		"build/",
		"config/secrets.json",
		"notes.txt",
	}

	m := Matcher{patterns: patterns}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Ignore inside .kk directory",
			path:     ".kk/something",
			expected: true,
		},
		{
			name:     "Ignore nested inside .kk directory",
			path:     "subdir/.kk/something",
			expected: false,
		},
		{
			name:     "Wildcard match suffix",
			path:     "data/file.bin",
			expected: true,
		},
		{
			name:     "Wildcard match base only",
			path:     "file.bin",
			expected: true,
		},
		{
			name:     "No wildcard match other suffix",
			path:     "data/file.bin.txt",
			expected: false,
		},
		{
			name:     "Directory exact match",
			path:     "build",
			expected: true,
		},
		{
			name:     "Directory prefix match",
			path:     "build/debug/exe",
			expected: true,
		},
		{
			name:     "Directory prefix mismatch on start",
			path:     "src/build/exe",
			expected: false,
		},
		{
			name:     "Exact file path match",
			path:     "config/secrets.json",
			expected: true,
		},
		{
			name:     "Exact file path mismatch on folder",
			path:     "other/config/secrets.json",
			expected: false,
		},
		{
			name:     "Base name match",
			path:     "docs/notes.txt",
			expected: true,
		},
		{
			name:     "Base name match root",
			path:     "notes.txt",
			expected: true,
		},
		{
			name:     "Normal non-ignored file",
			path:     "src/main.go",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.Ignored(tt.path)
			if result != tt.expected {
				t.Errorf("Ignored(%q) = %v; expected %v", tt.path, result, tt.expected)
			}
		})
	}
}
