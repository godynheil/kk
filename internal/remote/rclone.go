package remote

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/storage"
)

type Rclone struct {
	name         string
	binary       string
	remote       string
	objectRoot   string
	manifestRoot string
	verifyMode   string
	runCmd       func(args ...string) error
	transfers    int
	bufferSizeMB int
}

func NewRclone(name string, cfg core.RemoteConfig) (*Rclone, error) {
	if cfg.Remote == "" {
		return nil, fmt.Errorf("rclone remote %s requires remote target", name)
	}
	binary, err := ResolveRcloneBinary(cfg.Binary)
	if err != nil {
		return nil, err
	}

	transfers := 4
	if cfg.RcloneTransfers > 0 {
		transfers = cfg.RcloneTransfers
	}
	if env := os.Getenv("KK_RCLONE_TRANSFERS"); env != "" {
		if n, err := strconv.Atoi(env); err == nil && n > 0 {
			transfers = n
		}
	}

	bufferSizeMB := 16
	if cfg.BufferSizeMB > 0 {
		bufferSizeMB = cfg.BufferSizeMB
	}
	if env := os.Getenv("KK_RCLONE_BUFFER_SIZE"); env != "" {
		if mb, err := strconv.Atoi(env); err == nil && mb > 0 {
			bufferSizeMB = mb
		}
	}

	return &Rclone{
		name:         name,
		binary:       binary,
		remote:       strings.TrimRight(cfg.Remote, "/"),
		objectRoot:   DefaultObjectRoot(cfg.ObjectRoot),
		manifestRoot: DefaultManifestRoot(cfg.ManifestRoot),
		verifyMode:   DefaultVerifyMode(cfg.VerifyMode),
		transfers:    transfers,
		bufferSizeMB: bufferSizeMB,
	}, nil
}

func ResolveRcloneBinary(value string) (string, error) {
	if value == "" || value == "rclone" {
		path, err := exec.LookPath("rclone")
		if err != nil {
			return "", fmt.Errorf("rclone not found in PATH")
		}
		return path, nil
	}
	if !filepath.IsAbs(value) {
		return "", fmt.Errorf("custom rclone path must be absolute: %s", value)
	}
	info, err := os.Stat(value)
	if err != nil {
		return "", fmt.Errorf("rclone binary not found: %s", value)
	}
	if info.IsDir() {
		return "", fmt.Errorf("rclone path is a directory: %s", value)
	}
	return value, nil
}

func (r *Rclone) Name() string { return r.name }

func (r *Rclone) Check() error {
	return r.run("mkdir", r.remote)
}

func (r *Rclone) objectRemotePath(oid string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", r.remote, r.objectRoot, oid[:2], oid[2:4], oid)
}

func (r *Rclone) manifestRemotePath(repoID string) string {
	return fmt.Sprintf("%s/%s/%s.json", r.remote, r.manifestRoot, repoID)
}

func (r *Rclone) PutObject(repo core.RepoInfo, p core.Pointer, localPath string) error {
	if ok, _ := r.HasObject(repo, p); ok {
		return nil
	}
	remoteFinal := r.objectRemotePath(p.OID)
	remoteTmp := remoteFinal + ".tmp"
	if err := r.run("copyto", localPath, remoteTmp); err != nil {
		return err
	}
	if r.verifyMode == "download" {
		if err := r.verifyByDownload(p, remoteTmp); err != nil {
			_ = r.run("deletefile", remoteTmp)
			return err
		}
	}
	if err := r.run("moveto", remoteTmp, remoteFinal); err != nil {
		_ = r.run("deletefile", remoteTmp)
		return err
	}
	return nil
}

