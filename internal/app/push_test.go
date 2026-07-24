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
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/git"
)

func TestParsePushArgsParsesEqualsRemote(t *testing.T) {
	remoteNames, all, syncWorkingDir, workers, gitArgs := ParsePushArgs([]string{"--remote=backup", "origin", testMainBranch})
	if all {
		t.Fatal("expected all-remotes to be false")
	}
	if syncWorkingDir {
		t.Fatal("expected sync-working-dir to be false")
	}
	if workers != 0 {
		t.Fatalf("expected workers to be 0 (unset), got %d", workers)
	}
	if !reflect.DeepEqual(remoteNames, []string{"backup"}) {
		t.Fatalf("unexpected remote names: %#v", remoteNames)
	}
	if !reflect.DeepEqual(gitArgs, []string{"origin", testMainBranch}) {
		t.Fatalf("unexpected git args: %#v", gitArgs)
	}
}

func TestParsePushArgsSyncWorkingDir(t *testing.T) {
	remoteNames, all, syncWorkingDir, workers, gitArgs := ParsePushArgs([]string{"--sync-working-dir", "origin"})
	if all {
		t.Fatal("expected all-remotes to be false")
	}
	if !syncWorkingDir {
		t.Fatal("expected sync-working-dir to be true")
	}
	if workers != 0 {
		t.Fatalf("expected workers to be 0 (unset), got %d", workers)
	}
	if len(remoteNames) != 0 {
		t.Fatalf("unexpected remote names: %#v", remoteNames)
	}
	if !reflect.DeepEqual(gitArgs, []string{"origin"}) {
		t.Fatalf("unexpected git args: %#v", gitArgs)
	}
}

func TestParsePushArgsWorkers(t *testing.T) {
	_, _, _, workers, _ := ParsePushArgs([]string{"--workers", "8", "origin"})
	if workers != 8 {
		t.Fatalf("expected workers=8, got %d", workers)
	}
	_, _, _, workers, _ = ParsePushArgs([]string{"--workers=16"})
	if workers != 16 {
		t.Fatalf("expected workers=16, got %d", workers)
	}
}

func TestPushRejectedHintsSuggestsPull(t *testing.T) {
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
	writeAndCommit(t, gcA, dirA, "file.txt", "initial", "initial commit")
	mustGit(t, gcA, "push", "-u", "origin", testMainBranch)

	dirB := filepath.Join(tmp, "local-b")
	_ = initKKTestRepo(t, dirB)
	gcB := git.New(dirB)
	mustGit(t, gcB, "remote", "add", "origin", bareDir)
	mustGit(t, gcB, "fetch", "origin")
	mustGit(t, gcB, "checkout", "-B", testMainBranch, "--track", "origin/"+testMainBranch)
	writeAndCommit(t, gcB, dirB, "file.txt", "by B", "B changes file")
	mustGit(t, gcB, "push", "origin", testMainBranch)

	writeAndCommit(t, gcA, dirA, "file.txt", "by A", "A changes file")

	pushErr := appA.Push([]string{"origin", testMainBranch})
	if pushErr == nil {
		t.Fatal("expected rejected push error, got nil")
	}
	if !strings.Contains(pushErr.Error(), "pull") {
		t.Fatalf("expected 'pull' hint in push error, got: %v", pushErr)
	}
}

func TestPushWithoutAnyDestinationUsesKKError(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)
	gc := git.New(dir)
	writeAndCommit(t, gc, dir, "file.txt", "content", "initial commit")

	err := app.Push(nil)
	if err == nil {
		t.Fatal("expected missing destination error")
	}
	if strings.Contains(err.Error(), "Either specify the URL") ||
		strings.Contains(err.Error(), "git remote add") {
		t.Fatalf("expected kk-level error, got raw git guidance: %v", err)
	}
	if !strings.Contains(err.Error(), "no push destination configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPushGitRemoteSetsUpstreamWhenMissing(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	bareDir := initBareRepo(t, filepath.Join(tmp, "bare.git"))

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)
	gc := git.New(dir)
	mustGit(t, gc, "remote", "add", "origin", bareDir)
	writeAndCommit(t, gc, dir, "file.txt", "content", "initial commit")
	head, err := gc.HeadCommit()
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := core.ReadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Remotes["origin"] = core.RemoteConfig{
		Type:        "git",
		DisplayName: "origin",
		Provider:    "git",
		URL:         bareDir,
		Pull:        true,
		Push:        true,
		Tags:        []string{"git"},
	}
	if err := core.WriteConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}

	if err := app.Push(nil); err != nil {
		t.Fatalf("push with git remote: %v", err)
	}
	remoteOut, err := gc.Output("ls-remote", "origin", "refs/heads/"+testMainBranch)
	if err != nil {
		t.Fatal(err)
	}
	remoteFields := strings.Fields(remoteOut)
	if len(remoteFields) == 0 || remoteFields[0] != head {
		t.Fatalf("remote %s = %q, want %s", testMainBranch, strings.TrimSpace(remoteOut), head)
	}
	upstream, err := gc.Output("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(upstream) != "origin/"+testMainBranch {
		t.Fatalf("upstream = %q, want origin/%s", strings.TrimSpace(upstream), testMainBranch)
	}
}

func TestPushSkipsGitPushWhenOnlyKKRemoteIsConfigured(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)
	gc := git.New(dir)
	writeAndCommit(t, gc, dir, "file.txt", "content", "initial commit")

	cfg, err := core.ReadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DefaultRemote = "backup"
	cfg.Remotes["backup"] = core.RemoteConfig{
		Type:         "local",
		DisplayName:  "backup",
		Provider:     "local",
		Path:         filepath.Join(tmp, "remote"),
		ObjectRoot:   "objects",
		ManifestRoot: "manifests",
		VerifyMode:   "local-hash",
		Pull:         true,
		Push:         true,
	}
	if err := core.WriteConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}

	if err := app.Push(nil); err != nil {
		t.Fatalf("expected kk-only push to succeed without a kk history remote, got: %v", err)
	}
}

