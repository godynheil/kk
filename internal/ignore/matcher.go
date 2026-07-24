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

package ignore

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/godynheil/kk/internal/core"
)

type Matcher struct {
	patterns []string
}

func Load(root string) Matcher {
	path := filepath.Join(root, core.KKIgnoreFile)
	f, err := os.Open(path) // #nosec G304 -- ignore file is read from the caller's repository root.
	if err != nil {
		return Matcher{}
	}
	defer func() {
		_ = f.Close()
	}()
	m := Matcher{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m.patterns = append(m.patterns, filepath.ToSlash(line))
	}
	return m
}

func (m Matcher) Ignored(path string) bool {
	path = filepath.ToSlash(filepath.Clean(path))
	if core.IsInsideKK(path) {
		return true
	}
	base := filepath.Base(path)
	for _, pattern := range m.patterns {
		if strings.HasSuffix(pattern, "/") {
			prefix := strings.TrimSuffix(pattern, "/")
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				return true
			}
			continue
		}
		if pattern == path || pattern == base {
			return true
		}
		if ok, _ := filepath.Match(pattern, path); ok {
			return true
		}
		if ok, _ := filepath.Match(pattern, base); ok {
			return true
		}
	}
	return false
}
