package app

import (
	"fmt"
	"path/filepath"
	"strings"
)

func safeRepoPath(root, input string) (string, string, error) {
	if input == "" || filepath.IsAbs(input) {
		return "", "", fmt.Errorf("invalid repository path: %q", input)
	}
	clean := filepath.Clean(filepath.FromSlash(input))
	if clean == "." {
		rootAbs, err := filepath.Abs(root)
		if err != nil {
			return "", "", err
		}
		return ".", rootAbs, nil
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("path escapes repository: %q", input)
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}
	abs := filepath.Join(rootAbs, clean)
	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil {
		return "", "", err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("path escapes repository: %q", input)
	}
	return filepath.ToSlash(rel), abs, nil
}
