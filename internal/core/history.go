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

package core

import (
	"fmt"
	"strings"
	"time"
)

type BranchHistory struct {
	BaseRef string   `json:"base_ref"`
	NextSeq int      `json:"next_seq"`
	Bundles []string `json:"bundles"`
	Tip     string   `json:"tip"`
}

type RefsSnapshot struct {
	Version       string                   `json:"version"`
	DefaultBranch string                   `json:"default_branch"`
	Branches      map[string]BranchHistory `json:"branches"`
	UpdatedAt     time.Time                `json:"updated_at"`
}

type BranchHistoryState struct {
	LastAppliedBundle string `json:"last_applied_bundle"`
	LastAppliedRef    string `json:"last_applied_ref"`
}

type RemoteHistoryState struct {
	Branches  map[string]BranchHistoryState `json:"branches"`
	UpdatedAt time.Time                     `json:"updated_at"`
}

type HistoryState struct {
	Version string                        `json:"version"`
	Remotes map[string]RemoteHistoryState `json:"remotes"`
}

func IsValidBundleName(name string) bool {
	if name == "full.bundle" {
		return true
	}
	if strings.HasPrefix(name, "inc-") && strings.HasSuffix(name, ".bundle") {
		if len(name) == 17 {
			digits := name[4:10]
			for _, r := range digits {
				if r < '0' || r > '9' {
					return false
				}
			}
			return true
		}
	}
	return false
}

func IsValidBranchName(name string) bool {
	if name == "" {
		return false
	}

	if strings.Contains(name, "..") || strings.Contains(name, "\\") {
		return false
	}

	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return false
	}

	if strings.Contains(name, "//") || strings.Contains(name, "@{") || name == "@" {
		return false
	}
	for _, r := range name {
		if r < 32 || r == 127 || r == ' ' || r == '~' || r == '^' || r == ':' || r == '?' || r == '*' || r == '[' {
			return false
		}
	}

	parts := strings.Split(name, "/")
	for _, p := range parts {
		if p == "" || strings.HasPrefix(p, ".") || strings.HasSuffix(p, ".") {
			return false
		}
	}
	return true
}

func (r RefsSnapshot) Validate() error {
	if r.DefaultBranch != "" && !IsValidBranchName(r.DefaultBranch) {
		return fmt.Errorf("unsafe default branch name: %q", r.DefaultBranch)
	}
	for branchName, branchSnap := range r.Branches {
		if !IsValidBranchName(branchName) {
			return fmt.Errorf("unsafe branch name: %q", branchName)
		}
		for _, bundleName := range branchSnap.Bundles {
			if !IsValidBundleName(bundleName) {
				return fmt.Errorf("unsafe bundle name: %q under branch %q", bundleName, branchName)
			}
		}
	}
	return nil
}
