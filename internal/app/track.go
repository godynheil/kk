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
)

func (a App) Track(patterns []string) error {
	if len(patterns) == 0 {
		return fmt.Errorf("usage: kk track <pattern...>")
	}
	tracks, err := core.ReadTracks(a.Root)
	if err != nil {
		return err
	}
	for _, pattern := range patterns {
		tracks = core.AddTrackPattern(tracks, pattern)
		fmt.Println("tracking", pattern)
	}
	return core.WriteTracks(a.Root, tracks)
}

func (a App) Untrack(patterns []string) error {
	if len(patterns) == 0 {
		return fmt.Errorf("usage: kk untrack <pattern...>")
	}
	tracks, err := core.ReadTracks(a.Root)
	if err != nil {
		return err
	}
	for _, pattern := range patterns {
		tracks = core.RemoveTrackPattern(tracks, pattern)
		fmt.Println("untracked", pattern)
	}
	return core.WriteTracks(a.Root, tracks)
}

func (a App) TrackList() error {
	tracks, err := core.ReadTracks(a.Root)
	if err != nil {
		return err
	}
	if len(tracks.Patterns) == 0 {
		fmt.Println("no tracked patterns")
		return nil
	}
	for _, pattern := range tracks.Patterns {
		fmt.Println(pattern)
	}
	return nil
}
