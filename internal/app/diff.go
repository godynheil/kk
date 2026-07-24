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

type DiffSummary struct {
	Mode     string            `json:"mode"`
	CodeOnly bool              `json:"code_only"`
	Files    []DiffFileSummary `json:"files"`
}

type DiffFileSummary struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	Language  string `json:"language"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

func (a App) Diff(args []string) error {
	jsonOut := hasFlag(args, "--json")
	summary := hasFlag(args, "--summary")
	staged := hasFlag(args, "--staged")
	file := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--file" && i+1 < len(args) {
			file = args[i+1]
		}
	}
	if file != "" {
		if _, ok := core.CodeLanguage(file); !ok {
			return fmt.Errorf("not a recognized code file: %s", file)
		}
		gitArgs := []string{"diff", "--unified=3"}
		if staged {
			gitArgs = append(gitArgs, "--cached")
		}
		gitArgs = append(gitArgs, "--", file)
		return git.New(a.Root).Run(gitArgs...)
	}
	if summary || jsonOut {
		return a.diffSummary(staged, jsonOut)
	}
	return a.diffSummary(staged, false)
}

func (a App) diffSummary(staged bool, jsonOut bool) error {
	client := git.New(a.Root)
	args := []string{"diff", "--numstat"}
	if staged {
		args = append(args, "--cached")
	}
	numstat, _, _ := client.Combined(args...)
	statusArgs := []string{"diff", "--name-status"}
	if staged {
		statusArgs = append(statusArgs, "--cached")
	}
	statusOut, _, _ := client.Combined(statusArgs...)
	statuses := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(statusOut), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			statuses[parts[len(parts)-1]] = parts[0]
		}
	}
	result := DiffSummary{Mode: "unstaged", CodeOnly: true, Files: []DiffFileSummary{}}
	if staged {
		result.Mode = "staged"
	}
	for _, line := range strings.Split(strings.TrimSpace(numstat), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}
		path := parts[2]
		lang, ok := core.CodeLanguage(path)
		if !ok {
			continue
		}
		result.Files = append(result.Files, DiffFileSummary{Path: path, Status: statuses[path], Language: lang, Additions: parseStat(parts[0]), Deletions: parseStat(parts[1])})
	}
	if jsonOut {
		b, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	for _, file := range result.Files {
		fmt.Printf("%s %s +%d -%d %s\n", file.Status, file.Path, file.Additions, file.Deletions, file.Language)
	}
	return nil
}

func parseStat(s string) int {
	if s == "-" {
		return 0
	}
	var n int
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}
