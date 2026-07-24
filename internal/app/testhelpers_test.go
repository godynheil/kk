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
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/godynheil/kk/internal/git"
)

const testMainBranch = "kk-test-main"

func mustGit(t *testing.T, client git.Client, args ...string) {
	t.Helper()
	_, stderr, err := client.Combined(args...)
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, stderr)
	}
}

func initKKTestRepo(t *testing.T, dir string) App {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	a := New(dir)
	if err := a.Init(); err != nil {
		t.Fatalf("kk init in %s: %v", dir, err)
	}
	gc := git.New(dir)
	mustGit(t, gc, "config", "user.email", "test@example.com")
	mustGit(t, gc, "config", "user.name", "KK Test")
	mustGit(t, gc, "checkout", "-B", testMainBranch)
	return a
}

func initBareRepo(t *testing.T, dir string) string {
	t.Helper()
	gitPath, err := git.GitExecutable()
	if err != nil {
		t.Skip("git not available:", err)
	}
	cmd := exec.Command(gitPath, "init", "--bare", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare %s: %v\n%s", dir, err, out)
	}
	return dir
}

func writeAndCommit(t *testing.T, gc git.Client, root, name, content, msg string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, gc, "add", name)
	mustGit(t, gc, "commit", "-m", msg)
}
