package remote

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/ignore"
)

var allowedKKFiles = map[string]bool{
	".kk/repo.json":   true,
	".kk/config.json": true,
	".kk/tracks.json": true,
}

var kkInternalDirs = map[string]bool{
	".kk/objects": true,
	".kk/tmp":     true,
	".kk/logs":    true,
	".kk/git":     true,
}

func WalkProjectFiles(root string) ([]string, error) {
	matcher := ignore.Load(root)
	var files []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkerr error) error {
		if walkerr != nil {
			return walkerr
		}
		relOrig, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel := filepath.ToSlash(relOrig)
		if rel == "." {
			return nil
		}

		if core.IsInsideKK(rel) {
			if d.IsDir() {
				if kkInternalDirs[rel] {
					return filepath.SkipDir
				}
				return nil
			}
			if allowedKKFiles[rel] {
				files = append(files, rel)
			}
			return nil
		}

		if matcher.Ignored(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !d.IsDir() {
			files = append(files, rel)
		}
		return nil
	})
	return files, err
}

func CommittedProjectFiles(root string, headFiles []string) []string {
	var files []string
	for _, f := range headFiles {
		files = append(files, filepath.ToSlash(f))
	}
	for _, meta := range []string{".kk/repo.json", ".kk/config.json", ".kk/tracks.json"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(meta))); err == nil {
			files = append(files, meta)
		}
	}
	return files
}

func ExistingProjectFiles(root string, files []string) []string {
	var out []string
	for _, f := range files {
		rel := filepath.ToSlash(f)
		if core.IsInsideKK(rel) && !allowedKKFiles[rel] {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err == nil {
			out = append(out, rel)
		}
	}
	return out
}
