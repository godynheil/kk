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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/registry"
)

func TestConnectProjectRegistersExistingRepo(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("APPDATA", temp)
	root := filepath.Join(temp, "repo")
	if err := os.MkdirAll(filepath.Join(root, core.KKDir), 0o755); err != nil {
		t.Fatal(err)
	}
	info := core.RepoInfo{RepoID: "repo-1", Name: "MyGame"}
	if err := core.WriteRepoInfo(root, info); err != nil {
		t.Fatal(err)
	}

	result, err := ConnectProject(root)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "connected" {
		t.Fatalf("expected connected action, got %q", result.Action)
	}

	data, err := os.ReadFile(registry.DefaultPath())
	if err != nil {
		t.Fatal(err)
	}
	var reg registry.Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		t.Fatal(err)
	}
	if len(reg.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(reg.Projects))
	}
	if reg.Projects[0].RepoID != "repo-1" {
		t.Fatalf("unexpected repo id: %q", reg.Projects[0].RepoID)
	}
}

func TestConnectProjectReimportsMovedRepoByRepoID(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("APPDATA", temp)
	oldRoot := filepath.Join(temp, "old-repo")
	newRoot := filepath.Join(temp, "new-repo")
	if err := os.MkdirAll(filepath.Join(newRoot, core.KKDir), 0o755); err != nil {
		t.Fatal(err)
	}
	info := core.RepoInfo{RepoID: "repo-1", Name: "MyGame"}
	if err := core.WriteRepoInfo(newRoot, info); err != nil {
		t.Fatal(err)
	}
	reg := registry.Registry{
		Projects: []registry.Project{{
			RepoID: "repo-1",
			Name:   "MyGame",
			Path:   oldRoot,
		}},
	}
	if err := registry.Save("", reg); err != nil {
		t.Fatal(err)
	}

	result, err := ConnectProject(newRoot)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "reimported" {
		t.Fatalf("expected reimported action, got %q", result.Action)
	}

	loaded, err := registry.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(loaded.Projects))
	}
	if !samePath(loaded.Projects[0].Path, newRoot) {
		t.Fatalf("expected updated path %q, got %q", newRoot, loaded.Projects[0].Path)
	}
}