func (r *Rclone) GetObject(repo core.RepoInfo, p core.Pointer, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0o750); err != nil {
		return err
	}
	tmp := localPath + ".tmp"
	if err := r.run("copyto", r.objectRemotePath(p.OID), tmp); err != nil {
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

func (r *Rclone) HasObject(repo core.RepoInfo, p core.Pointer) (bool, error) {
	remotePath := r.objectRemotePath(p.OID)
	out, err := r.runCapture("lsf", remotePath)
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(out) == "" {
		return false, nil
	}
	if r.verifyMode == "download" {
		if err := r.verifyByDownload(p, remotePath); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (r *Rclone) ReadManifest(repo core.RepoInfo) (core.Manifest, error) {
	remotePath := r.manifestRemotePath(repo.RepoID)
	out, err := r.runCapture("lsf", remotePath)
	if err != nil {
		return EmptyManifest(repo), nil
	}
	if strings.TrimSpace(out) == "" {
		return EmptyManifest(repo), nil
	}

	tmp, err := os.CreateTemp("", "kk-manifest-*.json")
	if err != nil {
		return core.Manifest{}, err
	}
	path := tmp.Name()
	_ = tmp.Close()
	defer func() {
		_ = os.Remove(path)
	}()

	var copyErr error
	for attempt := 1; attempt <= 5; attempt++ {
		copyErr = r.run("copyto", remotePath, path)
		if copyErr == nil {
			break
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	if copyErr != nil {
		return EmptyManifest(repo), nil
	}

	data, err := os.ReadFile(path) // #nosec G304 -- path is a temp file created by this function for a downloaded manifest.
	if err != nil {
		return core.Manifest{}, err
	}
	var m core.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return core.Manifest{}, err
	}
	return m, nil
}

func (r *Rclone) WriteManifest(repo core.RepoInfo, manifest core.Manifest) error {
	sort.Slice(manifest.Objects, func(i, j int) bool { return manifest.Objects[i].OID < manifest.Objects[j].OID })
	tmp, err := os.CreateTemp("", "kk-manifest-write-*.json")
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
	manifestPath := r.manifestRemotePath(repo.RepoID)
	remoteTmp := manifestPath + ".tmp"
	if err := r.run("copyto", path, remoteTmp); err != nil {
		return err
	}
	return r.run("moveto", remoteTmp, manifestPath)
}

func (r *Rclone) HasProject(repo core.RepoInfo) (bool, error) {
	if err := r.Check(); err != nil {
		return false, nil
	}
	return true, nil
}

func (r *Rclone) ReadRemoteRepoInfo(repo core.RepoInfo) (core.RepoInfo, bool, error) {
	remotePath := fmt.Sprintf("%s/.kk/repo.json", r.remote)
	out, err := r.runCapture("lsf", remotePath)
	if err != nil {
		return core.RepoInfo{}, false, nil
	}
	if strings.TrimSpace(out) == "" {
		return core.RepoInfo{}, false, nil
	}

	tmp, err := os.CreateTemp("", "kk-remote-repo-*.json")
	if err != nil {
		return core.RepoInfo{}, false, err
	}
	path := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(path) }()

	var copyErr error
	for attempt := 1; attempt <= 5; attempt++ {
		copyErr = r.run("copyto", remotePath, path)
		if copyErr == nil {
			break
		}
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	if copyErr != nil {
		return core.RepoInfo{}, false, nil
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

func (r *Rclone) SyncProjectFiles(repo core.RepoInfo, localRoot string, files []string, workers int,
	onStart func(workerID int, file string),
	onProgress func(workerID, done, total int, file string),
) (SyncStats, error) {
	err := ConcurrentFiles(files, workers, onStart, onProgress, func(rel string) error {
		clean, err := safeRemoteRel(rel)
		if err != nil {
			return err
		}
		src, err := safeJoinUnder(localRoot, clean)
		if err != nil {
			return err
		}
		dst := fmt.Sprintf("%s/%s", r.remote, filepath.ToSlash(clean))
		return r.run("copyto", src, dst)
	})
	return SyncStats{Changed: len(files)}, err
}

func (r *Rclone) DownloadProjectFiles(repo core.RepoInfo, localRoot string, workers int,
	onFiles func(total int),
	onStart func(workerID int, file string),
	onProgress func(workerID, done, total int, file string),
) error {

	files, err := r.listProjectFiles()
	if err != nil {

		if onFiles != nil {
			onFiles(0)
		}
		return r.run("copy",
			"--exclude", r.objectRoot+"/**",
			"--exclude", r.manifestRoot+"/**",
			"--exclude", core.HistoryRoot+"/**",
			r.remote, localRoot,
		)
	}

	if onFiles != nil {
		onFiles(len(files))
	}

	return ConcurrentFiles(files, workers, onStart, onProgress, func(rel string) error {
		clean, err := safeRemoteRel(rel)
		if err != nil {
			return err
		}
		src := r.remote + "/" + filepath.ToSlash(clean)
		dst, err := safeJoinUnder(localRoot, clean)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return err
		}
		return r.run("copyto", src, dst)
	})
}

func (r *Rclone) DownloadFiles(repo core.RepoInfo, localRoot string, files []string, workers int) error {
	return ConcurrentFiles(files, workers, nil, nil, func(rel string) error {
		clean, err := safeRemoteRel(rel)
		if err != nil {
			return err
		}
		src := r.remote + "/" + filepath.ToSlash(clean)
		dst, err := safeJoinUnder(localRoot, clean)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return err
		}
		_ = r.run("copyto", src, dst)
		return nil
	})
}

func (r *Rclone) listProjectFiles() ([]string, error) {
	cmd := exec.Command(r.binary, "lsf", "--recursive", // #nosec G204 -- r.binary is resolved to rclone by ResolveRcloneBinary; arguments are not shell-expanded.
		"--exclude", r.objectRoot+"/**",
		"--exclude", r.manifestRoot+"/**",
		"--exclude", core.HistoryRoot+"/**",
		r.remote)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			clean, err := safeRemoteRel(line)
			if err != nil {
				return nil, err
			}
			files = append(files, filepath.ToSlash(clean))
		}
	}
	return files, nil
}

func (r *Rclone) historyRemotePath(filename string) string {
	return fmt.Sprintf("%s/%s/%s", r.remote, core.HistoryRoot, filename)
}

func (r *Rclone) PutHistoryBundle(repo core.RepoInfo, branch, bundleName, localPath string) error {
	return r.run("copyto", localPath, r.historyRemotePath(branch+"/"+bundleName))
}

func (r *Rclone) GetHistoryBundle(repo core.RepoInfo, branch, bundleName, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0o750); err != nil {
		return err
	}
	remotePath := r.historyRemotePath(branch + "/" + bundleName)
	var lastErr error
	for attempt := 1; attempt <= 5; attempt++ {
		lastErr = r.run("copyto", remotePath, localPath)
		if lastErr == nil {
			return nil
		}
		fmt.Printf("kk: [%s] warning: bundle %s download attempt %d failed: %v, retrying...\n", r.name, bundleName, attempt, lastErr)
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	return fmt.Errorf("failed to download bundle after 5 attempts: %w", lastErr)
}

func (r *Rclone) HasHistoryBundle(repo core.RepoInfo, branch, bundleName string) (bool, error) {
	remotePath := r.historyRemotePath(branch + "/" + bundleName)
	out, err := r.runCapture("lsf", remotePath)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func (r *Rclone) PutRefsSnapshot(repo core.RepoInfo, snap core.RefsSnapshot) error {
	tmp, err := os.CreateTemp("", "kk-history-refs-*.json")
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
	return r.run("copyto", path, r.historyRemotePath("refs.json"))
}

func (r *Rclone) GetRefsSnapshot(repo core.RepoInfo) (core.RefsSnapshot, bool, error) {
	remotePath := r.historyRemotePath("refs.json")
	out, err := r.runCapture("lsf", remotePath)
	if err != nil {
		return core.RefsSnapshot{}, false, nil
	}
	if strings.TrimSpace(out) == "" {
		return core.RefsSnapshot{}, false, nil
	}

	tmp, err := os.CreateTemp("", "kk-history-refs-fetch-*.json")
	if err != nil {
		return core.RefsSnapshot{}, false, err
	}
	path := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(path) }()

	var copyErr error
	for attempt := 1; attempt <= 5; attempt++ {
		copyErr = r.run("copyto", remotePath, path)
		if copyErr == nil {
			break
		}
		fmt.Printf("kk: [%s] warning: refs.json download attempt %d failed: %v, retrying...\n", r.name, attempt, copyErr)
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	if copyErr != nil {
		return core.RefsSnapshot{}, false, fmt.Errorf("downloading refs.json after lsf check: %w", copyErr)
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

func (r *Rclone) verifyByDownload(p core.Pointer, remotePath string) error {
	dir, err := os.MkdirTemp("", "kk-rclone-verify-*")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()
	local := filepath.Join(dir, p.OID)
	if err := r.run("copyto", remotePath, local); err != nil {
		return err
	}
	actualOID, actualSize, err := storage.HashFile(local)
	if err != nil {
		return err
	}
	if actualOID != p.OID || actualSize != p.Size {
		return fmt.Errorf("rclone remote verification failed for %s", p.OID)
	}
	return nil
}

func (r *Rclone) buildArgs() []string {
	args := []string{}
	if r.transfers != 4 {
		args = append(args, "--transfers", strconv.Itoa(r.transfers))
	}
	if r.bufferSizeMB != 16 {
		args = append(args, "--buffer-size", fmt.Sprintf("%dM", r.bufferSizeMB))
	}
	return args
}

func (r *Rclone) run(args ...string) error {
	if r.runCmd != nil {
		return r.runCmd(args...)
	}
	fullArgs := append(r.buildArgs(), args...)
	cmd := exec.Command(r.binary, fullArgs...) // #nosec G204 -- r.binary is resolved to rclone by ResolveRcloneBinary; arguments are not shell-expanded.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func (r *Rclone) runCapture(args ...string) (string, error) {
	if r.runCmd != nil {
		err := r.runCmd(args...)
		if err != nil {
			if os.IsNotExist(err) {
				return "", nil
			}
			return "", err
		}
		if len(args) > 0 && args[0] == "lsf" {
			return filepath.Base(args[len(args)-1]), nil
		}
		return "", nil
	}
	fullArgs := append(r.buildArgs(), args...)
	cmd := exec.Command(r.binary, fullArgs...) // #nosec G204 -- r.binary is resolved to rclone by ResolveRcloneBinary; arguments are not shell-expanded.
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
