# Git History via Object Storage (No GitHub Required)

KK can store and sync the complete Git commit history — all branches, commits,
and tree snapshots — through your existing object storage driver (Google Drive,
NAS, rclone). No GitHub, GitLab, or any external git hosting service is needed.

---

## When is this used?

Automatically, whenever **no `type=git` remote** is configured.  
If you have not run `kk remote add git …`, every `kk push` bundles and uploads
the history alongside the large-file objects.

If a git remote *is* configured, the existing pointer-history sync via
`git push` / `git pull` continues unchanged.

---

## Remote storage layout

```
<remote-root>/
  history/
    refs.json            ← branch tips + ordered bundle list
    main/
      full.bundle          ← base bundle: complete history to the first pushed tip
      inc-000001.bundle    ← incremental: commits since full.bundle
      inc-000002.bundle    ← incremental: commits since inc-000001
  objects/ …             ← large-file blobs (unchanged)
  manifests/ …           ← object manifest (unchanged)
```

> **`history/` is never downloaded into your working tree.**  
> All drivers (`local`, `rclone`, `drive`) skip this folder when cloning or
> syncing project files.

---

## Incremental bundles

KK uses native **git bundles** to pack history:

| Push # | Bundle created | Contents |
|--------|---------------|----------|
| 1 | `full.bundle` | Entire history up to current HEAD (`git bundle create --all`) |
| 2 | `inc-000001.bundle` | Commits since `full.bundle` tip (`git bundle create --all ^<prev-tip>`) |
| 3 | `inc-000002.bundle` | Commits since `inc-000001` tip |
| … | `inc-NNNNNN.bundle` | … |

`refs.json` tracks the **ordered bundle list** and the **base_ref** (tip SHA of
the last upload) so each push creates only the smallest possible delta.

### Bundle Compaction (Squashing)

To prevent an infinite chain of incremental downloads on future clones, KK automatically squashes the bundle chain after **100 incremental bundles**. When this limit is reached:
1. The next push generates a new `full.bundle` containing the complete history up to the current branch tip.
2. The remote list of bundles for the branch is reset to just this new `full.bundle`.
3. Subsequent pushes resume the incremental chain starting at `inc-000001.bundle`.
4. Clients fetching from this remote will automatically detect that the last applied bundle is no longer in the remote bundle list, and cleanly fall back to applying the new `full.bundle`.

---

## Command reference

### Push (auto-triggered)

```bash
kk push
```

After uploading large files, kk automatically:
1. Checks `refs.json` on the remote for the `base_ref` (last known tip).
2. Creates `full.bundle` (first push) or `inc-NNNNNN.bundle` (incremental).
3. Uploads the bundle to `<remote>/history/<branch>/`.
4. Updates `refs.json` with the new bundle, the new `base_ref`, and the current branch map.

Sample output:
```
kk: [origin] creating history bundle (inc-000002.bundle) for branch main...
kk: [origin] uploading history bundle for main...
kk: [origin] history pushed (inc-000002.bundle, branch main)
```

---

### Fetch (download bundles, no merge)

```bash
kk fetch
```

Downloads all bundles not yet applied locally and places the remote branch tips
under `refs/remotes/kk-history/*` without touching any local branch.

Progress is shown per bundle:
```
kk: [origin] fetching 1 history bundle(s) for branch main...
kk: [origin] downloading bundle 1/1 (inc-000002.bundle) for branch main...
kk: [origin] history fetched:
kk:   refs/remotes/kk-history/main    → 3f9a12b4c6d1
kk:   refs/remotes/kk-history/feature → a8b2c4d6e001
kk: fetch complete (default branch: main)
kk: to merge into the current branch, run: kk pull
```

When a git-type remote IS configured, `kk fetch` delegates to `git fetch`.

---

### Pull (fetch + merge)

```bash
kk pull               # fetch history bundles then merge current branch
kk pull --no-merge    # fetch bundles only — inspect before merging
```

**Without `--no-merge`** (default): after applying all bundles, KK attempts to
merge `refs/remotes/kk-history/<currentBranch>` into the current branch.

If the merge creates conflicts:
```
kk: merge conflict after history fetch — resolve all conflicts,
  stage the results with 'kk stage', then run 'kk commit' to complete the merge.
  To fetch without merging next time, use: kk pull --no-merge
```

**With `--no-merge`**:
```
kk: history fetched (--no-merge) — run 'kk pull' without --no-merge to merge
```

After merging, `kk pull` automatically materialises any new pointer files that
arrived with the merged commits, then runs `kk fsck`.

---

### Clone with history

```bash
# Lightweight clone — single snapshot commit, no git history (fast)
kk clone local:/NAS/KK/MyGame

# Full clone — restores all branches and commits
kk clone local:/NAS/KK/MyGame --history

# Full clone + materialise large files immediately
kk clone drive:<folder-id> --history --pull
```

