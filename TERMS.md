# Terms of Service

**Effective Date:** May 21, 2026  
**Licensor:** Godynheil A. Quisto — godynheil@quisto.ph

Welcome to KK ("Software"). By downloading, installing, or using KK, you agree to be bound by these Terms of Service ("Terms"). If you do not agree, do not use the Software.

---

## 1. Acceptance of Terms
By executing the KK binary, cloning the repository, or otherwise making use of the Software or its source code, you confirm that you have read and agreed to these Terms and that you have the legal capacity to enter into this agreement.

---

## 2. Description of Software
KK ("KK Version Control System") is a **source-available** command-line tool for large-file version control, distributed under a **Modified GNU General Public License, Version 2.0 (Modified GPL 2.0)**. It manages a local embedded Git repository, SHA-256-addressed object storage, and synchronization with user-configured remote storage drivers (Google Drive, local/NAS paths, rclone-supported destinations, and in a future version, SSH).

There is no hosted "KK Service." All operations run on your local machine and communicate directly with the remote drivers you configure. The Licensor provides no cloud infrastructure.

---

## 3. License and Permitted Use

### 3.1 Non-Commercial Use
KK is distributed under the **Modified GPL 2.0**. The following uses are permitted at no charge:
- Personal use
- Educational and academic use
- Research and evaluation use
- Development, testing, and non-production experimentation
- Noncommercial indie development, **provided** the Licensed Work is not used by a company, studio, agency, contractor, publisher, client project, paid production team, or other commercial production environment

### 3.2 Commercial Use — Restricted
**Commercial use of the Software, specifically selling the software or charging for providing distribution or support services, is prohibited under the Modified GPL 2.0.**

If you wish to charge others, sell the software, or provide paid distribution/support services, you must ask for and obtain a separate business/commercial license from the original developer (Licensor).

Unauthorized commercial use is a violation of these Terms and the Modified GPL 2.0, and may expose you to legal liability.

### 3.3 Attribution and License Retention
Any permitted distribution or modification of the Licensed Work must retain the original copyright notice, the `LICENSE.MD` file, and a clear reference to the source repository.

---

## 4. Trademarks
This License does not grant permission to use the names, trademarks, service marks, logos, branding, or trade dress of the Licensor, except as reasonably necessary to identify the origin of the Licensed Work.

The names **"KK"**, **"KK Version Control System"**, and **"kk"** may not be used to imply endorsement, sponsorship, or affiliation without prior written permission from the Licensor (godynheil@quisto.ph).

---

## 5. User Responsibilities
Because KK operates locally and depends entirely on your configuration, you are solely responsible for:

- **Data Backup:** Maintaining independent backups of your `.kk/git` database, `.kk/objects` blob cache, and all configuration files. KK does not provide a backup service.
- **Data Verification:** Verifying that the Software performs correctly in your environment before relying on it for any important or production workflow.
- **Credential Security:** Protecting all OAuth tokens, SSH keys, passwords, API keys, and `.kk/config.json`. These secrets are never transmitted to the Licensor.
- **Repository Integrity:** Ensuring that only authorized users have access to your project directory and configured remote drivers.
- **Third-Party Compliance:** Abiding by the Terms of Service, Acceptable Use Policies, and API usage limits of any third-party services you integrate with KK (e.g., Google Drive, MEGA, GitHub, rclone remotes).
- **Dependency Management:** Keeping the `rclone` binary (if used) and other local dependencies up to date.

---

## 6. Assumption of Risk
You acknowledge that version-control tools, file synchronization tools, cloud storage systems, remote object stores, and large-file management systems may affect source code, binary assets, project files, backups, and production workflows.

You assume **all risk** associated with installing, configuring, modifying, integrating, deploying, and using the Licensed Work. You are responsible for maintaining independent backups and validating that the Software performs correctly in your environment.

---

## 7. Acceptable Use
You agree **not** to use the Software to:
- Store, transmit, or distribute illegal, infringing, or harmful content.
- Circumvent any security or access control mechanism of a third-party service.
- Circumvent, remove, or disable the Commercial Use restriction in violation of the Modified GPL 2.0.
- Use the bundled Google OAuth Client ID for any purpose other than authorizing your own KK installation against Google Drive.

---

## 8. Google OAuth Client ID
The Software ships with a **bundled Google OAuth Client ID** to simplify Google Drive setup. This Client ID identifies the KK application registered in Google Cloud Console. It does **not** grant the Licensor any access to your Google account or files.

You may substitute your own credentials via the `KK_GOOGLE_CLIENT_ID` and `KK_GOOGLE_CLIENT_SECRET` environment variables.

The Licensor reserves the right to rotate, revoke, or replace the bundled Client ID at any time without notice. The Licensor is not liable for any disruption this may cause.

---

