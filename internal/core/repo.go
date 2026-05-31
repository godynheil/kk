package core

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func NewRepoInfo(root string) (RepoInfo, error) {
	id, err := newUUID()
	if err != nil {
		return RepoInfo{}, err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	return RepoInfo{RepoID: id, Name: filepath.Base(abs), CreatedAt: time.Now().UTC()}, nil
}

func ReadRepoInfo(root string) (RepoInfo, error) {
	var info RepoInfo
	data, err := os.ReadFile(filepath.Join(root, RepoFile)) // #nosec G304 -- repo metadata is read from the caller's repository root.
	if err != nil {
		return info, err
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return info, err
	}
	if info.RepoID == "" {
		return info, fmt.Errorf("repo_id is missing in %s", RepoFile)
	}
	return info, nil
}

func WriteRepoInfo(root string, info RepoInfo) error {
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(root, RepoFile), data, 0o600)
}

func newUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32]), nil
}

func IsInsideKK(path string) bool {
	clean := filepath.ToSlash(filepath.Clean(path))
	return clean == KKDir || strings.HasPrefix(clean, KKDir+"/")
}
