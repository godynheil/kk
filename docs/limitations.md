# Limitations

KK is a **client-side file management tool** built on top of Git. It is
intentionally focused: it handles large binary files and project snapshots on
your chosen storage driver. It is _not_ a hosted service, a code-review
platform, or a replacement for GitHub or GitLab.

Understanding these limits up front prevents surprises.

---

## GitHub / GitLab / Bitbucket - pointer history only

Git remote support (`kk remote add git`) syncs KK pointer history to a normal
git host while keeping binary objects on KK storage. Core push, pull, clone,
and branch workflows have been thoroughly tested on GitHub and GitLab.

Pull requests, concurrent pushes, branch divergence, branch protection behavior,
webhooks, and CI integration are external git-host workflows that KK does not
manage directly — those are handled by the git host itself.

KK can optionally push the **pointer history** stored in `.kk/git` to any git
hosting service (GitHub, GitLab, Bitbucket, Gitea, etc.) using a `type=git`
remote:

```bash
kk remote add git github https://github.com/your-username/MyGame.git
kk push   # pointer history synced to github; binary blobs go to your KK object remote
```

However, KK does **not**:

- Push or pull binary blobs to/from a git host — they stay on your KK object remote
- Create pull requests or code reviews
- Support webhooks, CI triggers, or branch protection rules from a git host via KK
- Show issues or project boards

**What it does instead:** KK mirrors the entire project (source files + large
objects + KK metadata) to your own storage driver — Google Drive, a local NAS,
or any `rclone`-supported provider. Teammates clone large files directly from
that driver, not from a git host.

If you use GitHub/GitLab for code review or CI, run them in parallel:

1. Add a KK git remote: `kk remote add git github https://github.com/your-username/MyGame.git`
2. Push pointer history: `kk push` (syncs automatically from then on)
3. Large files stay on your KK object remote; only the lightweight pointer files
   land on GitHub.

See [`docs/how-it-works.md`](how-it-works.md) → "Adding a Git remote" for the
full workflow.

---

## Remote history sync

`kk push` uploads a **file snapshot** to the driver. History then travels in one
of two modes:

- If a `type=git` remote is configured, KK syncs lightweight pointer history
  from `.kk/git` with `git push` / `git pull`.
- If no `type=git` remote is configured, KK stores full Git history as bundles
  under `history/<branch>/` on the object remote.

What a teammate gets from a default object-remote `kk clone` is:

- All source files and large-file pointer files
- All `.kk/` metadata (config, tracks, repo identity)
- A **single fresh git commit** ("kk clone: initial snapshot")

Use `kk clone <spec> --history` to restore full bundle history from object
storage when the project was pushed without a git remote. Or use a git host for
pointer-history transport:

```bash
kk remote add git github https://github.com/your-username/MyGame.git
kk push   # syncs pointer history to github automatically from here on
```

When no git remote is configured, `kk pull` fetches history bundles, merges the
current branch from `refs/remotes/kk-history/<branch>`, materialises new pointer
files, and runs `kk fsck`. Use `kk pull --no-merge` or `kk fetch` to download
bundles without merging.

See [`docs/history-bundles.md`](history-bundles.md) for the full object-storage
history flow.

---

## `kk pull` ≠ syncing large files

| Command | What it actually does |
|---|---|
| `kk pull` | With a git remote, runs `git pull` for pointer history. Without a git remote, fetches and merges storage history bundles. Then materialises new pointer files. |
| `kk pull --no-merge` / `kk fetch` | Fetches storage history bundles without merging when no git remote is configured. |
| `kk pull-file .` | Downloads and materialises **all large-file objects** in the current HEAD. |
| `kk pull-file <file>` | Downloads and materialises a **single large file**. |
| `kk pull-file --all` | Identical to `kk pull-file .`. |

**In a standard KK workflow, `kk pull-file .` is the command you want for large
files already referenced by the current HEAD.** Use `kk pull` when you also need
new pointer history from a git remote or storage history bundles.

---

## No server — no central tracking

KK runs no server and keeps no central state. Everything lives on:

- **Your local machine** — `.kk/git/` (git history), `.kk/objects/` (large-file
  blobs)
- **Your chosen driver** — Google Drive, NAS, rclone provider — project mirror,
  objects, manifests

Because KK is fully peer-to-peer via your driver:

- There is no authoritative "latest version" enforced by a server
- There is no lock system — two teammates can push conflicting states
- There is no notification when the driver content changes

---

## No access control

KK cannot restrict who reads or writes your driver. Anyone with access to the
driver folder can overwrite or delete files without KK being notified.

Use the driver's own sharing settings (Google Drive folder permissions, POSIX
permissions on a NAS, etc.) to control access. See the warning in
`docs/how-it-works.md` → "⚠️ No server-side deletion tracking" for details.

---

## No binary merge / conflict resolution

Git cannot merge binary files (textures, audio, video, 3D meshes). If two
teammates modify the same large file and commit independently, KK will track
two different objects under two different OIDs. Resolving which version wins is
a manual step — KK does not offer a merge strategy for binary content.

---

## History is local only

Branching, merging, `kk log`, `kk diff`, and `kk fsck` all operate on the
**local** `.kk/git` repository. If the local repo is lost or corrupted there is
no way to recover git history from the driver — the driver stores file
content, not git objects.

**Recommendation:** For teams, use `kk push` regularly so the project snapshot
on the driver stays current. Keep the `.kk/git/` directory healthy (it is
excluded from the driver mirror by design). If long-term git history matters to
your team, mirror it to a private GitHub/GitLab repo in addition to using KK.

---

## Object pruning is manual

KK does not automatically remove large objects from the local cache or the
remote when a branch is deleted or a file is no longer referenced. Objects
accumulate until you explicitly prune them:

```bash
kk objects prune --dry-run   # preview which local objects are orphaned
kk objects prune             # remove them
```

Remote objects are never automatically deleted — you must manage unused objects
on the driver yourself.

---

## No automatic cross-remote replication

Uploading an object to one push-enabled remote does **not** automatically copy
it to other configured remotes. If you run multiple remotes (e.g. a team NAS
and a cloud backup), objects can become out-of-sync between them.

Use `kk objects sync` to replicate all live objects across every push-enabled
remote, or pass `--sync` to `kk pull` / `kk pull-file` to replicate on the fly:

```bash
kk objects sync            # full replication scan across all push-enabled remotes
kk pull --sync             # replicate while pulling
```

See [`docs/object-syncing.md`](object-syncing.md) for the complete guide.

---

## Summary

| Capability | KK | GitHub/GitLab |
|---|---|---|
| Large binary file storage | ✅ | ❌ (size limits) |
| Content-addressed deduplication | ✅ | ❌ |
| Git history (local) | ✅ | ✅ |
| Remote git history sync (pointer history) | ✅ (`kk remote add git`) | ✅ |
| Binary blob sync to git host | ❌ | ❌ |
| Cross-remote object replication | ✅ (`kk objects sync`, `--sync`) | ❌ |
| Pull requests / code review | ❌ | ✅ |
| Branch protection / CI hooks | ❌ | ✅ |
| Server-side access control | ❌ | ✅ |
| Deletion event tracking | ❌ | ✅ |
| Binary conflict resolution | ❌ | ❌ |
| Works without internet | ✅ (local NAS) | ❌ |
| Self-hosted / no vendor lock-in | ✅ | Optional |
