# KK

KK is a Go-first, Git-compatible large-file version-control wrapper for game projects.

It keeps normal source code in Git and stores large binary assets as verified SHA-256 objects in one or more remotes such as a NAS/local folder, Google Drive, MEGA, or any `rclone`-supported provider.

> **What KK is not:** KK does not require GitHub, GitLab, or any other
> git hosting service. It has no server and no pull requests. It is a
> client-side tool for managing large files and project snapshots on your own
> storage driver. Optionally, you can push pointer history to GitHub / GitLab
> with `kk remote add git`. Core pointer-history push, pull, clone, and branch
> workflows have been thoroughly tested on GitHub and GitLab. Pull-request,
> CI, webhook, and branch protection workflows are handled by the git host and
> are not managed by KK. See
> [`docs/limitations.md`](docs/limitations.md) for a full breakdown.

### Push / Pull in one line

| Intent | Command |
|---|---|
| Upload your work to the driver | `kk push` |
| Download large files after cloning | `kk pull-file .` |
| Clone on a new machine | `kk clone drive:<id> --pull` |
| Sync pointer history to GitHub/GitLab | `kk pull` _(requires `kk remote add git`)_ |
| Replicate/sync objects across remotes | `kk objects sync` or `kk pull --sync` |
| Add a GitHub/GitLab remote | `kk remote add git github https://github.com/your-username/MyGame.git` |
| Switch from storage bundles → GitHub | `kk remote migrate to-git github https://github.com/your-username/MyGame.git` |
| Switch from GitHub → storage bundles | `kk remote migrate to-storage --yes` |

The CLI command is:

```bash
kk
```

The name can be treated as **KK** publicly. Internally, the original idea was **Kernel Kommando**.

See [`docs/glossary.md`](docs/glossary.md) for definitions of all core terms (Driver, Pointer, Object, Manifest, etc.).

## Current goal

This repository is the CLI/core foundation.

The important rules are:

1. Git remains the source of truth for commits, branches, merges, and history.
2. KK controls how large files enter Git history.
3. Git commits pointer files for large assets, not the real bytes.
4. Large objects are stored by SHA-256.
5. Remote upload/download must verify SHA-256 and byte size.
6. An object must not be pruned while any reachable Git commit still points to it.


## Contributing & Commit messages

Please follow the project's commit message conventions documented in `docs/commit-message-guide.md`. The guide is KK-specific and explains how to annotate pointer commits, include pointer SHAs and sizes in commit bodies, and how to indicate upload status for large assets.

A minimal commitlint config has been added as `commitlint.config.js` to help enforce Conventional Commits in CI or local tooling. For a quick Windows PowerShell setup (developer convenience), see the full instructions in `docs/commit-message-guide.md`, or use the snippet:

```powershell
# from repo root - optional developer steps
npm init -y
npm install --save-dev @commitlint/cli @commitlint/config-conventional husky commitizen
npx husky install
npx husky add .husky/commit-msg "npx --no -- commitlint --edit $1"
```

Using local hooks is optional but recommended for faster feedback; CI enforcement is strongly encouraged.

## Project structure

```text
kk/
  cmd/
    kk/
      main.go              CLI executable entry point

  internal/
    app/                   command orchestration and output
      app.go
      init.go
      status.go
      track.go
      add.go
      commit.go
      push.go
      pull.go
      clone.go
      fsck.go
      diff.go
      remote.go
      objects.go
      stage.go
      progress.go          terminal progress bars (ProgressBar, MultiProgressBar)
      workers.go           resolveWorkers — --workers flag / KK_WORKERS env

    core/                  KK models, config, repo rules, pointer format
      constants.go
      types.go
      config.go
      repo.go
      pointer.go
      track.go
      code.go

    git/                   Git adapter using .kk/git
      client.go

    storage/               local SHA-256 object storage
      hash.go
      paths.go
      local.go

    remote/                remote drivers and manifests
      remote.go
      local.go
      rclone.go
      drive.go
      manifest.go
      sync.go
      concurrent.go        ConcurrentFiles worker-pool helper

    ignore/                .kkignore parsing and matching
      matcher.go

    registry/              user project registry
      projects.go

  docs/
    how-it-works.md
    remote-layout.md
    pointer-format.md
    data-deduplication.md
    ui-integration.md
    wrapper-checklist.md
    commit-message-guide.md
    installation.md

  scripts/
    smoke-test.sh

  README.md
  Makefile
  LICENSE
  go.mod
```

