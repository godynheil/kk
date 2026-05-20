# KK Version Control

`kk` is a Git-compatible large-file version control tool designed for game projects.

It keeps source code and small text files in Git, while storing large binary assets in verified object storage such as NAS folders, SSH servers, Google Drive, MEGA, S3, or any storage backend supported by `rclone`.

KK Version Control is intended for projects that contain both code and large assets, such as:

- game projects;
- animation projects;
- 3D asset libraries;
- media-heavy software projects;
- teams that need download-on-demand asset workflows.

## Status

KK Version Control is in early development.

The current design focuses on:

- a Go-based CLI named `kk`;
- Git metadata stored inside `.kk/git`;
- no required root `.git/` folder;
- large-file pointer files committed to Git;
- content-addressed asset objects stored by SHA-256;
- download-on-demand materialization;
- verified upload and download;
- flexible multi-remote configuration;
- future Flutter desktop UI support.

## Core Idea

Git is good at versioning code and small text files. It is not ideal for large binary assets.

KK Version Control uses Git for history, branches, commits, and diffs, but stores large binary files outside Git.

A large file committed through `kk` becomes a small pointer file:

```text
version kk-lfs-v1
oid sha256:9f86d081884c7d659a2feaa0c55ad015...
size 104857600
```

The real file is stored separately:

```text
.kk/objects/9f/86/9f86d081884c7d659a2feaa0c55ad015...
```

Remote object storage uses the same content-addressed layout:

```text
remote-root/
  kk-remote.json
  objects/
    9f/
      86/
        9f86d081884c7d659a2feaa0c55ad015...
  manifests/
    <repo_id>.json
```

## Project Layout

A KK project looks like this:

```text
my-game/
  .kk/
    git/
    objects/
    tmp/
    logs/
    repo.json
    config.json
    tracks.json

  .kkignore
  Assets/
  Source/
  README.md
```

KK does not need to create a normal root `.git/` directory. Git metadata is stored in:

```text
.kk/git/
```

Internally, `kk` calls Git like this:

```bash
git --git-dir=.kk/git --work-tree=. status
git --git-dir=.kk/git --work-tree=. add file
git --git-dir=.kk/git --work-tree=. commit -m "message"
```

## Why Not Git LFS?

KK Version Control is not Git LFS.

Git LFS is a standard and mature solution for large files. KK explores a different workflow:

- no required Git LFS installation;
- custom object remotes;
- multiple remotes per project;
- rclone-based cloud storage support;
- download-on-demand materialization;
- game-project-focused asset workflows;
- future CLI/TUI/Flutter UI integration.

Use Git LFS if you need compatibility with existing Git LFS hosting.

Use KK if you want a custom Git-compatible workflow for code plus large assets, especially with flexible storage targets.

## Installation

Build from source:

```bash
go build -o kk ./cmd/kk
```

On Windows:

```powershell
go build -o kk.exe ./cmd/kk
```

Then place `kk` or `kk.exe` somewhere on your `PATH`.

## Requirements

Required:

- Go, for building from source;
- Git, installed separately and available on `PATH`.

Optional:

- `rclone`, for Google Drive, MEGA, Dropbox, OneDrive, S3, Backblaze B2, and other cloud storage providers.

KK invokes Git and rclone as external tools. It does not relicense them.

## Basic Usage

Create a new project:

```bash
mkdir my-game
cd my-game
kk init
```

Track large file patterns:

```bash
kk track "*.mp4"
kk track "*.zip"
kk track "*.psd"
kk track "*.fbx"
kk track "*.blend"
kk track "*.wav"
```

Add files:

```bash
kk add Source/main.go
kk add Assets/intro.mp4
```

Commit:

```bash
kk commit -m "Initial project"
```

Check status:

```bash
kk status
```

Pull a large file on demand:

```bash
kk pull-file Assets/intro.mp4
```

Turn a materialized file back into a pointer:

```bash
kk dematerialize Assets/intro.mp4
```

Verify local and remote object integrity:

```bash
kk fsck
```

## Download-on-Demand Workflow

KK is designed around download-on-demand.

A committed asset can remain as a small pointer file in the working tree. When the real file is needed, run:

```bash
kk pull-file path/to/asset
```

After editing or reviewing, the file can be dematerialized:

```bash
kk dematerialize path/to/asset
```

This helps avoid keeping every large asset materialized on every machine.

## Remote Storage

KK supports flexible remote configuration through `.kk/config.json`.

A project can have one or more remotes.

Example:

```json
{
  "version": "kk-local-config-v1",
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

## Supported Remote Types

Planned or supported backend types:

```text
local
  Local folder, external drive, NAS, synced cloud folder.

ssh
  Remote server over SSH.

rclone
  Any rclone-supported provider, such as Google Drive, MEGA, Dropbox,
  OneDrive, S3, Backblaze B2, WebDAV, FTP, SFTP, and others.