func TestPushOnBranchFallsBackFromPullOnlyDefaultToPushEnabledStorage(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)
	gc := git.New(dir)
	writeAndCommit(t, gc, dir, "main.txt", "main", "initial commit")
	mustGit(t, gc, "checkout", "-b", "feature")
	writeAndCommit(t, gc, dir, "feature.txt", "feature", "feature commit")

	backupRoot := filepath.Join(tmp, "remote", filepath.Base(dir))
	cfg, err := core.ReadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DefaultRemote = "origin"
	cfg.Remotes["origin"] = core.RemoteConfig{
		Type:         "local",
		DisplayName:  "origin",
		Provider:     "local",
		Path:         filepath.Join(tmp, "pull-only-origin"),
		ObjectRoot:   "objects",
		ManifestRoot: "manifests",
		VerifyMode:   "local-hash",
		Pull:         true,
		Push:         false,
	}
	cfg.Remotes["backup"] = core.RemoteConfig{
		Type:         "local",
		DisplayName:  "backup",
		Provider:     "local",
		Path:         backupRoot,
		ObjectRoot:   "objects",
		ManifestRoot: "manifests",
		VerifyMode:   "local-hash",
		Pull:         true,
		Push:         true,
		Priority:     10,
	}
	if err := core.WriteConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}

	if err := app.Push(nil); err != nil {
		t.Fatalf("expected storage-only branch push to use backup, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(backupRoot, "feature.txt")); err != nil {
		t.Fatalf("expected feature file on push-enabled storage remote: %v", err)
	}
	if _, err := os.Stat(filepath.Join(backupRoot, "history", "feature", "full.bundle")); err != nil {
		t.Fatalf("expected feature branch history bundle on push-enabled storage remote: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "pull-only-origin", "feature.txt")); !os.IsNotExist(err) {
		t.Fatalf("pull-only default remote should not receive pushed files, stat err=%v", err)
	}
}

func TestPushLeavesUnchangedRemoteFilesUntouched(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)
	gc := git.New(dir)
	writeAndCommit(t, gc, dir, "file.txt", "content", "initial commit")

	remoteRoot := filepath.Join(tmp, "remote", filepath.Base(dir))
	cfg, err := core.ReadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DefaultRemote = "backup"
	cfg.Remotes["backup"] = core.RemoteConfig{
		Type:         "local",
		DisplayName:  "backup",
		Provider:     "local",
		Path:         remoteRoot,
		ObjectRoot:   "objects",
		ManifestRoot: "manifests",
		VerifyMode:   "local-hash",
		Pull:         true,
		Push:         true,
	}
	if err := core.WriteConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}

	if err := app.Push(nil); err != nil {
		t.Fatalf("first push: %v", err)
	}
	remoteFile := filepath.Join(remoteRoot, "file.txt")
	infoBefore, err := os.Stat(remoteFile)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)

	if err := app.Push(nil); err != nil {
		t.Fatalf("second push: %v", err)
	}
	infoAfter, err := os.Stat(remoteFile)
	if err != nil {
		t.Fatal(err)
	}
	if !infoAfter.ModTime().Equal(infoBefore.ModTime()) {
		t.Fatalf("unchanged remote file was rewritten: before=%s after=%s", infoBefore.ModTime(), infoAfter.ModTime())
	}
}

func TestPushOnlyChecksFilesChangedSinceLastPush(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)
	gc := git.New(dir)
	writeAndCommit(t, gc, dir, "one.txt", "one", "add one")
	writeAndCommit(t, gc, dir, "two.txt", "two", "add two")

	remoteRoot := filepath.Join(tmp, "remote", filepath.Base(dir))
	cfg, err := core.ReadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DefaultRemote = "backup"
	cfg.Remotes["backup"] = core.RemoteConfig{
		Type:         "local",
		DisplayName:  "backup",
		Provider:     "local",
		Path:         remoteRoot,
		ObjectRoot:   "objects",
		ManifestRoot: "manifests",
		VerifyMode:   "local-hash",
		Pull:         true,
		Push:         true,
	}
	if err := core.WriteConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}

	if err := app.Push(nil); err != nil {
		t.Fatalf("first push: %v", err)
	}
	remoteOne := filepath.Join(remoteRoot, "one.txt")
	remoteTwo := filepath.Join(remoteRoot, "two.txt")
	oneBefore, err := os.Stat(remoteOne)
	if err != nil {
		t.Fatal(err)
	}
	twoBefore, err := os.Stat(remoteTwo)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)

	writeAndCommit(t, gc, dir, "one.txt", "one changed", "change one")
	if err := app.Push(nil); err != nil {
		t.Fatalf("second push: %v", err)
	}

	oneAfter, err := os.Stat(remoteOne)
	if err != nil {
		t.Fatal(err)
	}
	twoAfter, err := os.Stat(remoteTwo)
	if err != nil {
		t.Fatal(err)
	}
	if !oneAfter.ModTime().After(oneBefore.ModTime()) {
		t.Fatalf("changed file was not rewritten: before=%s after=%s", oneBefore.ModTime(), oneAfter.ModTime())
	}
	if !twoAfter.ModTime().Equal(twoBefore.ModTime()) {
		t.Fatalf("unchanged file was checked/re-written: before=%s after=%s", twoBefore.ModTime(), twoAfter.ModTime())
	}
}