## Installation

KK depends on:

1. `git` for commits, branches, merges, and history
2. `rclone` only for rclone-backed cloud/object remotes such as MEGA, S3, Dropbox, or Google Drive via rclone
3. `kk` itself, built from this repository

Native Google Drive support uses `kk setup gdrive` and does not require rclone.

### 1. Install Git

Verify Git is installed:

```bash
git --version
```

If that command fails, install Git first, then confirm it is available on your
PATH before continuing.

### 2. Install rclone (optional)

Verify `rclone` is installed:

```bash
rclone version
```

If that command fails, install and configure `rclone` outside KK only when you
plan to use `kk remote add rclone`. KK expects a working `rclone` binary on your
PATH unless you explicitly configure a custom binary path.

Create and test a remote:

```bash
rclone config
```

Example remote names:

```text
gdrive
mega
```

Test Google Drive:

```bash
rclone lsd gdrive:
rclone mkdir gdrive:kk-lfs/my-game
```

Test MEGA:

```bash
rclone lsd mega:
rclone mkdir mega:kk-lfs/my-game
```

### 3. Build and install kk

#### Windows

From the project root:

```bash
go build -o kk.exe ./cmd/kk
```

Run it:

```bash
./kk.exe version
```

or:

```bash
./kk.exe --version
```

Register `kk` on your user PATH so you can run it from any directory:



**Windows (recommended — edits user PATH in registry automatically):**

```powershell
# build first
go build -o kk.exe .\cmd\kk

# register on user PATH (idempotent — safe to run more than once)
.\kk.exe install-path
```

Open a **new terminal** and verify:

```powershell
Get-Command kk
kk
```



`kk install-path` is **idempotent** on Windows: if the directory is already
present in the user PATH it prints a confirmation and makes no changes.

---

#### Linux/macOS/Git Bash

```bash
go build -o kk ./cmd/kk
```

Run:

```bash
./kk
```

Optionally add it to your PATH:

```bash
./kk install-path
```

### 4. Verify the toolchain

After installation, confirm all required tools are available:

```bash
git --version
rclone version
kk
```

## Makefile usage

If `make` is installed:

```bash
make build
make test
make smoke
make fmt
make lint
make clean
```

The Makefile means:

```makefile
build:
	go build -o kk ./cmd/kk

test:
	go test ./...

smoke:
	./scripts/smoke-test.sh

fmt:
	gofmt -w ./cmd ./internal

lint:
	golangci-lint run ./...

clean:
	rm -f kk kk.exe
```

If Git Bash says `make: command not found`, run the commands manually:

```bash
go build -o kk.exe ./cmd/kk
go test ./...
gofmt -w ./cmd ./internal
golangci-lint run ./...
./scripts/smoke-test.sh
```

The smoke test is Bash-based, so on Windows use Git Bash, MSYS2, or WSL for `scripts/smoke-test.sh`.

### Windows PowerShell (no `make` required)

A `make.ps1` script is included at the repository root as a drop-in replacement for `make` on Windows. Run it from a PowerShell terminal in the project root:

```powershell
# Instead of: make build
.\make.ps1 build

# Build GitHub release assets: kk.exe, portable zip, checksums, release notes
.\make.ps1 build-all

# Instead of: make test
.\make.ps1 test

# Instead of: make lint
.\make.ps1 lint

# Instead of: make clean
.\make.ps1 clean

# Instead of: make portable-windows
.\make.ps1 portable-windows
```

`.\make.ps1 build-all` creates:

```text
dist/kk.exe
dist/kk.zip
dist/kk-portable.zip
dist/SHA256SUMS.txt
dist/GITHUB_RELEASE.md
```

