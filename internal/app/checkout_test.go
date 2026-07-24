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

	"github.com/godynheil/kk/internal/git"
)

func TestCheckoutBranchSwitchWithModifiedPointer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)
	gc := git.New(dir)

	if err := app.Track([]string{"*.bin"}); err != nil {
		t.Fatalf("track failed: %v", err)
	}

	mustGit(t, gc, "add", "-f", ".kkignore", ".kk/tracks.json")
	mustGit(t, gc, "commit", "-m", "setup tracking")

	if err := os.WriteFile(filepath.Join(dir, "asset.bin"), []byte("version 1 of binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := app.Add([]string{"asset.bin"}); err != nil {
		t.Fatalf("kk add failed: %v", err)
	}
	if err := app.Commit([]string{"-m", "add asset.bin v1"}); err != nil {
		t.Fatalf("kk commit failed: %v", err)
	}

	if err := app.Git([]string{"checkout", "-b", "feature"}); err != nil {
		t.Fatalf("create branch feature failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "asset.bin"), []byte("version 2 of binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := app.Add([]string{"asset.bin"}); err != nil {
		t.Fatalf("kk add failed: %v", err)
	}
	if err := app.Commit([]string{"-m", "add asset.bin v2"}); err != nil {
		t.Fatalf("kk commit failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "asset.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "version 2 of binary" {
		t.Fatalf("expected version 2 of binary on feature branch, got %q", string(data))
	}

	if err := app.Git([]string{"checkout", testMainBranch}); err != nil {
		t.Fatalf("switching back to %s failed: %v", testMainBranch, err)
	}

	data, err = os.ReadFile(filepath.Join(dir, "asset.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "version 1 of binary" {
		t.Fatalf("expected version 1 of binary on %s branch, got %q", testMainBranch, string(data))
	}

	if err := app.Git([]string{"checkout", "feature"}); err != nil {
		t.Fatalf("switching back to feature failed: %v", err)
	}

	data, err = os.ReadFile(filepath.Join(dir, "asset.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "version 2 of binary" {
		t.Fatalf("expected version 2 of binary on feature branch again, got %q", string(data))
	}
}

func TestCheckoutBranchSwitchWithDeletedFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)
	gc := git.New(dir)

	if err := app.Track([]string{"*.bin"}); err != nil {
		t.Fatalf("track failed: %v", err)
	}

	mustGit(t, gc, "add", "-f", ".kkignore", ".kk/tracks.json")
	mustGit(t, gc, "commit", "-m", "setup tracking")

	if err := os.WriteFile(filepath.Join(dir, "asset.bin"), []byte("version 1 of binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := app.Add([]string{"asset.bin"}); err != nil {
		t.Fatalf("kk add failed: %v", err)
	}
	if err := app.Commit([]string{"-m", "add asset.bin v1"}); err != nil {
		t.Fatalf("kk commit failed: %v", err)
	}

	if err := app.Git([]string{"checkout", "-b", "feature"}); err != nil {
		t.Fatalf("create branch feature failed: %v", err)
	}

	if err := os.Remove(filepath.Join(dir, "asset.bin")); err != nil {
		t.Fatal(err)
	}
	if err := app.Add([]string{"asset.bin"}); err != nil {
		t.Fatalf("kk add failed: %v", err)
	}
	if err := app.Commit([]string{"-m", "delete asset.bin"}); err != nil {
		t.Fatalf("kk commit failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "asset.bin")); err == nil || !os.IsNotExist(err) {
		t.Fatalf("expected asset.bin to be deleted on feature branch")
	}

	if err := app.Git([]string{"checkout", testMainBranch}); err != nil {
		t.Fatalf("switching back to %s failed: %v", testMainBranch, err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "asset.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "version 1 of binary" {
		t.Fatalf("expected version 1 of binary on %s branch, got %q", testMainBranch, string(data))
	}

	if err := app.Git([]string{"checkout", "feature"}); err != nil {
		t.Fatalf("switching back to feature failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "asset.bin")); err == nil || !os.IsNotExist(err) {
		t.Fatalf("expected asset.bin to be deleted on feature branch again")
	}
}

func TestCheckoutBranchSwitchRollbackOnDirtyFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)
	gc := git.New(dir)

	if err := app.Track([]string{"*.bin"}); err != nil {
		t.Fatalf("track failed: %v", err)
	}

	mustGit(t, gc, "add", "-f", ".kkignore", ".kk/tracks.json")
	mustGit(t, gc, "commit", "-m", "setup tracking")

	if err := os.WriteFile(filepath.Join(dir, "asset.bin"), []byte("version 1 of binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := app.Add([]string{"asset.bin"}); err != nil {
		t.Fatalf("kk add failed: %v", err)
	}
	if err := app.Commit([]string{"-m", "add asset.bin v1"}); err != nil {
		t.Fatalf("kk commit failed: %v", err)
	}

	if err := app.Git([]string{"checkout", "-b", "feature"}); err != nil {
		t.Fatalf("create branch feature failed: %v", err)
	}
	if err := app.Git([]string{"checkout", testMainBranch}); err != nil {
		t.Fatalf("switch back to %s failed: %v", testMainBranch, err)
	}

	if err := app.Git([]string{"checkout", "feature"}); err != nil {
		t.Fatalf("switch to feature failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "asset.bin"), []byte("version 2 on feature"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := app.Add([]string{"asset.bin"}); err != nil {
		t.Fatalf("kk add failed: %v", err)
	}
	if err := app.Commit([]string{"-m", "add asset.bin v2 on feature"}); err != nil {
		t.Fatalf("kk commit failed: %v", err)
	}
	if err := app.Git([]string{"checkout", testMainBranch}); err != nil {
		t.Fatalf("switch to %s failed: %v", testMainBranch, err)
	}

	if err := os.WriteFile(filepath.Join(dir, "asset.bin"), []byte("dirty user modifications"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := app.Git([]string{"checkout", "feature"})
	if err == nil {
		t.Fatal("expected error switching branches with dirty file, got nil")
	}

	data, err := os.ReadFile(filepath.Join(dir, "asset.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "dirty user modifications" {
		t.Fatalf("expected dirty user modifications to be preserved, got %q", string(data))
	}
}
