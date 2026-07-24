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
	"context"
	"crypto/md5" // #nosec G501 -- Google Drive exposes MD5Checksum metadata; this is compatibility comparison, not security hashing.
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/gdrive"
	"github.com/godynheil/kk/internal/storage"
)

type Drive struct {
	name           string
	folderID       string
	authPath       string
	objectRoot     string
	manifestRoot   string
	verifyMode     string
	client         *gdrive.Client
	chunkSizeBytes int64
	folderCacheMu  sync.Mutex
	folderCache    map[string]string
}

var (
	driveObjectIndexTimeout = 15 * time.Second
	driveObjectIndexPoll    = 500 * time.Millisecond
)

func NewDrive(name string, cfg core.RemoteConfig) (*Drive, error) {
	if cfg.DriveFolderID == "" {
		return nil, fmt.Errorf("drive remote %s requires drive_folder_id", name)
	}
	if cfg.DriveAuthPath == "" {
		return nil, fmt.Errorf("drive remote %s requires drive_auth_path", name)
	}

	chunkSizeMB := 8
	if cfg.ChunkSizeMB > 0 {
		chunkSizeMB = cfg.ChunkSizeMB
	}
	if env := os.Getenv("KK_GDRIVE_CHUNK_SIZE"); env != "" {
		if mb, err := strconv.Atoi(env); err == nil && mb > 0 {
			chunkSizeMB = mb
		}
	}

	uploadTimeout := 300 * time.Second
	if cfg.UploadTimeoutSeconds > 0 {
		uploadTimeout = time.Duration(cfg.UploadTimeoutSeconds) * time.Second
	}
	if env := os.Getenv("KK_GDRIVE_UPLOAD_TIMEOUT"); env != "" {
		if seconds, err := strconv.Atoi(env); err == nil && seconds > 0 {
			uploadTimeout = time.Duration(seconds) * time.Second
		}
	}

	client := gdrive.NewClientWithTimeout(cfg.DriveAuthPath, uploadTimeout, cfg.DisableConnectionPool)

	return &Drive{
		name:           name,
		folderID:       cfg.DriveFolderID,
		authPath:       cfg.DriveAuthPath,
		objectRoot:     DefaultObjectRoot(cfg.ObjectRoot),
		manifestRoot:   DefaultManifestRoot(cfg.ManifestRoot),
		verifyMode:     DefaultVerifyMode(cfg.VerifyMode),
		client:         client,
		chunkSizeBytes: int64(chunkSizeMB) * 1024 * 1024,
		folderCache:    map[string]string{"": cfg.DriveFolderID},
	}, nil
}

func (d *Drive) Name() string { return d.name }

func (d *Drive) Check() error {
	ctx := context.Background()
	return d.client.CheckFolder(ctx, d.folderID)
}

func (d *Drive) PutObject(repo core.RepoInfo, p core.Pointer, localPath string) error {
	ctx := context.Background()
	if ok, _ := d.HasObject(repo, p); ok {
		return nil
	}
	parentID, err := d.ensureFolderChain(ctx, d.objectRoot, p.OID[:2], p.OID[2:4])
	if err != nil {
		return err
	}

	var uploaded gdrive.File
	info, err := os.Stat(localPath)
	if err != nil {
		return err
	}

	if info.Size() > d.chunkSizeBytes {
		uploaded, err = d.client.UploadFileChunked(ctx, parentID, p.OID, localPath, "application/octet-stream", d.chunkSizeBytes)
	} else {
		uploaded, err = d.client.UploadFile(ctx, parentID, p.OID, localPath, "application/octet-stream")
	}
	if err != nil {
		return err
	}
	if d.verifyMode == "download" {
		tmpDir, err := os.MkdirTemp("", "kk-drive-verify-*")
		if err != nil {
			return err
		}
		defer func() {
			_ = os.RemoveAll(tmpDir)
		}()
		tmpPath := filepath.Join(tmpDir, p.OID)
		if err := d.client.DownloadFile(ctx, uploaded.ID, tmpPath); err != nil {
			_ = d.client.DeleteFile(ctx, uploaded.ID)
			return err
		}
		actualOID, actualSize, err := storage.HashFile(tmpPath)
		if err != nil {
			_ = d.client.DeleteFile(ctx, uploaded.ID)
			return err
		}
		if actualOID != p.OID || actualSize != p.Size {
			_ = d.client.DeleteFile(ctx, uploaded.ID)
			return fmt.Errorf("Drive remote verification failed for %s", p.OID)
		}
	}
	if _, ok, err := d.waitForObjectIndexed(ctx, repo, p.OID); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("Drive object not visible after upload: %s", p.OID)
	}
	return nil
}

