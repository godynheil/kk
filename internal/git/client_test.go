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

package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitExecutable(t *testing.T) {
	exe, err := GitExecutable()
	if err != nil {
		t.Fatalf("expected to resolve git executable, got %v", err)
	}
	if exe == "" {
		t.Error("expected non-empty git executable path")
	}
}

func TestGitClientLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	c := New(tempDir)

	// 1. Initially not initialized
	if c.IsInitialized() {
		t.Error("expected repository to be uninitialized initially")
	}

	// EnsureRepository returns error if not initialized
	err := c.EnsureRepository()
	if err == nil {
		t.Error("expected EnsureRepository to return error for uninitialized repo")
	}

	// Create .kk/git folder structure required by git client
	err = os.MkdirAll(filepath.Join(tempDir, ".kk", "git"), 0o750)
	if err != nil {
		t.Fatal(err)
	}

	// 2. Initialize repo
	err = c.InitMain()
	if err != nil {
		t.Fatalf("InitMain failed: %v", err)
	}

	if !c.IsInitialized() {
		t.Error("expected IsInitialized to be true after InitMain")
	}

	err = c.EnsureRepository()
	if err != nil {
		t.Errorf("EnsureRepository failed after init: %v", err)
	}

	// Configure mock git user
	_ = c.Run("config", "user.email", "test@example.com")
	_ = c.Run("config", "user.name", "KK Test")

	// 3. Current branch verification
	branch := c.CurrentBranch()
	if branch != "main" {
		t.Errorf("expected current branch 'main', got %q", branch)
	}

	// 4. Staged files checks (no commit yet, empty HEAD)
	if c.HasHEAD() {
		t.Error("expected HasHEAD() to be false with no commits")
	}

	// Write a file and stage it
	testFile := "document.txt"
	absTestFile := filepath.Join(tempDir, testFile)
	err = os.WriteFile(absTestFile, []byte("git wrapper test content"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Run("add", testFile)
	if err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	staged, err := c.StagedFiles()
	if err != nil {
		t.Fatalf("StagedFiles failed: %v", err)
	}
	if len(staged) != 1 || staged[0] != testFile {
		t.Errorf("expected staged files to contain %q, got %v", testFile, staged)
	}

	// Commit the staged file
	err = c.Run("commit", "-m", "initial commit")
	if err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	if !c.HasHEAD() {
		t.Error("expected HasHEAD() to be true after first commit")
	}

	commit1, err := c.HeadCommit()
	if err != nil {
		t.Fatalf("failed to get head commit: %v", err)
	}
	if len(commit1) == 0 {
		t.Error("expected non-empty commit hash")
	}

	// 5. HeadFiles check
	headFiles, err := c.HeadFiles()
	if err != nil {
		t.Fatalf("HeadFiles failed: %v", err)
	}
	if len(headFiles) != 1 || headFiles[0] != testFile {
		t.Errorf("expected HEAD files to be [%s], got %v", testFile, headFiles)
	}

	// 6. DeletedFiles check
	err = os.Remove(absTestFile)
	if err != nil {
		t.Fatal(err)
	}

	deleted, err := c.DeletedFiles("")
	if err != nil {
		t.Fatalf("DeletedFiles failed: %v", err)
	}
	if len(deleted) != 1 || deleted[0] != testFile {
		t.Errorf("expected deleted files to contain %q, got %v", testFile, deleted)
	}

	// Restore file and commit changes
	err = os.WriteFile(absTestFile, []byte("second revision"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_ = c.Run("add", testFile)
	err = c.Run("commit", "-m", "second commit")
	if err != nil {
		t.Fatalf("second commit failed: %v", err)
	}

	commit2, err := c.HeadCommit()
	if err != nil {
		t.Fatalf("failed to get second head commit: %v", err)
	}

	// 7. IsAncestor check
	if !c.IsAncestor(commit1, commit2) {
		t.Errorf("expected %s to be ancestor of %s", commit1, commit2)
	}
	if c.IsAncestor(commit2, commit1) {
		t.Errorf("expected %s NOT to be ancestor of %s", commit2, commit1)
	}

	// 8. ChangedFiles check
	changed, err := c.ChangedFiles(commit1, commit2)
	if err != nil {
		t.Fatalf("ChangedFiles failed: %v", err)
	}
	if len(changed) != 1 || changed[0] != testFile {
		t.Errorf("expected changed files between commits to be [%s], got %v", testFile, changed)
	}

	// 9. ShowHeadFile check
	showContent, err := c.ShowHeadFile(testFile)
	if err != nil {
		t.Fatalf("ShowHeadFile failed: %v", err)
	}
	if showContent != "second revision" {
		t.Errorf("expected show content %q, got %q", "second revision", showContent)
	}

	// 10. ShowIndexFile check
	indexBytes, err := c.ShowIndexFile(testFile)
	if err != nil {
		t.Fatalf("ShowIndexFile failed: %v", err)
	}
	if string(indexBytes) != "second revision" {
		t.Errorf("expected show index bytes %q, got %q", "second revision", string(indexBytes))
	}

	// 11. HasRemotes check
	if c.HasRemotes() {
		t.Error("expected HasRemotes() to be false initially")
	}

	// Add mock remote
	_ = c.Run("remote", "add", "origin", "https://github.com/mock/repo.git")
	if !c.HasRemotes() {
		t.Error("expected HasRemotes() to be true after adding remote")
	}
}

func TestRunBatchedStagesAllItemsAcrossSmallBatches(t *testing.T) {
	tempDir := t.TempDir()
	c := New(tempDir)

	if err := os.MkdirAll(filepath.Join(tempDir, ".kk", "git"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := c.InitMain(); err != nil {
		t.Fatalf("InitMain failed: %v", err)
	}

	var files []string
	for i := 0; i < 12; i++ {
		name := filepath.ToSlash(filepath.Join("nested", "file-with-a-long-name-"+string(rune('a'+i))+".txt"))
		if err := os.MkdirAll(filepath.Dir(filepath.Join(tempDir, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
		files = append(files, name)
	}

	if err := c.runBatched([]string{"add", "--"}, files, c.commandLineLen([]string{"add", "--"})+40); err != nil {
		t.Fatalf("runBatched failed: %v", err)
	}

	staged, err := c.StagedFiles()
	if err != nil {
		t.Fatalf("StagedFiles failed: %v", err)
	}
	if len(staged) != len(files) {
		t.Fatalf("expected %d staged files, got %d: %v", len(files), len(staged), staged)
	}
}
