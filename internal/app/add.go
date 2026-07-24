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

package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/git"
	"github.com/godynheil/kk/internal/ignore"
	"github.com/godynheil/kk/internal/storage"
)

func (a App) Add(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kk add <file-or-dir...>")
	}
	client := git.New(a.Root)
	if err := client.EnsureRepository(); err != nil {
		return err
	}
	tracks, err := core.ReadTracks(a.Root)
	if err != nil {
		return err
	}
	files, err := a.expandFiles(args)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		fmt.Println("nothing to add")
		return nil
	}
	store := storage.New(a.Root)
	var converted []string
	for _, file := range files {
		if _, err := os.Stat(filepath.Join(a.Root, file)); os.IsNotExist(err) {
			continue
		}
		if core.ShouldTrack(file, tracks) {
			_, isCode := core.CodeLanguage(file)
			if isCode {
				fmt.Printf("code %s (storing as regular file)\n", file)
			} else {
				if err := a.convertToPointer(store, file); err != nil {
					return err
				}
				converted = append(converted, file)
			}
		}
	}
	if err := client.RunBatched([]string{"add", "--"}, files); err != nil {
		return err
	}
	for _, file := range converted {
		if err := a.materialize(file, false, false); err != nil {
			fmt.Printf("kk: warning: could not materialize %s after staging: %v\n", file, err)
		}
	}
	return nil
}

func (a App) expandFiles(args []string) ([]string, error) {
	matcher := ignore.Load(a.Root)
	seen := map[string]bool{}
	var out []string
	client := git.New(a.Root)
	for _, input := range args {
		clean, abs, err := safeRepoPath(a.Root, input)
		if err != nil {
			return nil, err
		}
		if matcher.Ignored(clean) || core.IsInsideKK(clean) {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				deleted, dErr := client.DeletedFiles(clean)
				if dErr == nil && len(deleted) > 0 {
					for _, file := range deleted {
						cleanedFile := filepath.Clean(file)
						if !seen[cleanedFile] {
							seen[cleanedFile] = true
							out = append(out, cleanedFile)
						}
					}
					continue
				}
			}
			return nil, err
		}
		if !info.IsDir() {
			if !seen[clean] {
				seen[clean] = true
				out = append(out, clean)
			}
			continue
		}
		err = filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(a.Root, path)
			if err != nil {
				return err
			}
			rel = filepath.Clean(rel)
			if rel == "." {
				return nil
			}
			if d.IsDir() {
				if matcher.Ignored(rel) || core.IsInsideKK(rel) {
					return filepath.SkipDir
				}
				return nil
			}
			if matcher.Ignored(rel) || core.IsInsideKK(rel) {
				return nil
			}
			if !seen[rel] {
				seen[rel] = true
				out = append(out, rel)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		deleted, dErr := client.DeletedFiles(clean)
		if dErr == nil && len(deleted) > 0 {
			for _, file := range deleted {
				cleanedFile := filepath.Clean(file)
				if !seen[cleanedFile] {
					seen[cleanedFile] = true
					out = append(out, cleanedFile)
				}
			}
		}
	}
	return out, nil
}

func (a App) convertToPointer(store storage.Store, rel string) error {
	rel, path, err := safeRepoPath(a.Root, rel)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	isPointer := false
	if info.Size() <= 4096 {
		f, err := os.Open(path) // #nosec G304 -- path is constrained by safeRepoPath before opening.
		if err == nil {
			buf := make([]byte, 1024)
			n, _ := io.ReadFull(f, buf)
			_ = f.Close()
			_, isPointer = core.ParsePointerBytes(buf[:n])
		}
	}
	if isPointer {
		fmt.Println("pointer", rel)
		return nil
	}
	p, err := store.StoreObjectFromFile(path)
	if err != nil {
		return err
	}
	if err := store.VerifyObject(p); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(core.FormatPointer(p)), 0o600); err != nil {
		return err
	}
	fmt.Printf("large %s -> sha256:%s (%d bytes)\n", rel, p.OID, p.Size)
	return nil
}