func (d *Drive) GetObject(repo core.RepoInfo, p core.Pointer, localPath string) error {
	ctx := context.Background()
	file, ok, err := d.findObject(ctx, repo, p.OID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("Drive object not found: %s", p.OID)
	}
	if err := d.client.DownloadFile(ctx, file.ID, localPath); err != nil {
		return err
	}
	actualOID, actualSize, err := storage.HashFile(localPath)
	if err != nil {
		return err
	}
	if actualOID != p.OID || actualSize != p.Size {
		_ = os.Remove(localPath)
		return fmt.Errorf("download verification failed for %s", p.OID)
	}
	return nil
}

func (d *Drive) HasObject(repo core.RepoInfo, p core.Pointer) (bool, error) {
	ctx := context.Background()
	file, ok, err := d.findObject(ctx, repo, p.OID)
	if err != nil || !ok {
		return false, err
	}
	if d.verifyMode != "download" {
		return true, nil
	}
	tmpDir, err := os.MkdirTemp("", "kk-drive-has-*")
	if err != nil {
		return false, err
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()
	tmpPath := filepath.Join(tmpDir, p.OID)
	if err := d.client.DownloadFile(ctx, file.ID, tmpPath); err != nil {
		return false, err
	}
	actualOID, actualSize, err := storage.HashFile(tmpPath)
	if err != nil {
		return false, err
	}
	return actualOID == p.OID && actualSize == p.Size, nil
}

func (d *Drive) ReadManifest(repo core.RepoInfo) (core.Manifest, error) {
	ctx := context.Background()
	parentID, err := d.ensureFolderChain(ctx, d.manifestRoot)
	if err != nil {
		return core.Manifest{}, err
	}
	file, ok, err := d.client.FindChild(ctx, parentID, repo.RepoID+".json", false)
	if err != nil {
		return core.Manifest{}, err
	}
	if !ok {
		return EmptyManifest(repo), nil
	}
	tmp, err := os.CreateTemp("", "kk-drive-manifest-*.json")
	if err != nil {
		return core.Manifest{}, err
	}
	path := tmp.Name()
	_ = tmp.Close()
	defer func() {
		_ = os.Remove(path)
	}()
	if err := d.client.DownloadFile(ctx, file.ID, path); err != nil {
		return core.Manifest{}, err
	}
	data, err := os.ReadFile(path) // #nosec G304 -- path is a temp file created by this function for a downloaded manifest.
	if err != nil {
		return core.Manifest{}, err
	}
	var manifest core.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return core.Manifest{}, err
	}
	return manifest, nil
}

