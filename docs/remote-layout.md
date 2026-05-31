# Remote Layout

Every remote's `path` (local), `remote` (rclone), or `drive_folder_id` (Drive)
points **directly to the project folder**. All data lives at the root of that
folder — no additional project-name subdirectory is inserted.

```text
<project-folder>/             ← what path / remote / drive_folder_id points to
  objects/
    ab/
      cd/
        abcdef...             ← large file blob (SHA-256 addressed)
  manifests/
    <repo_id>.json            ← object manifest for status and UI
  history/                    ← bundle history when no git remote is configured
    refs.json
    <branch>/
      full.bundle
      inc-000001.bundle
  .kk/                        ← project file mirror: kk metadata
    repo.json                 ← repo identity (repo_id, name, created_at)
    config.json               ← remote configuration
    tracks.json               ← tracked file patterns
  Config/
  Content/
  <other project files>       ← everything not in .kkignore
```

### ⚠️ No server-side deletion tracking

> **Warning:** KK does not run any server and does not monitor your driver for
> external changes. Anyone with write access to the remote can delete or overwrite
> objects, manifests, and project files without KK being notified. The driver
> (Google Drive, NAS, rclone provider, etc.) is solely responsible for
> access control. KK has no way to enforce or audit permissions on your behalf.
>
> Restrict access at the driver level and run `kk fsck` regularly to verify that
> all referenced objects are still present on the remote.

### Rules

- **`objects/`** — immutable blobs addressed by SHA-256. Downloaded bytes are
  always verified locally before use.
- **`manifests/`** — per-repo JSON manifest listing all uploaded objects.
  Used by status, UI, and fsck commands.
- **`history/`** — git bundle history used only when no `type=git` remote is
  configured. This folder is skipped during project-file clone/download.
- **Project files** (directly under the project folder) — a full mirror of the
  local working directory, uploaded on every `kk push`. Files matching
  `.kkignore` are excluded. The following `.kk/` internals are also excluded
  from the mirror: `objects/`, `tmp/`, `logs/`, `git/`.

### Name collision guard

On push, kk reads `.kk/repo.json` from the remote project folder.
If a file exists with a **different** `repo_id`, the push is aborted with an
error. This prevents one project from silently overwriting another when two
repos happen to share the same folder.

---

## Cloning from a remote

`kk clone` can reconstruct a local project from the object-remote layout above.
It can also clone pointer history from a standard git remote with `git:<url>`.

### Remote spec formats

| Spec | Driver |
|------|---------|
| `git:https://host/your-username/MyGame.git` | Standard git remote for pointer history. Large objects still come from configured KK object remotes. |
| `local:/path/to/ProjectName` | Local filesystem / NAS — the path IS the project folder |
| `rclone:<remote>:<path>/ProjectName` | Any rclone-supported provider — the full path IS the project folder |
| `drive:<project-folder-id>` | Native Google Drive. Pass the project folder's ID from its Drive URL. KK resolves the project name automatically via the Drive API. |

For `drive:` remotes, open the project folder in Google Drive after the first
`kk push` and copy the ID from the URL:

```
https://drive.google.com/drive/folders/<project-folder-id>
```

That ID is what teammates pass to `kk clone drive:<project-folder-id>`.

### What `kk clone` reads from the remote

```text
<project-folder>/
  .kk/                     ← DownloadProjectFiles() copies this entire tree
    repo.json              ← recovered to preserve repo_id
    config.json            ← discarded; a fresh one is written for this machine
    tracks.json            ← kept as-is (track patterns are machine-agnostic)
  <source files …>         ← pointer files + regular files
  objects/                 ← large-file blobs (not downloaded during clone;
  manifests/                  downloaded on demand by kk pull-file)
```

The `objects/` and `manifests/` subtrees are **not** downloaded during clone.
Large-file objects are fetched lazily by `kk pull-file` (or eagerly with
`kk clone --pull`), using the same remote that was used to clone.

For object remotes, pass `--history` to restore full Git history from
`history/<branch>/*.bundle`; otherwise KK creates a single initial snapshot
commit. For `git:` remotes, the normal git clone supplies pointer history.
