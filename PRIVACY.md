# Privacy Policy

**Effective Date:** May 21, 2026  
**Licensor:** Godynheil A. Quisto — godynheil@quisto.ph

## 1. Overview
KK ("KK Version Control System") is a free software command-line tool for large-file version control, distributed under **AGPL-3.0-or-later**. It operates entirely on your local machine and your chosen storage drivers. There is no centralized "KK Service," and the Licensor does not collect, transmit, or store any personal data, repository data, or usage information.

---

## 2. What Data KK Stores — and Where
All data processed by KK is stored either on your local device or on the remote storage drivers that **you** configure and control. The following local paths are created inside your project's `.kk/` directory and your OS user configuration directory:

| Location | Contents |
|---|---|
| `.kk/git/` | Embedded bare Git repository (commit history, tree objects, refs) |
| `.kk/objects/` | SHA-256-addressed large-file blob cache |
| `.kk/config.json` | Remote driver configuration (types, folder IDs, auth file paths) |
| `.kk/tracks.json` | Glob patterns of tracked large-file paths |
| `.kk/repo.json` | Repository metadata (repo ID, name, creation timestamp) |
| `.kk/logs/` | Local operational log files |
| `.kk/tmp/` | Temporary staging files (created and deleted during operations) |
| `<OS config dir>/KK/gdrive/<name>.json` | Google Drive OAuth tokens (see §4) |

> On Windows the OS config dir is `%APPDATA%`. On macOS/Linux it is `~/.config`.

The Licensor has no access to any of these locations. All data remains exclusively under your control.

---

## 3. No Telemetry
KK collects **zero** telemetry, analytics, crash reports, usage statistics, or diagnostics of any kind. The codebase contains no network calls to the Licensor's infrastructure, no "phone-home" behavior, and no third-party tracking SDKs.

---

## 4. Google Drive OAuth & Bundled Client ID

### 4.1 Bundled OAuth Application
When you run `kk setup gdrive`, KK uses a **bundled default Google OAuth Client ID** to initiate the authorization flow on your behalf. This Client ID identifies the KK application registered in Google Cloud Console. It does **not** grant the Licensor any access to your Google account or files — it is solely used to identify the application during the standard OAuth 2.0 consent flow.

You may supply your own OAuth credentials at any time by setting the following environment variables before running `kk setup gdrive`:
```
KK_GOOGLE_CLIENT_ID=<your-client-id>
KK_GOOGLE_CLIENT_SECRET=<your-client-secret>
```

### 4.2 How the OAuth Flow Works
KK implements the OAuth 2.0 Authorization Code Grant with **PKCE** (Proof Key for Code Exchange, RFC 7636) and CSRF state validation:
1. A temporary local HTTP server is opened on `127.0.0.1` at a **randomly chosen ephemeral port** to receive the redirect callback. This port is never exposed beyond your local machine.
2. Your browser is directed to Google's authorization endpoint.
3. After you grant consent, Google redirects back to `127.0.0.1` with an authorization code.
4. KK exchanges the code for tokens directly with Google (`oauth2.googleapis.com`) — no proxy, no Licensor server is involved.
5. The local HTTP server is shut down immediately after the callback is received.

### 4.3 Scopes Requested
By default, KK requests only the `https://www.googleapis.com/auth/drive.file` scope. This restricts KK's access to **only the files and folders that KK itself creates** in your Google Drive. It cannot read, modify, or delete any other files in your Drive.

However, if you need to clone or synchronize folders or Shared Drives (Team Drives) shared with your Google account by other users, KK needs permission to interact with files not created by the application itself. In these cases, you can explicitly opt-in to request the full drive scope (`https://www.googleapis.com/auth/drive`) by running:
```bash
kk setup gdrive --scope full
```
This grants KK the necessary access to find and sync shared resources in your Drive. All operations and credentials remain strictly local to your machine.

### 4.4 Token Storage
The resulting access token and refresh token are written to your OS user configuration directory (e.g., `%APPDATA%\KK\gdrive\default.json` on Windows) with restrictive file permissions (`0600` — readable only by your user account). These tokens are **never transmitted to the Licensor or any third party**.

---
 
## 5. Other Remote Drivers
KK supports additional remote drivers that you configure:

- **Local / NAS:** Files are transferred directly over your local filesystem or network share. No external network traffic.
- **rclone:** KK invokes the `rclone` binary installed on your system as a subprocess. Any data transmitted goes directly between your machine and the rclone-configured destination. KK does not intercept or log the transferred data. Consult [rclone's documentation](https://rclone.org/) for your specific remote.
- **SSH (planned):** Not yet implemented. This policy will be updated when SSH support is added.

You are solely responsible for securing credentials (OAuth tokens, passwords, SSH keys) for all configured drivers. Please consult the privacy policies of any third-party services you connect.

---

## 6. Your Responsibilities
Consistent with the AGPL-3.0-or-later **NO WARRANTY** and **Limitation of Liability** clauses, you acknowledge that:
- You are solely responsible for backing up your data and verifying that the Software performs correctly in your environment.
- Unauthorized access to your remote environments caused by compromised local tokens or misconfigured access controls is your responsibility.
- The Licensor has no ability to recover, restore, or access your data.

---

## 7. Children's Privacy
KK is a developer tool not intended for use by persons under 13 years of age (or the applicable age of digital consent in your jurisdiction). The Licensor does not knowingly collect data from minors.

---

## 8. Disclaimer of Liability
**THE LICENSED WORK IS PROVIDED "AS IS" AND "AS AVAILABLE", WITHOUT WARRANTY OF ANY KIND, WHETHER EXPRESS, IMPLIED, STATUTORY, OR OTHERWISE.**

TO THE MAXIMUM EXTENT PERMITTED BY APPLICABLE LAW, THE LICENSOR SHALL NOT BE LIABLE FOR ANY CLAIM, DAMAGE, LOSS, COST, EXPENSE, OR LIABILITY ARISING FROM THE USE OF THE SOFTWARE. THIS INCLUDES, BUT IS NOT LIMITED TO, DATA LOSS, ASSET CORRUPTION, SECURITY INCIDENTS, UNAUTHORIZED ACCESS, OR DISRUPTION FROM THIRD-PARTY SERVICES.

If liability cannot be fully excluded under applicable law, the Licensor's total cumulative liability shall not exceed the greater of: (1) the amount you paid to the Licensor in the twelve months before the claim arose, or (2) **USD $100**.

Full warranty disclaimers and liability limitations are detailed in the `LICENSE.MD` file.

---

## 9. Changes to This Policy
Revisions will be committed directly to the project repository with a dated commit message. Continued use of the Software after a revision constitutes acceptance of the updated policy.

---

## 10. Contact
For privacy-related inquiries or legal notices, contact:

**Godynheil A. Quisto**  
godynheil@quisto.ph

For general questions or bug reports, please open an issue in the public project repository.