func (d *Drive) WriteManifest(repo core.RepoInfo, manifest core.Manifest) error {
	sort.Slice(manifest.Objects, func(i, j int) bool { return manifest.Objects[i].OID < manifest.Objects[j].OID })
	ctx := context.Background()
	parentID, err := d.ensureFolderChain(ctx, d.manifestRoot)
	if err != nil {
		return err
	}
	if existing, ok, err := d.client.FindChild(ctx, parentID, repo.RepoID+".json", false); err == nil && ok {
		if err := d.client.DeleteFile(ctx, existing.ID); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	tmp, err := os.CreateTemp("", "kk-drive-manifest-write-*.json")
	if err != nil {
		return err
	}
	path := tmp.Name()
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(path)
		return err
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		_ = os.Remove(path)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	defer func() {
		_ = os.Remove(path)
	}()
	_, err = d.client.UploadFile(ctx, parentID, repo.RepoID+".json", path, "application/json")
	return err
}

func (d *Drive) HasProject(repo core.RepoInfo) (bool, error) {
	return true, d.Check()
}

func (d *Drive) ReadRemoteRepoInfo(repo core.RepoInfo) (core.RepoInfo, bool, error) {
	ctx := context.Background()
	deadline := time.Now().Add(15 * time.Second)
	for {
		info, ok, err := d.readRemoteRepoInfoOnce(ctx)
		if err != nil || ok || time.Now().After(deadline) {
			return info, ok, err
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (d *Drive) readRemoteRepoInfoOnce(ctx context.Context) (core.RepoInfo, bool, error) {
	parentID, ok, err := d.findFolderChain(ctx, d.folderID, ".kk")
	if err != nil {
		return core.RepoInfo{}, false, err
	}
	if !ok {
		return core.RepoInfo{}, false, nil
	}
	file, ok, err := d.client.FindChild(ctx, parentID, "repo.json", false)
	if err != nil || !ok {
		return core.RepoInfo{}, false, err
	}
	tmp, err := os.CreateTemp("", "kk-drive-repo-*.json")
	if err != nil {
		return core.RepoInfo{}, false, err
	}
	path := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(path) }()
	if err := d.client.DownloadFile(ctx, file.ID, path); err != nil {
		return core.RepoInfo{}, false, err
	}
	data, err := os.ReadFile(path) // #nosec G304 -- path is a temp file created by this function for downloaded repo metadata.
	if err != nil {
		return core.RepoInfo{}, false, err
	}
	var info core.RepoInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return core.RepoInfo{}, false, err
	}
	return info, true, nil
}

func (d *Drive) SyncProjectFiles(repo core.RepoInfo, localRoot string, files []string, workers int,
	onStart func(workerID int, file string),
	onProgress func(workerID, done, total int, file string),
) (SyncStats, error) {
	ctx := context.Background()
	cache := newDriveSyncCache(d.folderID)
	var stats syncStatsCounter

	err := ConcurrentFiles(files, workers, onStart, onProgress, func(rel string) error {
		parts := strings.Split(rel, "/")
		fileName := parts[len(parts)-1]
		dirParts := parts[:len(parts)-1]

		parentID, err := d.ensureFolderChainCached(ctx, cache, dirParts...)
		if err != nil {
			return fmt.Errorf("ensuring folder for %s: %w", rel, err)
		}
		localPath, err := safeJoinUnder(localRoot, rel)
		if err != nil {
			return err
		}
		changed, err := d.uploadOrReplaceCached(ctx, cache, parentID, fileName, localPath)
		if err != nil {
			return err
		}
		if changed {
			stats.addChanged()
		} else {
			stats.addSkipped()
		}
		return nil
	})
	return stats.snapshot(), err
}

type syncStatsCounter struct {
	mu      sync.Mutex
	changed int
	skipped int
}

func (s *syncStatsCounter) addChanged() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.changed++
}

func (s *syncStatsCounter) addSkipped() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.skipped++
}

func (s *syncStatsCounter) snapshot() SyncStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return SyncStats{Changed: s.changed, Skipped: s.skipped}
}

type folderPromise struct {
	ready chan struct{}
	id    string
	err   error
}

type childPromise struct {
	ready    chan struct{}
	children map[string]gdrive.File
	err      error
}

type driveSyncCache struct {
	mu             sync.Mutex
	folders        map[string]string
	folderPromises map[string]*folderPromise
	childMaps      map[string]map[string]gdrive.File
	childPromises  map[string]*childPromise
}

func newDriveSyncCache(rootID string) *driveSyncCache {
	return &driveSyncCache{
		folders: map[string]string{
			"": rootID,
		},
		folderPromises: map[string]*folderPromise{},
		childMaps:      map[string]map[string]gdrive.File{},
		childPromises:  map[string]*childPromise{},
	}
}

func (d *Drive) ensureFolderChainCached(ctx context.Context, cache *driveSyncCache, names ...string) (string, error) {
	parentID := d.folderID
	var prefix []string
	for _, name := range names {
		prefix = append(prefix, name)
		key := strings.Join(prefix, "/")

		cache.mu.Lock()
		if id := cache.folders[key]; id != "" {
			parentID = id
			cache.mu.Unlock()
			continue
		}

		p := cache.folderPromises[key]
		if p != nil {
			cache.mu.Unlock()
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-p.ready:
			}
			if p.err != nil {
				return "", p.err
			}
			parentID = p.id
			continue
		}

		p = &folderPromise{
			ready: make(chan struct{}),
		}
		cache.folderPromises[key] = p
		cache.mu.Unlock()

		folder, err := d.client.EnsureFolder(ctx, parentID, name)

		cache.mu.Lock()
		if err != nil {
			p.err = err
			close(p.ready)
			delete(cache.folderPromises, key)
			cache.mu.Unlock()
			return "", err
		}

		p.id = folder.ID
		close(p.ready)

		cache.folders[key] = folder.ID
		parentID = folder.ID
		cache.mu.Unlock()
	}
	return parentID, nil
}

