package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/godynheil/kk/internal/git"
)

func TestAddStagesDeletions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)
	gc := git.New(dir)

	mustGit(t, gc, "add", ".kkignore")
	mustGit(t, gc, "commit", "-m", "commit kkignore")

	writeAndCommit(t, gc, dir, "keep.txt", "keep me", "add keep.txt")
	writeAndCommit(t, gc, dir, "delete_me.txt", "delete me", "add delete_me.txt")

	err := os.Remove(filepath.Join(dir, "delete_me.txt"))
	if err != nil {
		t.Fatal(err)
	}

	staged, err := gc.StagedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(staged) > 0 {
		t.Fatalf("expected no staged files, got: %v", staged)
	}

	err = app.Add([]string{"."})
	if err != nil {
		t.Fatalf("kk add . failed: %v", err)
	}

	staged, err = gc.StagedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(staged) != 1 || staged[0] != "delete_me.txt" {
		t.Fatalf("expected delete_me.txt to be staged, got: %v", staged)
	}

	err = app.Commit([]string{"-m", "commit deletion"})
	if err != nil {
		t.Fatalf("kk commit failed: %v", err)
	}

	staged, err = gc.StagedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(staged) > 0 {
		t.Fatalf("expected no staged files after commit, got: %v", staged)
	}
}

func TestAddSpecificDeletedFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)
	gc := git.New(dir)

	writeAndCommit(t, gc, dir, "delete_me.txt", "delete me", "add delete_me.txt")

	err := os.Remove(filepath.Join(dir, "delete_me.txt"))
	if err != nil {
		t.Fatal(err)
	}

	err = app.Add([]string{"delete_me.txt"})
	if err != nil {
		t.Fatalf("kk add delete_me.txt failed: %v", err)
	}

	staged, err := gc.StagedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(staged) != 1 || staged[0] != "delete_me.txt" {
		t.Fatalf("expected delete_me.txt to be staged, got: %v", staged)
	}
}

func TestAddNonExistentFileReturnsError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)

	err := app.Add([]string{"does_not_exist.txt"})
	if err == nil {
		t.Fatal("expected error adding non-existent file, got nil")
	}
}

func TestStatusTreatsCleanMaterializedPointerAsClean(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	tmp := t.TempDir()
	t.Setenv("APPDATA", tmp)

	dir := filepath.Join(tmp, "repo")
	app := initKKTestRepo(t, dir)
	gc := git.New(dir)

	if err := app.Track([]string{"*.bin"}); err != nil {
		t.Fatalf("track failed: %v", err)
	}
	mustGit(t, gc, "add", "-f", ".kkignore", ".kk/tracks.json")
	mustGit(t, gc, "commit", "-m", "setup tracking")

	assetPath := filepath.Join(dir, "asset.bin")
	if err := os.WriteFile(assetPath, []byte("binary version 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := app.Add([]string{"asset.bin"}); err != nil {
		t.Fatalf("kk add failed: %v", err)
	}
	if err := app.Commit([]string{"-m", "add asset"}); err != nil {
		t.Fatalf("kk commit failed: %v", err)
	}

	raw, _, err := gc.Combined("status", "--porcelain")
	if err != nil {
		t.Fatalf("raw git status failed: %v", err)
	}
	if !strings.Contains(raw, "asset.bin") {
		t.Fatalf("expected raw git status to see materialized asset.bin as modified, got %q", raw)
	}

	filtered, _, err := app.gitStatusOutput([]string{"status", "--porcelain"})
	if err != nil {
		t.Fatalf("kk git status failed: %v", err)
	}
	if strings.TrimSpace(filtered) != "" {
		t.Fatalf("expected kk git status to hide clean materialized asset.bin, got %q", filtered)
	}

	if err := os.WriteFile(assetPath, []byte("dirty user edit"), 0o644); err != nil {
		t.Fatal(err)
	}
	filtered, _, err = app.gitStatusOutput([]string{"status", "--porcelain"})
	if err != nil {
		t.Fatalf("kk git status after edit failed: %v", err)
	}
	if !strings.Contains(filtered, "asset.bin") {
		t.Fatalf("expected kk git status to report dirty edited asset.bin, got %q", filtered)
	}
}
