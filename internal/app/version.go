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
	"fmt"
	"os/exec"
	"strings"

	"github.com/godynheil/kk/internal/git"
)

var BuildVersion = "dev"

var BuildDate = ""

var BuildDateLocal = ""

func (a App) Version() error {
	display := strings.TrimSpace(BuildVersion)
	if display == "" {
		display = "dev"
	}

	if display == "dev" {
		if gp, err := git.GitExecutable(); err == nil {
			if out, err := exec.Command(gp, "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil { // #nosec G204 -- gp is resolved by GitExecutable; arguments are fixed.
				branch := strings.TrimSpace(string(out))
				if branch != "" {
					display = strings.ReplaceAll(branch, "/", "-")
				}
			}
		}
	}

	display = strings.ReplaceAll(display, "/", "-")

	utc := strings.TrimSpace(BuildDate)
	if utc != "" {
		utcFmt := strings.Replace(utc, "T", " ", 1)
		utcFmt = strings.TrimSuffix(utcFmt, "Z") + " UTC"

		local := strings.TrimSpace(BuildDateLocal)
		if local != "" {
			localFmt := strings.Replace(local, "T", " ", 1)
			fmt.Printf("kk %s (built %s | %s)\n", display, utcFmt, localFmt)
		} else {
			fmt.Printf("kk %s (built %s)\n", display, utcFmt)
		}
	} else {
		fmt.Printf("kk %s\n", display)
	}
	if gv, err := gitVersion(); err == nil {
		fmt.Println(gv)
	}
	return nil
}

func gitVersion() (string, error) {
	gitPath, err := git.GitExecutable()
	if err != nil {
		return "", fmt.Errorf("get kk history engine version: %w", err)
	}
	out, err := exec.Command(gitPath, "--version").Output() // #nosec G204 -- gitPath is resolved by GitExecutable; arguments are fixed.
	if err != nil {
		return "", fmt.Errorf("get kk history engine version: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
