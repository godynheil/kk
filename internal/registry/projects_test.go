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

package registry

import (
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultPath(t *testing.T) {
	path := DefaultPath()
	if path == "" {
		t.Fatal("expected non-empty default path")
	}
	if !filepath.IsAbs(path) && path != "." {
		// user config dir might fail to determine, but it should fall back safely or be an absolute path
		t.Logf("default path: %s", path)
	}
}

func TestLoadSave(t *testing.T) {
	tempDir := t.TempDir()
	regPath := filepath.Join(tempDir, "projects.json")

	// 1. Loading non-existent file returns empty registry with no error
	reg, err := Load(regPath)
	if err != nil {
		t.Fatalf("expected no error loading non-existent path, got %v", err)
	}
	if len(reg.Projects) != 0 {
		t.Fatalf("expected empty registry, got %d projects", len(reg.Projects))
	}

	// 2. Saving a registry
	testProj := Project{
		RepoID:   "test-repo-id-123",
		Name:     "Test Project",
		Path:     tempDir,
		LastSeen: time.Now().Truncate(time.Second), // truncate to avoid nanosecond precision diffs in JSON
	}
	reg.Projects = append(reg.Projects, testProj)

	err = Save(regPath, reg)
	if err != nil {
		t.Fatalf("expected no error on save, got %v", err)
	}

	// 3. Load it back and verify
	loadedReg, err := Load(regPath)
	if err != nil {
		t.Fatalf("expected no error on load, got %v", err)
	}
	if len(loadedReg.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(loadedReg.Projects))
	}

	loadedProj := loadedReg.Projects[0]
	if loadedProj.RepoID != testProj.RepoID || loadedProj.Name != testProj.Name || loadedProj.Path != testProj.Path {
		t.Errorf("expected %+v, got %+v", testProj, loadedProj)
	}
	if !loadedProj.LastSeen.Equal(testProj.LastSeen) {
		t.Errorf("expected LastSeen %v, got %v", testProj.LastSeen, loadedProj.LastSeen)
	}
}

func TestNormalizeProjectPath(t *testing.T) {
	// Test empty path error
	_, err := NormalizeProjectPath("   ")
	if err == nil {
		t.Error("expected error for empty project path")
	}

	// Test absolute path resolution
	path := "."
	norm, err := NormalizeProjectPath(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !filepath.IsAbs(norm) {
		t.Errorf("expected absolute path, got %q", norm)
	}
}

func TestUpsert(t *testing.T) {
	reg := Registry{}

	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	proj1 := Project{
		RepoID:   "repo-1",
		Name:     "Project One",
		Path:     tempDir1,
		LastSeen: time.Now(),
	}

	// 1. Insert new project
	reg.Upsert(proj1)
	if len(reg.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(reg.Projects))
	}

	// 2. Update existing project by RepoID
	proj1Updated := Project{
		RepoID:   "repo-1",
		Name:     "Project One Updated",
		Path:     tempDir2, // path changed but RepoID remains same
		LastSeen: time.Now(),
	}
	reg.Upsert(proj1Updated)
	if len(reg.Projects) != 1 {
		t.Fatalf("expected still 1 project after update, got %d", len(reg.Projects))
	}
	if reg.Projects[0].Name != "Project One Updated" {
		t.Errorf("expected updated name to be %q, got %q", "Project One Updated", reg.Projects[0].Name)
	}

	// 3. Update existing project by Path (RepoID changed)
	proj2PathUpdated := Project{
		RepoID:   "repo-2", // different repo id
		Name:     "Project Two",
		Path:     tempDir2, // same path as proj1Updated
		LastSeen: time.Now(),
	}
	reg.Upsert(proj2PathUpdated)
	if len(reg.Projects) != 1 {
		t.Fatalf("expected still 1 project after path-match update, got %d", len(reg.Projects))
	}
	if reg.Projects[0].RepoID != "repo-2" {
		t.Errorf("expected updated RepoID to be %q, got %q", "repo-2", reg.Projects[0].RepoID)
	}
}

func TestFindByRepoIDAndPath(t *testing.T) {
	tempDir := t.TempDir()
	reg := Registry{
		Projects: []Project{
			{
				RepoID:   "id-a",
				Name:     "Proj A",
				Path:     tempDir,
				LastSeen: time.Now(),
			},
		},
	}

	// Find by RepoID success
	p, found := reg.FindByRepoID("id-a")
	if !found || p.Name != "Proj A" {
		t.Errorf("expected to find 'Proj A' by ID, got found=%v, name=%q", found, p.Name)
	}

	// Find by RepoID failure
	_, found = reg.FindByRepoID("id-nonexistent")
	if found {
		t.Error("expected to not find nonexistent ID")
	}

	// Find by Path success
	p, found = reg.FindByPath(tempDir)
	if !found || p.RepoID != "id-a" {
		t.Errorf("expected to find 'Proj A' by Path, got found=%v, ID=%q", found, p.RepoID)
	}

	// Find by Path failure
	_, found = reg.FindByPath(filepath.Join(tempDir, "does-not-exist"))
	if found {
		t.Error("expected to not find nonexistent Path")
	}
}
