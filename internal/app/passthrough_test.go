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
)

func TestRunAcceptsWrappedGitCommandsWithoutGitSubcommand(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	initKKTestRepo(t, dir)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", "note.txt"},
		{"commit", "-m", "initial"},
		{"branch", "--show-current"},
		{"rev-parse", "HEAD"},
		{"log", "-1", "--pretty=%s"},
		{"show-ref", "--verify", "--quiet", "refs/heads/" + testMainBranch},
		{"checkout", "-B", "feature"},
	} {
		if err := Run(args); err != nil {
			t.Fatalf("kk %v failed: %v", args, err)
		}
	}
}

func TestRunRoutesGitShapedStatusAndRemoteFormsToWrappedGit(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	initKKTestRepo(t, dir)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	for _, args := range [][]string{
		{"status", "--porcelain"},
		{"status", "--short"},
		{"remote"},
		{"remote", "-v"},
		{"remote", "list"},
	} {
		if err := Run(args); err != nil {
			t.Fatalf("kk %v failed: %v", args, err)
		}
	}
}