## 9. No Warranties
**THE LICENSED WORK IS PROVIDED "AS IS" AND "AS AVAILABLE", WITHOUT WARRANTY OF ANY KIND, WHETHER EXPRESS, IMPLIED, STATUTORY, OR OTHERWISE.**

To the maximum extent permitted by applicable law, the Licensor disclaims all warranties, including but not limited to:
- Warranties of merchantability, fitness for a particular purpose, title, non-infringement, accuracy, availability, reliability, security, compatibility, and error-free operation.
- Warranties that the Software will meet your requirements, operate without interruption, preserve data, prevent data loss, prevent security incidents, or be free from defects or vulnerabilities.

---

## 10. Limitation of Liability
**TO THE MAXIMUM EXTENT PERMITTED BY APPLICABLE LAW, THE LICENSOR SHALL NOT BE LIABLE FOR ANY CLAIM, DAMAGE, LOSS, COST, EXPENSE, OR LIABILITY ARISING FROM OR RELATED TO THE LICENSED WORK OR THESE TERMS, WHETHER BASED ON CONTRACT, TORT, NEGLIGENCE, STRICT LIABILITY, OR ANY OTHER LEGAL THEORY.**

This limitation includes, but is not limited to, liability for:
- Data loss, corrupted files, missing files, failed uploads, failed downloads, or repository corruption
- Remote storage failure or cloud provider issues
- Business interruption, lost profits, lost revenue, lost savings, loss of goodwill, or loss of use
- Security incidents or unauthorized access
- Third-party tool failures (including rclone, Git, and Google Drive API)
- Indirect, incidental, special, consequential, exemplary, or punitive damages

**If liability cannot be fully excluded under applicable law, the Licensor's total cumulative liability shall be limited to the greater of:**
1. the amount you paid directly to the Licensor for use of the Licensed Work during the twelve (12) months before the claim arose; or
2. **USD $100**.

---

## 11. Indemnification
You agree to indemnify, defend, and hold harmless the Licensor and Contributors from and against any and all third-party claims, liabilities, damages, losses, penalties, costs, and expenses (including reasonable attorneys' fees) arising out of or related to:
- Your use or misuse of the Software.
- Your violation of these Terms or the Modified GPL 2.0.
- Your violation of any third-party service provider's terms or policies.
- Any content or data you store, process, or transmit using the Software.

---

## 12. Third-Party Software and Services
KK integrates with or invokes third-party software and APIs. These are not part of the Licensed Work and are governed by their own license terms.

| Integration | Notes |
|---|---|
| **Google Drive API** | Governed by [Google's Terms of Service](https://policies.google.com/terms). |
| **rclone** | An independent open-source binary invoked as a subprocess; governed by its own license. |
| **Git** | KK embeds a bare Git repository and may invoke the system `git` binary for pass-through commands. |
| **SSH (planned)** | Not yet implemented; terms will be updated when available. |

The Licensor does not endorse, control, audit, or warrant these third-party services or tools and is not responsible for any damages or losses caused by them.

---

## 13. Open-Source Dependencies
KK's source code may incorporate open-source libraries governed by their own respective licenses, available in the project repository. Nothing in these Terms supersedes the rights granted to you by those licenses.

---

## 14. Termination
Any use of the Licensed Work in violation of these Terms or the Modified GPL 2.0 automatically terminates your rights under this License.

Upon termination, you must stop using the Licensed Work and destroy any copies in your possession or control, unless you have obtained a separate commercial license from the Licensor.

Sections 4 (Trademarks), 6 (Assumption of Risk), 9 (No Warranties), 10 (Limitation of Liability), 11 (Indemnification), and 14 (Termination) survive termination.

---

## 15. Modifications to the Software and Terms
- **Software:** The Licensor may modify, deprecate, or discontinue features at any time without notice.
- **Terms:** These Terms may be updated at any time. Updates will be reflected in the project repository with a dated commit. Continued use of the Software after a revision constitutes acceptance of the revised Terms.

---

## 16. Governing Law and Dispute Resolution
These Terms shall be governed by and construed in accordance with the laws of the jurisdiction in which the Licensor resides, without regard to conflict of law principles. Disputes shall first be attempted to be resolved through good-faith negotiation. If negotiation fails, disputes shall be submitted to the competent courts of that jurisdiction.

---

## 17. Severability
If any provision of these Terms is found to be invalid, illegal, or unenforceable, the remaining provisions shall continue in full force and effect.

---

## 18. Entire Agreement
These Terms, together with the `LICENSE.MD` file and `PRIVACY.md`, constitute the entire agreement between you and the Licensor regarding your use of the Software and supersede all prior agreements, representations, or understandings.

---

## 19. Contact
For commercial licensing, royalty terms, studio licensing, enterprise licensing, support terms, or any legal notices:

**Godynheil A. Quisto**  
godynheil@quisto.ph

For general questions or bug reports, please open an issue in the public project repository.
