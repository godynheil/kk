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

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/remote"
	"github.com/godynheil/kk/internal/storage"
)

type FsckResult struct {
	OK      bool               `json:"ok"`
	Objects []FsckObjectResult `json:"objects"`
}

type FsckObjectResult struct {
	Path    string `json:"path"`
	OID     string `json:"oid"`
	Size    int64  `json:"size"`
	Status  string `json:"status"`
	Remote  string `json:"remote,omitempty"`
	Message string `json:"message,omitempty"`
}

func (a App) Fsck(jsonOut bool) error {
	objects, err := a.requiredObjectsInHEAD()
	if err != nil {
		return err
	}
	store := storage.New(a.Root)
	cfg, _ := core.ReadConfig(a.Root)
	info, _ := core.ReadRepoInfo(a.Root)
	result := FsckResult{OK: true, Objects: []FsckObjectResult{}}
	for _, obj := range objects {
		p := core.Pointer{OID: obj.OID, Size: obj.Size}
		row := FsckObjectResult{Path: obj.Path, OID: obj.OID, Size: obj.Size}
		if store.HasObject(p) {
			row.Status = "ok-local"
			result.Objects = append(result.Objects, row)
			continue
		}
		found := false
		for _, target := range cfg.PullRemotes() {
			driver, err := remote.New(target.Name, target.Config)
			if err != nil {
				continue
			}
			ok, _ := driver.HasObject(info, p)
			if ok {
				row.Status = "ok-remote"
				row.Remote = target.Name
				found = true
				break
			}
		}
		if !found {
			row.Status = "missing"
			row.Message = "object not available locally or on pull-enabled remotes"
			result.OK = false
		}
		result.Objects = append(result.Objects, row)
	}
	if jsonOut {
		b, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(b))
		if !result.OK {
			return fmt.Errorf("fsck failed")
		}
		return nil
	}
	for _, row := range result.Objects {
		if row.Remote != "" {
			fmt.Printf("%s %s %s remote=%s\n", row.Status, row.OID, row.Path, row.Remote)
		} else {
			fmt.Printf("%s %s %s\n", row.Status, row.OID, row.Path)
		}
		if row.Message != "" {
			fmt.Println("  ", row.Message)
		}
	}
	if !result.OK {
		return fmt.Errorf("fsck failed")
	}
	fmt.Println("fsck ok")
	return nil
}
