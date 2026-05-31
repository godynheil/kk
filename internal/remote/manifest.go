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
