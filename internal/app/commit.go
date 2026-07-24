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

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/git"
)

func (a App) Commit(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kk commit -m <message>")
	}
	client := git.New(a.Root)
	if err := client.EnsureRepository(); err != nil {
		return err
	}
	if err := a.validateStagedLargeFiles(); err != nil {
		return err
	}
	return client.Run(append([]string{"commit"}, args...)...)
}

func (a App) validateStagedLargeFiles() error {
	client := git.New(a.Root)
	tracks, err := core.ReadTracks(a.Root)
	if err != nil {
		return err
	}
	files, err := client.StagedFiles()
	if err != nil {
		return err
	}
	for _, file := range files {
		if !core.ShouldTrack(file, tracks) {
			continue
		}
		data, err := client.ShowIndexFile(file)
		if err != nil {
			continue
		}
		if _, ok := core.ParsePointerBytes(data); ok {
			continue
		}
		return fmt.Errorf("staged large file is not a kk pointer: %s; use kk add %s", file, file)
	}
	return nil
}
