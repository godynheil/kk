# How It Works

[Home](../README.md) | [Installation](installation.md) | [How To](../How_to.md) | [How It Works](how-it-works.md) | [License](../LICENSE.MD)

KK is a wrapper around Git. It does not replace Git history. It controls how large files enter the repository.

> **Scope:** KK is a client-side file management and large-object storage tool.
> It does **not** require GitHub, GitLab, or any other git hosting service.
> There is no server and no pull requests. Git history stays local by default.
> Optionally, you can push pointer history to a GitHub / GitLab remote with
> `kk remote add git` — binary objects are always stored separately on your KK
> storage driver (Google Drive, NAS, rclone, etc.).
> See [`docs/limitations.md`](limitations.md) for the full list of what KK does
> and does not do.

## Daily workflow

```
┌──────────────────────────────────────────────────────────────────┐
│  Machine A (author)                                              │
│                                                                  │
│  kk add .          → stage files, convert large ones to pointers│
│  kk commit -m "…"  → record snapshot in local git history       │
│  kk push           → upload file mirror + large objects          │
│                           │                                      │
└───────────────────────────┼──────────────────────────────────────┘
                            │  driver (Google Drive / NAS / rclone)
┌───────────────────────────┼──────────────────────────────────────┐
│  Machine B (teammate)     │                                      │
│                           ▼                                      │
│  kk clone <spec>   → download mirror, create local git commit   │
│  kk pull-file .    → materialise all large files on demand      │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Push (Machine A)

```bash
kk add .
kk commit -m "update assets"
kk push                    # syncs file mirror + uploads large objects
```

`kk push` does three things:
1. Mirrors the committed project files to `<driver>/<ProjectName>/`
2. Uploads any new SHA-256 objects for large files
3. _(Optional)_ Pushes pointer history to any push-enabled `type=git` KK remote (GitHub, GitLab, etc.)
4. _(When no git remote)_ Bundles and uploads the full commit history (branches + commits) to the
   object storage driver as incremental git bundles under `<driver>/<ProjectName>/history/<branch>/`

Large objects (binary blobs) are **never** sent to git remotes — only the lightweight
pointer files that already live in `.kk/git` history are synced there.

### Get updates (Machine B)

**First time on a new machine:**

```bash
kk clone drive:<project-folder-id> --pull
# or
kk clone local:/Volumes/NAS/KK/MyGame --pull
```

**After the project is already cloned — get latest large files:**

```bash
kk pull-file .          # download and materialise all pointer files
kk pull-file --all      # identical
kk pull-file Assets/    # only files under a directory
```

> ⚠️ **`kk pull` is not the command for syncing large files.**
> `kk pull` wraps `git pull` and only syncs git commit history from a git
> remote (GitHub / GitLab). In a KK-only workflow with no git remote configured,
> `kk pull` instead downloads and merges history bundles from the object storage
> remote automatically:
> ```
> kk pull            # merge history from remote (via bundles if no git remote)
> kk pull --no-merge # download bundles only; don't merge (inspect first)
> kk fetch           # synonym for pull --no-merge (fetch-only, no merge)
> ```
> Use `kk pull-file .` to download large files at any time.
> If you add a git remote with `kk remote add git`, `kk pull` will pull from it
> instead.

## Hidden Git database

KK stores the Git database in `.kk/git` and calls Git with:

```bash
git --git-dir=.kk/git --work-tree=. <command>
```

That lets KK keep the project root free of a `.git/` folder while still using Git's proven history model.

## Pointer files

Tracked large files are replaced by text pointers before staging:

```text
version kk-lfs-1.0.0
oid sha256:<hash>
size <bytes>
```

The real bytes are stored in `.kk/objects` under a content-addressed path.

### Default Tracking Behavior
By default, if no custom tracking patterns are configured in `.kk/tracks.json`, KK automatically tracks **all files that are not recognized as code** (e.g., `.unitypackage`, `.uasset`, `.fbx`, `.png`, `.wav`, etc.).
Code files with extensions like `.go`, `.cs`, `.cpp`, `.h`, `.html`, `.json`, etc. are always exempt and travel as regular files.

If you specify custom tracking patterns using `kk track <pattern>`, KK will switch to only tracking files that match your custom patterns (while still keeping code files exempt).


## Download on demand

A checked-out project can contain only pointer files. `kk pull-file path`
downloads and verifies the real object only when needed. `kk pull-file --force`
re-downloads the object even if it is already cached locally.

### Materialise a single file

```bash
kk pull-file Assets/cinematic.mp4
```

### Materialise everything at once

Pass `.` (the working-tree root) or the `--all` flag to materialise every
pointer file that HEAD still has un-expanded in the working tree:

```bash
kk pull-file .          # materialize all pointer files
kk pull-file --all      # identical to .
```

Both forms skip files that are already materialized (real bytes on disk).
Add `--force` to re-download and overwrite even those:

```bash
kk pull-file --force .
kk pull-file --force --all
```

### Materialise a directory subtree

```bash
kk pull-file Assets/         # only pointer files under Assets/
kk pull-file Content/Levels/ # only pointer files under Content/Levels/
```

### Control concurrency

All multi-file forms (`.`, `--all`, directory, or multiple explicit paths)
run with concurrent workers and display a live progress bar:

```
  pulling    [████████████░░░░░░░░░░░░░░░░░░] 3/8 (37%)
  ├─ [1] Assets/cinematic.mp4
  ├─ [2] Content/Textures/hero_diffuse.tga
  ├─ [3] Audio/music_theme.wav
  └─ [4] ·
