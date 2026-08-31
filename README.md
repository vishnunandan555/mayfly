<p align="center">
  <img src="assets/icon.png" width="64" height="64" alt="MayFly Logo" />
</p>

<h1 align="center">MayFly</h1>

<p align="center">
  <strong>Zero-Dependency Ephemeral Secrets Workspace & In-Memory Process Injector</strong><br />
  <em>Built 100% from first principles using the Go standard library. 0 external dependencies.</em>
</p>

<p align="center">
  <a href="https://github.com/vishnunandan555/mayfly/releases/tag/v0.0.5"><img src="https://img.shields.io/badge/version-v0.0.5-blue?style=flat" alt="Version v0.0.5" /></a>
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

<p align="center">
  <a href="FEATURES.md"><strong>Complete Features Matrix →</strong></a> |
  <a href="docs/"><strong>Explore Full Documentation →</strong></a> |
  <a href="STDLIB.md"><strong>Zero-Dependency Matrix (STDLIB.md) →</strong></a> |
  <a href="deps-proof.txt"><strong>Dependency Verification Proof →</strong></a>
</p>

---

## The Problem MayFly Solves

Modern development workflows rely heavily on plaintext `.env` files stored directly on disk:

```env
STRIPE_SECRET_KEY=sk_live_9876543210
DATABASE_URL=postgres://admin:pass@localhost:5432/production
OPENAI_API_KEY=sk-proj-1234567890
```

### The Vulnerability: Disk-Bound Plaintext Secrets
When you execute `npm install`, `pip install`, or `cargo build`, third-party packages and install scripts run with full user privileges. Malicious dependencies or compromised supply-chain packages can quietly traverse your workspace, scrape `.env` files from disk, and exfiltrate production credentials before your application code ever starts.

### The MayFly Paradigm: Zero-Disk, Memory-Only Secret Lifecycle
MayFly eliminates `.env` files from your filesystem entirely:

1. **Authenticated Encrypted Storage**: All project secrets are encrypted at rest in a single binary vault (`~/.mayfly/vault.enc`) using **AES-256-GCM** authenticated encryption, protected by an **RFC 8018 PBKDF2-HMAC-SHA256** key derivation function (600,000 iterations).
2. **Volatile RAM Injection**: When you execute `mf <command>` (e.g., `mf npm run dev` or `mf python app.py`), MayFly decrypts the required secrets directly into **volatile memory (RAM)** and overlays them into the spawned child process environment.
3. **Guaranteed Zero-Disk Footprint**: Secrets never touch the disk. When the process terminates, all in-memory buffers are immediately overwritten with zeros via memory zeroization routines. File scrapers find zero credentials on disk.

```text
┌── [EPHEMERAL EXECUTION FLOW] ───────────────────────────────────────────────────────────┐
│ 1. Developer runs:           $ mf npm run dev                                           │
│ 2. MayFly unlocks vault:     AES-256-GCM decrypted in RAM (0 disk writes)               │
│ 3. Child process spawned:    Next.js / Node / Python receives secrets in process.env    │
│ 4. Filesystem state:         0 plaintext .env files on disk (Scrapers find nothing)     │
│ 5. Process termination:      Memory buffers zeroed immediately                          │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```

> 📚 **Deep Dive in the Docs:** For an in-depth threat analysis, attack trees, and OS security boundaries, see the [Why MayFly & Threat Model Documentation](docs/content/docs/why-mayfly.mdx).

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

#### Native PowerShell / cmd.exe (recommended on Windows):
```powershell
git clone https://github.com/vishnunandan555/mayfly.git
cd mayfly
$binDir = Join-Path $env:USERPROFILE ".local\bin"
New-Item -ItemType Directory -Force -Path $binDir | Out-Null
$env:CGO_ENABLED = "0"
go build -trimpath -ldflags="-s -w -buildid=" -o (Join-Path $binDir "mayfly.exe") .\cmd\mayfly
Copy-Item (Join-Path $binDir "mayfly.exe") (Join-Path $binDir "mf.exe")
$env:Path += ";$binDir"
```

#### Make-based build (Git Bash / MSYS2 / WSL / `choco install make`):
```bash
git clone https://github.com/vishnunandan555/mayfly.git
cd mayfly
make install
```

> `make` targets depend on Unix tooling (`mkdir -p`, `ln -sf`, `sha256sum`, `$(HOME)`) and are not native to plain `cmd.exe` or PowerShell without Git Bash/MSYS2/WSL or a Windows `make` installation.

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

See [STDLIB.md](STDLIB.md) for the complete 13-entry standard library substitution matrix.

---

## Documentation Hub

Explore the full interactive documentation, security architecture, and language integration guides in [`docs/`](docs/):

| Guide / Reference | Description | Link |
| :--- | :--- | :--- |
| 📋 **Features Matrix** | Complete capability matrix, security guarantees, & CLI specifications | [Features Specification](FEATURES.md) |
| 🚀 **Quickstart** | 2-minute setup, installation, and first secret injection | [Quickstart Guide](docs/content/docs/quickstart.mdx) |
| 🛡️ **Why MayFly & Threat Model** | In-depth analysis of supply-chain attacks & memory isolation | [Security Architecture](docs/content/docs/why-mayfly.mdx) |
| 🧠 **Core Architecture** | Vault format, binary layout, KDF derivation, and inode identity | [Security Model & Internals](docs/content/docs/architecture/security-model.mdx) |
| 💻 **CLI Reference** | Complete guide to all 12 CLI subcommands and options | [CLI Command Overview](docs/content/docs/cli/overview.mdx) |
| 🌐 **Universal Framework Guides** | Integration guides for Node.js, Python, Go, Rust, Docker, and more | [Language Guides](docs/content/docs/guides/universal.mdx) |
| 🔬 **Zero-Dependency Audit** | Comprehensive standard library substitution matrix and audit | [Zero-Dep Audit](docs/content/docs/reference/zero-dependency-audit.mdx) |

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