func (d *Drive) uploadOrReplaceCached(ctx context.Context, cache *driveSyncCache, parentID, name, localPath string) (bool, error) {
	existing, ok, err := d.cachedChild(ctx, cache, parentID, name)
	if err != nil {
		return false, err
	}
	if ok {
		same, err := sameDriveFileContent(existing, localPath)
		if err != nil {
			return false, err
		}
		if same {
			return false, nil
		}
		if err := d.client.DeleteFile(ctx, existing.ID); err != nil {
			return false, err
		}
	}
	uploaded, err := d.client.UploadFile(ctx, parentID, name, localPath, "")
	if err != nil {
		return false, err
	}
	d.setCachedChild(cache, parentID, name, uploaded)
	return true, nil
}

func (d *Drive) cachedChild(ctx context.Context, cache *driveSyncCache, parentID, name string) (gdrive.File, bool, error) {
	cache.mu.Lock()
	children := cache.childMaps[parentID]
	if children != nil {
		file, ok := children[name]
		cache.mu.Unlock()
		return file, ok, nil
	}

	p := cache.childPromises[parentID]
	if p != nil {
		cache.mu.Unlock()
		select {
		case <-ctx.Done():
			return gdrive.File{}, false, ctx.Err()
		case <-p.ready:
		}
		if p.err != nil {
			return gdrive.File{}, false, p.err
		}
		cache.mu.Lock()
		children = cache.childMaps[parentID]
		file, ok := children[name]
		cache.mu.Unlock()
		return file, ok, nil
	}

	p = &childPromise{
		ready: make(chan struct{}),
	}
	cache.childPromises[parentID] = p
	cache.mu.Unlock()

	files, err := d.client.ListChildren(ctx, parentID)

	cache.mu.Lock()
	if err != nil {
		p.err = err
		close(p.ready)
		delete(cache.childPromises, parentID)
		cache.mu.Unlock()
		return gdrive.File{}, false, err
	}

	children = make(map[string]gdrive.File, len(files))
	for _, file := range files {
		children[file.Name] = file
	}
	p.children = children
	close(p.ready)

	cache.childMaps[parentID] = children
	file, ok := children[name]
	cache.mu.Unlock()
	return file, ok, nil
}

func (d *Drive) setCachedChild(cache *driveSyncCache, parentID, name string, file gdrive.File) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	children := cache.childMaps[parentID]
	if children == nil {
		children = map[string]gdrive.File{}
		cache.childMaps[parentID] = children
	}
	children[name] = file
}

func sameDriveFileContent(file gdrive.File, localPath string) (bool, error) {
	if file.MD5Checksum == "" {
		return false, nil
	}
	sum, err := md5File(localPath)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(file.MD5Checksum, sum), nil
}

func md5File(path string) (string, error) {
	f, err := os.Open(path) // #nosec G304 -- caller supplies app-managed local file paths for Drive checksum comparison.
	if err != nil {
		return "", err
	}
	defer func() {
		_ = f.Close()
	}()
	h := md5.New() // #nosec G401 -- Google Drive's MD5Checksum field requires MD5 for provider metadata comparison.
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (d *Drive) DownloadProjectFiles(repo core.RepoInfo, localRoot string, workers int,
	onFiles func(total int),
	onStart func(workerID int, file string),
	onProgress func(workerID, done, total int, file string),
) error {
	ctx := context.Background()
	skip := map[string]bool{
		d.objectRoot:     true,
		d.manifestRoot:   true,
		core.HistoryRoot: true,
	}

	entries, err := d.listDriveFilesRecursive(ctx, d.folderID, "", skip)
	if err != nil {
		return err
	}

	if onFiles != nil {
		onFiles(len(entries))
	}

	pathToID := make(map[string]string, len(entries))
	filePaths := make([]string, 0, len(entries))
	for _, e := range entries {
		pathToID[e.relPath] = e.fileID
		filePaths = append(filePaths, e.relPath)
	}

	return ConcurrentFiles(filePaths, workers, onStart, onProgress, func(rel string) error {
		fileID := pathToID[rel]
		dst, err := safeJoinUnder(localRoot, rel)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return err
		}
		return d.client.DownloadFile(ctx, fileID, dst)
	})
}