```

```bash
kk pull-file --workers 8 .          # 8 concurrent downloads
kk pull-file --workers 1 --all      # serial (useful for rate-limited remotes)
KK_WORKERS=8 kk pull-file .         # via environment variable
```

After a successful `kk checkout` or `kk switch`, KK scans the new `HEAD` for
pointer files and materializes them automatically. During download it prints
remote checks, fallback decisions, and the remote that ultimately served the
object.

## Adding a Google Drive remote

KK supports two Google Drive drivers. Pick one — you don't need both.

### Native Google Drive (`drive` — recommended)

`kk setup gdrive` authenticates with Google, creates the `KK/<ProjectName>/`
folder on your Drive, registers the remote, and prints the **project folder ID**
directly — no extra steps needed.

```bash
# Inside your kk repo — opens a browser for Google OAuth:
kk setup gdrive

# Skip the interactive account selection menu by naming a saved profile:
kk setup gdrive --account work

# Output (example):
# Using saved Google Drive account "work".
# kk: Drive project folder-id: 1DVMqBeX-hIj7XrQkUBVFKIVTP4wP5DbV
# kk: Run 'kk push' to upload your project to Drive.
#     Share the project folder ID with teammates so they can clone:
#     kk clone drive:1DVMqBeX-hIj7XrQkUBVFKIVTP4wP5DbV
# Remote ready.

# Then push:
kk push
```

> **What gets saved in `config.json`:**
> `drive_folder_id` is the **project folder** ID (`KK/<ProjectName>/`) — not the
> KK root. This means teammates only need access to the project folder itself,
> not the parent `KK/` root, and the ID you see in config.json is exactly the
> one you pass to `kk clone`.

Need access to Shared Drives or externally shared folders?

```bash
kk setup gdrive --scope full
# Or combined with a saved profile:
kk setup gdrive --account work --scope full
```

### Google Drive via rclone (`rclone`)

Requires [rclone](https://rclone.org) installed and a Drive remote configured
(`rclone config`).

```bash
# --remote must point to the PROJECT folder, not just a KK root.
# Minimal:
kk remote add rclone gdrive --remote gdrive:KK/MyGame --push true --pull true

# Full (with optional metadata):
kk remote add rclone gdrive \
  --display-name "Google Drive" \
  --role primary \
  --provider google-drive \
  --binary rclone \
  --remote gdrive:KK/MyGame \
  --verify-mode download \
  --priority 20 \
  --pull true \
  --push true \
  --tag cloud
