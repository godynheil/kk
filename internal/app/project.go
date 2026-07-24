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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/registry"
)

type ProjectConnectResult struct {
	Action  string           `json:"action"`
	Project registry.Project `json:"project"`
}

func (a App) Project(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kk project <connect|reimport|list>")
	}
	switch args[0] {
	case "connect":
		return a.projectConnect("connect", args[1:])
	case "reimport":
		return a.projectConnect("reimport", args[1:])
	case "list":
		return a.projectList(hasFlag(args[1:], "--json"))
	default:
		return fmt.Errorf("unknown project command: %s", args[0])
	}
}

func (a App) projectConnect(verb string, args []string) error {
	jsonOut := hasFlag(args, "--json")
	args = removeFlags(args, "--json")
	path := a.Root
	if len(args) > 1 {
		return fmt.Errorf("usage: kk project %s [path] [--json]", verb)
	}
	if len(args) == 1 {
		path = args[0]
	}
	result, err := ConnectProject(path)
	if err != nil {
		return err
	}
	if jsonOut {
		b, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	fmt.Printf("%s %s (%s)\n", result.Action, result.Project.Name, result.Project.Path)
	return nil
}

func (a App) projectList(jsonOut bool) error {
	reg, err := registry.Load("")
	if err != nil {
		return err
	}
	if jsonOut {
		b, _ := json.MarshalIndent(reg, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	for _, project := range reg.Projects {
		fmt.Printf("%s %s %s\n", project.RepoID, project.Name, project.Path)
	}
	return nil
}

func syncProjectRegistry(root string) error {
	_, err := ConnectProject(root)
	return err
}

func ConnectProject(root string) (ProjectConnectResult, error) {
	normalized, err := registry.NormalizeProjectPath(root)
	if err != nil {
		return ProjectConnectResult{}, err
	}
	info, err := core.ReadRepoInfo(normalized)
	if err != nil {
		if os.IsNotExist(err) {
			return ProjectConnectResult{}, fmt.Errorf("not a kk repository: %s", normalized)
		}
		return ProjectConnectResult{}, err
	}
	project := registry.Project{
		RepoID:   info.RepoID,
		Name:     projectName(info, normalized),
		Path:     normalized,
		LastSeen: time.Now().UTC(),
	}
	reg, err := registry.Load("")
	if err != nil {
		return ProjectConnectResult{}, err
	}
	action := "connected"
	if existing, ok := reg.FindByRepoID(project.RepoID); ok {
		if samePath(existing.Path, project.Path) {
			action = "refreshed"
		} else {
			action = "reimported"
		}
	} else if _, ok := reg.FindByPath(project.Path); ok {
		action = "updated"
	}
	reg.Upsert(project)
	if err := registry.Save("", reg); err != nil {
		return ProjectConnectResult{}, err
	}
	return ProjectConnectResult{Action: action, Project: project}, nil
}

func projectName(info core.RepoInfo, path string) string {
	if info.Name != "" {
		return info.Name
	}
	return filepath.Base(path)
}

func samePath(a string, b string) bool {
	left, err := registry.NormalizeProjectPath(a)
	if err != nil {
		return false
	}
	right, err := registry.NormalizeProjectPath(b)
	if err != nil {
		return false
	}
	return left == right
}