func (d *Drive) DownloadFiles(repo core.RepoInfo, localRoot string, files []string, workers int) error {
	ctx := context.Background()
	cache := newDriveSyncCache(d.folderID)
	return ConcurrentFiles(files, workers, nil, nil, func(rel string) error {
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) == 0 {
			return nil
		}
		dirParts := parts[:len(parts)-1]
		filename := parts[len(parts)-1]
		parentID, err := d.ensureFolderChainCached(ctx, cache, dirParts...)
		if err != nil {
			return nil
		}
		f, ok, err := d.cachedChild(ctx, cache, parentID, filename)
		if err != nil || !ok {
			return err
		}
		dst, err := safeJoinUnder(localRoot, rel)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return err
		}
		return d.client.DownloadFile(ctx, f.ID, dst)
	})
}

type driveFileEntry struct {
	relPath string
	fileID  string
}

func (d *Drive) listDriveFilesRecursive(ctx context.Context, folderID, prefix string, skipNames map[string]bool) ([]driveFileEntry, error) {
	children, err := d.client.ListChildren(ctx, folderID)
	if err != nil {
		return nil, err
	}
	var entries []driveFileEntry
	for _, child := range children {
		if skipNames != nil && skipNames[child.Name] {
			continue
		}
		rel := child.Name
		if _, err := safeRemoteRel(child.Name); err != nil {
			return nil, fmt.Errorf("unsafe drive entry name %q: %w", child.Name, err)
		}
		if strings.ContainsAny(child.Name, "/\\") {
			return nil, fmt.Errorf("unsafe drive entry name %q: contains path separators", child.Name)
		}
		if prefix != "" {
			rel = prefix + "/" + child.Name
		}
		if child.MimeType == "application/vnd.google-apps.folder" {
			sub, err := d.listDriveFilesRecursive(ctx, child.ID, rel, nil)
			if err != nil {
				return nil, err
			}
			entries = append(entries, sub...)
		} else {
			entries = append(entries, driveFileEntry{relPath: rel, fileID: child.ID})
		}
	}
	return entries, nil
}

func (d *Drive) PutHistoryBundle(repo core.RepoInfo, branch, bundleName, localPath string) error {
	ctx := context.Background()
	parentID, err := d.ensureFolderChain(ctx, core.HistoryRoot, branch)
	if err != nil {
		return err
	}

	if existing, ok, err := d.client.FindChild(ctx, parentID, bundleName, false); err == nil && ok {
		_ = d.client.DeleteFile(ctx, existing.ID)
	}
	_, err = d.client.UploadFile(ctx, parentID, bundleName, localPath, "application/octet-stream")
	return err
}

func (d *Drive) GetHistoryBundle(repo core.RepoInfo, branch, bundleName, localPath string) error {
	ctx := context.Background()
	parentID, ok, err := d.findFolderChain(ctx, d.folderID, core.HistoryRoot, branch)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("drive: history folder not found on remote for branch %s", branch)
	}
	file, ok, err := d.client.FindChild(ctx, parentID, bundleName, false)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("drive: bundle %s not found on remote", bundleName)
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0o750); err != nil {
		return err
	}
	return d.client.DownloadFile(ctx, file.ID, localPath)
}

func (d *Drive) HasHistoryBundle(repo core.RepoInfo, branch, bundleName string) (bool, error) {
	ctx := context.Background()
	parentID, ok, err := d.findFolderChain(ctx, d.folderID, core.HistoryRoot, branch)
	if err != nil || !ok {
		return false, err
	}
	_, ok, err = d.client.FindChild(ctx, parentID, bundleName, false)
	return ok, err
}

