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
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/gdrive"
	"github.com/godynheil/kk/internal/git"
)

func TestExplainDriveFolderResolveErrorRestrictedScope(t *testing.T) {
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := gdrive.SaveAuth(authPath, gdrive.Auth{
		ClientID:     "client",
		RefreshToken: "refresh",
		Scope:        "https://www.googleapis.com/auth/drive.file",
	}); err != nil {
		t.Fatalf("SaveAuth: %v", err)
	}

	err := explainDriveFolderResolveError(&gdrive.APIError{
		Op:         "get Drive folder",
		Status:     "404 Not Found",
		StatusCode: http.StatusNotFound,
	}, authPath)

	msg := err.Error()
	if !strings.Contains(msg, "restricted drive.file scope") {
		t.Fatalf("expected restricted-scope hint, got:\n%s", msg)
	}
	if !strings.Contains(msg, "kk setup gdrive --auth-only --scope full") {
		t.Fatalf("expected full-scope setup command, got:\n%s", msg)
	}
}

func TestExplainDriveFolderResolveErrorFullScope(t *testing.T) {
	authPath := filepath.Join(t.TempDir(), "auth.json")
	if err := gdrive.SaveAuth(authPath, gdrive.Auth{
		ClientID:     "client",
		RefreshToken: "refresh",
		Scope:        "https://www.googleapis.com/auth/drive",
	}); err != nil {
		t.Fatalf("SaveAuth: %v", err)
	}

	err := explainDriveFolderResolveError(&gdrive.APIError{
		Op:         "get Drive folder",
		Status:     "404 Not Found",
		StatusCode: http.StatusNotFound,
	}, authPath)

	msg := err.Error()
	if strings.Contains(msg, "restricted drive.file scope") {
		t.Fatalf("did not expect restricted-scope hint for full scope, got:\n%s", msg)
	}
	if !strings.Contains(msg, "share is granted to this Google account") {
		t.Fatalf("expected account access hint, got:\n%s", msg)
	}
}

