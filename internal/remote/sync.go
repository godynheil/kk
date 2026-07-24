// Copyright (C) 2026 Godynheil A. Quisto <godynheil@quisto.ph>
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

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
