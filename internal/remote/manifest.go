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

package remote

import (
	"time"

	"github.com/godynheil/kk/internal/core"
)

func EmptyManifest(repo core.RepoInfo) core.Manifest {
	return core.Manifest{
		Version:     core.ManifestVersion,
		RepoID:      repo.RepoID,
		ProjectName: repo.Name,
		UpdatedAt:   time.Now().UTC(),
		Objects:     []core.ManifestObject{},
	}
}

func UpsertManifestObject(m core.Manifest, p core.Pointer) core.Manifest {
	if m.Version == "" {
		m.Version = core.ManifestVersion
	}
	found := false
	for i := range m.Objects {
		if m.Objects[i].OID == p.OID {
			m.Objects[i].Size = p.Size
			m.Objects[i].UploadedAt = time.Now().UTC()
			m.Objects[i].Verified = true
			found = true
			break
		}
	}
	if !found {
		m.Objects = append(m.Objects, core.ManifestObject{
			OID:        p.OID,
			Size:       p.Size,
			UploadedAt: time.Now().UTC(),
			Verified:   true,
		})
	}
	m.UpdatedAt = time.Now().UTC()
	return m
}
