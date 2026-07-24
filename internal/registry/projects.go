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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Registry struct {
	Projects []Project `json:"projects"`
}

type Project struct {
	RepoID   string    `json:"repo_id"`
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	LastSeen time.Time `json:"last_seen"`
}

func DefaultPath() string {
	cfg, err := os.UserConfigDir()
	if err != nil || cfg == "" {
		cfg = "."
	}
	return filepath.Join(cfg, "KK", "projects.json")
}

func Load(path string) (Registry, error) {
	if path == "" {
		path = DefaultPath()
	}
	var reg Registry
	data, err := os.ReadFile(path) // #nosec G304 -- registry path is either DefaultPath or an explicit caller-provided registry file.
	if err != nil {
		if os.IsNotExist(err) {
			return reg, nil
		}
		return reg, err
	}
	return reg, json.Unmarshal(data, &reg)
}

func Save(path string, reg Registry) error {
	if path == "" {
		path = DefaultPath()
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func (r *Registry) Upsert(project Project) {
	for i := range r.Projects {
		existingPath, err := NormalizeProjectPath(r.Projects[i].Path)
		if err != nil {
			existingPath = filepath.Clean(r.Projects[i].Path)
		}
		projectPath, err := NormalizeProjectPath(project.Path)
		if err != nil {
			projectPath = filepath.Clean(project.Path)
		}
		if r.Projects[i].RepoID == project.RepoID || existingPath == projectPath {
			r.Projects[i] = project
			return
		}
	}
	r.Projects = append(r.Projects, project)
}

func NormalizeProjectPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("project path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func (r Registry) FindByRepoID(repoID string) (Project, bool) {
	for _, project := range r.Projects {
		if project.RepoID == repoID {
			return project, true
		}
	}
	return Project{}, false
}

func (r Registry) FindByPath(path string) (Project, bool) {
	normalized, err := NormalizeProjectPath(path)
	if err != nil {
		return Project{}, false
	}
	for _, project := range r.Projects {
		projectPath, err := NormalizeProjectPath(project.Path)
		if err != nil {
			continue
		}
		if projectPath == normalized {
			return project, true
		}
	}
	return Project{}, false
}