# remote added gdrive
#     Teammates can clone with:
#     kk clone rclone:gdrive:KK/MyGame
```

> `--remote gdrive:KK/MyGame` points **directly to the project folder** on
> Drive. kk push/pull operate at that path without appending any project name.

### Comparison

| | Native `drive` | rclone |
|---|---|---|
| Requires rclone | No | Yes |
| Setup command | `kk setup gdrive` | `rclone config` → `kk remote add rclone …` |
| Clone syntax | `kk clone drive:<folder-id>` | `kk clone rclone:gdrive:KK/MyGame` |
| Shared Drive support | Yes (`--scope full`) | Yes (via rclone config) |

## Adding a Git remote (GitHub / GitLab / Gitea)

Git remote support (`kk remote add git`) is available for pointer-history sync.
Core push, pull, clone, and branch workflows have been thoroughly tested on
GitHub and GitLab. Pull requests, concurrent pushes, branch divergence,
CI/webhook integration, and branch protection behavior are handled by the git
host and are not managed by KK.

You can register any git hosting service as a KK **git remote**. KK will then
automatically push the lightweight pointer history (`.kk/git`) to that remote
on every `kk push`. Binary objects are **never** sent to git remotes —
they always go to a KK object remote (local/drive/rclone).

> **Why?** This lets developers and teams use GitHub or GitLab as an optional
> code-review or CI mirror while keeping large binary files out of the git
> repository. Teammates clone large files with `kk clone`, not `git clone`.

### Add a git remote

```bash
# Positional URL form
kk remote add git github https://github.com/your-username/MyGame.git

# Named flag form
kk remote add git github --url https://github.com/your-username/MyGame.git

# With explicit push/pull flags (both default to true)
kk remote add git github https://github.com/your-username/MyGame.git --push true --pull true

# GitLab example
kk remote add git gitlab https://gitlab.com/your-username/MyGame.git

# Self-hosted Gitea
kk remote add git origin https://git.example.com/your-username/MyGame.git
```

KK automatically detects the provider from the URL (`github`, `gitlab`,
`bitbucket`, `azure-devops`, `gitea`) and records it in `.kk/config.json`.
The actual URL is stored in `.kk/git/config` by git itself.

### How push works with a git remote

```bash
kk push
# kk: [github] syncing pointer history to git remote...
# kk: [github] pointer history synced
# kk: [gdrive] syncing 5 file(s) (4 workers)...
# ...object upload as usual...
```

1. KK syncs pointer history to every push-enabled `type=git` remote.
2. KK uploads object blobs to every push-enabled non-git remote (local/drive/rclone).
3. If the current branch has no upstream yet, KK tries to set one with
   `git push -u <remote> <branch>` and then uses normal `git push` on later
   runs.

### First push to a new git remote

For a new branch, `kk push` bootstraps upstream tracking automatically:

```bash
kk push
# From then on, kk push syncs automatically.
```

### How pull works with a git remote

```bash
kk pull
# Runs git pull against all configured git remotes (pointer history only).
# Does NOT download any binary objects — use kk pull-file for that.
```

```bash
kk pull-file .   # download large-file objects (always via the KK object remote)
```

### Listing and removing git remotes

```bash
kk remote list
# default: gdrive
# github  type=git provider=github url=https://github.com/your-username/MyGame.git push=true pull=true
# gdrive  type=drive ...

kk remote remove github   # removes from .kk/config.json AND .kk/git/config
kk remote rename github gh
```

### kk remote check for git remotes

```bash
kk remote check github
# ok github  git remote configured (https://github.com/your-username/MyGame.git)
```

---



`.kk/config.json` can define any number of remotes. Pulls can fallback by priority. Pushes are explicit and respect each remote's `push` flag.

Every remote's `path` / `remote` / `drive_folder_id` points **directly to the
project folder**. All data for that project lives at the root of that folder —
projects on different remotes are fully isolated:

```text
<project-folder>/         ← what --path / --remote / drive_folder_id point to
  objects/ab/cd/<oid>     ← large file blobs
  manifests/<repo_id>.json← object manifest
  .kk/                    ← project file mirror root
  <source files …>
```

See `docs/remote-layout.md` for the complete layout specification.

## Project file sync

On every `kk push`, KK mirrors the working directory to the configured project
folder on the remote (`--path`, `--remote`, or `drive_folder_id`). This makes
the remote a self-contained backup of the project — source files, configuration,
and kk metadata — without relying on GitHub or GitLab.

### Two sync modes

| Mode | Flag | What gets synced |
|---|---|---|
| Committed only **(default)** | _(none)_ | Files tracked in current HEAD + `.kk/` metadata |
| Working directory | `--sync-working-dir` | Everything on disk that isn't excluded |

Use `--sync-working-dir` when you explicitly want to include uncommitted local
changes in the remote snapshot:

```bash
kk push                    # default: syncs committed files only
kk push --sync-working-dir # opt-in: syncs entire working directory
```

In both modes the following are excluded: files matching `.kkignore`, and the
`.kk/` internals `objects/`, `tmp/`, `logs/`, `git/`. The three config files
`.kk/repo.json`, `.kk/config.json`, and `.kk/tracks.json` **are always
included** (they are not tracked by git, so the committed-only sync adds them
explicitly) so that `kk clone` can reconstruct the repository.

### Concurrent sync and upload

File sync and object uploads both run with concurrent workers for better
throughput. The default is **4 workers**. Override via flag or environment
variable:

```bash
# Flag (overrides env var)
kk push --workers 8

