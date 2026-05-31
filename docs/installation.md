# Installation & Setup

[Home](../README.md) | [Installation](installation.md) | [How To](../How_to.md) | [How It Works](how-it-works.md) | [License](../LICENSE.MD)

This document collects everything needed to get KK working on a developer machine: Git, the KK CLI (build or download), and the optional `rclone` binary used for cloud remotes. It supplements the project's `README.md` with additional platform tips (Windows PowerShell) and developer tooling required by the `Makefile` such as `goversioninfo` and `golangci-lint`.

If you haven't already, read the top-level `README.md` for conceptual docs (how-it-works, remote-layout, pointer-format) and examples. This page focuses on practical, copy-paste steps.

Supported platforms
- Windows (PowerShell / Git Bash / WSL)
- macOS
- Linux

Prerequisites
- Git (core dependency)
- Go toolchain (Go 1.23 or newer as declared in `go.mod`)
- Optional: `rclone` (for rclone-backed remotes such as Google Drive or MEGA)
- Optional developer tools: `goversioninfo` (Windows resource generation), `golangci-lint`, `make` (or run the commands manually)

Quick checks

PowerShell (Windows):

```powershell
git --version
go version
rclone version   # optional
```

If any command fails, install the related tool and ensure it is on your PATH.

1) Install Git

- Windows: Download and install from https://git-scm.com/download/win (Git for Windows). Choose options that add Git to your PATH.
- macOS: Install via Homebrew `brew install git` or from https://git-scm.com.
- Linux: Use your distro package manager, e.g. `sudo apt install git`.

Verify:

```bash
git --version
```

2) Install Go (required to build KK)

- Download from https://go.dev/dl/ and install a Go version >= 1.23 as declared in `go.mod`.
- On Windows you can use the MSI installer; on macOS use the pkg or Homebrew `brew install go`; on Linux use your distro package manager or download the tarball.

Verify:

```bash
go version
```

3) Optional: Install rclone (for rclone-backed remotes)

- rclone is optional if you only use local/NAS remotes or native Google Drive via `kk setup gdrive`. If you plan to use MEGA, S3-like providers, Dropbox, OneDrive, SFTP, or Google Drive via rclone, install `rclone`:

- Windows: Download & unpack from https://rclone.org/downloads/ or use Chocolatey/Scoop:

```powershell
# Chocolatey (if installed)
choco install rclone
# or Scoop
scoop install rclone
```

- macOS: `brew install rclone`
- Linux: follow distro package or https://rclone.org/install/ (curl install script or package manager)

Configure a remote (example: Google Drive):

```bash
rclone config
rclone lsd gdrive:
rclone mkdir gdrive:KK
```

4) Developer extras required by `Makefile` (Windows build specifics)

- goversioninfo: The Makefile runs `goversioninfo` inside `cmd/kk` to create a Windows resource (`resource.syso`) from `versioninfo.json`. Install it if you need to run `make build` on Windows:

```powershell
go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
# Ensure $GOPATH\bin or $(go env GOPATH)/bin is on your PATH so `goversioninfo` is invocable
```

- golangci-lint (optional): used by `make lint`/CI

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

- make: On Windows, use Git Bash/MSYS2/WSL or install GNU Make (choco install make) or run the build commands manually.

5) Build KK (from source)

From the repository root:

Windows (PowerShell):

```powershell
# Option A: direct build
go build -o kk.exe ./cmd/kk

# Option B: use Makefile (preferred for resource embedding)
make build
```

If using `make build` on Windows, `goversioninfo` must be available so `cmd/kk/resource.syso` can be generated.

macOS / Linux:

```bash
go build -o kk ./cmd/kk
# or
make build
```

After build, run:

```bash
# Windows
./kk.exe --version
# macOS/Linux
./kk --version
```

5a) Use a prebuilt executable (recommended for regular users)

If you are a regular user and don't want to build from source, use a prebuilt `kk` executable. This may be shipped with the project (for example `kk.exe` at the repository root) or distributed via release artifacts.

Windows (prebuilt `kk.exe` included in repo or downloaded):

```powershell
# Run directly from the folder that contains kk.exe
.\kk.exe --version

# Register the binary's folder on the user PATH (idempotent)
.\kk.exe install-path

# Alternatively, copy the EXE to a permanent folder and add that folder to your PATH
# Example (manual): create %USERPROFILE%\bin, copy executable, then open a new terminal
New-Item -ItemType Directory -Force "$env:USERPROFILE\bin"
Copy-Item .\kk.exe "$env:USERPROFILE\bin\kk.exe"
# Add to user PATH if not already present (PowerShell - persistent)
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$env:USERPROFILE\bin", "User")
# Open a new terminal to pick up the updated PATH and run
kk --version
```

macOS / Linux (prebuilt `kk` binary):

```bash
# Download or move the prebuilt 'kk' to a directory on your PATH
chmod +x kk
sudo mv kk /usr/local/bin/kk
kk --version
```

Security notes for prebuilt binaries
- Prefer official release artifacts (e.g., GitHub releases) from a trusted publisher.
- Verify checksums or signatures if they are published alongside a release.
- If you run a prebuilt EXE that came from an untrusted source, consider building from source instead.


6) Install `kk` on your PATH (convenience)

The project provides `kk install-path` which edits your user PATH on Windows. From a PowerShell session in the repo root after building:

