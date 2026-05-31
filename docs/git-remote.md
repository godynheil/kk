# Git Remote Documentation

This document explains how KK (KK-Private) handles remote storage with and without git remotes, and how code files vs. large files are managed.

## Table of Contents

1. [Overview](#overview)
2. [How KK Works Without Git Remotes](#how-kk-works-without-git-remotes)
3. [How KK Works With Git Remotes](#how-kk-works-with-git-remotes)
4. [Code Files vs Large Files](#code-files-vs-large-files)
5. [Repository Structure](#repository-structure)
6. [Common Workflows](#common-workflows)
7. [Migrating Between History Modes](#migrating-between-history-modes)
8. [Configuration](#configuration)

---

## Overview

KK is a version control system designed for projects with large binary assets. It combines:

- **Git** for version controlling code and pointer files
- **Object Remotes** (local, rclone, Google Drive) for storing large binary files

When you work with KK:
- **Code files** are stored directly in git as regular files
- **Large files** are converted to lightweight **pointer files** in git
- The actual binary content is stored on object remotes

---

## How KK Works Without Git Remotes

### Repository Setup

Without git remotes, KK uses only object remotes for collaboration:

```
kk init
kk setup gdrive        # or: kk remote add rclone backup --remote gdrive:KK/MyGame
```

### How Files Are Handled

| File Type | Stored In | Example |
|-----------|-----------|---------|
| Code files | `.kk/git` (git objects) | `main.go`, `script.py` |
| Large files | Object remote (Drive/rclone/local) | `video.mp4`, `model.fbx` |

### Pushing Without Git Remotes

When you run `kk push`:

1. **Pointer History**: Git history is bundled and uploaded to object remotes
2. **Large Files**: Binary objects are uploaded to object remotes
3. **Project Files**: Project metadata and config are synced

```
kk push
# → Uploads binary objects to object remote(s)
# → Bundles and uploads git history to object remote(s)
# → Syncs project files to object remote(s)
```

### Pulling Without Git Remotes

When you run `kk pull`:

1. **History**: Downloads and applies history bundles from object remotes
2. **Large Files**: Downloads binary objects to `.kk/objects/`
3. **Materialization**: Expands pointer files to their actual content

```
kk pull
# → Downloads and applies history bundles
# → Downloads binary objects
# → Materializes pointer files
```

### Cloning Without Git Remotes

```
kk clone rclone:gdrive:KK/MyGame
# or
kk clone drive:<folder-id>
# or
kk clone local:/path/to/project
```

**Process:**
1. Downloads project files from object remote
2. Creates KK directory structure
3. Downloads binary objects (if `--pull` specified)
4. Initializes `.kk/git` with bundled history

---

## How KK Works With Git Remotes

### Repository Setup

With git remotes, KK uses a hybrid approach:

```
kk init
kk remote add git origin https://github.com/your-username/MyGame.git
kk setup gdrive        # Object remote for large files
```

### Adding a Git Remote

```
kk remote add git origin https://github.com/your-username/MyGame.git
```

KK will:
1. **Verify connectivity** by running `git ls-remote`
2. Add the remote to `.kk/git/config`
3. Store metadata in `.kk/config.json`

### How Files Are Handled

| File Type | Stored In | Example |
|-----------|-----------|---------|
| Code files | Git remote (GitHub/GitLab/Gitea) | `main.go`, `script.py` |
| Large files | Pointer files in git + binaries on object remote | `video.mp4` → pointer in git, binary on Drive |
| `.kk/config.json` | Git remote | Remote configurations |
| `.kk/tracks.json` | Git remote | File tracking patterns |

### What Gets Pushed to Git vs Object Remotes

**Git remote receives:**
- All source code files (based on `codeExtensions` map)
- Pointer files (text references to large files)
- Configuration files (`.kk/config.json`, `.kk/tracks.json`)
- Git history

**Object remote receives:**
- Binary content of large files
- Manifests tracking what objects exist
- (Optional) History bundles if no git remote exists

### Pushing With Git Remotes

When you run `kk push`:

```
kk push
# → Syncs pointer history to git remote (git push)
# → Uploads binary objects to object remote(s)
# → (Optional) History bundles if no git remote
```

### Pulling With Git Remotes

When you run `kk pull`:

```
kk pull
# → Pulls pointer history from git remote (git pull)
# → Downloads binary objects from object remote(s)
# → Materializes pointer files
```

### Cloning From Git

```
kk clone git:https://github.com/your-username/MyGame.git
kk clone git:https://github.com/your-username/MyGame.git --pull
kk clone git:https://github.com/your-username/MyGame.git --account myprofile --pull
kk clone git:https://gitlab.com/your-username/MyGame.git --branch kk-test --pull
```

**Process:**
1. **Git clone**: Uses standard git to clone the repository
2. **File copy**: Copies files (excluding `.git` directory) to destination
3. **KK setup**: Initializes KK structure if not present
4. **Remote config**: Adds git remote to `.kk/git`
5. **Object check**: Verifies object remotes accessibility
6. **Materialization**: Downloads and expands pointer files (if `--pull`)

**What you get:**
- All source code files as regular files
- Pointer files for large assets
- `.kk/config.json` with remote configurations
- `.kk/tracks.json` with tracking patterns

---

## Code Files vs Large Files

### Code Files

Code files are defined by the `codeExtensions` map in `internal/core/code.go`. These files are **always stored as regular files in git**:

| Extensions | Language |
|------------|----------|
| `.go` | Go |
| `.py` | Python |
| `.js`, `.jsx` | JavaScript |
| `.ts`, `.tsx` | TypeScript |
| `.rs` | Rust |
| `.cpp`, `.c`, `.h` | C/C++ |
| `.java` | Java |
| `.php` | PHP |
| `.rb` | Ruby |
| `.swift` | Swift |
| `.kt` | Kotlin |
| `.md` | Markdown |
| `.json`, `.yaml`, `.toml` | Config files |
| `.txt`, `.csv`, `.tsv` | Plain text files |
| `.log` | Log files |
| `.tex`, `.rst`, `.adoc` | Documentation formats |
| `.properties`, `.cfg`, `.conf` | Configuration files |
| `.pem`, `.crt`, `.key` | Certificate/Key files |
| `.nix` | Nix expressions |
| ... and 100+ more |

### Large Files (Pointers)

Files that match patterns in `.kk/tracks.json` and are **NOT code files** are converted to pointers:

**Pointer format:**
```
version kk-lfs-1.0.0
oid sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
size 104857600
```

**Example tracking patterns:**
```json
{
  "patterns": [
    "*.mp4",
    "*.wav",
    "*.mp3",
    "*.png",
    "*.jpg",
    "*.jpeg",
    "*.psd",
    "*.fbx",
    "*.obj",
    "*.blend",
    "*.zip",
    "*.tar.gz"
  ]
}
```

### Adding Files

```
# Add code file - stored as regular file
kk add src/main.go
# Output: code src/main.go (storing as regular file)

# Add large file - converted to pointer
kk add Assets/video.mp4
# Output: large Assets/video.mp4 -> sha256:9f86... (104857600 bytes)
```

### Why This Matters

- **GitHub/GitLab/Gitea** can see your source code normally
- **Repository size stays small** - only text files in git
- **Large files** are accessible from object remotes
- **Collaborators** can clone with regular git and get code, then use KK for assets

---

## Repository Structure

```
MyProject/
├── .kk/                          # KK metadata
│   ├── git/                      # KK's internal git repository
│   │   ├── HEAD
│   │   ├── objects/              # Git objects
│   │   ├── refs/                 # Git references
│   │   └── config                # Git config (includes git remotes)
│   ├── objects/                  # Local cache of large files
│   │   └── ab/cd/<hash>         # Content-addressed storage
│   ├── config.json               # Remote configurations
│   ├── tracks.json               # File tracking patterns
│   ├── repo.json                 # Repository metadata (repo_id, name)
│   ├── tmp/                      # Temporary files
│   └── logs/                     # Operation logs
├── .kkignore                     # Exclusion patterns
├── src/                          # Source code (stored in git)
│   └── main.go
├── Assets/                       # Large assets (pointers in git)
│   ├── video.mp4                 # Pointer file
│   └── model.fbx                 # Pointer file
└── README.md                     # Documentation (stored in git)
```

---

## Common Workflows

> 💡 **Note on Tracking:** By default, if no custom tracking patterns are configured, KK automatically tracks **all files that are not recognized as code** (e.g., `.unitypackage`, `.uasset`, `.fbx`, `.png`, `.wav`, etc.). Code files always stay as regular files. Explicitly running `kk track` is only required if you want to limit tracking to specific custom patterns.

### Scenario 1: Solo Developer, Local Storage

```bash
# Initialize
kk init
kk track "*.mp4" "*.fbx" "*.blend"

# Work
kk add src/main.go Assets/video.mp4
kk commit -m "Add video player"

# Backup to NAS
kk remote add local nas --path /Volumes/NAS/KK/MyProject
kk push --remote nas
```

### Scenario 2: Team with GitHub + Google Drive

```bash
# Initialize (developer 1)
kk init
kk remote add git origin https://github.com/your-username/MyGame.git
kk setup gdrive
kk track "*.mp4" "*.fbx" "*.blend"

# Work
kk add src/main.go Assets/video.mp4
kk commit -m "Initial commit"
kk push  # Pushes code to GitHub, binaries to Drive

# Clone (developer 2)
kk clone git:https://github.com/your-username/MyGame.git --pull
# → Gets code from GitHub
# → Downloads binaries from Drive
```

### Scenario 3: Team Without GitHub

```bash
# Initialize
kk init
kk setup gdrive

# Work
kk add src/main.go Assets/video.mp4
kk commit -m "Initial commit"
kk push  # Pushes everything to Drive (includes history bundles)

# Clone
kk clone drive:<folder-id> --pull
# → Gets everything from Drive
```

### Scenario 4: Using Existing Git Repository

```bash
# Clone existing repo
git clone https://github.com/your-username/MyGame.git
cd project

# Initialize KK
kk init --here

# Setup remotes
kk remote add git origin https://github.com/your-username/MyGame.git
kk setup gdrive

# Track large files
kk track "*.mp4" "*.fbx"

# Add files
kk add .  # Code files stay regular, large files become pointers
kk commit -m "Add large assets with KK"
kk push
```

---

## Migrating Between History Modes

KK stores commit history in exactly one of two modes. You can switch at any time using `kk remote migrate`.

| Mode | Active when | History travels via |
|---|---|---|
| **Storage bundles** | No `type=git` remote in config | `full.bundle` + `inc-*.bundle` on your object remote |
| **Git remote** | A `type=git` remote exists | `git push` / `git pull` to GitHub, GitLab, etc. |

> **No history is lost** in either direction. Migration is safe-by-default:
> `to-storage` uploads a fresh bundle chain _before_ modifying any config;
> `to-git` rolls back the `.kk/git` remote entry if config write fails.

---

### Scenario 5: Storage bundles → GitHub (`to-git`)

A solo developer has been using Google Drive bundles. The team grows and wants GitHub for code review.

```bash
# Step 1 — verify current mode (no git remote listed)
kk remote list
# default: nas
# nas type=local provider=nas pull=true push=true priority=10

# Step 2 — migrate (connectivity is checked before any config changes)
kk remote migrate to-git github https://github.com/your-username/MyGame.git
```

Output:

```
kk: checking git remote accessibility...
kk: pushing all local branches to "github"...
To https://github.com/your-username/MyGame.git
 * [new branch]  main -> main
 * [new branch]  feature/ai -> feature/ai
kk: [github] all branches pushed

kk: migration complete → git remote "github" registered (https://github.com/your-username/MyGame.git)
kk: future 'kk push' will sync pointer history to "github" via git push
kk: existing history bundles on object remote(s) are kept but will not be extended
```

```bash
# Step 3 — confirm new state
kk remote list
# default: nas
# nas    type=local provider=nas    pull=true push=true priority=10
# github type=git  provider=github  url=https://github.com/your-username/MyGame.git push=true pull=true

# Step 4 — from now on, kk push syncs via git push
kk push
# kk: [github] syncing pointer history to git remote...
# kk: [github] pointer history synced
# kk: [nas] ...

# Step 5 — teammates clone from GitHub
kk clone git:https://github.com/your-username/MyGame.git --pull
```

#### If auth isn't set up yet

`kk remote migrate to-git` registers the remote even if the initial push fails
(e.g. SSH key not yet added to GitHub). The recovery command is printed:

```
kk: warning: initial git push failed: exit status 128
kk:          Once auth is set up, run:
kk:          git --git-dir=.kk/git push --all github
```

Run that command once, then `kk push` works normally.

#### Read-only mirror variant

```bash
# Register GitHub as a read-only pull source only (no push)
kk remote migrate to-git mirror https://github.com/your-username/MyGame.git --push false --pull true
```

---

### Scenario 6: GitHub → Storage bundles (`to-storage`)

The team is leaving GitHub and going to a self-hosted NAS only.

```bash
# Step 1 — verify current mode
kk remote list
# default: github
# github type=git  provider=github  url=https://github.com/your-username/MyGame.git push=true pull=true
# nas    type=local provider=nas    pull=true push=true priority=10

# Step 2 — preview the migration (shows what will change, asks to confirm)
kk remote migrate to-storage
```

Output:

```
kk: This will:
kk:   1. Upload a full history bundle to: nas
kk:   2. Remove git remote(s): github (https://github.com/your-username/MyGame.git)
kk:   Future 'kk push' will upload incremental bundles instead of using git push.

Proceed? [y/N] y
kk: creating initial history bundle(s)...
kk: [nas] creating history bundle (full.bundle)...
kk: [nas] uploading history bundle...
kk: [nas] history pushed (full.bundle, 2 branch(es))
kk: default remote set to "nas"

kk: migration complete → commit history will now travel via storage bundles
kk: teammates can restore history with: kk clone <spec> --history
```

```bash
# Step 3 — confirm new state
kk remote list
# default: nas
# nas type=local provider=nas pull=true push=true priority=10

# Step 4 — future pushes create incremental bundles automatically
kk push
# kk: [nas] creating history bundle (inc-000001.bundle)...
# kk: [nas] history pushed (inc-000001.bundle, 2 branch(es))

# Step 5 — teammates clone from NAS with full history
kk clone local:/NAS/KK/my-game --history --pull
```

#### Targeting one git remote when multiple exist

```bash
# Remove only the gitlab remote; keep github
kk remote migrate to-storage --remote gitlab --yes

# After this, github still exists and handles history;
# gitlab is removed from both .kk/git and .kk/config.json
```

#### Scripted / non-interactive use

```bash
# Skip the confirmation prompt (useful in CI or automation)
kk remote migrate to-storage --yes
```

---

### Idempotency & safety guarantees

Both commands are **safe to run more than once**. If the project is already in the
target mode, the command prints a notice and exits cleanly — no error, no changes.

| Situation | Behaviour |
|---|---|
| `to-git` — a git remote **already exists** | **No-op** — prints names of existing git remote(s) and exits with success |
| `to-git` — git URL unreachable | Error — config is **not changed** |
| `to-git` — initial `git push --all` fails | Non-fatal warning + manual recovery command printed; remote stays registered |
| `to-storage` — **no git remote exists** | **No-op** — prints "already in storage-bundle mode" and exits with success |
| `to-storage` — no non-git push remote found | Error — nowhere to store the bundles |
| `to-storage` — bundle upload fails | Error — config is **not changed** (history is safe) |
| User answers `N` at the confirm prompt | Cancelled — no changes made |

Example of running `to-git` when already in git-remote mode:

```
$ kk remote migrate to-git github https://github.com/your-username/MyGame.git
kk: already in git-remote mode (remote(s): github) — nothing to migrate
```

Example of running `to-storage` when already in storage-bundle mode:

```
$ kk remote migrate to-storage
kk: already in storage-bundle mode — nothing to migrate
```

---

## Configuration

### `.kk/config.json`

```json
{
  "version": "kk-local-config-1.0.0",
  "default_remote": "origin",
  "remotes": {
    "origin": {
      "type": "git",
      "url": "https://github.com/your-username/MyGame.git",
      "provider": "github",
      "pull": true,
      "push": true,
      "tags": ["git"]
    },
    "gdrive-backup": {
      "type": "drive",
      "drive_folder_id": "1ABC...",
      "drive_auth_path": "...",
      "provider": "google-drive",
      "pull": true,
      "push": true,
      "priority": 10
    },
    "nas": {
      "type": "local",
      "path": "/Volumes/NAS/KK/MyProject",
      "verify_mode": "local-hash",
      "pull": true,
      "push": true,
      "priority": 20
    }
  }
}
```

### Remote Types

| Type | Description | Used For |
|------|-------------|----------|
| `git` | Git hosting (GitHub, GitLab, Gitea) | Pointer history, code files |
| `drive` | Google Drive | Large file storage |
| `rclone` | Any rclone-supported storage | Large file storage |
| `local` | Local filesystem or network share | Large file storage |

### Remote Priority

Lower numbers = higher priority. KK tries remotes in order when downloading objects:

```
priority: 10  → Studio NAS (fast, local)
priority: 20  → Google Drive (slower, cloud)
priority: 50  → Archive storage (slowest)
```

---

## Summary

| Feature | Without Git Remote | With Git Remote | After `migrate to-git` | After `migrate to-storage` |
|---------|-------------------|-----------------|-----------------------|---------------------------|
| **Code storage** | Git (local) + history bundles | Git remote (GitHub/GitLab) | Git remote (GitHub/GitLab) | Git (local) + history bundles |
| **Large file storage** | Object remotes only | Object remotes | Object remotes | Object remotes |
| **Pointer history** | Bundled to object remotes | Git remote | Git remote | Bundled to object remotes |
| **Clone command** | `rclone:`, `drive:`, `local:` | `git:`, `rclone:`, `drive:`, `local:` | `git:`, + object remotes | `rclone:`, `drive:`, `local:` |
| **Switch history mode** | `kk remote migrate to-git` | `kk remote migrate to-storage` | `kk remote migrate to-storage` | `kk remote migrate to-git` |
| **Repository visibility** | Private to KK users | Public/visible to all git users | Public/visible to all git users | Private to KK users |
| **Collaboration** | Object remote access | Git + object remote access | Git + object remote access | Object remote access |
| **Git compatibility** | Full (KK manages git) | Full (regular git workflow) | Full (regular git workflow) | Full (KK manages git) |