# Environment variable (applies to every kk operation in the session)
KK_WORKERS=8 kk push
KK_WORKERS=1 kk push   # serial — useful for debugging or throttled remotes
```

During a push KK shows a live multi-line display: one summary bar plus one row
per active worker:

```
  syncing    [████████████░░░░░░░░░░░░░░░░░░] 12/20 (60%)
  ├─ [1] src/Assets/hero_diffuse.png
  ├─ [2] src/Assets/banner.tga
  ├─ [3] .kk/config.json
  └─ [4] docs/README.md
```

The same format is used for the object-upload phase:

```
  uploading  [████████░░░░░░░░░░░░░░░░░░░░░░] 4/10 (40%)
  ├─ [1] ab12cd34ef56 (200.0 MB)
  ├─ [2] 78901abc2345 (45.2 MB)
  ├─ [3] deadbeef9012 (1.2 GB)
  └─ [4] cafe00001234 (88.0 MB)
```

When `--pull` is used with `kk clone`, the same worker pool and display is used
while materialising large files.

### Name collision guard

Before syncing, KK reads `.kk/repo.json` from the remote project mirror. If
the file exists and contains a different `repo_id`, the push is aborted:

```text
remote folder "MyGameProject" already belongs to project "MyGameProject"
(repo_id: <other-id>); rename your project or use a different remote root
```

This prevents one project from overwriting another project's files when two
repos share the same name on the same remote.

## kk clone

Because `kk push` stores a complete project snapshot on the remote, `kk clone`
can reconstruct the full project on any machine — no GitHub or GitLab required.

### Remote spec formats

```bash
kk clone local:/Volumes/NAS/KK/MyGame    [<dest>] [--remote-name nas]    [--pull] [--history] [--workers N]
kk clone rclone:gdrive:KK/MyGame         [<dest>] [--remote-name gdrive] [--pull] [--history] [--workers N]
kk clone drive:<project-folder-id>       [<dest>] [--remote-name gdrive] [--pull] [--history] [--workers N]
                                                   [--account <profile>]
                                                   [--here]   clone into the current directory
                                                   [--force]  with --here: skip non-empty check
```

#### Clone into the current directory (`--here`)

By default `kk clone` creates a **new subdirectory** named after the project.
Use `--here` to clone directly into the current working directory instead —
useful when you place `kk.exe` inside a folder and want the project to live
there without a nested subdirectory:

```
MyGame/
  kk.exe          ← place kk here first
```

```bash
cd MyGame
kk clone drive:<project-folder-id> --account work --here
```

The directory must be **empty except for the kk binary** (`kk.exe`, `kk`, or
`kk-portable.exe`). If other files are present, KK stops with an error:

```
kk clone: --here: directory "MyGame" is not empty
  unexpected file(s): SomeFile.uasset, ...
  Remove them first, or re-run with --force to clone anyway.
```

Add `--force` to skip the check and clone anyway:

```bash
kk clone drive:<project-folder-id> --here --force
```

For `drive:` remotes, pass the **project folder ID** — the same ID printed by
`kk setup gdrive` and visible at the end of the folder's Google Drive URL:

```
https://drive.google.com/drive/folders/1DVMqBeX-hIj7XrQkUBVFKIVTP4wP5DbV
                                        ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
                                        kk clone drive:<this-id>
```

#### Multiple Google Drive accounts

If you have more than one Google Drive profile saved locally (e.g. personal and
work), use `--account <profile>` to choose which credentials to use:

```bash
# List saved profiles
kk accounts

# Delete one saved profile from the local auth cache
kk accounts --delete work

# Delete all saved profiles from the local auth cache
kk accounts --delete-all

# Clone using the "work" profile
kk clone drive:<project-folder-id> --account work

