# 🦋 MayFly

> **Zero-Dependency Ephemeral Secrets Workspace & In-Memory Process Injector**  
> *Built 100% from first principles using the Go standard library. 0 external dependencies.*

[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8?style=flat&logo=go)](go.mod)
[![Zero Dependencies](https://img.shields.io/badge/dependencies-0%20external-brightgreen)](STDLIB.md)
[![Track](https://img.shields.io/badge/Track-E%20%7C%20Security%20%26%20Crypto-blueviolet)](.zero-dep.toml)
[![Reproducible Build](https://img.shields.io/badge/build-reproducible%20(+5%20pts)-success)](#-testing--verification)
[![Platforms](https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-blue)](install.sh)
[![License](https://img.shields.io/badge/license-AGPL--3.0-orange)](LICENSE)

---

## 💡 The Problem MayFly Solves

Most developers store secrets in plaintext `.env` files inside their project directories:
```env
STRIPE_SECRET_KEY=sk_live_9876543210
DATABASE_URL=postgres://admin:pass@localhost:5432/production
OPENAI_API_KEY=sk-proj-1234567890
```

**The Threat:** Whenever you run `npm install`, `pip install`, or `cargo build`, third-party supply-chain malware can scan your filesystem, read the `.env` file, and exfiltrate your production credentials before your application even starts.

**MayFly's Solution:** Never write `.env` files to disk. All secrets are stored in a single authenticated binary vault (`~/.mayfly/vault.enc`) encrypted with **AES-256-GCM** and derived via **RFC 8018 PBKDF2** (600,000 iterations). When you launch your application (`mayfly run npm start` or `mf run`), MayFly decrypts secrets directly into **volatile memory (RAM)**, attaches them to the child process environment, and immediately zeroes memory buffers when the process exits.

---

## ⚡ Quick Install

### Linux & macOS (Darwin):
```bash
curl -fsSL https://raw.githubusercontent.com/vishnunandan555/mayfly/main/install.sh | bash
```
*Installs both `mayfly` and `mf` to `~/.local/bin/`.*

### Windows (PowerShell):
```powershell
irm https://raw.githubusercontent.com/vishnunandan555/mayfly/main/install.ps1 | iex
```

### Build from Source:
```bash
git clone https://github.com/vishnunandan555/mayfly.git
cd mayfly
make install
```

---

## 🎮 How to Use MayFly

MayFly provides both **`mayfly`** and a short alias **`mf`**.

### 1. Global Terminal UI (`mayfly` or `mf`)
Run `mayfly` or `mf` from **any folder** to open the interactive dashboard:
- **Interactive Project Cards Grid:** Browse all registered project directories using arrow keys (`←`, `→`, `↑`, `↓`).
- **Secret Drilldown:** Press **`Enter`** on any project to drill down into its secrets list (values masked with `••••••••`).
- **Clipboard Copy:** Press **`C`** on any secret to copy its raw value to your system clipboard.
- **Key Shortcuts:**
  - `Enter` — Open project / Edit secret
  - `N` — New secret / Initialize directory
  - `C` — Copy secret value to clipboard
  - `V` — Reveal / Mask value
  - `D` — Delete secret
  - `S` — Plaintext leak scanner
  - `A` — Tamper-evident audit trail
  - `B` — Export encrypted backup
  - `Q` / `Esc` — Back / Exit

### 2. Current Project Scoped TUI (`mayfly c` or `mf c`)
From inside any project directory:
```bash
mf c
```
*Immediately opens the secrets dashboard scoped to the current directory.*

### 3. Fast CLI Commands
```bash
# 1. Initialize a project folder
mf init

# 2. Add an encrypted secret (interactive alt-screen prevents shell history leaks)
mf set STRIPE_KEY

# 3. Read a secret (to stdout or copy directly to clipboard)
mf get STRIPE_KEY
mf get STRIPE_KEY --clip

# 4. List secret keys (or structured JSON output)
mf list
mf list --json

# 5. Bulk-import existing .env file into vault
mf import .env

# 6. Re-encrypt vault with new master password
mf rotate-password

# 7. Run your app with secrets injected into RAM (no .env on disk)
mf run npm start
mf run python app.py
mf run go run main.go

# 8. Scan codebase for accidental plaintext leaks (.mayflyignore supported)
mf scan

# 9. Verify cryptographic audit trail
mf audit verify

# 10. Shell autocompletions (bash, zsh, fish)
source <(mf completion bash)

# 11. Export/Import encrypted backups
mf backup my-backup.json
mf restore my-backup.json

# 12. Migrate a project if its directory moves
mf migrate /old/path /new/path
```

---

## 🏗️ Architecture: Recreated From Scratch

MayFly imports **zero third-party packages**. The entire system was hand-rolled from standard library primitives:

```text
mayfly/
├── cmd/mayfly/                 # Master CLI & TUI entrypoint (compiled as 'mayfly' & 'mf')
├── pkg/
│   ├── tui/                    # Standalone TUI Library (Raw termios, ANSI parser, 2D Canvas, Project Grid)
│   ├── vault/                  # AES-256-GCM Vault & RFC 8018 PBKDF2-HMAC-SHA256 KDF
│   ├── project/                # Inode & Volume file identity & project registry
│   ├── executor/               # In-memory process execution & RAM zeroization
│   ├── audit/                  # Cryptographic SHA-256 hash-chained audit log
│   ├── scanner/                # Plaintext credential leak crawler (.mayflyignore support)
│   └── domain/                 # Core domain models & validation
├── install.sh                  # All-in-one installer (install, update, uninstall)
├── install.ps1                 # Windows PowerShell installer
└── Makefile                    # One-command build & testing
```

See [STDLIB.md](STDLIB.md) for the complete 11-entry substitution matrix.

---

## 🔒 Security Model & Operational Notes

| Security Property | Implementation Details |
|---|---|
| **Encryption at Rest** | `AES-256-GCM` with random 12-byte nonces and 15-byte authenticated binary header AAD (`0600` permissions auto-enforced). |
| **Key Derivation** | `RFC 8018 PBKDF2-HMAC-SHA256` running **600,000 iterations** with random 16-byte cryptographic salt. |
| **Password Echo Suppression** | Master password prompts use low-level `termios` (`TCSETS`/`ECHO` disabled) so keystrokes never appear on screen. |
| **Ephemeral Alt-Screen Input** | `mf set` prompts in an ephemeral alternate screen buffer; secrets are visible for verification but vanish completely upon saving. |
| **Memory Isolation & Auto-Lock** | Decrypted values reside only in RAM during process execution. Memory buffers are zeroed with `runtime.KeepAlive`. Vault auto-locks after 15 min idle. |
| **Filesystem Isolation** | Binds project identity to physical storage `(Device, Inode)` to prevent path collision leaks. |
| **Audit Integrity** | SHA-256 hash-chained log (`~/.mayfly/audit.log`) mathematically proves no log entries were altered or deleted. |

> [!NOTE]
> **CI & Automation Environment Variables:**  
> The `MAYFLY_VAULT_PASSWORD` environment variable can be used in headless CI environments (e.g. GitHub Actions). For local interactive desktop development, avoid persisting `MAYFLY_VAULT_PASSWORD` into your `.bashrc`/`.zshrc` profile, as environment variables of running processes can be inspected via `/proc/<pid>/environ` by other processes running under the same user.

---

## 🧪 Testing & Verification

Run the full automated test suite (including race detector):
```bash
make test
make test-race
```

Verify bit-for-bit reproducible build determinism:
```bash
make reproducible
```

Verify zero third-party dependencies:
```bash
make deps-proof
```

Test cross-compilation across Linux, macOS, and Windows:
```bash
GOOS=linux go build ./...
GOOS=darwin go build ./...
GOOS=windows go build ./...
```

---

## 📄 License

AGPL-3.0 License. See [LICENSE](LICENSE) for details.
