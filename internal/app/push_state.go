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
	"os"
	"path/filepath"
	"time"

	"github.com/godynheil/kk/internal/core"
)

type pushState struct {
	Version string                     `json:"version"`
	Remotes map[string]remotePushState `json:"remotes"`
}

type remotePushState struct {
	HeadCommit string    `json:"head_commit"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func readPushState(root string) (pushState, error) {
	state := pushState{
		Version: "kk-push-state-1.0.0",
		Remotes: map[string]remotePushState{},
	}
	data, err := os.ReadFile(filepath.Join(root, core.PushStateFile)) // #nosec G304 -- push state is read from the caller's repository root.
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	if state.Remotes == nil {
		state.Remotes = map[string]remotePushState{}
	}
	return state, nil
}

func writePushState(root string, state pushState) error {
	if state.Version == "" {
		state.Version = "kk-push-state-1.0.0"
	}
	if state.Remotes == nil {
		state.Remotes = map[string]remotePushState{}
	}
	path := filepath.Join(root, core.PushStateFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}