# Clone using the "personal" profile
kk clone drive:<project-folder-id> --account personal
```

Profiles are created when you run `kk setup gdrive` and enter a profile name.
The profile name is the filename without `.json` in the KK auth cache directory.
If `--account` is omitted, KK uses the `default` profile; if that does not
exist it will prompt you to choose interactively.

```
https://drive.google.com/drive/folders/1DVMqBeX-hIj7XrQkUBVFKIVTP4wP5DbV
                                        ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
                                        kk clone drive:<this-id>
```

KK resolves the project name from the Drive API automatically and stores the
**project folder ID** directly in `config.json` — not the parent folder ID.

The `--workers` option controls how many files are downloaded concurrently when
`--pull` is used. It respects the `KK_WORKERS` environment variable as a
fallback (default: 4).


### Prerequisites

| Backend | One-time setup |
|---------|---------------|
| `local` | Mount the NAS / share; provide a reachable path |
| `rclone` | Run `rclone config` to create a named remote |
| `drive` (Machine A, inside project) | `kk setup gdrive` — creates the Drive folder, saves auth to `~/.config/KK/gdrive/default.json`, writes `drive_folder_id` to `.kk/config.json`, and prints the clone command |
| `drive` (Machine B, before cloning) | `kk setup gdrive --auth-only` — saves OAuth credentials only; no repo or project folder needed. After auth, clone with `kk clone drive:<project-folder-id>` |

### What clone does

1. **Check access** — calls `driver.Check()` before touching the disk; fails fast if the remote is unreachable.
2. **Verify project** — reads `<remote>/<ProjectName>/.kk/repo.json` to confirm the project exists and to recover the original `repo_id`.
3. **Download files** — copies the entire project mirror to `<dest>/`, including pointer files, source files, and `.kk/` metadata.
4. **Initialise git** — runs `git init` inside `.kk/git/`.
5. **Restore history** _(optional, `--history`)_ — downloads the incremental history bundles from `<remote>/history/<branch>/` and unbundles them into `.kk/git`, restoring all branches and commits. Without `--history` a single `kk clone: initial snapshot` commit is created instead.
6. **Write fresh config** — the downloaded `config.json` belonged to another machine; it is discarded and replaced with a new one that registers the clone source as the single remote (`origin` by default).
7. **Restore repo identity** — writes back the recovered `repo.json` so the local repo shares the same `repo_id` as the source and can therefore address the same large-file objects on the remote.
8. **Materialise** _(optional)_ — if `--pull` is given, downloads and verifies every large-file object referenced by pointer files in `HEAD`.

### Full example — local NAS remote

**Machine A** (the original developer):

```bash
# 1. Create the project
mkdir MyGame && cd MyGame
kk init

# 2. Add a large asset and some source code
kk track "*.mp4"
cp ~/raw/cinematic.mp4 Assets/cinematic.mp4
echo "v0.1" > version.txt

# 3. Stage and commit
kk add Assets/cinematic.mp4 version.txt
kk commit -m "add cinematic and version"

# 4. Register the NAS as a remote and push
# --path points directly to the project folder.
kk remote add local nas --path /Volumes/NAS/KK/MyGame --push true --pull true
# remote added nas
#     Teammates can clone with:
#     kk clone local:/Volumes/NAS/KK/MyGame
kk push

# Remote now contains:
#   /Volumes/NAS/KK/MyGame/                      ← project folder (= --path)
#     .kk/repo.json                               ← repo_id preserved here
#     .kk/config.json
#     .kk/tracks.json
#     Assets/cinematic.mp4                        ← pointer file (tiny)
#     version.txt
#   /Volumes/NAS/KK/MyGame/objects/ab/cd/<oid>   ← real 200 MB file
#   /Volumes/NAS/KK/MyGame/manifests/<id>.json
```

**Machine B** (a new team member, NAS already mounted at `/Volumes/NAS`):

```bash
# No git URL needed. Clone directly from the NAS backup.
kk clone local:/Volumes/NAS/KK/MyGame

# Output:
# kk: checking remote "origin" (local) ...
# kk: remote "origin" is accessible
# kk: found project "MyGame" (repo_id: a1b2c3d4-...)
# kk: downloading project files from "origin" ...
# kk: project files downloaded
# kk: staging files ...
# kk: clone complete -> /home/user/MyGame

cd MyGame

# At this point Assets/cinematic.mp4 is still a pointer file.
# Large-file objects have NOT been downloaded yet.
kk fsck
# ok-remote <oid>  Assets/cinematic.mp4  remote=origin
# (object exists on remote, not yet local)

