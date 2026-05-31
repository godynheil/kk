package remote

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/storage"
)

func TestRcloneHasObjectVerifiesDownloadedContent(t *testing.T) {
	root := t.TempDir()
	remoteRoot := filepath.Join(root, "remote")
	repo := core.RepoInfo{Name: "test-project", RepoID: "test-repo-id"}

	objectPath := filepath.Join(remoteRoot, "objects", "ab", "cd", "abcd")
	if err := os.MkdirAll(filepath.Dir(objectPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(objectPath, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := &Rclone{
		name:       "test",
		remote:     "fake",
		objectRoot: "objects",
		verifyMode: "download",
		runCmd:     fakeRcloneRunner(remoteRoot),
	}

	ok, err := r.HasObject(repo, core.Pointer{OID: "abcd", Size: 4})
	if err == nil {
		t.Fatal("expected verification error for corrupt remote object")
	}
	if ok {
		t.Fatal("expected corrupt object to be reported as unavailable")
	}
}

func TestRcloneHasObjectAcceptsVerifiedContent(t *testing.T) {
	root := t.TempDir()
	remoteRoot := filepath.Join(root, "remote")
	data := []byte("good")
	oid, size := storage.HashBytes(data)
	repo := core.RepoInfo{Name: "test-project", RepoID: "test-repo-id"}

	objectPath := filepath.Join(remoteRoot, "objects", oid[:2], oid[2:4], oid)
	if err := os.MkdirAll(filepath.Dir(objectPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(objectPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	r := &Rclone{
		name:       "test",
		remote:     "fake",
		objectRoot: "objects",
		verifyMode: "download",
		runCmd:     fakeRcloneRunner(remoteRoot),
	}

	ok, err := r.HasObject(repo, core.Pointer{OID: oid, Size: size})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected verified object to be present")
	}
}

func fakeRcloneRunner(remoteRoot string) func(args ...string) error {
	return func(args ...string) error {
		if len(args) == 0 {
			return fmt.Errorf("missing command")
		}
		switch args[0] {
		case "lsf":
			if len(args) != 2 {
				return fmt.Errorf("invalid lsf args: %v", args)
			}
			_, err := os.Stat(fakeRclonePath(remoteRoot, args[1]))
			return err
		case "copyto":
			if len(args) != 3 {
				return fmt.Errorf("invalid copyto args: %v", args)
			}
			src := args[1]
			dst := args[2]
			if len(src) >= 5 && src[:5] == "fake/" {
				src = fakeRclonePath(remoteRoot, src)
			}
			data, err := os.ReadFile(src)
			if err != nil {
				return err
			}
			return os.WriteFile(dst, data, 0o644)
		default:
			return fmt.Errorf("unsupported command: %s", args[0])
		}
	}
}

func fakeRclonePath(remoteRoot, remotePath string) string {
	return filepath.Join(remoteRoot, filepath.FromSlash(remotePath[len("fake/"):]))
}
