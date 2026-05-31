# Object Syncing & Replication

KK supports robust large-file object syncing and replication across multiple configured storage remotes. When working with multiple remotes (for example, a team `origin` and an offsite `backup`), some remotes might lack certain object versions. 

KK provides two methods to ensure your objects are fully replicated across all push-enabled remotes:
1. **Full History/Active Replication** using the `kk objects sync` command.
2. **On-Demand Replication** using the `--sync` flag during pulls.

---

## 1. Standalone Replication (`kk objects sync`)

The `kk objects sync` command scans all live large-file objects across all branches, tags, and commits in your repository. It identifies which of your push-enabled object remotes (excluding Git remotes) are missing these objects, and synchronizes them.

### Command Usage
```bash
kk objects sync [--workers N] [--verbose]
```

### How it works:
1. **Metadata Scan**: KK retrieves all live objects in the repository history.
2. **Presence Verification**: For each remote, KK first checks its fast JSON manifest to verify object presence. If missing, it double-checks with a direct network query (`HasObject`) to be absolutely certain.
3. **Local Cache Fallback**: If an object to be synced is not present in your local cache (`.kk/objects/`), KK automatically locates another remote that contains the object, downloads it to your local cache first, and then replicates it to the missing target remotes.
4. **Multi-threaded Uploads**: Supports the `--workers N` flag to perform transfers concurrently.

---

## 2. On-Demand Sync on Pull

You can also replicate missing objects on the fly while pulling or materializing files. By appending the `--sync` flag, KK ensures any large-file object you fetch is immediately uploaded to all other push-enabled remotes that do not have it yet.

### Commands with Sync support:
- **`kk pull --sync`**: Merges history and pulls all newly introduced large-file pointer objects, automatically replicating them to other remotes.
- **`kk pull-file --sync <file...>`**: Materializes specific large files and replicates their corresponding objects to other remotes.
- **`kk pull-file --sync .`** or **`kk pull-file --sync --all`**: Materializes all pointer files in HEAD and replicates them.

### Example:
```bash
kk pull --sync --workers 4
```

---

## 3. Configuration & Workflows

To use object syncing, configure multiple remotes in your `.kk/config` file with `push = true` and `type` other than `"git"`.

### Example `.kk/config`
```toml
[remote "origin"]
type = "rclone"
path = "gdrive:kk-team-bucket"
push = true
pull = true

[remote "backup"]
type = "rclone"
path = "aws-s3:kk-offsite-backup"
push = true
pull = false
```

In this setup, running:
```bash
kk pull --sync
```
will fetch new large files from `origin` (since `pull = true`), download them, and instantly push/replicate them to `backup` (since `push = true` on `backup`), ensuring your secondary backup remote is always up-to-date with minimal effort.