# Download one file on demand:
kk pull-file Assets/cinematic.mp4
# kk: object <oid> not in local cache; checking 1 pull remote(s)
# kk: downloading <oid> from origin
# materialized Assets/cinematic.mp4 using remote origin

# Or download everything at once by passing --pull to clone:
kk clone local:/Volumes/NAS/KK/MyGame MyGame2 --pull
# ... automatically materialises all pointer files after cloning
```

### Full example — rclone (Google Drive)

**One-time setup** (each machine, done once):

```bash
rclone config
# → create a remote named "gdrive" pointing to Google Drive
```

**Machine A** — push:

```bash
# --remote points directly to the project folder.
kk remote add rclone gdrive --remote gdrive:KK/MyGame --push true --pull true
# remote added gdrive
#     Teammates can clone with:
#     kk clone rclone:gdrive:KK/MyGame
kk push
```

**Machine B** — clone:

```bash
kk clone rclone:gdrive:KK/MyGame
# kk: checking remote "origin" (rclone) ...
# kk: remote "origin" is accessible
# kk: found project "MyGame" (repo_id: a1b2c3d4-...)
# kk: downloading project files from "origin" ...
# kk: project files downloaded
# kk: staging files ...
# kk: clone complete -> ./MyGame

cd MyGame
kk pull-file Assets/cinematic.mp4   # fetch large file on demand
```

### Full example — native Google Drive

The Google Drive folder-id is **not** something you type or invent. It is a
unique identifier that Google assigns to every Drive folder. KK creates the
folder automatically during `kk setup gdrive` and prints the id so you can
share it with teammates.

**Machine A — one-time setup inside the project directory:**

```bash
cd MyGame          # must be inside a kk repo
kk setup gdrive

# Output:
# Connect Google Drive
# Opening browser for authorization...
# kk: Google Drive auth saved to ~/.config/KK/gdrive/default.json
# Testing upload...
# Testing download...
# kk: Drive folder-id: 1xR9gP3kTwNq2Lm8vZoUeAb7cYs4FjDh
# kk: Run 'kk push' to create the project folder on Drive.
#     Afterwards, open Google Drive, navigate to KK/MyGame,
#     copy the folder ID from the URL, and share it with teammates:
#     kk clone drive:<project-folder-id>
# Remote ready.
```

`kk setup gdrive` does three things:
1. **OAuth**: opens a browser, authenticates with Google, and saves the refresh
   token to `~/.config/KK/gdrive/default.json`.
2. **Creates folder**: ensures `My Drive → KK/` exists, and records the ID of
   that `KK/` folder in `.kk/config.json` as `drive_folder_id`. The per-project
   subfolder (`KK/<ProjectName>/`) is created automatically on the first
   `kk push`.
3. **Tests round-trip**: uploads and downloads a probe file to confirm access.

You can also retrieve the KK root folder-id later:

```bash
kk remote list --json
# Look for: "drive_folder_id": "1xR9gP3kTwNq2Lm8vZoUeAb7cYs4FjDh"
# This is the KK/ root folder. After your first push, the project subfolder
# is visible in Drive under KK/<ProjectName>.
```

Open the **project subfolder** in Google Drive's web interface and copy the ID
from the URL — that is the ID to share with teammates:

```
https://drive.google.com/drive/folders/9zK4mBvXpL7wNcJeRd2oTsUf1YgHiQa3
                                        ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
                                        kk clone drive:<this-id>
```

**Machine A — push the project:**

```bash
kk push
# kk: 3 committed file(s) will be synced
# kk: [gdrive] syncing 3 file(s) (4 workers)...
#   syncing    [██████████████████████████████] 3/3 (100%)
#   ├─ [1] .kk/repo.json
#   ├─ [2] Assets/cinematic.mp4
#   └─ [3] version.txt                (progress bar clears, then:)
# kk: [gdrive] synced 3 file(s)
# kk: [gdrive] checking 1 object(s)...
#   checking   [██████████████████████████████] 1/1 (100%)  ab12cd34ef56
# kk: [gdrive] 1 to upload, 0 already on remote
# kk: [gdrive] uploading 1 object(s) (4 workers)...
#   uploading  [██████████████████████████████] 1/1 (100%)
#   └─ [1] ab12cd34ef56 (200.0 MB)
# kk: [gdrive] uploaded 1 object(s)
# kk: [gdrive] objects complete — 1 uploaded, 0 skipped
```

**Machine B — one-time setup (auth only, no project needed):**

```bash
# Machine B has no kk repo yet. Use --auth-only to save credentials
# without touching any project config or creating Drive folders.
kk setup gdrive --auth-only

