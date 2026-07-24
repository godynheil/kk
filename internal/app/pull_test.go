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
	"strings"
	"testing"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/git"
	"github.com/godynheil/kk/internal/storage"
)

func TestPullSurfacesMergeConflictHint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	bareDir := initBareRepo(t, filepath.Join(tmp, "bare.git"))

	dirA := filepath.Join(tmp, "local-a")
	appA := initKKTestRepo(t, dirA)
	_ = appA
	gcA := git.New(dirA)
	mustGit(t, gcA, "remote", "add", "origin", bareDir)
	writeAndCommit(t, gcA, dirA, "asset.txt", "initial", "initial commit")
	mustGit(t, gcA, "push", "-u", "origin", testMainBranch)

	dirB := filepath.Join(tmp, "local-b")
	appB := initKKTestRepo(t, dirB)
	gcB := git.New(dirB)
	mustGit(t, gcB, "remote", "add", "origin", bareDir)
	mustGit(t, gcB, "fetch", "origin")
	mustGit(t, gcB, "checkout", "-B", testMainBranch, "--track", "origin/"+testMainBranch)

	writeAndCommit(t, gcA, dirA, "asset.txt", "version from A", "A changes asset")
	mustGit(t, gcA, "push", "origin", testMainBranch)

	writeAndCommit(t, gcB, dirB, "asset.txt", "version from B", "B changes asset")

	pullErr := appB.Pull([]string{"--no-rebase"})
	if pullErr == nil {
		t.Fatal("expected error from conflicting pull, got nil")
	}
	if !strings.Contains(pullErr.Error(), "merge conflict") {
		t.Fatalf("expected merge-conflict hint in error, got: %v", pullErr)
	}
	if !strings.Contains(pullErr.Error(), "exit status") {
		t.Fatalf("expected wrapped exit-status error, got: %v", pullErr)
	}
}

func TestPullSucceedsOnFastForward(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	bareDir := initBareRepo(t, filepath.Join(tmp, "bare.git"))

	dirA := filepath.Join(tmp, "local-a")
	_ = initKKTestRepo(t, dirA)
	gcA := git.New(dirA)
	mustGit(t, gcA, "remote", "add", "origin", bareDir)
	writeAndCommit(t, gcA, dirA, "note.txt", "hello", "initial commit")
	mustGit(t, gcA, "push", "-u", "origin", testMainBranch)

	dirB := filepath.Join(tmp, "local-b")
	appB := initKKTestRepo(t, dirB)
	gcB := git.New(dirB)
	mustGit(t, gcB, "remote", "add", "origin", bareDir)
	mustGit(t, gcB, "fetch", "origin")
	mustGit(t, gcB, "checkout", "-B", testMainBranch, "--track", "origin/"+testMainBranch)

	writeAndCommit(t, gcA, dirA, "note.txt", "world", "second commit")
	mustGit(t, gcA, "push", "origin", testMainBranch)

	if err := appB.Pull(nil); err != nil {
		t.Fatalf("expected clean pull to succeed, got: %v", err)
	}
}

func TestPullTemporarilyDematerializesCleanLargeFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	bareDir := initBareRepo(t, filepath.Join(tmp, "bare.git"))

	dirA := filepath.Join(tmp, "local-a")
	appA := initKKTestRepo(t, dirA)
	gcA := git.New(dirA)
	mustGit(t, gcA, "remote", "add", "origin", bareDir)
	mustTrackBin(t, appA, gcA)

	assetContent := []byte("binary version 1")
	if err := os.WriteFile(filepath.Join(dirA, "asset.bin"), assetContent, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := appA.Add([]string{"asset.bin"}); err != nil {
		t.Fatalf("kk add failed: %v", err)
	}
	if err := appA.Commit([]string{"-m", "add asset"}); err != nil {
		t.Fatalf("kk commit failed: %v", err)
	}
	mustGit(t, gcA, "push", "-u", "origin", testMainBranch)

	pointer, ok, err := pointerFromWorkingFile(dirA, "asset.bin")
	if err != nil || ok {
		t.Fatalf("expected local-a asset.bin to remain materialized, ok=%v err=%v", ok, err)
	}
	headPointerText, err := gcA.ShowHeadFile("asset.bin")
	if err != nil {
		t.Fatal(err)
	}
	pointer, ok = core.ParsePointerText(headPointerText)
	if !ok {
		t.Fatalf("expected asset.bin in HEAD to be a pointer, got %q", headPointerText)
	}

	dirB := filepath.Join(tmp, "local-b")
	appB := initKKTestRepo(t, dirB)
	gcB := git.New(dirB)
	mustGit(t, gcB, "remote", "add", "origin", bareDir)
	mustGit(t, gcB, "fetch", "origin")
	_ = os.Remove(filepath.Join(dirB, ".kkignore"))
	_ = os.Remove(filepath.Join(dirB, ".kk", "tracks.json"))
	mustGit(t, gcB, "checkout", "-B", testMainBranch, "--track", "origin/"+testMainBranch)

	copyCachedObject(t, dirA, dirB, pointer.OID)
	if err := os.WriteFile(filepath.Join(dirB, "asset.bin"), assetContent, 0o644); err != nil {
		t.Fatal(err)
	}
	raw, _, err := gcB.Combined("status", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, "asset.bin") {
		t.Fatalf("expected raw git status to see materialized asset.bin as modified, got %q", raw)
	}

	writeAndCommit(t, gcA, dirA, "note.txt", "world", "second commit")
	mustGit(t, gcA, "push", "origin", testMainBranch)

	if err := appB.Pull(nil); err != nil {
		t.Fatalf("expected pull with clean materialized asset to succeed, got: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(dirB, "asset.bin")); err != nil || string(got) != string(assetContent) {
		t.Fatalf("expected asset.bin to be rematerialized after pull, got %q err=%v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dirB, "note.txt")); err != nil {
		t.Fatalf("expected note.txt after pull: %v", err)
	}
}

func mustTrackBin(t *testing.T, app App, gc git.Client) {
	t.Helper()
	if err := app.Track([]string{"*.bin"}); err != nil {
		t.Fatalf("track failed: %v", err)
	}
	mustGit(t, gc, "add", "-f", ".kkignore", ".kk/tracks.json")
	mustGit(t, gc, "commit", "-m", "setup tracking")
}

func copyCachedObject(t *testing.T, srcRoot, dstRoot, oid string) {
	t.Helper()
	srcStore := storage.New(srcRoot)
	dstStore := storage.New(dstRoot)
	srcPath := srcStore.ObjectPath(oid)
	dstPath := dstStore.ObjectPath(oid)
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dstPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
