package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunAcceptsWrappedGitCommandsWithoutGitSubcommand(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	initKKTestRepo(t, dir)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", "note.txt"},
		{"commit", "-m", "initial"},
		{"branch", "--show-current"},
		{"rev-parse", "HEAD"},
		{"log", "-1", "--pretty=%s"},
		{"show-ref", "--verify", "--quiet", "refs/heads/" + testMainBranch},
		{"checkout", "-B", "feature"},
	} {
		if err := Run(args); err != nil {
			t.Fatalf("kk %v failed: %v", args, err)
		}
	}
}

func TestRunRoutesGitShapedStatusAndRemoteFormsToWrappedGit(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	initKKTestRepo(t, dir)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	for _, args := range [][]string{
		{"status", "--porcelain"},
		{"status", "--short"},
		{"remote"},
		{"remote", "-v"},
		{"remote", "list"},
	} {
		if err := Run(args); err != nil {
			t.Fatalf("kk %v failed: %v", args, err)
		}
	}
}
