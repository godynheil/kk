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
)

func TestVerifyUploadedFileIsPointerNotWholeFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)
	gc := git.New(dir)

	// 1. Track *.bin files
	if err := app.Track([]string{"*.bin"}); err != nil {
		t.Fatalf("track failed: %v", err)
	}

	// 2. Add test.bin with content
	testFile := "test.bin"
	binContent := "hello large file - this should remain a pointer on the remote"

	// Write file, run kk add, and commit
	if err := os.WriteFile(filepath.Join(dir, testFile), []byte(binContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := app.Add([]string{testFile}); err != nil {
		t.Fatalf("kk add failed: %v", err)
	}
	mustGit(t, gc, "commit", "-m", "add binary file")

	// Verify it's materialized locally (contains the actual binary content)
	localBytes, err := os.ReadFile(filepath.Join(dir, testFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(localBytes) != binContent {
		t.Fatalf("expected local file to be materialized, got: %s", string(localBytes))
	}

	// 3. Set up local remote
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

	// 4. Run kk push
	if err := app.Push(nil); err != nil {
		t.Fatalf("push failed: %v", err)
	}

	// 5. Check the contents of the remote mirror
	remoteMirrorFile := filepath.Join(remoteRoot, testFile)
	remoteBytes, err := os.ReadFile(remoteMirrorFile)
	if err != nil {
		t.Fatalf("could not read remote mirror file: %v", err)
	}

	t.Logf("Remote mirror file content:\n%s", string(remoteBytes))

	// Verify if it contains pointer text or the whole file
	if strings.Contains(string(remoteBytes), "version kk-lfs-1.0.0") {
		t.Log("SUCCESS: Remote mirror has the pointer file, NOT the whole file!")
	} else if string(remoteBytes) == binContent {
		t.Error("FAILURE: Remote mirror has the WHOLE FILE (materialized bytes) instead of the pointer!")
	} else {
		t.Errorf("Unexpected content in remote mirror file: %q", string(remoteBytes))
	}

	// 6. Check the Git repository committed content (what gets uploaded to the Git remote)
	gitContent, err := gc.ShowHeadFile(testFile)
	if err != nil {
		t.Fatalf("could not read file from Git: %v", err)
	}
	t.Logf("Git HEAD file content:\n%s", gitContent)

	if !strings.Contains(gitContent, "version kk-lfs-1.0.0") {
		t.Errorf("Git remote commit content is NOT a pointer! Content: %q", gitContent)
	} else {
		t.Log("SUCCESS: Git commit contains the pointer, NOT the whole file!")
	}
}
