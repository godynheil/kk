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
	"github.com/godynheil/kk/internal/git"
)

func TestObjectsSyncWithMultipleRemotes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)
	gc := git.New(dir)

	// Enable tracking for .bin files
	mustTrackBin(t, app, gc)

	// Create and commit a large file
	assetContent := []byte("sync test file content")
	if err := os.WriteFile(filepath.Join(dir, "file.bin"), assetContent, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := app.Add([]string{"file.bin"}); err != nil {
		t.Fatalf("kk add failed: %v", err)
	}
	if err := app.Commit([]string{"-m", "add file.bin"}); err != nil {
		t.Fatalf("kk commit failed: %v", err)
	}

	remoteARoot := filepath.Join(tmp, "remoteA")
	remoteBRoot := filepath.Join(tmp, "remoteB")

	cfg, err := core.ReadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}

	// remoteA is push/pull enabled
	cfg.Remotes["remoteA"] = core.RemoteConfig{
		Type:         "local",
		DisplayName:  "remoteA",
		Provider:     "local",
		Path:         remoteARoot,
		ObjectRoot:   "objects",
		ManifestRoot: "manifests",
		VerifyMode:   "local-hash",
		Pull:         true,
		Push:         true,
	}

	// remoteB is pull-only initially (so push won't target it)
	cfg.Remotes["remoteB"] = core.RemoteConfig{
		Type:         "local",
		DisplayName:  "remoteB",
		Provider:     "local",
		Path:         remoteBRoot,
		ObjectRoot:   "objects",
		ManifestRoot: "manifests",
		VerifyMode:   "local-hash",
		Pull:         true,
		Push:         false,
	}

	cfg.DefaultRemote = "remoteA"
	if err := core.WriteConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}

	// Push to remoteA
	if err := app.Push(nil); err != nil {
		t.Fatalf("expected push to succeed, got: %v", err)
	}

	// Verify object is in remoteA but not in remoteB
	headPointerText, err := gc.ShowHeadFile("file.bin")
	if err != nil {
		t.Fatal(err)
	}
	pointer, ok := core.ParsePointerText(headPointerText)
	if !ok {
		t.Fatalf("expected file.bin in HEAD to be a pointer, got %q", headPointerText)
	}

	remoteAObjectPath := filepath.Join(remoteARoot, "objects", pointer.OID[0:2], pointer.OID[2:4], pointer.OID)
	if _, err := os.Stat(remoteAObjectPath); err != nil {
		t.Fatalf("expected object in remoteA: %v", err)
	}

	remoteBObjectPath := filepath.Join(remoteBRoot, "objects", pointer.OID[0:2], pointer.OID[2:4], pointer.OID)
	if _, err := os.Stat(remoteBObjectPath); err == nil {
		t.Fatal("expected object to NOT be in remoteB yet")
	}

	// Now make remoteB push-enabled, and run sync
	cfg, err = core.ReadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	rB := cfg.Remotes["remoteB"]
	rB.Push = true
	cfg.Remotes["remoteB"] = rB
	if err := core.WriteConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}

	// Run ObjectsSync
	if err := app.ObjectsSync([]string{"--verbose"}); err != nil {
		t.Fatalf("expected ObjectsSync to succeed, got: %v", err)
	}

	// Verify object is now successfully in remoteB
	if _, err := os.Stat(remoteBObjectPath); err != nil {
		t.Fatalf("expected object in remoteB after sync: %v", err)
	}
}