func (d *Drive) PutRefsSnapshot(repo core.RepoInfo, snap core.RefsSnapshot) error {
	ctx := context.Background()
	parentID, err := d.ensureFolderChain(ctx, core.HistoryRoot)
	if err != nil {
		return err
	}

	if existing, ok, err := d.client.FindChild(ctx, parentID, "refs.json", false); err == nil && ok {
		_ = d.client.DeleteFile(ctx, existing.ID)
	}
	tmp, err := os.CreateTemp("", "kk-drive-refs-*.json")
	if err != nil {
		return err
	}
	path := tmp.Name()
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(path)
		return err
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		_ = os.Remove(path)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	defer func() { _ = os.Remove(path) }()
	_, err = d.client.UploadFile(ctx, parentID, "refs.json", path, "application/json")
	return err
}

func (d *Drive) GetRefsSnapshot(repo core.RepoInfo) (core.RefsSnapshot, bool, error) {
	ctx := context.Background()
	parentID, ok, err := d.findFolderChain(ctx, d.folderID, core.HistoryRoot)
	if err != nil {
		return core.RefsSnapshot{}, false, err
	}
	if !ok {
		return core.RefsSnapshot{}, false, nil
	}
	file, ok, err := d.client.FindChild(ctx, parentID, "refs.json", false)
	if err != nil || !ok {
		return core.RefsSnapshot{}, false, err
	}
	tmp, err := os.CreateTemp("", "kk-drive-refs-fetch-*.json")
	if err != nil {
		return core.RefsSnapshot{}, false, err
	}
	path := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(path) }()
	if err := d.client.DownloadFile(ctx, file.ID, path); err != nil {
		return core.RefsSnapshot{}, false, err
	}
	data, err := os.ReadFile(path) // #nosec G304 -- path is a temp file created by this function for downloaded refs metadata.
	if err != nil {
		return core.RefsSnapshot{}, false, err
	}
	var snap core.RefsSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return core.RefsSnapshot{}, false, err
	}
	return snap, true, nil
}

func (d *Drive) ensureFolderChain(ctx context.Context, names ...string) (string, error) {
	d.folderCacheMu.Lock()
	if d.folderCache == nil {
		d.folderCache = map[string]string{"": d.folderID}
	}
	parentID := d.folderID
	var prefix []string
	for _, name := range names {
		prefix = append(prefix, name)
		key := strings.Join(prefix, "/")
		if id, ok := d.folderCache[key]; ok {
			parentID = id
			continue
		}
		d.folderCacheMu.Unlock()
		folder, err := d.client.EnsureFolder(ctx, parentID, name)
		if err != nil {
			return "", err
		}
		d.folderCacheMu.Lock()
		d.folderCache[key] = folder.ID
		parentID = folder.ID
	}
	d.folderCacheMu.Unlock()
	return parentID, nil
}

func (d *Drive) findFolderChain(ctx context.Context, parentID string, names ...string) (string, bool, error) {
	for _, name := range names {
		folder, ok, err := d.client.FindChild(ctx, parentID, name, true)
		if err != nil {
			return "", false, err
		}
		if !ok {
			return "", false, nil
		}
		parentID = folder.ID
	}
	return parentID, true, nil
}

func (d *Drive) findObject(ctx context.Context, repo core.RepoInfo, oid string) (gdrive.File, bool, error) {
	parentID, err := d.ensureFolderChain(ctx, d.objectRoot, oid[:2], oid[2:4])
	if err != nil {
		return gdrive.File{}, false, err
	}
	return d.client.FindChild(ctx, parentID, oid, false)
}

func (d *Drive) waitForObjectIndexed(ctx context.Context, repo core.RepoInfo, oid string) (gdrive.File, bool, error) {
	return waitForDriveObject(ctx, driveObjectIndexTimeout, driveObjectIndexPoll, func(ctx context.Context) (gdrive.File, bool, error) {
		return d.findObject(ctx, repo, oid)
	})
}

func waitForDriveObject(
	ctx context.Context,
	timeout time.Duration,
	poll time.Duration,
	lookup func(context.Context) (gdrive.File, bool, error),
) (gdrive.File, bool, error) {
	deadline := time.Now().Add(timeout)
	for {
		file, ok, err := lookup(ctx)
		if err != nil || ok || time.Now().After(deadline) {
			return file, ok, err
		}
		timer := time.NewTimer(poll)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return gdrive.File{}, false, ctx.Err()
		case <-timer.C:
		}
	}
}