Upload `kk.exe`, `kk.zip`, `kk-portable.zip`, and `SHA256SUMS.txt` as GitHub release assets. Copy the contents of `dist/GITHUB_RELEASE.md` into the GitHub release description.

### Windows code signing

`make.ps1 build`, `make.ps1 portable`, and `make.ps1 portable-windows` can sign the generated `.exe` with Microsoft `signtool`. Signing is optional and is skipped unless a certificate is configured through environment variables or `.env`.

For a PFX certificate:

```powershell
$env:KK_SIGN_CERT_PATH = "C:\certs\kk-code-signing.pfx"
$env:KK_SIGN_CERT_PASSWORD = "pfx-password"
.\make.ps1 portable
```

For a certificate already installed in the Windows certificate store:

```powershell
$env:KK_SIGN_CERT_SHA1 = "certificate-thumbprint"
.\make.ps1 build
```

Optional settings are `KK_SIGNTOOL_PATH`, `KK_SIGN_TIMESTAMP_URL`, `KK_SIGN_DIGEST_ALG`, and `KK_SIGN_MACHINE_STORE`. See `.env.example` for the full list.

## Quick start

Create a project:

```bash
mkdir my-game
cd my-game
../kk.exe init
```

On non-Windows:

```bash
../kk init
```

`kk init` creates:

```text
.kk/
  git/          Git database; replaces root .git/
  objects/      local large-object cache
  tmp/
  logs/
  repo.json     stable project identity
  config.json   local KK config and remotes
  tracks.json   large-file tracking patterns
.kkignore       KK ignore rules
```

KK does not create a root `.git/` folder and does not create a root `.gitignore` file.

Internally, it calls Git like this:

```bash
git --git-dir=.kk/git --work-tree=. <command>
```

## Default branch

New repositories should use:

```text
main
```

not `master`.

The Git initialization flow should prefer:

```bash
git init --initial-branch=main
```

and fall back to:

```bash
git init
git symbolic-ref HEAD refs/heads/main
```

for older Git versions.

## Tracking large files

Track file patterns:

```bash
kk track "*.mp4"
kk track "*.zip"
kk track "*.psd"
kk track "*.fbx"
```

List tracked patterns:

```bash
kk track list
```

Untrack a pattern:

```bash
kk untrack "*.mp4"
```

Tracked patterns are stored in:

```text
.kk/tracks.json
```

## Adding files

Use:

```bash
kk add <file-or-folder>
```

not raw `git add` for large files.

When `kk add` sees a file matching a tracked pattern:

1. It hashes the real file with SHA-256.
2. It stores the real bytes in `.kk/objects/`.
3. It verifies the local object.
4. It replaces the working file with a text pointer.
5. It stages the pointer with Git.

Pointer format:

```text
version kk-lfs-1.0.0
oid sha256:<64 lowercase hex chars>
size <decimal byte count>
```

Example:

```text
version kk-lfs-1.0.0
oid sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
size 104857600
```

The real object path is:

```text
.kk/objects/9f/86/9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08
```

## Commit and push

Commit:

```bash
kk commit -m "Add assets"
```

Push:

```bash
kk push
```

Push to one remote:

```bash
kk push --remote studio-nas
```

Push to all push-enabled remotes:

```bash
kk push --all-remotes
```

Control how many files and objects are uploaded/synced in parallel (default: 4):

```bash
kk push --workers 8          # more parallelism for fast NAS or high-bandwidth links
kk push --workers 1          # serial uploads for debugging or rate-limited remotes
KK_WORKERS=8 kk push         # set via environment variable instead of flag
```

The `--workers` flag and `KK_WORKERS` environment variable apply to both the file-sync phase and the object-upload phase. The `--workers` flag takes precedence over `KK_WORKERS`.

Before pushing Git history, KK should upload and verify all required large objects.

## Download on demand

Materialize a pointer file back into real bytes:

```bash
kk pull-file Assets/intro.mp4
kk pull-file --force Assets/intro.mp4
```