func TestPullWithSyncReplicatesToMissingRemotes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	bareDir := initBareRepo(t, filepath.Join(tmp, "bare.git"))

	// Setup Repo A (pushes to remoteA)
	dirA := filepath.Join(tmp, "local-a")
	appA := initKKTestRepo(t, dirA)
	gcA := git.New(dirA)
	mustGit(t, gcA, "remote", "add", "origin", bareDir)

	mustTrackBin(t, appA, gcA)

	assetContent := []byte("on-demand replication binary data")
	if err := os.WriteFile(filepath.Join(dirA, "file.bin"), assetContent, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := appA.Add([]string{"file.bin"}); err != nil {
		t.Fatalf("kk add failed: %v", err)
	}
	if err := appA.Commit([]string{"-m", "add file.bin"}); err != nil {
		t.Fatalf("kk commit failed: %v", err)
	}
	mustGit(t, gcA, "push", "-u", "origin", testMainBranch)

	remoteARoot := filepath.Join(tmp, "remoteA")
	remoteBRoot := filepath.Join(tmp, "remoteB")

	cfgA, err := core.ReadConfig(dirA)
	if err != nil {
		t.Fatal(err)
	}
	cfgA.Remotes["remoteA"] = core.RemoteConfig{
		Type:         "local",
		DisplayName:  "remoteA",
		Provider:     "local",
		Path:         remoteARoot,
		ObjectRoot:   "objects",
		ManifestRoot: "manifests",
		VerifyMode:   "local-hash",
		Pull:         true,
		Push:         true,
	}
	cfgA.DefaultRemote = "remoteA"
	if err := core.WriteConfig(dirA, cfgA); err != nil {
		t.Fatal(err)
	}

	if err := appA.Push(nil); err != nil {
		t.Fatalf("push from A failed: %v", err)
	}

	headPointerText, err := gcA.ShowHeadFile("file.bin")
	if err != nil {
		t.Fatal(err)
	}
	pointer, ok := core.ParsePointerText(headPointerText)
	if !ok {
		t.Fatalf("expected file.bin in HEAD to be a pointer, got %q", headPointerText)
	}

	// Setup Repo B (pulls from bare, pulls objects from remoteA, pushes/syncs objects to remoteB)
	dirB := filepath.Join(tmp, "local-b")
	appB := initKKTestRepo(t, dirB)
	gcB := git.New(dirB)
	mustGit(t, gcB, "remote", "add", "origin", bareDir)
	mustGit(t, gcB, "fetch", "origin")

	_ = os.Remove(filepath.Join(dirB, ".kkignore"))
	_ = os.Remove(filepath.Join(dirB, ".kk", "tracks.json"))
	mustGit(t, gcB, "checkout", "-B", testMainBranch, "--track", "origin/"+testMainBranch)

	cfgB, err := core.ReadConfig(dirB)
	if err != nil {
		t.Fatal(err)
	}
	// Pull from remoteA, Push/Sync to remoteB
	cfgB.Remotes["remoteA"] = core.RemoteConfig{
		Type:         "local",
		DisplayName:  "remoteA",
		Provider:     "local",
		Path:         remoteARoot,
		ObjectRoot:   "objects",
		ManifestRoot: "manifests",
		VerifyMode:   "local-hash",
		Pull:         true,
		Push:         false,
	}
	cfgB.Remotes["remoteB"] = core.RemoteConfig{
		Type:         "local",
		DisplayName:  "remoteB",
		Provider:     "local",
		Path:         remoteBRoot,
		ObjectRoot:   "objects",
		ManifestRoot: "manifests",
		VerifyMode:   "local-hash",
		Pull:         false,
		Push:         true,
	}
	if err := core.WriteConfig(dirB, cfgB); err != nil {
		t.Fatal(err)
	}

	// Run pull with --sync on Repo B
	if err := appB.Pull([]string{"--sync", "--verbose"}); err != nil {
		t.Fatalf("pull with sync failed: %v", err)
	}

	// Verify object was pulled/materialized locally in B
	materializedPath := filepath.Join(dirB, "file.bin")
	localContent, err := os.ReadFile(materializedPath)
	if err != nil {
		t.Fatalf("failed to read materialized file: %v", err)
	}
	if string(localContent) != string(assetContent) {
		t.Fatalf("unexpected local content: %q", localContent)
	}

	// Verify object was on-the-fly replicated to remoteB
	remoteBObjectPath := filepath.Join(remoteBRoot, "objects", pointer.OID[0:2], pointer.OID[2:4], pointer.OID)
	if _, err := os.Stat(remoteBObjectPath); err != nil {
		t.Fatalf("expected object in remoteB after sync-pull: %v", err)
	}
}