Without `--history`, `kk clone` creates a single `kk clone: initial snapshot`
commit. This is sufficient for `kk fsck` and `kk pull-file` but loses branch
and commit metadata.

With `--history`:
1. Downloads all bundles from `<remote>/history/<branch>/` for all branches.
2. Applies `full.bundle` first (no prerequisites), then each `inc-*.bundle` in order.
3. Point HEAD at `<defaultBranch>`, populate the index with `git reset --mixed HEAD`.
4. Records the applied bundle state in `.kk/history-state.json` so the next
   `kk fetch` / `kk pull` only downloads new bundles.

Sample output:
```
kk: [--history] restoring history from 2 branch(es)...
kk: [--history] downloading bundle 1/1 (full.bundle) for branch feature...
kk: [--history] downloading bundle 1/2 (full.bundle) for branch main...
kk: [--history] downloading bundle 2/2 (inc-000001.bundle) for branch main...
```

---

## Local state file: `.kk/history-state.json`

KK records per-remote progress in `.kk/history-state.json` so that subsequent
fetches only download new incremental bundles.

```json
{
  "version": "kk-history-state-1.0.0",
  "remotes": {
    "origin": {
      "branches": {
        "main": {
          "last_applied_bundle": "inc-000002.bundle",
          "last_applied_ref": "3f9a12b4c6d1..."
        }
      },
      "updated_at": "2026-05-24T12:00:00Z"
    }
  }
}
```

This file lives alongside `.kk/push-state.json` and is committed to the project
file mirror on the next `kk push`.

---

## Typical team workflow (no GitHub)

```
Machine A (owner)
  kk init
  kk remote add local nas --path /NAS/KK/MyGame --push true --pull true
  kk add .  &&  kk commit -m "initial assets"
  kk push
  # → uploads files + full.bundle + refs.json

Machine B (teammate)
  kk clone local:/NAS/KK/MyGame --history --pull
  # → full history + all branches restored; large files materialised

  # ... time passes, Machine A pushes more commits ...

Machine B
  kk fetch            # see what changed (no merge yet)
  kk pull             # merge + materialise new pointer files

  # Machine B can also contribute through the same storage remote.
  kk add .
  kk commit -m "update from Machine B"
  kk push
  # → uploads Machine B's branch tip and a new history bundle

Machine A
  kk pull             # fetches and merges Machine B's bundle on the current branch
```

Storage clones register `origin` as both pull-enabled and push-enabled by
default. Older clones that still have `origin.push=false` are upgraded
automatically on first `kk push` when `origin` is the only storage remote.

---

## Notes and limitations

- **Full re-bundle** on the very first push; all subsequent pushes are **incremental**.
- If the remote `base_ref` is unreachable from the local repo (e.g. after a force-push
  or repo reset), kk will create a fresh `full.bundle` to recover.
- **Branch divergence**: `kk pull` attempts an auto-merge; if it fails, resolve
  conflicts normally (`kk stage`, `kk commit`) and re-run `kk push`.
- The `--history` flag on `kk clone` downloads and applies ALL bundles in the chain.
  For large repos with many commits, prefer `kk clone` (no flag) then `kk pull` later.
- History bundles use the standard `git bundle` format. They can be inspected with:
  ```bash
  git --git-dir=.kk/git bundle verify /path/to/full.bundle
  git --git-dir=.kk/git bundle list-heads /path/to/full.bundle
  ```

---

## Switching history mode

You can switch between the storage-bundle mode and a git-hosting remote at any
time using `kk remote migrate`.

### Already using bundles? Add GitHub/GitLab (`to-git`)

```bash
# All local branches are pushed to the git remote in one step.
kk remote migrate to-git github https://github.com/your-username/MyGame.git

# After this, kk push syncs history via git push; bundles are no longer created.
```

- Existing `history/<branch>/` bundles on the object remote are left in place and ignored.
- Future teammates clone with `kk clone git:https://github.com/your-username/MyGame.git --pull`.

### Already using GitHub? Move to bundles (`to-storage`)

```bash
# Interactively confirm, then migrate.
kk remote migrate to-storage

# Or skip the prompt in scripts:
kk remote migrate to-storage --yes

# Target one specific git remote when several exist:
kk remote migrate to-storage --remote gitlab --yes
```

- A fresh `full.bundle` is created on every push-enabled object remote **before**
  any config changes — if the upload fails, nothing is modified.
- After migration, `kk push` resumes bundle-based history (incremental bundles as above).
- Teammates restore full history with `kk clone <spec> --history`.

See [`docs/GIT_REMOTE.md`](git-remote.md#migrating-between-history-modes) for complete step-by-step examples.