# Output:
# Connect Google Drive
# Opening browser for authorization...
# kk: Google Drive auth saved to ~/.config/KK/gdrive/default.json
# kk: auth-only mode — skipping folder setup.
#     Ask the project owner for the Drive project folder ID, then run:
#     kk clone drive:<project-folder-id>
```

**Machine B — clone using the project folder ID shared by Machine A:**

```bash
# Machine A pushed the project. The project folder now exists at KK/MyGame in Drive.
# Open that folder in the browser, copy its ID from the URL, and clone:
kk clone drive:9zK4mBvXpL7wNcJeRd2oTsUf1YgHiQa3 --pull

# kk reads ~/.config/KK/gdrive/default.json for credentials automatically.
# kk: resolved folder → project="MyGame"  parent-id=1xR9gP3kTwNq2Lm8vZoUeAb7cYs4FjDh
# kk: checking remote "origin" (drive) ...
# kk: remote "origin" is accessible
# kk: found project "MyGame" (repo_id: a1b2c3d4-...)
# kk: downloading project files from "origin" ...
# kk: project files downloaded
# kk: staging files ...
# kk: --pull: materialising large files ...
#   pulling    [██████████████████████████████] 1/1 (100%)
#   └─ [1] Assets/cinematic.mp4
# kk: materialised 1/1 object(s)
# kk: materialisation complete
# kk: clone complete -> ./MyGame
```

### After cloning

```bash
cd MyGame

# Check which large files are local vs. remote-only
kk fsck

