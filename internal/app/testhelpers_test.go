package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/godynheil/kk/internal/git"
)

const testMainBranch = "kk-test-main"

func mustGit(t *testing.T, client git.Client, args ...string) {
	t.Helper()
	_, stderr, err := client.Combined(args...)
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, stderr)
	}
}

func initKKTestRepo(t *testing.T, dir string) App {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	a := New(dir)
	if err := a.Init(); err != nil {
		t.Fatalf("kk init in %s: %v", dir, err)
	}
	gc := git.New(dir)
	mustGit(t, gc, "config", "user.email", "test@example.com")
	mustGit(t, gc, "config", "user.name", "KK Test")
	mustGit(t, gc, "checkout", "-B", testMainBranch)
	return a
}

func initBareRepo(t *testing.T, dir string) string {
	t.Helper()
	gitPath, err := git.GitExecutable()
	if err != nil {
		t.Skip("git not available:", err)
	}
	cmd := exec.Command(gitPath, "init", "--bare", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare %s: %v\n%s", dir, err, out)
	}
	return dir
}

func writeAndCommit(t *testing.T, gc git.Client, root, name, content, msg string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, gc, "add", name)
	mustGit(t, gc, "commit", "-m", msg)
}
