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
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/storage"
)

type Local struct {
	name         string
	root         string
	objectRoot   string
	manifestRoot string
}

func NewLocal(name string, cfg core.RemoteConfig) (*Local, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("local remote %s requires path", name)
	}
	return &Local{
		name:         name,
		root:         cfg.Path,
		objectRoot:   DefaultObjectRoot(cfg.ObjectRoot),
		manifestRoot: DefaultManifestRoot(cfg.ManifestRoot),
	}, nil
}

func (l *Local) Name() string { return l.name }

func (l *Local) Check() error {
	return os.MkdirAll(l.root, 0o750)
}

func (l *Local) objectPath(oid string) string {
	return filepath.Join(l.root, l.objectRoot, oid[:2], oid[2:4], oid)
}

func (l *Local) manifestPath(repoID string) string {
	return filepath.Join(l.root, l.manifestRoot, repoID+".json")
}

func (l *Local) PutObject(repo core.RepoInfo, p core.Pointer, localPath string) error {
	if ok, _ := l.HasObject(repo, p); ok {
		return nil
	}
	dst := l.objectPath(p.OID)
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	if err := copyFile(localPath, tmp); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	actualOID, actualSize, err := storage.HashFile(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if actualOID != p.OID || actualSize != p.Size {
		_ = os.Remove(tmp)
		return fmt.Errorf("local remote verification failed for %s", p.OID)
	}
	return os.Rename(tmp, dst)
}

func (l *Local) GetObject(repo core.RepoInfo, p core.Pointer, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0o750); err != nil {
		return err
	}
	tmp := localPath + ".tmp"
	if err := copyFile(l.objectPath(p.OID), tmp); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	actualOID, actualSize, err := storage.HashFile(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if actualOID != p.OID || actualSize != p.Size {
		_ = os.Remove(tmp)
		return fmt.Errorf("download verification failed for %s", p.OID)
	}
	return os.Rename(tmp, localPath)
}

func (l *Local) HasObject(repo core.RepoInfo, p core.Pointer) (bool, error) {
	path := l.objectPath(p.OID)
	info, err := os.Stat(path)
	if err != nil {
		return false, nil
	}
	return info.Size() == p.Size, nil
}

func (l *Local) ReadManifest(repo core.RepoInfo) (core.Manifest, error) {
	path := l.manifestPath(repo.RepoID)
	data, err := os.ReadFile(path) // #nosec G304 -- path is built from configured local remote root and manifest filename.
	if err != nil {
		return EmptyManifest(repo), nil
	}
	var m core.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return core.Manifest{}, err
	}
	return m, nil
}

func (l *Local) WriteManifest(repo core.RepoInfo, manifest core.Manifest) error {
	path := l.manifestPath(repo.RepoID)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	sort.Slice(manifest.Objects, func(i, j int) bool { return manifest.Objects[i].OID < manifest.Objects[j].OID })
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func (l *Local) HasProject(repo core.RepoInfo) (bool, error) {
	_, err := os.Stat(l.root)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func (l *Local) ReadRemoteRepoInfo(repo core.RepoInfo) (core.RepoInfo, bool, error) {
	path := filepath.Join(l.root, core.RepoFile)
	data, err := os.ReadFile(path) // #nosec G304 -- path is built from configured local remote root and repo metadata filename.
	if os.IsNotExist(err) {
		return core.RepoInfo{}, false, nil
	}
	if err != nil {
		return core.RepoInfo{}, false, err
	}
	var info core.RepoInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return core.RepoInfo{}, false, err
	}
	return info, true, nil
}

func (l *Local) SyncProjectFiles(repo core.RepoInfo, localRoot string, files []string, workers int,
	onStart func(workerID int, file string),
	onProgress func(workerID, done, total int, file string),
) (SyncStats, error) {
	var changed atomic.Int64
	var skipped atomic.Int64
	err := ConcurrentFiles(files, workers, onStart, onProgress, func(rel string) error {
		src, err := safeJoinUnder(localRoot, rel)
		if err != nil {
			return err
		}
		dst, err := safeJoinUnder(l.root, rel)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return err
		}
		if same, err := sameFileContent(src, dst); err != nil {
			return err
		} else if same {
			skipped.Add(1)
			return nil
		}
		if err := copyFile(src, dst); err != nil {
			return err
		}
		changed.Add(1)
		return nil
	})
	return SyncStats{Changed: int(changed.Load()), Skipped: int(skipped.Load())}, err
}

