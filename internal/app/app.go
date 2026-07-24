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

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/godynheil/kk/internal/core"
	"github.com/godynheil/kk/internal/git"
)

type App struct {
	Root string
}

func New(root string) App {
	if root == "" {
		root = "."
	}
	return App{Root: root}
}

func findKKRoot(start string) (string, bool) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return start, false
	}
	dir := abs
	for {
		if _, err := os.Stat(filepath.Join(dir, core.KKGitDir, "HEAD")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return start, false
}

func rootForCommand(cmd string) string {
	switch cmd {
	case "init", "clone", "version", "--version", "setup", "set-up",
		"accounts", "install-path", "help", "--help", "-h":
		return "."
	default:
		if root, ok := findKKRoot("."); ok {
			return root
		}
		return "."
	}
}

func Run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}
	app := New(rootForCommand(args[0]))

	cmd := args[0]
	rest := args[1:]
	if shouldShowActivity(args) {
		_, _ = fmt.Fprintf(os.Stderr, "kk: processing %s...\n", cmd)
	}
	switch cmd {
	case "version", "--version":
		return app.Version()
	case "init":
		return app.Init()
	case "setup", "set-up":
		return app.Setup(rest)
	case "accounts":
		return app.Accounts(rest)
	case "status":
		if isGitStatusPassthrough(rest) {
			return app.Git(args)
		}
		return app.Status(hasFlag(rest, "--json"))
	case "track":
		if len(rest) > 0 && rest[0] == "list" {
			return app.TrackList()
		}
		return app.Track(rest)
	case "untrack":
		return app.Untrack(rest)
	case "add":
		return app.Add(rest)
	case "commit":
		return app.Commit(rest)
	case "push":
		return app.Push(rest)
	case "pull":
		return app.Pull(rest)
	case "fetch":
		return app.Fetch(rest)
	case "pull-file":
		return app.PullFile(rest)
	case "dematerialize":
		return app.Dematerialize(rest)
	case "fsck":
		return app.Fsck(hasFlag(rest, "--json"))
	case "objects":
		return app.Objects(rest)
	case "diff":
		return app.Diff(rest)
	case "remote":
		if isGitRemotePassthrough(rest) {
			return app.Git(args)
		}
		return app.Remote(rest)
	case "project":
		return app.Project(rest)
	case "stage":
		return app.Stage(rest)
	case "unstage":
		return app.Unstage(rest)
	case "discard":
		return app.Discard(rest)
	case "install-path":
		return app.InstallPath()
	case "clone":
		return app.Clone(rest)
	case "git":
		return app.Git(rest)
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		return app.Git(args)
	}
}

func shouldShowActivity(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch args[0] {
	case "help", "--help", "-h", "version", "--version":
		return false
	}
	return !hasFlag(args, "--json")
}

func hasFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

func isGitStatusPassthrough(args []string) bool {
	for _, arg := range args {
		if arg == "--short" || arg == "-s" || arg == "--porcelain" || arg == "-z" ||
			strings.HasPrefix(arg, "--porcelain=") {
			return true
		}
	}
	return false
}

func isGitRemotePassthrough(args []string) bool {
	if len(args) == 0 {
		return true
	}
	return strings.HasPrefix(args[0], "-")
}

func removeFlags(args []string, flags ...string) []string {
	flagSet := map[string]bool{}
	for _, f := range flags {
		flagSet[f] = true
	}
	var out []string
	for _, arg := range args {
		if !flagSet[arg] {
			out = append(out, arg)
		}
	}
	return out
}

func stringFlag(args []string, flag string) (string, error) {
	for i, arg := range args {
		if arg != flag {
			continue
		}
		if i+1 >= len(args) {
			return "", fmt.Errorf("%s requires a value", flag)
		}
		return args[i+1], nil
	}
	return "", nil
}