# Materialise specific files on demand
kk pull-file Assets/*.mp4

# Show project status (pointer files appear as regular files once materialised)
kk status
```

For storage clones (`local`, `rclone`, and `drive`), the registered `origin`
remote supports both **pull** and **push** by default. A cloned machine can
become a full contributor without adding a GitHub/GitLab remote:

```bash
cd MyGame
kk pull

# Make changes on main or a feature branch.
kk add .
kk commit -m "update from clone"
kk push
```

`kk push` uploads the committed project mirror, large objects, and storage
history bundles for every local branch. Teammates can then run `kk pull` on the
same branch to fetch and merge those bundles. Older clones that were created
with `origin.push=false` are upgraded automatically on first `kk push` when
`origin` is the only storage remote.


### Shared Drives and Shared Folders

If you need to clone or work with a repository located on a **Shared Drive (Team Drive)** or within a **shared folder** that has been shared with your Google account by another user, the default restricted access scope (`drive.file`) is insufficient. Under `drive.file`, KK can only interact with files that it created itself on your account.

To access these shared files and folders, you must authorize KK with the full drive access scope:

1. **Authorize with full scope:**
   If you have not set up your Google Drive account profile yet, or want to re-authenticate with the expanded scope, run:
   ```bash
   kk setup gdrive --auth-only --scope full
   ```
   *(Omitting `--auth-only` will also initialize a `KK` remote directory, but for cloning, `--auth-only` is recommended.)*

2. **Clone the shared folder or drive:**
   Once authorized, clone using the project folder's ID:
   ```bash
   kk clone drive:<project-folder-id> --pull
   ```

KK automatically includes the `supportsAllDrives=true` and `includeItemsFromAllDrives=true` parameters in all API queries, allowing it to seamlessly read, write, and list items in Shared Drives and external shared directories once authorized with the `full` scope.

## ⚠️ No server-side deletion tracking

> **Warning: KK does not host or control your storage driver.**
>
> KK is a client-side tool only. All files — project mirrors, large-file objects,
> and manifests — live directly on your chosen storage driver (Google Drive, a local NAS,
> an rclone-supported provider, etc.). KK itself runs no server and keeps no
> central record of who has access to your remote or what they do with it.
>
> **This means:**
> - Anyone with write access to your driver folder can **delete, overwrite, or
>   corrupt** objects, manifests, or project files at any time.
> - KK will **not** be notified of external deletions. If an object is removed
>   from the remote by another user (or by the driver's own garbage collection /
>   quota enforcement), `kk fsck` will report it as missing only after you try to
>   access it — there is no push or pull hook to alert you in advance.
> - Google Drive, rclone remotes, and local filesystems each have their own
>   access-control model. KK does not layer any additional permissions on top of
>   them.
>
> **Recommendations:**
> - **Restrict folder access** on the driver to only the collaborators who
>   need write access. Use the driver's own sharing / permission settings (e.g.
>   Google Drive folder sharing, POSIX permissions on a NAS).
> - **Keep a local copy** of critical large-file objects (`.kk/objects/`) on at
>   least one trusted machine, or use multiple remotes so that objects are
>   replicated across more than one driver.
> - **Audit the remote periodically** with `kk fsck` to detect any objects that
>   have gone missing from the remote before you actually need them.

## Commit history without GitHub / GitLab

KK can store and sync the full Git commit history (branches, commits, tree snapshots) through
your existing object-storage remote — Google Drive, a NAS, or any rclone target — with no
GitHub / GitLab account required.

**When does this activate?**  
Automatically whenever `kk push` runs and no `type=git` remote is configured (i.e. you have
not run `kk remote add git …`).

### How it works — incremental bundles

KK uses native **git bundles** (a compact binary format understood by Git itself) to pack and
transport history:

```
<remote-root>/
  history/
    refs.json            ← branch tips + bundle list metadata
    main/
      full.bundle          ← complete history up to the first push
      inc-000001.bundle    ← incremental: commits since full.bundle
      inc-000002.bundle    ← incremental: commits since inc-000001
```

- **First `kk push`** creates `full.bundle` (entire history).
- **Every subsequent `kk push`** creates the next `inc-NNNNNN.bundle` starting from the
  previous tip, keeping each upload small.
- `refs.json` lists the ordered bundle chain and the tip SHA used for the next incremental.

### Push history

```bash
kk push         # uploads objects + creates/uploads incremental history bundle automatically
```

Console output when history is bundled:

```
kk: [origin] creating history bundle (inc-000001.bundle) for branch main...
kk: [origin] uploading history bundle for main...
kk: [origin] history pushed (inc-000001.bundle, branch main)
```

### Fetch history (no merge)

```bash
kk fetch
```

Downloads all new bundles and applies them as remote-tracking refs
(`refs/remotes/kk-history/*`). Local branches are **not** touched.

### Pull history (fetch + merge)

```bash
kk pull                # fetch history bundles then merge current branch
kk pull --no-merge     # fetch bundles only — review before merging
```

If the merge produces conflicts:

```
kk: merge conflict after history fetch — resolve all conflicts,
  stage the results with 'kk stage', then run 'kk commit' to complete the merge.
  To fetch without merging next time, use: kk pull --no-merge
```

### Clone with full history

```bash
kk clone local:/Volumes/NAS/KK/MyGame --history
kk clone drive:<folder-id> --history --pull
```

`--history` downloads all bundles and restores the complete branch + commit graph into
`.kk/git`, replacing the default single-snapshot commit. Without `--history`, `kk clone`
creates a lightweight single commit that is enough for `kk pull-file` to work; history
can be fetched later with `kk fetch`.

### Remote layout for history

```text
<project-folder>/
  history/
    refs.json            ← { default_branch, branches, updated_at, … }
    main/
      full.bundle          ← base bundle (all commits up to first push)
      inc-000001.bundle    ← incremental on top of full.bundle
  objects/…              ← large-file blobs (unchanged)
  manifests/…            ← object manifest (unchanged)
  .kk/…                  ← project file mirror (unchanged)
```

The `history/` folder is created automatically on first push and is never downloaded
into the working tree (it is excluded from `kk clone` file downloads).

---

## Deduplication and retention

Large objects are content-addressed by SHA-256. If two branches point to the same file bytes, they share the same object. KK checks a remote before upload and skips uploading if that remote already has the object.

Objects are retained while any reachable Git commit still contains a pointer to them. `kk objects live` scans all refs with `git rev-list --all`. `kk objects prune --dry-run` shows local cache objects that are no longer reachable.

---

## Object Syncing and Replication

KK supports robust large-file object syncing and replication across multiple configured storage remotes. If you work with multiple remotes (for example, a team `origin` and an offsite `backup`), you can synchronize and actively replicate objects across all push-enabled remotes.

For a comprehensive guide, see [`docs/object-syncing.md`](object-syncing.md).

---

For a full list of term definitions, see [`docs/glossary.md`](glossary.md).
