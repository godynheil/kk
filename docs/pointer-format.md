# Pointer Format

A KK pointer is a small text file committed to Git in place of the real large
file:

```text
version kk-lfs-1.0.0
oid sha256:<64 lowercase hex chars>
size <decimal byte count>
```

The SHA-256 and size must match the real object bytes exactly. If either value
differs, KK refuses to trust the object.

## Pointer lifecycle

```text
(Optional) kk track "*.mp4" [Not required: non-code files are tracked by default!]
  ↓
kk add Assets/cinematic.mp4
  → real file hashed → pointer written to working tree → staged in git
  → real bytes stored at .kk/objects/<ab>/<cd>/<sha256>
  ↓
kk commit -m "add cinematic"
  → pointer file committed to git history

kk push
  → pointer stays in git; real bytes uploaded to remote
  → remote: <remote-root>/<ProjectName>/objects/<ab>/<cd>/<sha256>
  ↓
kk clone <remote-spec>   (on another machine)
  → pointer files downloaded as-is (still tiny text files)
  → real bytes NOT downloaded yet

kk pull-file Assets/cinematic.mp4   (or kk clone --pull)
  → real bytes fetched from remote + verified against sha256 + size
  → pointer file replaced by real file in working tree  (materialised)

kk dematerialize Assets/cinematic.mp4
  → real file replaced by pointer again (frees disk space)
```

## Where pointers are stored

| Location | What is stored |
|----------|---------------|
| Git history (`.kk/git`) | The pointer text file |
| Local cache (`.kk/objects/`) | The real bytes (content-addressed) |
| Remote (`objects/`) | The real bytes (content-addressed, same path structure) |
| Remote mirror (`<ProjectName>/`) | The pointer text file (synced by `kk push`) |

See `docs/remote-layout.md` for the full remote directory structure.