func TestStorageCloneCanPushBackToOriginOnBranch(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	sourceDir := filepath.Join(tmp, "source")
	sourceApp := initKKTestRepo(t, sourceDir)
	sourceGit := git.New(sourceDir)
	writeAndCommit(t, sourceGit, sourceDir, "main.txt", "main", "initial commit")

	remoteRoot := filepath.Join(tmp, "remote", filepath.Base(sourceDir))
	cfg, err := core.ReadConfig(sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DefaultRemote = "origin"
	cfg.Remotes["origin"] = core.RemoteConfig{
		Type:         "local",
		DisplayName:  "origin",
		Provider:     "local",
		Path:         remoteRoot,
		ObjectRoot:   "objects",
		ManifestRoot: "manifests",
		VerifyMode:   "local-hash",
		Pull:         true,
		Push:         true,
	}
	if err := core.WriteConfig(sourceDir, cfg); err != nil {
		t.Fatal(err)
	}
	if err := sourceApp.Push(nil); err != nil {
		t.Fatalf("source push: %v", err)
	}

	cloneDir := filepath.Join(tmp, "clone")
	cloneApp := New(tmp)
	if err := cloneApp.Clone([]string{"local:" + remoteRoot, cloneDir, "--history"}); err != nil {
		t.Fatalf("clone: %v", err)
	}

	clonedCfg, err := core.ReadConfig(cloneDir)
	if err != nil {
		t.Fatal(err)
	}
	origin, ok := clonedCfg.Remotes["origin"]
	if !ok {
		t.Fatal("expected cloned config to contain origin remote")
	}
	if !origin.Push {
		t.Fatal("expected cloned storage origin to be push-enabled")
	}
	origin.Push = false
	clonedCfg.Remotes["origin"] = origin
	if err := core.WriteConfig(cloneDir, clonedCfg); err != nil {
		t.Fatal(err)
	}

	cloneGit := git.New(cloneDir)
	mustGit(t, cloneGit, "config", "user.email", "clone@example.com")
	mustGit(t, cloneGit, "config", "user.name", "KK Clone Test")
	mustGit(t, cloneGit, "checkout", "-b", "feature")
	writeAndCommit(t, cloneGit, cloneDir, "feature.txt", "feature", "feature commit")

	if err := New(cloneDir).Push(nil); err != nil {
		t.Fatalf("clone branch push: %v", err)
	}
	repairedCfg, err := core.ReadConfig(cloneDir)
	if err != nil {
		t.Fatal(err)
	}
	if !repairedCfg.Remotes["origin"].Push {
		t.Fatal("expected push to repair legacy cloned origin push=false config")
	}
	if _, err := os.Stat(filepath.Join(remoteRoot, "feature.txt")); err != nil {
		t.Fatalf("expected cloned feature file pushed to origin storage: %v", err)
	}
	if _, err := os.Stat(filepath.Join(remoteRoot, "history", "feature", "full.bundle")); err != nil {
		t.Fatalf("expected cloned feature branch history bundle pushed to origin storage: %v", err)
	}
}

func TestGitCloneCanSelectBranch(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	bareDir := initBareRepo(t, filepath.Join(tmp, "bare.git"))

	sourceDir := filepath.Join(tmp, "source")
	_ = initKKTestRepo(t, sourceDir)
	sourceGit := git.New(sourceDir)
	writeAndCommit(t, sourceGit, sourceDir, "main.txt", "main", "initial commit")
	mustGit(t, sourceGit, "remote", "add", "origin", bareDir)
	mustGit(t, sourceGit, "push", "-u", "origin", testMainBranch)
	mustGit(t, sourceGit, "checkout", "-b", "kk-test-branch")
	writeAndCommit(t, sourceGit, sourceDir, "branch.txt", "branch", "branch commit")
	branchHead, err := sourceGit.HeadCommit()
	if err != nil {
		t.Fatal(err)
	}
	mustGit(t, sourceGit, "push", "-u", "origin", "kk-test-branch")

	cloneDir := filepath.Join(tmp, "clone")
	bareURLPath := filepath.ToSlash(bareDir)
	bareURL := "file://" + bareURLPath
	if !strings.HasPrefix(bareURLPath, "/") {
		bareURL = "file:///" + bareURLPath
	}
	if err := New(tmp).Clone([]string{"git:" + bareURL, cloneDir, "--branch", "kk-test-branch"}); err != nil {
		t.Fatalf("clone: %v", err)
	}

	cloneGit := git.New(cloneDir)
	if branch := cloneGit.CurrentBranch(); branch != "kk-test-branch" {
		t.Fatalf("branch = %q, want kk-test-branch", branch)
	}
	head, err := cloneGit.HeadCommit()
	if err != nil {
		t.Fatal(err)
	}
	if head != branchHead {
		t.Fatalf("HEAD = %s, want %s", head, branchHead)
	}
	if _, err := os.Stat(filepath.Join(cloneDir, "branch.txt")); err != nil {
		t.Fatalf("expected selected branch file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cloneDir, "main.txt")); err != nil {
		t.Fatalf("expected base branch file: %v", err)
	}
}

func TestGitCloneFailureBehavior(t *testing.T) {
	tmp := t.TempDir()

	dest1 := filepath.Join(tmp, "dest1")
	app1 := New(dest1)
	err := app1.cloneFromGit("git://invalid-url-that-fails.git", dest1, "origin", "", "", false, false, false, false, 1, core.RemoteConfig{})
	if err == nil {
		t.Fatal("expected failure on invalid git URL clone")
	}
	if _, statErr := os.Stat(dest1); !os.IsNotExist(statErr) {
		t.Fatalf("expected dest1 to be cleaned up, but it exists: %v", statErr)
	}

	dest2 := filepath.Join(tmp, "dest2")
	if err := os.MkdirAll(dest2, 0o750); err != nil {
		t.Fatal(err)
	}
	app2 := New(dest2)
	err = app2.cloneFromGit("git://invalid-url-that-fails.git", dest2, "origin", "", "", false, false, true, true, 1, core.RemoteConfig{})
	if err == nil {
		t.Fatal("expected failure on invalid git URL clone")
	}
	if _, statErr := os.Stat(dest2); os.IsNotExist(statErr) {
		t.Fatal("expected dest2 NOT to be cleaned up when here=true, but it does not exist")
	}

	dest3 := filepath.Join(tmp, "dest3")
	if err := os.MkdirAll(dest3, 0o750); err != nil {
		t.Fatal(err)
	}
	app3 := New(dest3)
	err = app3.cloneFromGit("git://invalid-url-that-fails.git", dest3, "origin", "", "", false, false, false, true, 1, core.RemoteConfig{})
	if err == nil {
		t.Fatal("expected failure on invalid git URL clone")
	}
	if _, statErr := os.Stat(dest3); os.IsNotExist(statErr) {
		t.Fatal("expected dest3 NOT to be cleaned up when destExisted=true, but it does not exist")
	}
}
