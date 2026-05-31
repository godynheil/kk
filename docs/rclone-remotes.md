# Setting Up Cloud Storage with Rclone

KK uses `rclone` as a universal adapter for any cloud storage provider.
One `rclone` remote config covers the rclone side; one `kk remote add rclone` command
covers the KK side. This guide walks through the full setup for every major provider.

> **Native Google Drive users:** If you only need Google Drive you can skip `rclone`
> entirely and use [`kk setup gdrive`](installation.md#option-a--native-google-drive-recommended-no-rclone-needed)
> instead — it handles OAuth automatically with no rclone needed.

---

## Table of contents

1. [Prerequisites](#1-prerequisites)
2. [How rclone remotes map to KK remotes](#2-how-rclone-remotes-map-to-kk-remotes)
3. [MEGA.nz](#3-meganz)
4. [Google Drive (via rclone)](#4-google-drive-via-rclone)
5. [Dropbox](#5-dropbox)
6. [OneDrive](#6-onedrive)
7. [Amazon S3 (and S3-compatible)](#7-amazon-s3-and-s3-compatible)
8. [Backblaze B2](#8-backblaze-b2)
9. [SFTP / SSH server](#9-sftp--ssh-server)
10. [Local path / NAS (no rclone needed)](#10-local-path--nas-no-rclone-needed)
11. [Multi-remote setup](#11-multi-remote-setup)
12. [Performance Tuning](#12-performance-tuning)
13. [Troubleshooting](#13-troubleshooting)

---

## 1. Prerequisites

### Install rclone

| Platform | Command |
|----------|---------|
| **Windows** (Chocolatey) | `choco install rclone` |
| **Windows** (Scoop) | `scoop install rclone` |
| **Windows** (manual) | Download from <https://rclone.org/downloads/>, unzip, add folder to PATH |
| **macOS** | `brew install rclone` |
| **Linux** | `sudo apt install rclone` or `curl https://rclone.org/install.sh | sudo bash` |

Verify:

```bash
rclone version
```

### Folder naming convention

KK's `--remote` value must point **directly to the project folder**, not to a root or bucket.
Use a consistent path like `<provider>:KK/<ProjectName>` so all projects live under one
KK namespace and are easy to share.

```
mega:KK/MyGame          ← project folder
mega:KK/MyGame/objects  ← managed by KK — do not create manually
mega:KK/MyGame/history  ← managed by KK — do not create manually
```

---

## 2. How rclone remotes map to KK remotes

```
rclone remote name     kk remote name    project path on storage
────────────────────   ───────────────   ───────────────────────────
mega                   mega-backup       mega:KK/MyGame
gdrive                 gdrive-backup     gdrive:KK/MyGame
dropbox                dropbox-main      dropbox:KK/MyGame
```

These are independent names — `rclone` needs the left column, `kk remote add` needs the middle.

**KK clone spec** format:

```bash
kk clone rclone:<rclone-remote-name>:<path>/<ProjectName>
#                └─ rclone remote    └─ path matching --remote used in kk remote add
```

---

## 3. MEGA.nz

MEGA provides 20 GB free storage with end-to-end encryption.

### Step 1 — Configure rclone for MEGA

```bash
rclone config
```

Interactive prompts (type the value shown after `>`):

```
e) Edit existing remote
n) New remote
q) Quit config
e/n/q> n

name> mega

Type of storage to configure.
...
Storage> mega           # or type the number shown next to "Mega"

User name (usually email):
user> your-email@example.com

Password. Leave blank to use RCLONE_MEGA_PASS env var.
y) Yes type in my own password
g) Generate random password
n) No leave this optional password blank
y/g/n> y
Enter the password:
password:
Confirm the password:
password:

Edit advanced config?
y) Yes
n) No (default)
y/n> n

--------------------
[mega]
type = mega
user = your-email@example.com
pass = *** ENCRYPTED ***
--------------------
y) Yes this is OK (default)
e) Edit this remote
d) Delete this remote
y/e/d> y
```

### Step 2 — Verify the rclone remote

```bash
# List the top-level of your MEGA account
rclone lsd mega:

# Create the KK namespace (one-time)
rclone mkdir mega:KK

# Create the project folder (one-time per project)
rclone mkdir mega:KK/MyGame

# Confirm it exists
rclone lsd mega:KK
#          -1 2026-05-24 00:00:00        -1 MyGame
```

### Step 3 — Add the KK remote

```bash
# Minimal — only required flags
kk remote add rclone mega-backup \
  --remote mega:KK/MyGame \
  --push true \
  --pull true

# Full — with optional metadata (recommended for teams)
kk remote add rclone mega-backup \
  --display-name "MEGA Backup" \
  --role backup \
  --provider mega \
  --binary rclone \
  --remote mega:KK/MyGame \
  --verify-mode download \
  --priority 20 \
  --pull true \
  --push true \
  --tag cloud \
  --tag backup
```

Expected output:

```
remote added mega-backup
    Teammates can clone with:
    kk clone rclone:mega:KK/MyGame
```

### Step 4 — Push and clone

```bash
# Upload all large files + history bundles
kk push

# Teammate clones on a new machine
kk clone rclone:mega:KK/MyGame
kk clone rclone:mega:KK/MyGame --pull       # clone + materialise large files
kk clone rclone:mega:KK/MyGame --history    # clone with full git history restored
```

### MEGA tips

- MEGA has a **transfer quota** (~10 GB/day on free accounts). Large pushes will
  hit this — upgrade or use `--workers 1` to slow the transfer.
- MEGA does **not** support server-side hashing; KK uses `--verify-mode download`
  which re-downloads to verify. This doubles transfer on upload.
- If `rclone` reports "Too many requests", add `--transfers 2` to your rclone config
  or lower `KK_WORKERS`:

  ```bash
  KK_WORKERS=1 kk push
  ```

---

## 4. Google Drive (via rclone)

Use this when you already manage rclone and want manual control, or when
the native `kk setup gdrive` flow does not suit your setup.

### Step 1 — Configure rclone for Google Drive

```bash
rclone config
```

```
n) New remote
name> gdrive
Storage> drive          # Google Drive

client_id>              # leave blank to use rclone's built-in app
client_secret>          # leave blank

scope> drive            # full access; use drive.file for a narrower scope

root_folder_id>         # leave blank (uses My Drive root)
service_account_file>   # leave blank (uses OAuth)

Edit advanced config? n

# rclone opens a browser for OAuth — log in and allow access.
```

### Step 2 — Verify

```bash
rclone lsd gdrive:
rclone mkdir gdrive:KK
rclone mkdir gdrive:KK/MyGame
```

### Step 3 — Add KK remote

```bash
kk remote add rclone gdrive-backup \
  --display-name "Google Drive" \
  --role primary \
  --provider google-drive \
  --binary rclone \
  --remote gdrive:KK/MyGame \
  --verify-mode download \
  --priority 10 \
  --pull true \
  --push true \
  --tag cloud
```

### Step 4 — Push and clone

```bash
kk push
kk clone rclone:gdrive:KK/MyGame --pull
```

---

## 5. Dropbox

### Step 1 — Configure rclone

```bash
rclone config
```

```
n) New remote
name> dropbox
Storage> dropbox

client_id>        # leave blank
client_secret>    # leave blank

# rclone opens a browser for OAuth — log in and allow.
```

### Step 2 — Verify

```bash
rclone lsd dropbox:
rclone mkdir dropbox:KK/MyGame
```

### Step 3 — Add KK remote

```bash
kk remote add rclone dropbox-main \
  --display-name "Dropbox" \
  --role primary \
  --provider dropbox \
  --binary rclone \
  --remote dropbox:KK/MyGame \
  --verify-mode download \
  --priority 10 \
  --pull true \
  --push true
```

### Step 4 — Push and clone

```bash
kk push
kk clone rclone:dropbox:KK/MyGame --pull
```

---

## 6. OneDrive

### Step 1 — Configure rclone

```bash
rclone config
```

```
n) New remote
name> onedrive
Storage> onedrive

client_id>        # leave blank
client_secret>    # leave blank

# rclone opens a browser — log in with your Microsoft account.
# Choose the drive/site type: personal OneDrive, SharePoint, etc.
```

### Step 2 — Verify

```bash
rclone lsd onedrive:
rclone mkdir "onedrive:KK/MyGame"
```

### Step 3 — Add KK remote

```bash
kk remote add rclone onedrive-main \
  --display-name "OneDrive" \
  --role primary \
  --provider onedrive \
  --binary rclone \
  --remote "onedrive:KK/MyGame" \
  --verify-mode download \
  --priority 10 \
  --pull true \
  --push true
```

### Step 4 — Push and clone

```bash
kk push
kk clone "rclone:onedrive:KK/MyGame" --pull
```

---

## 7. Amazon S3 (and S3-compatible)

Works with AWS S3, Cloudflare R2, Wasabi, MinIO, Backblaze B2 (S3 API), and any
S3-compatible endpoint.

### Step 1 — Configure rclone for AWS S3

```bash
rclone config
```

```
n) New remote
name> s3
Storage> s3                        # Amazon S3 Compliant Storage Providers

provider> AWS                      # or Cloudflare, Wasabi, Minio, etc.

env_auth> false                    # use explicit keys below

access_key_id> AKIAIOSFODNN7EXAMPLE
secret_access_key> wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

region> us-east-1                  # your bucket's region
endpoint>                          # leave blank for AWS; set for compatible providers

acl> private
```

For **Cloudflare R2**, set:
```
provider> Cloudflare
region> auto
endpoint> https://<account-id>.r2.cloudflarestorage.com
```

For **Wasabi**:
```
provider> Wasabi
endpoint> s3.wasabisys.com
```

For **MinIO** (self-hosted):
```
provider> Minio
endpoint> http://192.168.1.10:9000
```

### Step 2 — Verify

```bash
# Create a bucket first in the AWS/provider console, then:
rclone lsd s3:
rclone mkdir s3:my-kk-bucket/KK/MyGame
```

### Step 3 — Add KK remote

```bash
kk remote add rclone s3-main \
  --display-name "S3 Storage" \
  --role primary \
  --provider s3 \
  --binary rclone \
  --remote s3:my-kk-bucket/KK/MyGame \
  --verify-mode download \
  --priority 10 \
  --pull true \
  --push true
```

### Step 4 — Push and clone

```bash
kk push
kk clone rclone:s3:my-kk-bucket/KK/MyGame --pull
```

---

## 8. Backblaze B2

Backblaze B2 is one of the cheapest object stores (~$6/TB/month).

### Step 1 — Configure rclone

```bash
rclone config
```

```
n) New remote
name> b2
Storage> b2             # Backblaze B2 Cloud Storage

account> YOUR_APPLICATION_KEY_ID
key>     YOUR_APPLICATION_KEY
```

Get your Application Key ID and Key from the Backblaze console:
`Account → App Keys → Add a New Application Key`.

### Step 2 — Verify

```bash
rclone lsd b2:
# Create a bucket in the Backblaze console first, then:
rclone mkdir b2:my-kk-bucket/KK/MyGame
```

### Step 3 — Add KK remote

```bash
kk remote add rclone b2-main \
  --display-name "Backblaze B2" \
  --role primary \
  --provider b2 \
  --binary rclone \
  --remote b2:my-kk-bucket/KK/MyGame \
  --verify-mode download \
  --priority 10 \
  --pull true \
  --push true
```

### Step 4 — Push and clone

```bash
kk push
kk clone rclone:b2:my-kk-bucket/KK/MyGame --pull
```

---

## 9. SFTP / SSH server

Use any Linux/NAS box with SSH enabled — no special software needed on the server.

### Step 1 — Configure rclone

```bash
rclone config
```

```
n) New remote
name> nas-sftp
Storage> sftp

host> 192.168.1.100           # IP or hostname
user> kk-user
port> 22

# Authentication — choose one:
key_file> /home/you/.ssh/id_rsa    # SSH key (recommended)
# or leave key_file blank and enter a password when prompted
```

### Step 2 — Verify

```bash
rclone lsd nas-sftp:
rclone mkdir nas-sftp:/storage/KK/MyGame
```

### Step 3 — Add KK remote

```bash
kk remote add rclone nas-sftp \
  --display-name "Studio NAS (SFTP)" \
  --role primary \
  --provider sftp \
  --binary rclone \
  --remote "nas-sftp:/storage/KK/MyGame" \
  --verify-mode download \
  --priority 5 \
  --pull true \
  --push true
```

### Step 4 — Push and clone

```bash
kk push
kk clone "rclone:nas-sftp:/storage/KK/MyGame" --pull
```

---

## 10. Local path / NAS (no rclone needed)

For a local folder, USB drive, or network share mounted as a drive letter / path,
KK has a built-in `local` driver that does not use `rclone` at all.

```bash
# Linux / macOS — NAS mounted at /Volumes/NAS
kk remote add local nas \
  --path /Volumes/NAS/KK/MyGame \
  --push true \
  --pull true

# Windows — network share mapped to Z:\
kk remote add local nas \
  --path "Z:\KK\MyGame" \
  --push true \
  --pull true

# Windows — UNC path
kk remote add local nas \
  --path "\\STUDIO-NAS\kk-lfs\MyGame" \
  --push true \
  --pull true
```

Push and clone:

```bash
kk push
kk clone local:/Volumes/NAS/KK/MyGame --pull      # macOS/Linux
kk clone "local:Z:\KK\MyGame" --pull              # Windows
```

---

## 11. Multi-remote setup

KK supports any number of remotes simultaneously. A common pattern is a fast
primary (NAS or S3) plus a cloud backup (MEGA or B2):

```bash
# Primary — fast local NAS
kk remote add local nas \
  --path /Volumes/NAS/KK/MyGame \
  --priority 10 \
  --push true \
  --pull true

# Backup — MEGA cloud
kk remote add rclone mega-backup \
  --remote mega:KK/MyGame \
  --priority 20 \
  --push true \
  --pull true

# Set the NAS as default
kk remote set-default nas

# Push to all push-enabled remotes at once
kk push --all-remotes

# Pull from the fastest available remote (priority order)
kk pull-file Assets/intro.mp4
```

View all configured remotes:

```bash
kk remote list
kk remote list --json
```

Check connectivity for all remotes:

```bash
kk remote check --all
kk remote check --all --json
```

### Replicating objects across remotes

After pushing, you can ensure all push-enabled remotes hold a complete copy of every live object using `kk objects sync`:

```bash
kk objects sync            # scan + replicate missing objects across all push-enabled remotes
kk objects sync --workers 8  # multi-threaded
```

You can also replicate on the fly during a pull with the `--sync` flag:

```bash
kk pull --sync             # merge history and replicate newly pulled objects to other remotes
kk pull-file --sync .      # materialise all files and replicate their objects
```

For the full replication guide, see [`docs/object-syncing.md`](object-syncing.md).

---

## 12. Performance Tuning

For faster uploads on high-bandwidth connections, you can configure performance options when adding a remote:

### Upload Timeout

Increase the HTTP timeout for large file uploads (default: 300 seconds):

```bash
kk remote add rclone mega-backup \
  --remote mega:KK/MyGame \
  --upload-timeout 600 \
  --push true --pull true
```

Useful for:
- Slow connections that take longer to upload large files
- Unstable networks that may cause retries

### Chunk Size (Google Drive native)

Configure the chunk size for large file uploads to Google Drive (default: 8 MB) by editing `.kk/config.json` directly:

```json
{
  "remotes": {
    "gdrive-main": {
      "type": "drive",
      "drive_folder_id": "xxx",
      "drive_auth_path": "C:/Users/admin/.config/kk/gdrive/creds.json",
      "chunk_size_mb": 16,
      "push": true,
      "pull": true
    }
  }
}
```

Larger chunks (16-32 MB) can improve upload speed for large files on fast connections.
Smaller chunks (4-8 MB) are better for unstable connections.

### Rclone Transfers

Control the number of parallel file transfers for rclone remotes (default: 4):

```bash
kk remote add rclone mega-backup \
  --remote mega:KK/MyGame \
  --rclone-transfers 8 \
  --push true --pull true
```

- Higher values (8-16): Faster uploads on fast, stable connections
- Lower values (1-2): Better for rate-limited providers (MEGA) or unstable networks

### Buffer Size

Configure the memory buffer per transfer for rclone (default: 16 MB):

```bash
kk remote add rclone s3-main \
  --remote s3:my-bucket/KK/MyGame \
  --buffer-size 32 \
  --push true --pull true
```

Larger buffers reduce system call overhead and can improve throughput.

### Disable Connection Pooling

Disable HTTP connection pooling if you encounter connection issues by editing `.kk/config.json` directly:

```json
{
  "remotes": {
    "gdrive-main": {
      "type": "drive",
      "drive_folder_id": "xxx",
      "drive_auth_path": "C:/Users/admin/.config/kk/gdrive/creds.json",
      "disable_connection_pool": true,
      "push": true,
      "pull": true
    }
  }
}
```

Connection pooling is enabled by default and improves performance for most users.

### Environment Variable Overrides

For quick testing without editing `.kk/config.json`, use environment variables:

```bash
# Override defaults for testing
KK_GDRIVE_UPLOAD_TIMEOUT=600 kk push
KK_GDRIVE_CHUNK_SIZE=16 kk push
KK_RCLONE_TRANSFERS=8 kk push
KK_RCLONE_BUFFER_SIZE=32 kk push
```

### Full Example: High-Performance Remote

```bash
kk remote add rclone s3-fast \
  --display-name "S3 High-Performance" \
  --remote s3:my-bucket/KK/MyGame \
  --rclone-transfers 16 \
  --buffer-size 32 \
  --upload-timeout 600 \
  --push true \
  --pull true \
  --tag high-performance
```

---

## 13. Troubleshooting

### `rclone` not found

```
kk: rclone binary not found
```

Ensure `rclone` is on your PATH:

```bash
rclone version     # should print version
which rclone       # macOS/Linux
Get-Command rclone # Windows PowerShell
```

If rclone is installed at a custom path, pass it explicitly when adding the remote:

```bash
kk remote add rclone mega-backup \
  --binary "C:\Tools\rclone\rclone.exe" \
  --remote mega:KK/MyGame \
  --push true --pull true
```

### Authentication expired

For OAuth-based providers (Google Drive, Dropbox, OneDrive), the rclone token
can expire. Re-authenticate:

```bash
rclone config reconnect gdrive:
rclone config reconnect dropbox:
rclone config reconnect onedrive:
```

Then verify connectivity:

```bash
rclone lsd gdrive:
kk remote check --all
```

### MEGA transfer quota exceeded

MEGA free accounts have ~10 GB/day transfer quota. If you exceed it:

- Wait 24 hours for the quota to reset.
- Reduce parallel workers: `KK_WORKERS=1 kk push`
- Upgrade your MEGA plan, or add a second storage remote as overflow.

### Slow uploads

```bash
# Increase parallelism for fast connections
kk push --workers 8

# Decrease for rate-limited or unstable connections
kk push --workers 1

# Or set permanently via environment variable
export KK_WORKERS=4
```

### Object verify failure

If `kk fsck` reports a corrupted object:

```bash
kk fsck --json
```

The most common causes are interrupted uploads or MEGA's download-verify
requirement (objects uploaded via MEGA must be re-downloaded to verify —
this is normal and expected with `--verify-mode download`).

Force a re-upload of the specific object:

```bash
kk push   # kk detects missing/corrupt objects and re-uploads them
```

### Path contains spaces (Windows)

Always quote paths with spaces on Windows:

```bash
kk remote add local nas --path "D:\Game Assets\KK\MyGame" --push true --pull true
kk clone "local:D:\Game Assets\KK\MyGame"
```

### Verify a working rclone remote before adding to KK

```bash
# List directory — should show content
rclone lsd mega:KK

# Upload a test file
echo "test" | rclone rcat mega:KK/_kk_test.txt

# Download it back
rclone cat mega:KK/_kk_test.txt

# Clean up
rclone deletefile mega:KK/_kk_test.txt
```

If all three succeed, `kk remote add rclone ...` will work.

---

## Quick reference

| Provider | rclone type | KK `--provider` | `--verify-mode` | Clone spec prefix |
|----------|-------------|-----------------|-----------------|-------------------|
| MEGA | `mega` | `mega` | `download` | `rclone:mega:` |
| Google Drive | `drive` | `google-drive` | `download` | `rclone:gdrive:` |
| Dropbox | `dropbox` | `dropbox` | `download` | `rclone:dropbox:` |
| OneDrive | `onedrive` | `onedrive` | `download` | `rclone:onedrive:` |
| Amazon S3 | `s3` | `s3` | `download` | `rclone:s3:` |
| Backblaze B2 | `b2` | `b2` | `download` | `rclone:b2:` |
| SFTP | `sftp` | `sftp` | `download` | `rclone:nas-sftp:` |
| Local / NAS | _(built-in)_ | `nas` | `local-hash` | `local:` |

---

## Further reading

- [`docs/object-syncing.md`](object-syncing.md) — syncing and replicating objects across multiple remotes
- [`docs/remote-layout.md`](remote-layout.md) — folder structure KK creates on every remote
- [`docs/history-bundles.md`](history-bundles.md) — how commit history travels via storage bundles
- [`docs/GIT_REMOTE.md`](git-remote.md) — switching between bundle mode and GitHub/GitLab
- [`docs/installation.md`](installation.md) — full installation and first-time setup
- [rclone supported providers](https://rclone.org/overview/) — complete list of 70+ backends
