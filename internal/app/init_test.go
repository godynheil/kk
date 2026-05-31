package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/godynheil/kk/internal/core"
)

func TestInitCreatesKKGitWithoutRootGit(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("APPDATA", temp)
	root := filepath.Join(temp, "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	app := New(root)
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(root, core.KKGitDir)); err != nil {
		t.Fatalf("expected %s to exist: %v", core.KKGitDir, err)
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
		t.Fatal("expected root .git directory to be absent")
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected error checking root .git: %v", err)
	}
}