`kk pull-file` downloads the object only if it is missing from the local cache.
`kk pull-file --force` re-downloads the object even if it is already cached
locally.

If the working file is already materialized, `--force` falls back to the
pointer stored in `HEAD`, then re-downloads, re-verifies, and rewrites the
working copy.

During download, KK prints:
- which pull remotes it is checking
- when a remote does not have the object
- when a remote cannot be used
- which remote ultimately served the object

Turn a materialized file back into a pointer:

```bash
kk dematerialize Assets/intro.mp4
```

`kk dematerialize` must refuse to overwrite a file if the materialized bytes no longer match the committed object. This protects unsaved asset edits.

## Multi-remote config

`.kk/config.json` supports arbitrary remote names.

Remote names like `studio-nas`, `gdrive-backup`, and `mega-archive` are user-defined keys. Behavior comes from fields such as `type`, `role`, `provider`, `pull`, `push`, and `priority`.

For downloads, pull-enabled remotes are tried in priority order with fallback.

Example full config:

```json
{
  "version": "kk-local-config-1.0.0",
  "default_remote": "studio-nas",
  "remotes": {
    "studio-nas": {
      "type": "local",
      "display_name": "Studio NAS",
      "role": "primary",
      "provider": "nas",
      "path": "\\\\STUDIO-NAS\\kk-lfs\\my-game",
      "object_root": "objects",
      "manifest_root": "manifests",
      "verify_mode": "local-hash",
      "priority": 10,
      "pull": true,
      "push": true,
      "tags": ["primary", "studio", "lan"]
    },
    "gdrive-backup": {
      "type": "rclone",
      "display_name": "Google Drive Backup",
      "role": "backup",
      "provider": "google-drive",
      "binary": "rclone",
      "remote": "gdrive:kk-lfs/my-game",
      "object_root": "objects",
      "manifest_root": "manifests",
      "verify_mode": "download",
      "priority": 20,
      "pull": true,
      "push": true,
      "tags": ["backup", "cloud", "google"]
    },
    "mega-archive": {
      "type": "rclone",
      "display_name": "MEGA Archive",
      "role": "archive",
      "provider": "mega",
      "binary": "rclone",
      "remote": "mega:kk-lfs/my-game",
      "object_root": "objects",
      "manifest_root": "manifests",
      "verify_mode": "download",
      "priority": 50,
      "pull": true,
      "push": false,
      "tags": ["archive", "backup", "cold-storage"]
    }
  }
}
```

## Remote fields

```text
type
  Driver type: local, rclone, drive, ssh later.

display_name
  Human-readable name for command output and future tooling.

role
  Intended use: primary, backup, archive, cache, mirror.

provider
  Storage provider: nas, google-drive, mega, s3, custom, ssh.

binary
  For rclone remotes. Can be:
    rclone
    C:\Tools\rclone\rclone.exe

remote
  Rclone target, for example:
    gdrive:kk-lfs/my-game
    mega:kk-lfs/my-game

path
  Local driver root, for example:
    \\STUDIO-NAS\kk-lfs\my-game
    Z:\kk-lfs\my-game
    E:\kk-lfs\my-game

object_root
  Folder under the remote root where SHA-256 objects are stored.

manifest_root
  Folder under the remote root where per-project manifests are stored.

verify_mode
  download      for rclone; strict and backend-independent
  local-hash    for local/NAS
  remote-hash   for SSH servers with sha256sum

priority
  Lower number is tried first for pull fallback.

pull
  Whether KK can download from this remote.

push
  Whether KK can upload to this remote.

tags
  Flexible labels for filtering and grouping.
```

## Native Google Drive setup & Multiple Accounts

For the regular-user flow, run:

```bash
kk setup gdrive [--name <remote-name>] [--account <profile>] [--folder <folder-id>] [--scope <file|full>]
```