func (l *Local) DownloadProjectFiles(repo core.RepoInfo, localRoot string, workers int,
	onFiles func(total int),
	onStart func(workerID int, file string),
	onProgress func(workerID, done, total int, file string),
) error {
	srcRoot := l.root
	if _, err := os.Stat(srcRoot); os.IsNotExist(err) {
		return fmt.Errorf("local remote: project folder not found at %s", srcRoot)
	}
	skip := map[string]bool{
		l.objectRoot:     true,
		l.manifestRoot:   true,
		core.HistoryRoot: true,
	}

	var dirs []string
	var files []string
	err := filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, walkerr error) error {
		if walkerr != nil {
			return walkerr
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil || rel == "." {
			return nil
		}

		if d.IsDir() && filepath.Dir(rel) == "." && skip[rel] {
			return filepath.SkipDir
		}
		if d.IsDir() {
			dirs = append(dirs, filepath.ToSlash(rel))
			return nil
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return err
	}
	for _, rel := range dirs {
		dst, err := safeJoinUnder(localRoot, rel)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(dst, 0o750); err != nil {
			return err
		}
	}

	if onFiles != nil {
		onFiles(len(files))
	}

	return ConcurrentFiles(files, workers, onStart, onProgress, func(rel string) error {
		src, err := safeJoinUnder(srcRoot, rel)
		if err != nil {
			return err
		}
		dst, err := safeJoinUnder(localRoot, rel)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return err
		}
		return copyFile(src, dst)
	})
}

func (l *Local) DownloadFiles(repo core.RepoInfo, localRoot string, files []string, workers int) error {
	srcRoot := l.root
	if _, err := os.Stat(srcRoot); os.IsNotExist(err) {
		return fmt.Errorf("local remote: project folder not found at %s", srcRoot)
	}
	return ConcurrentFiles(files, workers, nil, nil, func(rel string) error {
		src, err := safeJoinUnder(srcRoot, rel)
		if err != nil {
			return err
		}
		dst, err := safeJoinUnder(localRoot, rel)
		if err != nil {
			return err
		}
		if _, err := os.Stat(src); os.IsNotExist(err) {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return err
		}
		return copyFile(src, dst)
	})
}

func (l *Local) historyPath(filename string) (string, error) {
	return safeJoinUnder(filepath.Join(l.root, core.HistoryRoot), filename)
}

func (l *Local) PutHistoryBundle(repo core.RepoInfo, branch, bundleName, localPath string) error {
	dst, err := l.historyPath(filepath.Join(branch, bundleName))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	return copyFile(localPath, dst)
}

func (l *Local) GetHistoryBundle(repo core.RepoInfo, branch, bundleName, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0o750); err != nil {
		return err
	}
	src, err := l.historyPath(filepath.Join(branch, bundleName))
	if err != nil {
		return err
	}
	return copyFile(src, localPath)
}

func (l *Local) HasHistoryBundle(repo core.RepoInfo, branch, bundleName string) (bool, error) {
	path, err := l.historyPath(filepath.Join(branch, bundleName))
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func (l *Local) PutRefsSnapshot(repo core.RepoInfo, snap core.RefsSnapshot) error {
	path, err := l.historyPath("refs.json")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func (l *Local) GetRefsSnapshot(repo core.RepoInfo) (core.RefsSnapshot, bool, error) {
	path, err := l.historyPath("refs.json")
	if err != nil {
		return core.RefsSnapshot{}, false, err
	}
	data, err := os.ReadFile(path) // #nosec G304 -- path is constrained by historyPath/safeJoinUnder.
	if os.IsNotExist(err) {
		return core.RefsSnapshot{}, false, nil
	}
	if err != nil {
		return core.RefsSnapshot{}, false, err
	}
	var snap core.RefsSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return core.RefsSnapshot{}, false, err
	}
	return snap, true, nil
}

func copyFile(src, dst string) error {
	if info, err := os.Lstat(src); err != nil {
		return err
	} else if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to copy symlink: %s", src)
	}
	in, err := os.Open(src) // #nosec G304 -- src is produced by safeJoinUnder or objectPath and symlinks are rejected with Lstat above.
	if err != nil {
		return err
	}
	defer func() {
		_ = in.Close()
	}()
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	out, err := os.CreateTemp(filepath.Dir(dst), "."+filepath.Base(dst)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := out.Name()
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func sameFileContent(a, b string) (bool, error) {
	aInfo, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	bInfo, err := os.Stat(b)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if aInfo.Size() != bInfo.Size() {
		return false, nil
	}
	aOID, aSize, err := storage.HashFile(a)
	if err != nil {
		return false, err
	}
	bOID, bSize, err := storage.HashFile(b)
	if err != nil {
		return false, err
	}
	return aOID == bOID && aSize == bSize, nil
}
