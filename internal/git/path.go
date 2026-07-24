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

package git

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	gitExecPath string
	gitExecErr  error
	resolveOnce sync.Once
)

func GitExecutable() (string, error) {
	resolveOnce.Do(func() {
		gitExecPath, gitExecErr = resolveGitExecutable()
	})
	return gitExecPath, gitExecErr
}

func resolveGitExecutable() (string, error) {
	if p, err := exec.LookPath("git"); err == nil {
		return p, nil
	}

	exe, err := os.Executable()
	if err != nil {
		return "", errors.New("kk history engine not found on PATH and failed to determine executable path")
	}
	dir := filepath.Dir(exe)

	for i := 0; i < 5; i++ {
		if runtime.GOOS == "windows" {
			cand := filepath.Join(dir, "PortableGit", "cmd", "git.exe")
			if _, err := os.Stat(cand); err == nil {
				return cand, nil
			}
			cand2 := filepath.Join(dir, "PortableGit", "mingw64", "bin", "git.exe")
			if _, err := os.Stat(cand2); err == nil {
				return cand2, nil
			}
		} else {
			cand := filepath.Join(dir, "PortableGit", "usr", "bin", "git")
			if _, err := os.Stat(cand); err == nil {
				return cand, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", errors.New("kk history engine not found on PATH and bundled history engine not found next to executable")
}
