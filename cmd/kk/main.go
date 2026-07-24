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

package main

import (
	"fmt"
	"os"
	"runtime"
	"syscall"

	"github.com/godynheil/kk/internal/app"
)

func init() {
	if runtime.GOOS == "windows" {
		handle := syscall.Handle(os.Stdout.Fd())
		var mode uint32
		if err := syscall.GetConsoleMode(handle, &mode); err == nil {
			mode |= 0x0004
			_, _, _ = syscall.NewLazyDLL("kernel32.dll").NewProc("SetConsoleMode").Call(uintptr(handle), uintptr(mode))
		}
	}
}

func main() {
	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "kk:", err)
		os.Exit(1)
	}
}