```powershell
# build first
go build -o kk.exe .\cmd\kk
.\kk.exe install-path
# open a new terminal and verify
Get-Command kk
kk --version
```

On macOS/Linux you may run `./kk install-path` or manually copy the binary to a directory on your PATH (for example `/usr/local/bin`).

7) Verify everything

```bash
git --version
go version
rclone version   # optional, only if you plan to use rclone backends
kk --version
```

8) Configure a remote in KK

There are two ways to add a Google Drive remote. Choose the one that fits your setup:

---

### Option A — Native Google Drive (recommended, no rclone needed)

`kk setup gdrive` handles OAuth, creates the `KK/` root folder on Drive, and
registers the remote in `.kk/config.json` automatically.

```bash
# Run once inside your kk repo (opens a browser for Google OAuth):
kk setup gdrive
# When prompted, give the profile a name (e.g. "work" or "personal").
# The default name is "default" — just press Enter to accept.

# Output:
# Connect Google Drive
# Opening browser for authorization...
# kk: Google Drive auth saved to ~/.config/KK/gdrive/default.json
# kk: Drive project folder-id: 1xR9gP3kTwNq2Lm8vZoUeAb7cYs4FjDh
# kk: Run 'kk push' to create the project folder on Drive.
#     Afterwards, share the project folder ID with teammates:
#     kk clone drive:<project-folder-id>
# Remote ready.

# Add a second Drive remote with a different account:
kk setup gdrive --name drive2 --account work

# Or connect to an existing shared folder directly:
kk setup gdrive --name shared --folder <shared-folder-id> --account work

# Teammates with a single account just clone normally:
kk clone drive:<project-folder-id>

# Teammates with multiple accounts use --account to pick a profile:
kk clone drive:<project-folder-id> --account work
kk clone drive:<project-folder-id> --account personal

# List your saved profiles:
kk accounts

# Delete one saved profile from the local auth cache:
kk accounts --delete personal

# Delete all saved profiles from the local auth cache:
kk accounts --delete-all

# Push your project:
kk push
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

When no `--folder` is specified, KK will use an existing Drive remote's folder structure as a base if one exists.

If you need access to Shared Drives or folders shared by others, authorize with
the full scope instead:

```bash
kk setup gdrive --scope full
```

> **Note:** `kk setup gdrive` registers the remote for you. You do **not** need
> to run `kk remote add` separately when using the native Drive backend.

---

### Option B — Google Drive via rclone

Use this if you already use rclone or want more control over the rclone config.

**Step 1 — configure rclone** (one-time, if not already done):

```bash
rclone config
# Suggested values:
#   name>    gdrive
#   storage> drive
#   scope>   drive
```

Verify the rclone remote works:

```bash
rclone lsd gdrive:
```

**Step 2 — add the remote to KK:**

```bash
# --remote must point to the PROJECT folder, not just a KK root.

# Minimal form — only required flags:
kk remote add rclone gdrive --remote gdrive:KK/MyGame --push true --pull true
# remote added gdrive
#     Teammates can clone with:
#     kk clone rclone:gdrive:KK/MyGame

# Full form — with optional metadata flags:
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

> `--remote gdrive:KK/MyGame` points **directly to the project folder**.
> kk push/pull operate at that path without inserting any additional subfolder.

**Step 3 — push:**

```bash
kk push
```

---

### Local NAS / external drive

```bash
# --path must point to the PROJECT folder, not just a KK root.
kk remote add local nas --path /Volumes/NAS/KK/MyGame --push true --pull true
# remote added nas
#     Teammates can clone with:
#     kk clone local:/Volumes/NAS/KK/MyGame

kk push
```

9) Running smoke tests and Makefile targets

- The project provides `scripts/smoke-test.sh` which is Bash-based — run it from Git Bash or WSL on Windows.
- Makefile useful targets (from repo root):

```bash
make build   # compile (runs goversioninfo on Windows)
make test    # go test ./...
make smoke   # ./scripts/smoke-test.sh
make fmt     # gofmt -w ./cmd ./internal
make lint    # golangci-lint run ./...
make clean   # remove built artifacts
```

Troubleshooting
- PATH issues: ensure `$GOPATH/bin` or `$(go env GOBIN)` is on your PATH so `go install`-ed tools are runnable.
- goversioninfo not found: either install or run `go build -o kk.exe ./cmd/kk` directly (skips resource embedding) if you don't need the Windows icon/version resource.
- If running `make` on Windows fails with `make: command not found`, use Git Bash, WSL, or run the build/test commands manually from PowerShell.
- If `rclone` remotes fail, test them directly with `rclone lsd <remote>:` and ensure network/auth is working.

Further reading and references
- `README.md` (root) — conceptual and command examples (already includes rclone and Google Drive samples)
- `docs/limitations.md` — what KK does and does not do (no hosted service, no server)
- `docs/object-syncing.md` — syncing and replicating large-file objects across multiple remotes
- `docs/remote-layout.md` — explains remote folder layout
- `docs/how-it-works.md` — internals and object lifecycle
- `docs/pointer-format.md` — pointer file format used inside Git
- `docs/wrapper-checklist.md` — expected behavior for wrapper commands like `kk add`, `kk commit`, `kk push`

If you want, I can also:
- add a small troubleshooting FAQ (common errors and commands)
- add a Windows-specific quickstart script that builds and registers `kk` automatically

