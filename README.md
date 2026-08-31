<p align="center">
  <img src="assets/icon.png" width="64" height="64" alt="MayFly Logo" />
</p>

<h1 align="center">MayFly</h1>

<p align="center">
  <strong>Zero-Dependency Ephemeral Secrets Workspace & In-Memory Process Injector</strong><br />
  <em>Built 100% from first principles using the Go standard library. 0 external dependencies.</em>
</p>

<p align="center">
  <a href="go.mod"><img src="https://img.shields.io/badge/go-1.22+-00ADD8?style=flat&logo=go" alt="Go Version" /></a>
  <a href="STDLIB.md"><img src="https://img.shields.io/badge/dependencies-0%20external-brightgreen" alt="Zero Dependencies" /></a>
  <a href=".zero-dep.toml"><img src="https://img.shields.io/badge/Track-E%20%7C%20Security%20%26%20Crypto-blueviolet" alt="Track" /></a>
  <a href="#testing--verification"><img src="https://img.shields.io/badge/build-reproducible-success" alt="Reproducible Build" /></a>
  <a href="install.sh"><img src="https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-blue" alt="Platforms" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-AGPL--3.0-orange" alt="License" /></a>
</p>

<p align="center">
  <img src="assets/demo.gif" width="760" alt="MayFly Live Demo" />
</p>

---

## The Problem MayFly Solves

Most developers store secrets in plaintext `.env` files inside their project directories:
```env
STRIPE_SECRET_KEY=sk_live_9876543210
DATABASE_URL=postgres://admin:pass@localhost:5432/production
OPENAI_API_KEY=sk-proj-1234567890
```

**The Threat:** Whenever you run `npm install`, `pip install`, or `cargo build`, third-party supply-chain malware can scan your filesystem, read the `.env` file, and exfiltrate your production credentials before your application even starts.

**MayFly's Solution:** Never write `.env` files to disk. All secrets are stored in a single authenticated binary vault (`~/.mayfly/vault.enc`) encrypted with **AES-256-GCM** and derived via **RFC 8018 PBKDF2** (600,000 iterations). When you launch your application (`mayfly run npm start` or `mf run`), MayFly decrypts secrets directly into **volatile memory (RAM)**, attaches them to the child process environment, and immediately zeroes memory buffers when the process exits.