By default, KK requests a narrow application-specific scope (`--scope file`, mapping to Google's `drive.file` scope), which restricts KK's access to only files and folders that KK itself creates or opens.

If you need to clone, pull, or push files located on **Shared Drives (Team Drives)** or **shared folders** that another Google account has shared with you, authorize KK with the full drive access scope:

```bash
kk setup gdrive --scope full
```

Pass `--account <profile>` to skip the interactive account selection menu and use a saved profile directly:

```bash
kk setup gdrive --account work
```

### Multiple Drive Remotes

You can add multiple Google Drive remotes to the same project with different names:

```bash
# First Drive remote
kk setup gdrive --name drive1 --account personal

# Second Drive remote (uses drive1's folder structure as base)
kk setup gdrive --name drive2 --account work

# Third Drive remote with specific folder
kk setup gdrive --name backup --folder ABC123 --account shared
```

When no `--folder` is specified, KK will use an existing Drive remote's folder structure as a base if one exists. This makes it easy to set up additional storage remotes while keeping all project files organized under the same Drive hierarchy.

### Using an Existing Shared Folder

If you have access to a shared Drive folder, you can connect to it directly:

```bash
kk setup gdrive --name shared --folder <shared-folder-id> --account work
```

At runtime, KK will ask for a Google OAuth desktop client ID if
`KK_GOOGLE_CLIENT_ID` is not already set. You can also preconfigure:

```bash
KK_GOOGLE_CLIENT_ID=...
KK_GOOGLE_CLIENT_SECRET=...
```

The setup flow opens the browser, authorizes Drive access with the selected scope, creates a project
folder under `KK/` in Google Drive, tests upload/download, and saves a native
`drive` remote in `.kk/config.json`.


### Multi-Account Support

KK supports managing and switching between multiple Google Drive accounts:

1. **Interactive Detection**: When you run `kk setup gdrive`, KK scans for existing local account profiles. If any are found, it lists them and prompts you to choose whether to use an existing account or log in to a new one online.
2. **Non-Interactive Selection**: Pass `--account <profile>` to skip the menu entirely and use a named saved account directly (e.g., `kk setup gdrive --account work`). An error is returned if the named profile does not exist.
3. **Connectivity Verification**: Selecting an existing account automatically validates its authorization online. If it has expired or failed, KK offers to log you in again online.
4. **Custom Profile Naming**: When connecting a new account, KK prompts you for a profile name (defaulting to `default`). You can specify any alphanumeric name (plus underscores/hyphens) to keep different accounts separate (e.g., `work`, `personal`).
5. **Cloning Auto-Resolution**: When running `kk clone drive:<project-folder-id>`, if the `default` profile is not set up but other profiles exist:
   - If only one account profile exists, KK automatically selects and uses it.
   - If multiple profiles exist, KK interactively prompts you to choose which account to use.

### Managing Accounts

To view all locally cached Google Drive account profiles, run:

```bash
kk accounts [--json]
kk accounts --delete <profile> [--json]
kk accounts --delete-all [--json]
```

This displays a table (or JSON) of saved profiles, including their configuration profiles, cached email addresses, display names, client IDs, and verification status:

Use `--delete <profile>` to remove one cached Google Drive account profile, or `--delete-all` to remove every cached profile from the local auth cache.

- **Active**: The account is authenticated and online.
- **Active (Offline)**: The account is authenticated, but currently offline (using cached access tokens before expiry).
- **Offline (Expired)**: The credentials are valid but expired, requiring online refresh.
- **Invalid Credentials**: Authorization has been revoked or is invalid.
- **Corrupted**: The local profile JSON file is invalid or corrupted.

## Configure an rclone remote in KK

After `rclone` is installed and the remote works, configure it in KK:

### Sample: Google Drive

Create the `rclone` remote:

```bash
rclone config
```

Suggested values:

```text
name> gdrive
storage> drive
scope> drive
root_folder_id>
service_account_file>
```

Verify it:

```bash
rclone lsd gdrive:
rclone mkdir gdrive:kk-lfs/my-game
```

Add it to KK:

```bash
kk remote add rclone gdrive-backup \
  --display-name "Google Drive Backup" \
  --role backup \
  --provider google-drive \
  --binary rclone \
  --remote gdrive:kk-lfs/my-game \
  --verify-mode download \
  --priority 20 \
  --pull true \
  --push true \
  --tag backup \
  --tag cloud
```

### Sample: MEGA

Create the `rclone` remote:

```bash
rclone config
```

Suggested values:

```text
name> mega
storage> mega
user> your-email@example.com
pass> ********
```

Verify it:

```bash
rclone lsd mega:
rclone mkdir mega:kk-lfs/my-game
```

Add it to KK:

```bash
kk remote add rclone mega-archive \
  --display-name "MEGA Archive" \
  --role archive \
  --provider mega \
  --binary rclone \
  --remote mega:kk-lfs/my-game \
  --verify-mode download \
  --priority 50 \
  --pull true \
  --push true \
  --tag archive \
  --tag cloud
```

For a custom rclone path on Windows:

```json
"binary": "C:\\Tools\\rclone\\rclone.exe"
```

## Switching history mode (`kk remote migrate`)

KK stores commit history in exactly one of two ways:

| Mode | How history travels | Activated when |
|---|---|---|
| **Storage bundles** | `full.bundle` + `inc-*.bundle` uploaded to your object remote | No `type=git` remote is configured |
| **Git remote** | `git push` / `git pull` to GitHub, GitLab, etc. | A `type=git` remote exists |

`kk remote migrate` lets you switch between modes at any time without losing history.

### Direction 1 — Storage bundles → Git remote (`to-git`)

Use this when the project currently stores history as bundle files and you want to
move to a proper git hosting platform so teammates can browse commits on GitHub/GitLab.

```bash
# Add GitHub as the git-history remote and immediately push all local branches.
kk remote migrate to-git github https://github.com/your-username/MyGame.git
```

Sample output:

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

**After migration:**
- `kk push` runs `git push github` for pointer history; binary objects still go to the object remote.
- Existing `history/<branch>/*.bundle` files on the object remote are left intact but ignored.
- Teammates clone with: `kk clone git:https://github.com/your-username/MyGame.git --pull`

Optional flags:

```bash
# Read-only mirror — pull from GitHub but never push back
kk remote migrate to-git mirror https://github.com/your-username/MyGame.git --push false --pull true
```

---

### Direction 2 — Git remote → Storage bundles (`to-storage`)

Use this when the project has a GitHub/GitLab remote and you want to stop using it
— for example, to go fully offline, move to a private NAS, or leave a git host.

```bash
# Preview what will happen, then confirm interactively
kk remote migrate to-storage
```

Sample output:

```
kk: This will:
kk:   1. Upload a full history bundle to: nas
kk:   2. Remove git remote(s): github (https://github.com/your-username/MyGame.git)
kk:   Future 'kk push' will upload incremental bundles instead of using git push.

Proceed? [y/N] y
kk: creating initial history bundle(s)...
kk: [nas] creating history bundle (full.bundle)...
kk: [nas] uploading history bundle...
kk: [nas] history pushed (full.bundle, branch main)
kk: [nas] history pushed (full.bundle, branch feature)
kk: default remote set to "nas"

kk: migration complete → commit history will now travel via storage bundles
kk: teammates can restore history with: kk clone <spec> --history
```

**After migration:**
- `kk push` creates incremental bundles on the object remote on every push.
- `kk clone local:/NAS/KK/my-game --history` restores all branches for teammates.
- If you had multiple git remotes (e.g. github + gitlab), pass `--remote <name>` to target just one.

Skip the confirmation prompt in scripts:

```bash
kk remote migrate to-storage --yes

# Target only a specific git remote
kk remote migrate to-storage --remote gitlab --yes
```

---

### Safety guarantees

| Situation | Behaviour |
|---|---|
| `to-git` called when a git remote **already exists** | **No-op** — prints which remote(s) are already registered and exits cleanly |
| `to-git` git URL unreachable | Error — config is **not changed** |
| `to-git` initial push fails (e.g. auth not set up) | Non-fatal warning + recovery command printed |
| `to-storage` called when **no git remote exists** | **No-op** — prints "already in storage-bundle mode" and exits cleanly |
| `to-storage` no non-git push remote found | Error — nowhere to store the bundles |
| `to-storage` bundle upload fails | Error — config is **not changed** (safe rollback) |
| User answers `N` at the confirm prompt | Cancelled — no changes made |

---

## Remote layout

Every remote uses the same logical layout:

```text
<remote-root>/
  kk-remote.json
  history/
    refs.json
    main/
      full.bundle
      inc-000001.bundle
    feature/
      full.bundle
  objects/
    ab/
      cd/
        abcdef...
  manifests/
    <repo_id>.json
```

Objects are immutable and addressed by SHA-256.

Manifests are useful for status and UI, but downloaded bytes must always be verified locally before use.

## Remote upload deduplication

Same bytes means same SHA-256.

Before upload, KK checks whether the target remote already has the object:

```text
for each required object:
  if remote has sha256 object with matching size/hash:
    skip upload
    ensure manifest entry
  else:
    upload
    verify
    ensure manifest entry
```

This prevents duplicate uploads when two branches or two files point to the same bytes.

## Branch-safe object retention

Example:

```text
Branch A deletes Assets/tree.fbx and commits.
Branch B still contains Assets/tree.fbx.
Both branches previously pointed to sha256:abc123.
```

The object `abc123` must remain because Branch B still references it.

KK must not delete an object while any reachable Git commit still points to it.

Reachable refs include:

```text
branches
tags
remote-tracking branches
other refs included by git rev-list --all
```

Commands:

```bash
kk objects live
kk objects live --json

kk objects refs <sha256>
kk objects refs <sha256> --json

kk objects prune --dry-run
kk objects prune --dry-run --json
kk objects prune

kk objects sync [--workers N] [--verbose]
```

`kk objects prune` should only delete local cache objects that are not referenced by any reachable Git commit.

`kk objects sync` scans all live large-file objects across all branches, tags, and commits in your repository, verifies their presence on all push-enabled non-Git remotes, and replicates any missing objects. For more details on active replication and on-demand sync-on-pull (`--sync`), see [`docs/object-syncing.md`](docs/object-syncing.md).

Remote prune is intentionally not automatic in the MVP.

## Code-only diff

Future UI should not show pointer noise for assets.

Use:

```bash
kk diff --code --summary --json
kk diff --code --file src/main.go
```

Intended behavior:

```text
Code Changes
  show normal source diffs

Asset Changes later
  show file path, old oid, new oid, size, upload status
```

## Current command checklist

Core:

```bash
kk init
kk status [--json]
kk track "*.mp4"
kk track list
kk untrack "*.mp4"
kk add <file-or-dir...>
kk commit -m "message"
kk push [--remote name] [--all-remotes] [--workers N] [--sync-working-dir]
kk pull [--sync] [--workers N]
kk pull-file [--force] [--sync] [--workers N] <file...>
kk dematerialize <file...>
kk fsck [--json]
```

Objects:

```bash
kk objects live [--json]
kk objects refs <sha256> [--json]
kk objects prune [--dry-run] [--json]
kk objects sync [--workers N] [--verbose]
```

Diff:

```bash
kk diff --code --summary [--json]
kk diff --code --file <path>
```

Remotes:

```bash
kk setup gdrive [--name <remote-name>] [--account <profile>] [--folder <folder-id>] [--scope <file|full>]
kk accounts [--json]
kk accounts --delete <profile> [--json]
kk accounts --delete-all [--json]
kk remote add local <name> ...
kk remote add rclone <name> ...
kk remote add git <name> <url>
kk remote list [--json]
kk remote set-default <name>
kk remote check <name|--all> [--json]
kk remote remove <name>
kk remote rename <old> <new>
kk remote migrate to-git <name> <url> [--push true|false] [--pull true|false]
kk remote migrate to-storage [--remote <name>] [--yes]
```

Staging helpers:

```bash
kk stage <path...>
kk unstage <path...>
kk discard <path...>
```

Raw Git passthrough:

```bash
kk git <raw git args>
kk log
kk branch
kk checkout <branch>
kk switch <branch>
```

Setup:

```bash
kk install-path
```

All passthrough commands must still inject:

```bash
git --git-dir=.kk/git --work-tree=.
```

## Wrapper priorities still to harden

Critical wrappers:

```text
kk add
  Must prevent raw large binaries from entering Git.

kk commit
  Must validate staged files before commit.

kk push
  Must upload/verify objects before Git push.

kk pull
  Must check object health after Git pull.

kk checkout / kk switch
  Now auto-materializes pointer files from the new HEAD after a successful switch.
  Still needs safety rules for dirty materialized files before branch changes.

kk reset / kk clean
  Must avoid deleting unverified object cache or unsaved materialized assets.

kk merge / kk rebase
  Must handle pointer conflicts and missing objects.
```

## Project identity for moved folders

Every KK repo should have:

```text
.kk/repo.json
```

Example:

```json
{
  "repo_id": "stable-uuid",
  "name": "MyGame",
  "created_at": "2026-05-20T00:00:00Z"
}
```

The project registry stores known KK projects in:

```text
%APPDATA%\KK\projects.json
```

Use `kk project connect [path]`, `kk project reimport [path]`, and
`kk project list [--json]` to refresh or inspect that registry. If a project
folder moves, reimport the new folder and KK will match it by `.kk/repo.json`
`repo_id`.

## Suggested next tasks for Codex

Use this order when continuing development locally:

1. Run `go test ./...` and fix any compile errors.
2. Run `scripts/smoke-test.sh` in Git Bash/WSL.
3. Confirm `kk init` creates `.kk/git`, `.kk/repo.json`, `.kk/config.json`, `.kk/tracks.json`, and `.kkignore`.
4. Confirm `kk add` converts tracked large files into pointers.
5. Harden `kk commit` so it blocks staged large binaries that are not KK pointers.
6. Confirm remote upload dedup: skip upload when the remote already has the SHA-256 object.
7. Confirm `kk objects live` scans all refs, not only HEAD.
8. Confirm `kk objects prune --dry-run` preserves objects still referenced by another branch.
9. Add or harden rclone driver tests using a local test remote if possible.
10. Add JSON output consistency for `status`, `fsck`, `remote list`, `remote check`, `objects`, and `diff`.
11. Add `kk checkout` / `kk switch` safety for dirty materialized files.
12. Add `kk reset` / `kk clean` safety.
13. Add asset diff view: `kk diff --assets --json`.
14. Add JSON Lines progress for long upload/download commands.

## Development rules for Codex

Keep these invariants:

```text
Pointer files are what Git stores for large assets.
Real large bytes live in .kk/objects and remotes.
Object identity is SHA-256 + size.
Never trust a remote object without verification.
Never delete an object while any reachable Git commit references it.
Do not let raw large binaries enter Git history.
```

Preferred package boundaries:

```text
cmd/kk
  tiny executable only

internal/app
  command parsing, orchestration, text/json output

internal/core
  data models and KK rules

internal/git
  all git exec calls

internal/storage
  local objects and hashing

internal/remote
  local/rclone/ssh drivers and manifests

internal/ignore
  .kkignore behavior

internal/registry
  user project registry for kk project connect/reimport/list
```

Avoid putting Git exec calls outside `internal/git`.
Avoid putting rclone exec calls outside `internal/remote`.
Avoid putting UI-specific behavior into `internal/core`.

## License note

KK is free software licensed under **AGPL-3.0-or-later**.

See [LICENSE.MD](LICENSE.MD) for the complete GNU Affero General Public License text and [DCO.md](DCO.md) for contribution certification requirements.

## Privacy Policy

Please review our [Privacy Policy](PRIVACY.md) to understand how KK operates entirely locally, collects zero telemetry, and safely manages integrations like Google Drive OAuth while shielding contributors from liability for user data or third-party storage outcomes.
