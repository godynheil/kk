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

import "time"

type RepoInfo struct {
	RepoID    string    `json:"repo_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type LocalConfig struct {
	Version       string                  `json:"version"`
	DefaultRemote string                  `json:"default_remote"`
	Remotes       map[string]RemoteConfig `json:"remotes"`
}

type RemoteConfig struct {
	Type        string `json:"type"`
	DisplayName string `json:"display_name,omitempty"`
	Role        string `json:"role,omitempty"`
	Provider    string `json:"provider,omitempty"`

	Path string `json:"path,omitempty"`

	URL string `json:"url,omitempty"`

	Binary string `json:"binary,omitempty"`
	Remote string `json:"remote,omitempty"`

	DriveFolderID string `json:"drive_folder_id,omitempty"`
	DriveAuthPath string `json:"drive_auth_path,omitempty"`

	SSH  string `json:"ssh,omitempty"`
	Root string `json:"root,omitempty"`

	ObjectRoot   string   `json:"object_root,omitempty"`
	ManifestRoot string   `json:"manifest_root,omitempty"`
	VerifyMode   string   `json:"verify_mode,omitempty"`
	Priority     int      `json:"priority,omitempty"`
	Pull         bool     `json:"pull"`
	Push         bool     `json:"push"`
	Tags         []string `json:"tags,omitempty"`

	UploadTimeoutSeconds  int  `json:"upload_timeout_seconds,omitempty"`
	ChunkSizeMB           int  `json:"chunk_size_mb,omitempty"`
	MaxConcurrentUploads  int  `json:"max_concurrent_uploads,omitempty"`
	BufferSizeMB          int  `json:"buffer_size_mb,omitempty"`
	RcloneTransfers       int  `json:"rclone_transfers,omitempty"`
	DisableConnectionPool bool `json:"disable_connection_pool,omitempty"`
}

type NamedRemoteConfig struct {
	Name   string
	Config RemoteConfig
}

type Tracks struct {
	Patterns []string `json:"patterns"`
}

type Pointer struct {
	OID  string
	Size int64
}

type RequiredObject struct {
	Path string `json:"path"`
	OID  string `json:"oid"`
	Size int64  `json:"size"`
}

type LiveObject struct {
	OID  string      `json:"oid"`
	Size int64       `json:"size"`
	Refs []ObjectRef `json:"refs"`
}

type ObjectRef struct {
	Commit string `json:"commit"`
	Path   string `json:"path"`
}

type PruneCandidate struct {
	OID  string `json:"oid"`
	Path string `json:"path"`
}

type Manifest struct {
	Version     string           `json:"version"`
	RepoID      string           `json:"repo_id"`
	ProjectName string           `json:"project_name"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Objects     []ManifestObject `json:"objects"`
}

type ManifestObject struct {
	OID        string    `json:"oid"`
	Size       int64     `json:"size"`
	UploadedAt time.Time `json:"uploaded_at"`
	Verified   bool      `json:"verified"`
}
