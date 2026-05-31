# Glossary

Canonical definitions for terms used throughout KK's code, documentation, and UI.

---

| Term | Definition |
|---|---|
| **Driver** | A pluggable implementation of the `remote.Driver` interface that knows how to store and retrieve KK objects, manifests, project files, and history bundles on a specific storage destination. The three built-in drivers are `local` (filesystem / NAS), `rclone` (any rclone-supported provider), and `drive` (native Google Drive). |
| **Git Remote** | A standard Git hosting URL (GitHub, GitLab, Gitea, etc.) added via `kk remote add git`. KK uses it to push and pull **pointer history** only — large-file blobs never travel through a git remote. |
| **History Bundle** | A standard `git bundle` file containing commit history for one or more branches. When no git remote is configured, KK stores bundles in `<driver>/<project>/history/<branch>/` so teammates can restore full git history via `kk clone --history` or `kk fetch`. |
| **History State** | `.kk/history-state.json` — records which history bundle was last applied from each remote, so `kk fetch` / `kk pull` downloads only new incremental bundles. |
| **KK Remote** | A named, user-configured entry in `.kk/config.json` that pairs a human-readable name with a driver type and its settings (path, folder ID, credentials, roles, etc.). Managed with `kk remote add / list / remove`. |
| **Local Cache** | The `.kk/objects/` directory on the local machine — a local copy of objects fetched from or staged for upload to a driver. `kk pull-file` populates it; `kk objects prune` cleans it. |
| **Manifest** | A per-project JSON file (`manifests/<repo_id>.json`) on the driver that lists every object the driver currently holds for this project. Used by `kk status`, `kk fsck`, and future UI tooling. |
| **Object / OID** | A content-addressed binary blob stored in the driver under `objects/<oid[:2]>/<oid[2:4]>/<oid>`. The OID is the lowercase hex SHA-256 digest of the file's content. Identical content across any project, branch, or machine always maps to the same OID. |
| **Pointer** | A small text file (stored in git history) that replaces a large binary. It records the file's SHA-256 OID, byte size, and the original filename. KK substitutes the real file for the pointer on demand (`kk pull-file`). |
| **Project Mirror** | The full copy of the committed working tree (source files, pointer files, and `.kk/` metadata) that `kk push` synchronises to the driver's project folder. Excludes `objects/`, `manifests/`, `tmp/`, `logs/`, and `git/`. |
| **Push State** | `.kk/push-state.json` — records the last HEAD commit successfully pushed to each named remote, enabling incremental `kk push` to upload only the files changed since that commit. |
| **Refs Snapshot** | A `refs.json` file stored alongside history bundles. It records the ordered bundle list, the current branch-tip SHAs, and the `base_ref` used to build the next incremental bundle. |
| **Repo ID** | A UUID generated once at `kk init` and stored in `.kk/repo.json`. Mirrors also contain this file; `kk push` uses it to detect project-name collisions before overwriting a remote folder. |
| **Stage** | The KK staging area, analogous to git's index. `kk add` converts tracked large files to pointers and adds them (along with all other files) to the embedded git index inside `.kk/git/`. |
| **Track** | Marking a file-glob pattern (e.g. `*.psd`, `Content/**/*.uasset`) so KK automatically converts matching files to pointers when staging with `kk add`. Managed with `kk track` / `kk untrack`. |

---

See also:
- [`docs/how-it-works.md`](how-it-works.md) — end-to-end workflow
- [`docs/object-syncing.md`](object-syncing.md) — syncing and replicating objects across multiple remotes
- [`docs/remote-layout.md`](remote-layout.md) — exact folder structure on every driver
- [`docs/pointer-format.md`](pointer-format.md) — pointer file schema
- [`docs/history-bundles.md`](history-bundles.md) — how history bundles work

