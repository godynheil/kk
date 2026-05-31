package remote

import (
	"fmt"

	"github.com/godynheil/kk/internal/core"
)

type Driver interface {
	Name() string
	Check() error
	PutObject(repo core.RepoInfo, p core.Pointer, localPath string) error
	GetObject(repo core.RepoInfo, p core.Pointer, localPath string) error
	HasObject(repo core.RepoInfo, p core.Pointer) (bool, error)
	ReadManifest(repo core.RepoInfo) (core.Manifest, error)
	WriteManifest(repo core.RepoInfo, manifest core.Manifest) error
	HasProject(repo core.RepoInfo) (bool, error)
	ReadRemoteRepoInfo(repo core.RepoInfo) (core.RepoInfo, bool, error)
	SyncProjectFiles(repo core.RepoInfo, localRoot string, files []string, workers int,
		onStart func(workerID int, file string),
		onProgress func(workerID, done, total int, file string)) (SyncStats, error)

	DownloadProjectFiles(repo core.RepoInfo, localRoot string, workers int,
		onFiles func(total int),
		onStart func(workerID int, file string),
		onProgress func(workerID, done, total int, file string)) error

	DownloadFiles(repo core.RepoInfo, localRoot string, files []string, workers int) error

	PutHistoryBundle(repo core.RepoInfo, branch, bundleName, localPath string) error

	GetHistoryBundle(repo core.RepoInfo, branch, bundleName, localPath string) error

	HasHistoryBundle(repo core.RepoInfo, branch, bundleName string) (bool, error)

	PutRefsSnapshot(repo core.RepoInfo, snap core.RefsSnapshot) error

	GetRefsSnapshot(repo core.RepoInfo) (core.RefsSnapshot, bool, error)
}

type SyncStats struct {
	Changed int
	Skipped int
}

func New(name string, cfg core.RemoteConfig) (Driver, error) {
	switch cfg.Type {
	case "local":
		return NewLocal(name, cfg)
	case "drive":
		return NewDrive(name, cfg)
	case "rclone":
		return NewRclone(name, cfg)
	case "ssh":
		return nil, fmt.Errorf("ssh driver is not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported remote type %q for %s", cfg.Type, name)
	}
}

func DefaultObjectRoot(v string) string {
	if v == "" {
		return "objects"
	}
	return v
}

func DefaultManifestRoot(v string) string {
	if v == "" {
		return "manifests"
	}
	return v
}

func DefaultVerifyMode(v string) string {
	if v == "" {
		return "local-hash"
	}
	return v
}