func printUsage() {
	fmt.Print(`KK - Git-compatible large-file version control for game projects

Core:
  kk init
  kk clone local:/path/to/remote-root/ProjectName [<dest>]
  kk clone rclone:<remote>:<path>/ProjectName [<dest>]
  kk clone drive:<folder-id> [<dest>]
           [--remote-name <name>]    remote name to register (default: origin)
           [--pull]                   materialise large files immediately
           [--history]                restore full commit history from the remote
                                      (requires project was pushed with history support)
           [--account <profile>]      Google Drive profile to use (see 'kk accounts')
           [--here]                   clone into the current directory (must be empty
                                      except for the kk binary itself)
           [--force]                  with --here: skip the non-empty directory check
  kk status [--json]
  kk track "*.mp4"
  kk track list
  kk untrack "*.mp4"
  kk add <file-or-dir...>
  kk commit -m "message"
  kk push [--remote name] [--all-remotes] [push args...]
  kk pull [--no-merge] [pull args...]
           --no-merge  fetch history bundles without merging (bundle-based remotes only)
  kk fetch             fetch history bundles without merging (no-git-remote mode only)

Large files:
  kk pull-file [--force] [--workers N] <file...>   Materialise specific files
  kk pull-file [--force] [--workers N] .            Materialise all pointer files (entire working tree)
  kk pull-file [--force] [--workers N] <dir>        Materialise all pointer files under a directory
  kk pull-file [--force] [--workers N] --all        Same as . (materialise everything)
  kk dematerialize <file...>
  kk fsck [--json]
  kk objects live [--json]
  kk objects refs <sha256> [--json]
  kk objects prune [--dry-run] [--json]
  kk objects sync [--workers N] [--verbose]


Remote:
  kk remote add local <name> --path <path> [--push true] [--pull true]
  kk remote add rclone <name> --remote <rclone:target> [--binary rclone|path]
  kk remote add git <name> <url>             Register any standard git remote
                                             Only pointer history is pushed — binary objects
                                             stay on your KK object remote(s).
  kk remote list [--json]
  kk remote set-default <name>
  kk remote check <name|--all> [--json]
  kk remote remove <name>
  kk remote rename <old> <new>

Projects:
  kk project connect [path] [--json]
  kk project reimport [path] [--json]
  kk project list [--json]

Diff and staging:
  kk diff --code --summary [--json]
  kk diff --code --file <path>
  kk stage <path...>
  kk unstage <path...>
  kk discard <path...>

History:
  kk log
  kk branch
  kk checkout <branch>

Setup:
  kk version             Show KK build version
  kk install-path        Add kk to user PATH (Windows: registry; Unix: prints instructions)
  kk setup gdrive [--name <remote-name>] [--account <profile>] [--folder <folder-id>] [--auth-only] [--scope file|full]
                         Native Google Drive setup for the current project.
                         --account <profile>  Use a saved account directly (skip interactive selection).
                         --auth-only          Authenticate only; skip folder/config setup.
                         --scope file|full    OAuth scope (default: file).
                         Options: [--auth-only] [--scope file|full] (use 'full' for shared folders/drives)
  kk accounts [--json]                     Show logged in/cached Google Drive accounts and their status
  kk accounts --delete <profile> [--json]  Delete one cached Google Drive account
  kk accounts --delete-all [--json]        Delete all cached Google Drive accounts
`)
}

func currentBranchLabel(client git.Client) string {
	if branch := client.CurrentBranch(); branch != "" {
		return branch
	}
	if commit, err := client.HeadCommit(); err == nil && len(commit) >= 7 {
		return commit[:7] + " (detached HEAD)"
	}
	return "(no commits yet)"
}

func boolArg(value string, fallback bool) bool {
	switch strings.ToLower(value) {
	case "true", "1", "yes", "y", "on":
		return true
	case "false", "0", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func fatalIfMissingValue(args []string, i int, flag string) error {
	if i+1 >= len(args) {
		return fmt.Errorf("%s requires a value", flag)
	}
	return nil
}