```text
┌── [LIVE WORKFLOW DEMONSTRATION] ────────────────────────────────────────────────┐
│ $ mf set STRIPE_SECRET=sk_live_9a8b7c6d5e4f3a2b1c                               │
│ [saved] Secret STRIPE_SECRET encrypted to vault (AES-256-GCM)                   │
│                                                                                 │
│ $ mf npm run dev                                                                │
│ [mayfly] unlocked vault in memory (600,000 PBKDF2 iterations)                   │
│ [mayfly] injected 3 secret(s) directly into volatile RAM                        │
│                                                                                 │
│ > my-app@0.1.0 dev                                                              │
│ > next dev                                                                      │
│   ▲ Next.js 15.5.24 - Local: http://localhost:3000                              │
│   ✓ Ready in 1.2s (Secrets active in process.env)                               │
│                                                                                 │
│ $ cat .env                                                                      │
│ cat: .env: No such file or directory  ← (Malicious package scanners find 0 keys)│
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## MayFly vs. The Alternatives

| Security & Workflow Feature | Plain `.env` | `direnv` | `dotenvx` | `1Password / Doppler` | **MayFly (`mf`)** |
| :--- | :---: | :---: | :---: | :---: | :---: |
| **In-Memory Injection (RAM only)** | ❌ Plaintext disk | ❌ Shell export | ❌ Plaintext disk | ⚠️ Partial | ✅ **100% Volatile RAM** |
| **Zero Disk Footprint (No `.env`)** | ❌ Files on disk | ❌ Files on disk | ❌ Files on disk | ⚠️ Requires daemon | ✅ **0 plaintext on disk** |
| **Protects from `npm/pip` disk scrapers**| ❌ Vulnerable | ❌ Vulnerable | ❌ Vulnerable | ⚠️ Partial | ✅ **Completely Neutralized** |
| **Zero Third-Party Dependencies** | ❌ Many packages | ❌ Many packages | ❌ npm bloat | ❌ Heavy SDKs/Daemons | ✅ **100% Pure Go Stdlib** |
| **100% Offline & Air-Gapped** | ✅ Offline | ✅ Offline | ⚠️ Hybrid | ❌ Cloud account / API | ✅ **Zero network calls** |
| **Hardware Inode Directory Binding** | ❌ Name only | ⚠️ Path string | ❌ None | ❌ Cloud workspace | ✅ **Physical `(Dev, Inode)`** |
| **Built-in Interactive Terminal UI** | ❌ None | ❌ None | ❌ None | ❌ Web dashboard | ✅ **Pure Go TUI Engine** |
| **Cryptographic Audit Log** | ❌ None | ❌ None | ❌ None | ⚠️ Cloud SIEM | ✅ **SHA-256 Hash Chain** |

---

## Quick Install

### Linux & macOS (Darwin):
```bash
curl -fsSL https://raw.githubusercontent.com/vishnunandan555/mayfly/main/install.sh | bash
```
*Auto-detects architecture, downloads release, cryptographically verifies published SHA-256 checksums, and installs `mayfly` and `mf` to `~/.local/bin/`.*

### Windows (PowerShell):
```powershell
irm https://raw.githubusercontent.com/vishnunandan555/mayfly/main/install.ps1 | iex
```
*Auto-verifies SHA-256 checksums via `Get-FileHash` before placing `mayfly.exe` and `mf.exe` into User PATH.*

### Offline Build from Source (0 External Dependencies):
```bash
git clone https://github.com/vishnunandan555/mayfly.git
cd mayfly
make install
```

---

## Framework One-Liner Quickstart

MayFly works out-of-the-box with any programming language, framework, or CLI tool:

| Framework / Stack | Run Command with MayFly | How It Works |
| :--- | :--- | :--- |
| **Next.js & React** | `mf npm run dev` | Injects into `process.env` in RAM; zero `.env.local` on disk |
| **Node.js & Express** | `mf node server.js` | Direct volatile RAM injection; no `dotenv.config()` needed |
| **Vite & Frontend** | `mf npx vite` | Secrets passed to build dev server without leaking to disk |
| **Python & FastAPI / Django** | `mf uvicorn main:app --reload` | Read via standard `os.environ.get()` |
| **Python & Poetry / Pipenv** | `mf poetry run python app.py` | Virtualenv child process inherits memory environment |
| **Go & Fiber / Gin** | `mf go run main.go` | Read via `os.Getenv()`; RAM zeroed on exit |
| **Rust & Actix / Axum** | `mf cargo run` | Read via `std::env::var()` |
| **Docker Compose** | `mf docker compose up` | Forwards host RAM environment into container services |
| **Prisma ORM** | `mf npx prisma migrate dev` | `DATABASE_URL` injected into DB migration CLI |

---

## Team Collaboration & Secret Sharing

MayFly is designed as an **air-gapped, 100% offline workstation secrets manager**. You don't need a third-party SaaS or paid cloud subscription to share secrets across engineering teams:

### 1. Exporting Encrypted Secrets for a Teammate
Export your project's encrypted vault entries into a portable file:
```bash
# Export all secrets in the current project into an encrypted backup
mf backup team-project.json
```

### 2. Sharing Safely
Transmit `team-project.json` via your team's existing encrypted communication channels (e.g. 1Password Vault, GPG-encrypted email, or secure team storage).

### 3. Importing on a New Machine
When a teammate clones the repository, they initialize the folder and restore the secrets:
```bash
cd my-project
mf init
mf restore team-project.json
```
Once restored, the teammate simply runs `mf npm run dev` with 100% memory isolation.

---

## How to Use MayFly

MayFly provides both **`mayfly`** and a short alias **`mf`**.

### 1. Global Terminal UI (`mayfly` or `mf`)
Run `mayfly` or `mf` from **any folder** to open the interactive dashboard:
- **Interactive Project Cards Grid:** Browse all registered project directories using arrow keys (`←`, `→`, `↑`, `↓`).
- **Secret Drilldown:** Press **`Enter`** on any project to drill down into its secrets list (values masked with `••••••••`).
- **Clipboard Copy:** Press **`C`** on any secret to copy its raw value to your system clipboard.
- **Key Shortcuts:**
  - `Enter`: Open project / Edit secret
  - `N`: New secret / Initialize directory
  - `C`: Copy secret value to clipboard
  - `V`: Reveal / Mask value
  - `D`: Delete secret
  - `S`: Plaintext leak scanner
  - `A`: Tamper-evident audit trail
  - `B`: Export encrypted backup
  - `Q` / `Esc`: Back / Exit

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

## Architecture: Recreated From Scratch

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

See [STDLIB.md](STDLIB.md) for the complete 12-entry substitution matrix.

---

## Security Model & Operational Notes

| Security Property | Implementation Details |
|---|---|
| **Encryption at Rest** | `AES-256-GCM` with random 12-byte nonces and 15-byte authenticated binary header AAD (`0600` permissions auto-enforced). |
| **Key Derivation** | `RFC 8018 PBKDF2-HMAC-SHA256` running **600,000 iterations** with random 16-byte cryptographic salt. |
| **Password Echo Suppression** | Master password prompts use low-level `termios` (`TCSETS`/`ECHO` disabled) so keystrokes never appear on screen. |
| **Ephemeral Alt-Screen Input** | `mf set` prompts in an ephemeral alternate screen buffer; secrets are visible for verification but vanish completely upon saving. |
| **Memory Isolation & Auto-Lock** | Decrypted values reside only in RAM during process execution. Memory buffers are zeroed with `runtime.KeepAlive`. Vault auto-locks after 15 min idle. |
| **Filesystem Isolation** | Binds project identity to physical storage `(Device, Inode)` to prevent path collision leaks. |
| **Audit Integrity** | SHA-256 hash-chained log (`~/.mayfly/audit.log`) mathematically proves no log entries were altered or deleted. |
| **Distribution Integrity** | Release binaries are deterministically compiled and cryptographically verified against published SHA-256 checksums in `install.sh`/`install.ps1`. |

> [!NOTE]
> **CI & Automation Environment Variables:**  
> The `MAYFLY_VAULT_PASSWORD` environment variable can be used in headless CI environments (e.g. GitHub Actions). For local interactive desktop development, avoid persisting `MAYFLY_VAULT_PASSWORD` into your `.bashrc`/`.zshrc` profile, as environment variables of running processes can be inspected via `/proc/<pid>/environ` by other processes running under the same user.

---

## Testing & Verification

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

## License

AGPL-3.0 License. See [LICENSE](LICENSE) for details.
