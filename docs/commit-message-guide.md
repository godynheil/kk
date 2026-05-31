# Commit Message Guide

This guide adapts conventional commit guidance for the KK project. It keeps the
familiar "Golden Format" and Chris Beams rules but adds KK-specific rules and
examples for pointer commits, large-object notes, and wrapper behavior.

## The Golden Format

```text
<type>(<scope>): <short summary>

[optional body]

[optional footer(s)]
```

Example:

```text
feat(models): add pointer for IMDB dataset

Add pointer file referencing a 2.1GB dataset. The object is stored in the KK
object store and uploaded to the remote during the next `kk push`.

Refs: pointer: sha256:abcd1234... size=2147483648
Closes #142
```

---

## Commit Types

| Type | When to use |
|---|---|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `style` | Formatting, no logic change |
| `refactor` | Code restructure, no feature/fix |
| `test` | Adding or fixing tests |
| `chore` | Build process, dependencies |
| `perf` | Performance improvement |

---

## The 7 Rules

1. Separate subject from body with a blank line.
2. Limit subject to 50 characters.
3. Capitalize the subject line.
4. Do not end the subject with a period.
5. Use imperative mood: "Add feature", not "Added feature".
6. Wrap body text at 72 characters.
7. Explain what and why, not how.

---

## Quick Mental Test

Ask yourself: "If applied, this commit will ____"

Good:
- `Add user authentication via JWT`
- `Fix null pointer in payment service`

Bad:
- `fixed stuff`
- `WIP`
- `asdfgh`

---

## KK-Specific Guidance

- Use `kk` in examples and documentation references when you mean the KK
  workflow, for example `kk add`, `kk commit`, `kk push`.
- Pointer commits should make it obvious that the commit changes a pointer, not
  raw large bytes stored in Git.
- When a commit introduces or updates pointer files, include the pointer SHA and
  object size in the body or footer.
- If a commit adds or updates content stored outside Git, mention whether the
  object was uploaded or is still pending upload.
- Prefer workflow-level explanations over raw Git internals because many Git
  operations are wrapped by `kk`.

Useful references:
- `docs/how-it-works.md` - end-to-end workflow including `kk clone`, push modes, and remote layout overview
- `docs/pointer-format.md` - pointer file schema and recommended commit-body fields
- `docs/data-deduplication.md` - how KK deduplicates uploaded content across remotes
- `docs/remote-layout.md` - exact directory structure on every driver (local, rclone, drive)
- `docs/wrapper-checklist.md` - which Git commands `kk` replaces or wraps

---

## Tooling Recommendations

Commit message quality is best enforced both locally and in CI.

Options:
- Option A - local Git hooks with Husky or plain hooks
- Option B - CI-only enforcement with commitlint
- Option C - both local hooks and CI

Minimal PowerShell setup:

```powershell
npm init -y
npm install --save-dev @commitlint/cli @commitlint/config-conventional husky commitizen
npx husky install
npx husky add .husky/commit-msg "npx --no -- commitlint --edit $1"
npm set-script commit "cz"
```

Notes:
- The project is Go-based; Node tooling is optional.
- If you do not want Node dev dependencies in the repo, run commitlint only in
  CI.

---

## Sample Subjects

### `feat` - New Features

```text
feat(models): add pointer for COCO-2017 dataset
feat(ui): show pointer badge in file browser
```

### `fix` - Bug Fixes

```text
fix(kk): prevent duplicate object uploads on kk push
fix(materialize): correctly materialize pointer when cache is cold
```

### `docs` - Documentation

```text
docs(kk): document pointer format and pointer lifecycle
docs(contributing): add KK commit message guide
docs(clone): document kk clone remote spec formats and examples
```

### `refactor` - Code Cleanup

```text
refactor(storage): extract remote lookup into helper
```

### `style` - Formatting

```text
style(cli): format kk help output
```

### `test` - Tests

```text
test(objects): add unit tests for deduplication logic
```

### `chore` - Maintenance

```text
chore(ci): run commitlint on PRs
```

### `perf` - Performance

```text
perf(store): optimize hash calculation for large files
```

### Breaking Change Example

```text
feat(remote)!: replace flat object layout with per-project folder

Objects and manifests are now stored under <remote-root>/<ProjectName>/
instead of directly at <remote-root>/. Existing remotes using the old flat
layout must be re-pushed.

BREAKING CHANGE: old flat remote layout is no longer read.
Closes #88
```

---

## Pointer Commits

Use pointer commits when you add or update large binary assets that KK stores
outside Git and references via pointer files.

What to include:
- In the subject: mention `pointer` or the affected dataset/asset.
- In the body: include the pointer SHA and size.
- In the footer: indicate whether the object was uploaded or is still pending.

Example footer lines:

```text
Pointer: sha256:<hex>
Size: 123456789
Uploaded: pending
```

---

## Bad vs Good

| Bad | Good |
|---|---|
| `fix bug` | `fix(auth): resolve token expiry crash` |
| `updated stuff` | `refactor(api): simplify error handling` |
| `WIP` | `feat(profile): add avatar upload (in progress)` |
| `changes` | `style(forms): normalize input border radius` |
| `it works now` | `fix(payment): handle declined card response` |

---

## First Commit

Most common:

```text
chore: initial commit
```

Other reasonable options:

```text
chore: project setup
chore: initialize project
feat: initial project scaffold
init: project setup
```

With body:

```text
chore: initial commit

- Go CLI scaffold (cmd/kk)
- Internal packages: core, storage, remote
- Basic KK workflow documentation
```

> **Note:** `kk clone` automatically creates an initial commit with the message
> `kk clone: initial snapshot`. Do not write over this commit — it is the
> baseline that `kk fsck` and `kk pull-file` use to locate pointer files in
> `HEAD`. Subsequent commits should follow the guide above.

