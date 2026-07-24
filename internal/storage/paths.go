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

package storage

import (
	"path/filepath"

	"github.com/godynheil/kk/internal/core"
)

type Store struct {
	Root string
}

func New(root string) Store {
	if root == "" {
		root = "."
	}
	return Store{Root: root}
}

func (s Store) ObjectPath(oid string) string {
	if len(oid) < 4 {
		return filepath.Join(s.Root, core.ObjectDir, "invalid", oid)
	}
	return filepath.Join(s.Root, core.ObjectDir, oid[:2], oid[2:4], oid)
}

func (s Store) TempPath(name string) string {
	return filepath.Join(s.Root, core.TmpDir, name)
}
