package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/git"
)

func TestRemoteMigrateToGit_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	bareDir := initBareRepo(t, filepath.Join(tmp, "bare.git"))

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)
	gc := git.New(dir)

	cfg, err := core.ReadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DefaultRemote = "local-storage"
	cfg.Remotes["local-storage"] = core.RemoteConfig{
		Type:         "local",
		DisplayName:  "local-storage",
		Provider:     "local",
		Path:         filepath.Join(tmp, "remote-storage"),
		ObjectRoot:   "objects",
		ManifestRoot: "manifests",
		VerifyMode:   "local-hash",
		Pull:         true,
		Push:         true,
	}
	if err := core.WriteConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}

	writeAndCommit(t, gc, dir, "file.txt", "content", "initial commit")

	err = app.Remote([]string{"migrate", "to-git", "github", bareDir})
	if err != nil {
		t.Fatalf("expected successful migration, got: %v", err)
	}

	cfg, err = core.ReadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}

	gitRemote, exists := cfg.Remotes["github"]
	if !exists {
		t.Fatal("expected 'github' remote to exist in config")
	}
	if gitRemote.Type != "git" {
		t.Fatalf("expected 'github' remote type to be 'git', got: %q", gitRemote.Type)
	}
	if gitRemote.URL != bareDir {
		t.Fatalf("expected 'github' remote URL to be %q, got %q", bareDir, gitRemote.URL)
	}

	if !gc.HasGitRemote("github") {
		t.Fatal("expected 'github' remote to be added to .kk/git")
	}
}

func TestRemoteAddGit_DefaultProviderIsServiceAgnostic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	bareDir := initBareRepo(t, filepath.Join(tmp, "bare.git"))
	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)

	err := app.Remote([]string{"add", "git", "origin", bareDir})
	if err != nil {
		t.Fatalf("expected git remote add to succeed, got: %v", err)
	}

	cfg, err := core.ReadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	gitRemote, exists := cfg.Remotes["origin"]
	if !exists {
		t.Fatal("expected 'origin' remote to exist in config")
	}
	if gitRemote.Type != "git" {
		t.Fatalf("expected remote type to be 'git', got %q", gitRemote.Type)
	}
	if gitRemote.Provider != "git" {
		t.Fatalf("expected service-agnostic provider 'git', got %q", gitRemote.Provider)
	}
}

func TestRemoteAddGit_ProviderOverride(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	bareDir := initBareRepo(t, filepath.Join(tmp, "bare.git"))
	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)

	err := app.Remote([]string{"add", "git", "mirror", bareDir, "--provider", "internal-git"})
	if err != nil {
		t.Fatalf("expected git remote add to succeed, got: %v", err)
	}

	cfg, err := core.ReadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	gitRemote := cfg.Remotes["mirror"]
	if gitRemote.Provider != "internal-git" {
		t.Fatalf("expected provider override to be preserved, got %q", gitRemote.Provider)
	}
}

func TestRemoteMigrateToGit_AlreadyHasGitRemote_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)

	cfg, err := core.ReadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Remotes["github"] = core.RemoteConfig{
		Type: "git",
		URL:  "https://github.com/foo/bar.git",
		Push: true,
		Pull: true,
	}
	if err := core.WriteConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}

	err = app.Remote([]string{"migrate", "to-git", "github", "https://github.com/foo/bar.git"})
	if err != nil {
		t.Fatalf("expected no-op to return nil, got: %v", err)
	}
}

func TestRemoteMigrateToGit_NameConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)

	cfg, err := core.ReadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Remotes["local-remote"] = core.RemoteConfig{
		Type: "local",
		Path: filepath.Join(tmp, "remote"),
	}
	if err := core.WriteConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}

	err = app.Remote([]string{"migrate", "to-git", "local-remote", "https://github.com/foo/bar.git"})
	if err == nil {
		t.Fatal("expected error due to name conflict, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected name conflict message, got: %v", err)
	}
}

func TestRemoteMigrateToStorage_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)
	gc := git.New(dir)

	cfg, err := core.ReadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.DefaultRemote = "github"
	cfg.Remotes["github"] = core.RemoteConfig{
		Type: "git",
		URL:  "https://github.com/foo/bar.git",
		Push: true,
		Pull: true,
	}
	storagePath := filepath.Join(tmp, "remote-storage")
	cfg.Remotes["local-storage"] = core.RemoteConfig{
		Type:         "local",
		DisplayName:  "local-storage",
		Provider:     "local",
		Path:         storagePath,
		ObjectRoot:   "objects",
		ManifestRoot: "manifests",
		VerifyMode:   "local-hash",
		Pull:         true,
		Push:         true,
	}
	if err := core.WriteConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}

	_ = gc.AddRemote("github", "https://github.com/foo/bar.git")

	writeAndCommit(t, gc, dir, "file.txt", "content", "initial commit")

	err = app.Remote([]string{"migrate", "to-storage", "--yes"})
	if err != nil {
		t.Fatalf("expected successful migration to-storage, got: %v", err)
	}

	cfg, err = core.ReadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}

	if _, exists := cfg.Remotes["github"]; exists {
		t.Fatal("expected 'github' remote to be removed from config")
	}

	if cfg.DefaultRemote != "local-storage" {
		t.Fatalf("expected default remote to switch to 'local-storage', got %q", cfg.DefaultRemote)
	}

	bundlePath := filepath.Join(storagePath, "history", testMainBranch, "full.bundle")
	if _, err := os.Stat(bundlePath); err != nil {
		t.Fatalf("expected history bundle at %q, got error: %v", bundlePath, err)
	}
}

func TestRemoteMigrateToStorage_AlreadyInStorageMode_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)

	err := app.Remote([]string{"migrate", "to-storage"})
	if err != nil {
		t.Fatalf("expected idempotent no-op to return nil, got: %v", err)
	}
}

func TestRemoteMigrateToStorage_NoNonGitRemote_Error(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)

	cfg, err := core.ReadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Remotes["github"] = core.RemoteConfig{
		Type: "git",
		URL:  "https://github.com/foo/bar.git",
		Push: true,
		Pull: true,
	}
	if err := core.WriteConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}

	err = app.Remote([]string{"migrate", "to-storage", "--yes"})
	if err == nil {
		t.Fatal("expected error due to missing object storage remote, got nil")
	}
	if !strings.Contains(err.Error(), "no push-enabled object remote found") {
		t.Fatalf("expected object remote error message, got: %v", err)
	}
}
