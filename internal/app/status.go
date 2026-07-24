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
	"strings"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/git"
)

type StatusResult struct {
	Initialized bool   `json:"initialized"`
	RepoID      string `json:"repo_id,omitempty"`
	Name        string `json:"name,omitempty"`
	Branch      string `json:"branch,omitempty"`
	Dirty       bool   `json:"dirty"`
	Raw         string `json:"raw,omitempty"`
}

func (a App) Status(jsonOut bool) error {
	client := git.New(a.Root)
	res := StatusResult{Initialized: client.IsInitialized()}
	if res.Initialized {
		if info, err := core.ReadRepoInfo(a.Root); err == nil {
			res.RepoID = info.RepoID
			res.Name = info.Name
			_ = syncProjectRegistry(a.Root)
		}
		res.Branch = client.CurrentBranch()
		raw, _, _ := a.gitStatusOutput([]string{"status", "--short", "--branch"})
		res.Raw = strings.TrimSpace(raw)
		for _, line := range strings.Split(res.Raw, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "##") {
				res.Dirty = true
				break
			}
		}
	}
	if jsonOut {
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	if !res.Initialized {
		return fmt.Errorf("not a kk repository")
	}
	fmt.Println(res.Raw)
	return nil
}