```

## Rclone Backend

The rclone backend allows KK to use cloud storage providers without writing a separate integration for each provider.

Example Google Drive remote:

```json
{
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
}
```

Example MEGA remote:

```json
{
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
```

The `binary` field can be:

```text
rclone
  Use rclone from PATH.

bundled
  Use rclone shipped beside kk.

C:\Tools\rclone\rclone.exe
  Use a custom rclone path.
```

Credentials should be managed by rclone, not stored in `.kk/config.json`.

## Multiple Remotes

A single game project may have several remotes.

Example roles:

```text
primary
  Main studio storage.

backup
  Secondary storage for redundancy.

archive
  Cold storage, often pull-only.

cache
  Regional or personal cache.

mirror
  Secondary copy of the primary remote.
```

Pull behavior:

```text
kk pull-file file
  Tries the default remote first.
  If unavailable, falls back to pull-enabled remotes by priority.
```

Push behavior:

```text
kk push
  Pushes to the default push-enabled remote.

kk push --remote name
  Pushes to a selected remote.

kk push --all-remotes
  Pushes to all push-enabled remotes.
```

## Integrity Model

KK uses SHA-256 and file size verification.

The main rule:

```text
Do not delete local cache after upload unless the remote object is verified.
Do not materialize a downloaded object unless SHA-256 and size match.
```

For rclone remotes, strict verification may download the uploaded temporary object back into a temporary local file, hash it, and only then publish it as a final remote object.

This is slower but safer across cloud providers that do not expose SHA-256 checksums consistently.

## Commands

Planned command set:

```text
kk init
kk status
kk track <pattern>
kk untrack <pattern>
kk track list

kk add <file...>
kk commit -m "message"
kk push
kk pull
kk clone

kk pull-file <file...>
kk dematerialize <file...>
kk fsck
kk prune

kk diff --code
kk diff --assets
kk diff --summary --json

kk remote add ...
kk remote remove <name>
kk remote rename <old> <new>
kk remote list
kk remote set-default <name>
kk remote check <name>
kk remote check --all
kk remote repair <name>
kk remote verify <name>
```

Future UI-oriented commands should support JSON:

```bash
kk status --json
kk fsck --json
kk remote list --json
kk remote check --json
kk diff --code --summary --json
```

Long-running commands should support JSON Lines:

```bash
kk push --json-lines
kk pull-file Assets/intro.mp4 --json-lines
kk fsck --json-lines
```

## Flutter UI Direction

A future Flutter desktop UI can call the `kk` CLI through Dart's `Process.run()` or `Process.start()`.

Example:

```dart
final result = await Process.run(
  'kk',
  ['status', '--json'],
  workingDirectory: projectPath,
);
```

Flutter should own the user experience. KK should own correctness, Git operations, hashing, remotes, and object verification.

## Terminal UI Direction

A terminal UI may be added later using Go libraries such as Bubble Tea.

Possible interactive flows:

```text
kk
  Opens project selector.

kk import
  Opens import wizard.

kk remote setup
  Opens remote configuration wizard.
```

The normal CLI should remain fully scriptable.

## Safety Notes

KK may modify files in your working tree.

Before using KK on important projects:

- keep independent backups;
- test workflows on disposable projects;
- verify remotes with `kk fsck`;
- avoid bypassing `kk add` for tracked large files;
- do not rely on a single cloud provider as your only backup.

## Relationship to Git

KK Version Control is not affiliated with Git.

Git is a separate version control system. KK invokes Git as an external command-line tool.

Avoid committing large tracked files with raw `git add`, because that may bypass KK pointer conversion.

Use:

```bash
kk add Assets/large-file.mp4
```

instead of:

```bash
git add Assets/large-file.mp4
```

## Relationship to rclone

rclone is a separate project and is governed by its own license.

KK may invoke rclone as an external tool or use a sidecar `rclone` binary if distributed that way.

KK does not store rclone credentials in `.kk/config.json`.

## License

KK Version Control is source-available under the Business Source License 1.1.

You may use KK Version Control for personal, educational, academic, research, evaluation, development, testing, and non-production purposes.

Commercial Production Use requires a separate commercial license before the applicable Change Date. This includes use by studios, companies, contractors, agencies, paid teams, or revenue-generating production workflows.

After the applicable Change Date, the relevant version becomes available under the Apache License, Version 2.0.

Commercial licensing contact:

```text
godynheil@quisto.ph
```

See [`LICENSE.MD`](LICENSE.MD) for details.

## Liability and Warranty

KK Version Control is provided as-is and without warranty.

You are responsible for validating the tool in your own environment, maintaining independent backups, and verifying that uploads, downloads, remotes, and workflows behave correctly.

See [`LICENSE.MD`](LICENSE.MD) for the full no-warranty and limitation-of-liability terms.

## Project Name

Product name:

```text
KK Version Control
```

CLI command:

```text
kk
```

## Contact

Commercial licensing and project contact:

```text
godynheil@quisto.ph
```
