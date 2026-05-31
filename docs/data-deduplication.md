# Data Deduplication and Object Retention

KK stores large-file bytes by SHA-256, not by filename or branch.

That gives content-addressed deduplication:

```text
same bytes -> same SHA-256 -> same object path -> upload once
```

Example:

```text
branch A: Assets/tree.fbx -> sha256:abc123
branch B: Assets/tree.fbx -> sha256:abc123
```

Only one object is stored:

```text
.kk/objects/ab/cd/abc123...
<remote-root>/<ProjectName>/objects/ab/cd/abc123...
```

All objects for a project are stored under the project's named folder on the
remote, keeping multiple projects on the same remote root fully isolated.

## Upload behavior

Before uploading an object, KK checks whether the target remote already has the same object ID.

If the remote already has it, upload is skipped and the remote manifest is confirmed/updated.

```text
for each required object:
  if remote has sha256 object:
    skip upload
    ensure manifest entry
  else:
    upload
    verify
    ensure manifest entry
```

## Branch deletion behavior

Deleting a file from one branch does not mean the object is safe to delete.

Example:

```text
branch A deletes Assets/tree.fbx
branch B still points to Assets/tree.fbx
```

The object must remain because branch B still references its pointer.

KK therefore distinguishes two ideas:

```text
upload deduplication
  avoid uploading same SHA-256 twice

reachability-aware pruning
  delete only objects no reachable commit still references
```

## Live objects

KK can scan all reachable Git commits:

```bash
kk objects live
kk objects live --json
```

A live object is any kk pointer found in commits reachable from:

```text
branches
tags
remote-tracking branches
other refs included by git rev-list --all
```

## Reference lookup

To find why an object is still retained:

```bash
kk objects refs <sha256>
kk objects refs <sha256> --json
```

## Local prune

Local cache pruning is conservative:

```bash
kk objects prune --dry-run
```

Only unreferenced local cache objects are candidates.

To actually prune:

```bash
kk objects prune
```

Remote prune is intentionally not enabled in this scaffold. Remote deletion should be added only after protected refs, grace periods, and multi-project remote ownership are designed.

## Remote object paths

Objects on the remote are stored under the project's named folder:

```text
<remote-root>/<ProjectName>/objects/<ab>/<cd>/<sha256>
```

The `<ProjectName>` segment matches `repo.Name` from `.kk/repo.json`. Objects
from different projects never share a folder, so deduplication is
per-project — two projects that happen to track the same bytes will each store
their own copy.

See `docs/remote-layout.md` for the complete directory structure on all
drivers.

## Object availability after kk clone

`kk clone` downloads the project file mirror (source files + pointer files) but
does **not** download large-file objects. After cloning, objects are fetched
lazily:

```bash
kk fsck                      # shows which objects are missing locally
kk pull-file Assets/video.mp4 # fetches and verifies one object on demand
kk clone <spec> --pull        # fetches all objects immediately after clone
```

The `repo_id` recovered by `kk clone` ensures the cloned repo addresses the
same object paths on the remote as the source machine, so `kk pull-file` works
against the original object store without any extra configuration.

---

## Cross-remote replication

When multiple push-enabled remotes are configured, objects uploaded to one remote
are **not automatically copied** to the others. Use `kk objects sync` to replicate
all live objects across every push-enabled remote, or pass `--sync` to `kk pull` /
`kk pull-file` to replicate objects on the fly as they are fetched.

See [`docs/object-syncing.md`](object-syncing.md) for the full replication guide.

