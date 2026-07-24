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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func (a App) InstallPath() error {
	if runtime.GOOS != "windows" {
		return installPathUnix()
	}
	return installPathWindows()
}

func installPathWindows() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not resolve kk executable path: %w", err)
	}
	exeDir := filepath.Dir(exePath)

	psReadCmd := `[Environment]::GetEnvironmentVariable('Path','User')`
	out, err := exec.Command("powershell", "-NoProfile", "-Command", psReadCmd).Output()
	if err != nil {
		return fmt.Errorf("could not read user PATH from registry: %w", err)
	}
	currentPath := strings.TrimSpace(string(out))

	for _, entry := range strings.Split(currentPath, ";") {
		if strings.EqualFold(strings.TrimSpace(entry), exeDir) {
			fmt.Printf("kk is already on PATH (%s)\n", exeDir)
			fmt.Println("No changes made.")
			return nil
		}
	}

	newPath := currentPath
	if newPath != "" && !strings.HasSuffix(newPath, ";") {
		newPath += ";"
	}
	newPath += exeDir

	psWriteCmd := `[Environment]::SetEnvironmentVariable('Path',$env:KK_NEW_PATH,'User')`
	cmd := exec.Command("powershell", "-NoProfile", "-Command", psWriteCmd)
	cmd.Env = append(os.Environ(), "KK_NEW_PATH="+newPath)
	if writeOut, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("could not update user PATH: %w\n%s", err, string(writeOut))
	}

	fmt.Printf("Added to user PATH: %s\n", exeDir)
	fmt.Println()
	fmt.Println("Open a new terminal (or restart your shell) and run:")
	fmt.Println("  kk")
	fmt.Println()
	fmt.Println("To verify:")
	fmt.Println("  Get-Command kk")
	return nil
}

func installPathUnix() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not resolve kk executable path: %w", err)
	}
	exeDir := filepath.Dir(exePath)

	fmt.Println("To add kk to your PATH, add the following line to your shell profile")
	fmt.Println("(~/.bashrc, ~/.zshrc, ~/.profile, etc.):")
	fmt.Println()
	fmt.Printf(`  export PATH="%s:$PATH"`+"\n", exeDir)
	fmt.Println()
	fmt.Println("Then reload your shell:")
	fmt.Println("  source ~/.bashrc   # or ~/.zshrc, etc.")
	fmt.Println()
	fmt.Println("Or move / symlink kk to a directory already on your PATH:")
	fmt.Printf("  sudo ln -sf %s /usr/local/bin/kk\n", exePath)
	return nil
}
