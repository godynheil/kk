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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func ReadTracks(root string) (Tracks, error) {
	var tracks Tracks
	data, err := os.ReadFile(filepath.Join(root, TracksFile)) // #nosec G304 -- tracks metadata is read from the caller's repository root.
	if err != nil {
		return Tracks{Patterns: []string{}}, err
	}
	if err := json.Unmarshal(data, &tracks); err != nil {
		return tracks, err
	}
	if tracks.Patterns == nil {
		tracks.Patterns = []string{}
	}
	return tracks, nil
}

func WriteTracks(root string, tracks Tracks) error {
	if tracks.Patterns == nil {
		tracks.Patterns = []string{}
	}
	data, err := json.MarshalIndent(tracks, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(root, TracksFile), data, 0o600)
}

func AddTrackPattern(tracks Tracks, pattern string) Tracks {
	pattern = strings.TrimSpace(filepath.ToSlash(pattern))
	if pattern == "" {
		return tracks
	}
	for _, existing := range tracks.Patterns {
		if existing == pattern {
			return tracks
		}
	}
	tracks.Patterns = append(tracks.Patterns, pattern)
	return tracks
}

func RemoveTrackPattern(tracks Tracks, pattern string) Tracks {
	pattern = strings.TrimSpace(filepath.ToSlash(pattern))
	var out []string
	for _, existing := range tracks.Patterns {
		if existing != pattern {
			out = append(out, existing)
		}
	}
	tracks.Patterns = out
	return tracks
}

func ShouldTrack(path string, tracks Tracks) bool {
	path = filepath.ToSlash(filepath.Clean(path))
	_, isCode := CodeLanguage(path)
	if len(tracks.Patterns) == 0 {
		return !isCode
	}
	base := filepath.Base(path)
	matched := false
	for _, pattern := range tracks.Patterns {
		if ok, _ := filepath.Match(pattern, path); ok {
			matched = true
			break
		}
		if ok, _ := filepath.Match(pattern, base); ok {
			matched = true
			break
		}
		if strings.HasSuffix(pattern, "/") {
			prefix := strings.TrimSuffix(pattern, "/")
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				matched = true
				break
			}
		}
	}
	if !matched {
		return false
	}
	return !isCode
}
