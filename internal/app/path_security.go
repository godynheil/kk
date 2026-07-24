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
