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

const (
	KKDir            = ".kk"
	KKGitDir         = ".kk/git"
	ObjectDir        = ".kk/objects"
	TmpDir           = ".kk/tmp"
	LogDir           = ".kk/logs"
	RepoFile         = ".kk/repo.json"
	ConfigFile       = ".kk/config.json"
	TracksFile       = ".kk/tracks.json"
	PushStateFile    = ".kk/push-state.json"
	HistoryStateFile = ".kk/history-state.json"
	KKIgnoreFile     = ".kkignore"

	HistoryRoot = "history"

	PointerVersion      = "kk-lfs-1.0.0"
	ConfigVersion       = "kk-local-config-1.0.0"
	ManifestVersion     = "kk-manifest-1.0.0"
	HistoryVersion      = "kk-history-1.0.0"
	HistoryStateVersion = "kk-history-state-1.0.0"
)
