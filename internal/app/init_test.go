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
	"os"
	"path/filepath"
	"testing"

	"github.com/godynheil/kk/internal/core"
)

func TestInitCreatesKKGitWithoutRootGit(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("APPDATA", temp)
	root := filepath.Join(temp, "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	app := New(root)
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(root, core.KKGitDir)); err != nil {
		t.Fatalf("expected %s to exist: %v", core.KKGitDir, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
		t.Fatal("expected root .git directory to be absent")
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected error checking root .git: %v", err)
	}
}
