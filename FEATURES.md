# MayFly Features Matrix

<p align="center">
  <img src="assets/icon.png" width="64" height="64" alt="MayFly Logo" />
</p>

<h1 align="center">MayFly Feature Specification & Capabilities</h1>

<p align="center">
  <strong>Zero-Dependency Ephemeral Secrets Workspace & In-Memory Process Injector</strong><br />
  <em>Built 100% from first principles using the Go standard library (0 external dependencies).</em>
</p>

<p align="center">
  <a href="https://github.com/vishnunandan555/mayfly/releases/tag/v0.0.5"><img src="https://img.shields.io/badge/version-v0.0.5-blue?style=flat" alt="Version v0.0.5" /></a>
  <a href="go.mod"><img src="https://img.shields.io/badge/go-1.22+-00ADD8?style=flat&logo=go" alt="Go Version" /></a>
  <a href="STDLIB.md"><img src="https://img.shields.io/badge/dependencies-0%20external-brightgreen" alt="Zero Dependencies" /></a>
  <a href=".zero-dep.toml"><img src="https://img.shields.io/badge/Track-E%20%7C%20Security%20%26%20Crypto-blueviolet" alt="Track" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-AGPL--3.0-orange" alt="License" /></a>
  <a href="install.sh"><img src="https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-blue" alt="Platforms" /></a>
</p>

---

## Table of Contents

1. [Executive Overview](#1-executive-overview)
2. [Core Architecture & Memory Injection](#2-core-architecture--memory-injection)
3. [Cryptographic Security & Vault Engine](#3-cryptographic-security--vault-engine)
4. [Hardware Inode & Project Identity](#4-hardware-inode--project-identity)
5. [Interactive Terminal UI (TUI)](#5-interactive-terminal-ui-tui)
6. [Plaintext Leak Scanner & Git Hooks](#6-plaintext-leak-scanner--git-hooks)
7. [Cryptographic Audit Trail (SHA-256 Hash Chain)](#7-cryptographic-audit-trail-sha-256-hash-chain)
8. [Developer Experience & Workspace Tooling](#8-developer-experience--workspace-tooling)
9. [Zero-Dependency Engineering (Standard Library Matrix)](#9-zero-dependency-engineering-standard-library-matrix)
10. [Comprehensive CLI Command Reference](#10-comprehensive-cli-command-reference)
11. [Cross-Platform & Operational Matrix](#11-cross-platform--operational-matrix)

---

## 1. Executive Overview

MayFly re-architects developer secret management from the ground up. By replacing disk-bound `.env` files with volatile in-memory process injection and AES-256-GCM encrypted binary storage, MayFly neutralizes supply-chain attacks (`npm install`, `pip install`, malicious build scripts) that target unencrypted credentials on developer machines.

```text
┌── [MAYFLY LIFECYCLE PARADIGM] ────────────────────────────────────────────────────────┐
│                                                                                       │
│   1. ENCRYPTED AT REST      2. VOLATILE DECRYPTION       3. IN-MEMORY INJECTION       │
│   ~/.mayfly/vault.enc   ──► AES-256-GCM decrypted   ──►  Injected into child RAM      │
│   (PBKDF2 600k rounds)      directly in memory           (exec.Cmd.Env overlay)       │
│                                                                  │                    │
│   5. DISK STATE: CLEAN      4. ZERO-RESIDUAL EXIT                ▼                    │
│   0 plaintext files on disk  RAM buffers wiped with  ◄── Process terminates           │
│   (Scrapers find nothing)    runtime.KeepAlive           (Next.js / Node / Python)    │
│                                                                                       │
└───────────────────────────────────────────────────────────────────────────────────────┘
```

### Feature Highlights at a Glance

| Feature Area | Key Capability | Primary Benefit |
| :--- | :--- | :--- |
| **Volatile RAM Injection** | Secrets decrypted strictly in process RAM | Zero plaintext `.env` files written to disk |
| **AEAD Vault Storage** | AES-256-GCM + RFC 8018 PBKDF2 (600,000 rounds) | GPU brute-force resistant encrypted store |
| **Physical Inode Binding** | Projects identified by storage `(Device, Inode)` | Immune to path tampering and symlink attacks |
| **Pure Go TUI Engine** | Custom double-buffered 2D character canvas | Rich project grid & secret drilldown with 0 deps |
| **Leak Scanner** | 15+ regex signatures + dangerous file detection | Catches unencrypted tokens before git commit |
| **Audit Hash Chain** | SHA-256 linked event blockchain | Mathematically provable tamper detection |
| **Zero Dependencies** | 100% Go standard library implementation | No external supply-chain risk whatsoever |

---

## 2. Core Architecture & Memory Injection

### 2.1 Transparent Process Wrapper
MayFly transparently wraps any development server, CLI tool, or build command without requiring SDK changes or code modifications.

- **Universal Framework Support**: Works natively with Next.js, Vite, Node.js, Python (Django/FastAPI/Poetry), Go, Rust (Cargo), Docker Compose, Prisma, Rails, and more.
- **Direct Invocation**: Prefix any command with `mf` or `mayfly`:
  ```bash
  mf npm run dev
  mf uvicorn main:app --reload
  mf cargo run
  mf docker compose up
  ```
- **Process Table Isolation**: Secrets are merged into the spawned child process environment table (`exec.Cmd.Env`) in volatile memory.

### 2.2 Memory Hygiene & Zeroization
- **Guaranteed Zero-Disk Footprint**: Secrets exist exclusively in memory buffers during the execution lifetime of the target command.
- **Volatile Buffer Wiping**: Upon command completion or process exit, memory buffers holding decrypted keys and plaintext payloads are immediately zeroed out.
- **Garbage Collector Safety**: Leverages `runtime.KeepAlive` to ensure cryptographic buffers are not prematurely collected before explicit zeroization routines complete.

---

## 3. Cryptographic Security & Vault Engine

```text
┌── [ENCRYPTED VAULT BINARY LAYOUT: ~/.mayfly/vault.enc] ────────────────────────────────┐
│  Magic Header   │ Salt (Random) │ Nonce (GCM) │ Ciphertext + Poly1305 Auth Tag (AEAD)  │
│  MAYFLY_VAULT_v1│    16 Bytes   │  12 Bytes   │       AES-256-GCM Encrypted Body       │
│    (15 Bytes)   │               │             │       (Projects, Keys, Payloads)       │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

### 3.1 Authenticated Encryption (AEAD)
- **Cipher**: `AES-256-GCM` (Galois/Counter Mode) providing both confidentiality and cryptographic integrity verification.
- **Nonces**: Cryptographically random 12-byte initialization vectors generated via `crypto/rand` per write operation.
- **Header Authentication**: A 15-byte magic header (`MAYFLY_VAULT_v1`) is bound into the GCM Additional Authenticated Data (AAD) to prevent format substitution attacks.

### 3.2 Key Derivation Function (KDF)
- **Standard**: RFC 8018 compliant **PBKDF2-HMAC-SHA256** implemented directly with Go standard library primitives (`crypto/hmac`, `crypto/sha256`).
- **Iteration Work Factor**: **600,000 rounds** (OWASP recommended baseline) to ensure robust resistance against offline GPU/ASIC brute-force attacks.
- **Salt Generation**: Unique 16-byte cryptographically secure salt generated for each vault initialization and re-encryption.

### 3.3 Storage Hardening & Operational Defenses
- **Strict File Permissions**: Vault file (`~/.mayfly/vault.enc`) automatically enforces POSIX `0600` (read/write by owner only).
- **Atomic File Replacement**: All disk persistence uses an atomic write pattern (`temp file -> fsync -> rename`) to eliminate corrupted state risks during sudden power loss.
- **Master Password Rotation**: `mf rotate-password` decrypts all existing entries, generates a fresh 16-byte cryptographic salt, and re-encrypts the entire vault under the new master password.
- **Soft Brute-Force Lockout**: 5 consecutive incorrect master password attempts trigger an automated 30-second lockout tracked in `~/.mayfly/meta.json`.
- **Inactivity Auto-Lock**: Decrypted master keys clear from memory after 15 minutes of inactivity.
- **Encrypted Snapshots**: `mf backup` and `mf restore` allow exporting and restoring password-protected encrypted snapshots for team disaster recovery and migration.

---

## 4. Hardware Inode & Project Identity

MayFly anchors secrets to the physical filesystem geometry rather than volatile directory path strings:

```text
┌── [PHYSICAL INODE RESOLUTION] ─────────────────────────────────────────────────────────┐
│                                                                                       │
│   Project Directory: /home/dev/projects/my-api                                       │
│          │                                                                            │
│          ├──► filepath.EvalSymlinks (Resolves canonical filesystem path)              │
│          └──► syscall.Stat_t (Extracts physical Device ID + Inode Number)             │
│                     │                                                                 │
│                     ▼                                                                 │
│               Physical Identity: (Dev: 2049, Ino: 5242881)                            │
│                                                                                       │
└───────────────────────────────────────────────────────────────────────────────────────┘
```

- **Physical Device & Inode Binding**: Uses `(Device, Inode)` on Linux/Darwin and `FileIndex` on Windows (`GetFileInformationByHandle`) to bind secrets to physical storage.
- **Symlink & Alias Immunity**: Resolves canonical paths through `filepath.EvalSymlinks` to protect against symlink-based privilege escalation or path hijacking.
- **Path Collision Protection**: Even if two folders share the same name or relative location, their cryptographic identity remains strictly isolated.
- **Project Directory Migration**: `mf migrate <OLD_PATH> <NEW_PATH>` gracefully updates hardware bindings when a project repository is moved across disks or parent directories.
- **In-Memory Registry Caching**: `~/.mayfly/projects.json` is cached in memory per process execution and invalidated on write for high-performance CLI operations.

---

## 5. Interactive Terminal UI (TUI)

MayFly contains a complete, zero-dependency Terminal UI framework built from raw terminal primitives:

```text
┌── [MAYFLY INTERACTIVE TUI DASHBOARD] ──────────────────────────────────────────────────┐
│  MayFly Secrets Workspace                             [1/3 Projects] [Vault: UNLOCKED]│
│ ┌──────────────────────┐ ┌──────────────────────┐ ┌──────────────────────┐            │
│ │ ▶ my-next-app        │ │ backend-api          │ │ payment-worker       │            │
│ │ /home/dev/web/app    │ │ /home/dev/api        │ │ /home/dev/workers    │            │
│ │ 4 Secrets (Dev:2049) │ │ 8 Secrets (Dev:2049) │ │ 3 Secrets (Dev:2049) │            │
│ └──────────────────────┘ └──────────────────────┘ └──────────────────────┘            │
│                                                                                       │
│  [Enter] Open Project  [N] New Secret  [C] Copy  [V] Reveal  [S] Scanner  [Q] Exit    │
└───────────────────────────────────────────────────────────────────────────────────────┘
```

### 5.1 First-Principles Terminal Engine
- **Raw Mode Controller**: Directly manipulates terminal line discipline using `syscall.SYS_IOCTL` (`TCGETS`/`TCSETS`) on Linux/macOS and Windows Console API (`SetConsoleMode`), bypassing external libraries.
- **Double-Buffered 2D Canvas**: Renders flicker-free character grids with differential cell updates.
- **ANSI Key Event Parser**: Real-time finite-state machine parsing multi-byte escape sequences (Arrow keys, Esc, Tab, Shift-Tab, UTF-8 runes).
- **Unicode Width Alignment**: Built-in East Asian Width boundary checker ensuring correct column alignment for wide characters and emojis.
- **`NO_COLOR` Standard**: Fully respects the `NO_COLOR` specification for accessibility and CI logs.

### 5.2 Interactive Dashboard Features
- **Project Cards Grid**: Responsive, multi-column visual grid of registered projects navigable via arrow keys (`←`, `→`, `↑`, `↓`).
- **Secret Drilldown & Masking**: Press `Enter` on any project to drill down into secrets; secret values default to masked bullets (`••••••••`).
- **Value Reveal Toggle**: Press `V` to toggle between masked and plaintext secret displays.
- **Project-Scoped Quick View**: Run `mf c` or `mf current` from any folder to instantly open the TUI scoped to the current directory.
- **Interactive Modals**: Full keyboard-navigated forms for creating, editing, and deleting secrets.

### 5.3 Leak-Proof Alt-Screen Prompts
- **Zero-History Ephemeral Screens**: `mf set` and `mf get` switch to the terminal alternate screen buffer (`\x1b[?1049h`). Secrets entered or viewed vanish completely upon exit without leaving artifacts in the terminal scrollback history.
- **OSC 52 Clipboard Copy**: Pure ANSI OSC 52 escape sequences (`\x1b]52;c;...`) copy secrets directly to the system clipboard from both TUI (`C` key) and CLI (`--clip` flag), with fallback to platform utilities (`xclip`, `wl-copy`, `pbcopy`, `clip.exe`).

---

## 6. Plaintext Leak Scanner & Git Hooks

MayFly features an integrated static analysis engine designed to detect accidental credential leaks before they are committed to version control.

```text
$ mf scan
[CRITICAL] web/server.js:14:22 - Detected OpenAI API Key (sk-proj-...)
[CRITICAL] config/database.js:8:15 - Detected PostgreSQL Connection URI with password
[WARNING]  deploy/Dockerfile:5:1 - Hardcoded ENV secret assignment
Found 2 critical issues, 1 warning.
```

### 6.1 Scanner Capabilities
- **Bounded Directory Crawler**: High-speed recursive filesystem traversal using `path/filepath.WalkDir`.
- **Ignore File Support**: Full support for `.mayflyignore` patterns to exclude legitimate fixtures and vendor directories (`node_modules`, `.git`, `dist`, `target`).
- **Dangerous File Detection**: Flags unencrypted sensitive files (`.env*`, `.pem`, `.key`, `.p12`, `.pfx`, `id_rsa`, `id_ecdsa`, `id_dsa`, `pip.conf`, `.npmrc`).
- **15+ Built-in Secret Signatures**:

| Credential Type | Detection Heuristic / Pattern | Severity |
| :--- | :--- | :--- |
| **AWS Access Key** | `AKIA[0-9A-Z]{16}` | `CRITICAL` |
| **AWS Secret Key** | Standard base64 40-character secret pattern | `CRITICAL` |
| **Stripe Live Key** | `sk_live_[0-9a-zA-Z]{24,}` | `CRITICAL` |
| **Stripe Restricted Key** | `rk_live_[0-9a-zA-Z]{24,}` | `CRITICAL` |
| **OpenAI Secret Key** | `sk-proj-[a-zA-Z0-9_-]{48,}` | `CRITICAL` |
| **Anthropic API Key** | `sk-ant-api03-[a-zA-Z0-9_-]{93,}` | `CRITICAL` |
| **GitHub Token** | `ghp_[a-zA-Z0-9]{36}`, `gho_`, `github_pat_` | `CRITICAL` |
| **Slack Bot / User Token** | `xoxb-[0-9]{11,}-[0-9]{11,}-[a-zA-Z0-9]{24}` | `CRITICAL` |
| **Twilio API Key** | `SK[0-9a-fA-F]{32}` | `CRITICAL` |
| **SendGrid API Key** | `SG\.[a-zA-Z0-9_-]{22}\.[a-zA-Z0-9_-]{43}` | `CRITICAL` |
| **Mailgun Private Key** | `key-[0-9a-zA-Z]{32}` | `CRITICAL` |
| **Database Connection Strings** | `postgres://`, `mysql://`, `mongodb+srv://` with credentials | `CRITICAL` |
| **Bearer & JWT Tokens** | `eyJhbGciOi...` base64 encoded JWT headers | `WARNING` |
| **NPM Auth Tokens** | `//registry.npmjs.org/:_authToken=...` | `CRITICAL` |
| **Dockerfile Secrets** | Hardcoded `ENV` declarations with sensitive variable names | `WARNING` |

### 6.2 CI/CD & Git Automation
- **Git Pre-Commit Hook**: `mf install-hook` installs a zero-dependency pre-commit script in `.git/hooks/pre-commit` that runs `mf scan` before every commit.
- **Hook Removal**: `mf uninstall-hook` cleanly uninstalls the hook.
- **Deterministic Exit Codes**:
  - `0`: Workspace clean.
  - `1`: Found `CRITICAL` secret leaks.
  - `2`: Found `WARNING` findings only.
- **Structured JSON Output**: `mf scan --json` for direct ingestion into CI security pipelines.
- **Severity Filtering**: `mf scan --severity CRITICAL` to filter out minor warnings.

---

## 7. Cryptographic Audit Trail (SHA-256 Hash Chain)

All vault operations generate an immutable, tamper-evident audit log stored in `~/.mayfly/audit.log`.

```text
┌── [SHA-256 HASH-CHAINED AUDIT LOG STRUCTURE] ─────────────────────────────────────────┐
│                                                                                       │
│  [Event 1: VAULT_INITIALIZED]                                                         │
│  Seq: 1  | Hash: a1b2... │ PrevHash: "00000000000000000000000000000000"              │
│                                  │                                                    │
│                                  ▼                                                    │
│  [Event 2: SECRET_SET]                                                                │
│  Seq: 2  | Hash: c3d4... │ PrevHash: a1b2... (Incorporates Event 1 Hash)              │
│                                  │                                                    │
│                                  ▼                                                    │
│  [Event 3: SECRET_INJECTED]                                                           │
│  Seq: 3  | Hash: e5f6... │ PrevHash: c3d4... (Incorporates Event 2 Hash)              │
│                                                                                       │
└───────────────────────────────────────────────────────────────────────────────────────┘
```

### 7.1 Integrity Guarantees
- **Blockchain-Style Hash Chaining**: Every log record calculates its SHA-256 digest over the current event data *and* the previous event's hash. Modifying or deleting any historical record breaks the cryptographic chain.
- **Cryptographic Verification**: `mf audit verify` traverses the entire log history, recalculating hashes from Genesis to verify zero tampering.
- **Windows UTF-8 BOM Sanitization**: Automatically strips Byte Order Marks (`\xef\xbb\xbf`) emitted by Windows text editors to prevent hash verification false positives.

### 7.2 Logged Actions
- `VAULT_INITIALIZED`, `VAULT_UNLOCKED`, `VAULT_PASSWORD_ROTATED`
- `PROJECT_INITIALIZED`, `PROJECT_MIGRATED`, `PROJECT_DELETED`
- `SECRET_SET`, `SECRET_GET`, `SECRET_DELETED`, `SECRET_IMPORTED`
- `SECRET_INJECTED` (with target command name and child process exit code)
- `SCAN_COMPLETED`, `BACKUP_CREATED`, `BACKUP_RESTORED`

### 7.3 Log Inspection & Filtering
- Human-readable aligned column view: `mf audit`
- Stream last *N* events: `mf audit --tail 20`
- Machine-readable JSON export: `mf audit --json`

---

## 8. Developer Experience & Workspace Tooling

### 8.1 Shell Environment Export (`mf env`)
Allows exporting project secrets as shell environment variables for legacy scripting workflows:
```bash
# Evaluate decrypted secrets directly into your current shell session
eval $(mf env)

# Export for Fish shell
eval (mf env --shell fish)

# Export as JSON object
mf env --shell json
```

### 8.2 Configuration Template Engine (`mf template`)
Renders configuration templates by replacing `{{ SECRET_NAME }}` placeholders with decrypted values in-memory:
```bash
# Render to stdout
mf template config.template.yaml

# Render directly to an output file
mf template config.template.json --output config.json
```

### 8.3 Cross-Project Key Diffing (`mf diff`)
Compares secret key sets between two projects to identify missing configuration variables without revealing secret values:
```bash
# Compare current project against another service
mf diff ../staging-api

# Compare two arbitrary projects
mf diff /path/to/project-a /path/to/project-b
```

### 8.4 Vault Health & Integrity Diagnostics (`mf status` / `mf check`)
- `mf status` (or `mf doctor`): Summarizes vault health, lock state, registered project count, secret totals, and audit chain length.
- `mf check`: Validates the binary vault header, verifies the entire audit log hash chain, and cleans up stale project registry entries pointing to deleted directories.

### 8.5 Bulk `.env` File Migration (`mf import`)
Enables one-command migration from legacy `.env` files into the encrypted vault:
```bash
# Import default .env
mf import

# Import a specific environment file and securely delete the plaintext original
mf import .env.production --delete
```

### 8.6 Headless CI & Automation Integration
- **`--password-stdin` Flag**: Securely pass the vault master password via standard input pipe:
  ```bash
  echo "$VAULT_PASS" | mf --password-stdin npm start
  ```
- **Process Memory Safety**: Using stdin is significantly safer than passing secrets via `MAYFLY_VAULT_PASSWORD` environment variables, which can be inspected via `/proc/<pid>/environ` by other processes running under the same user.

### 8.7 Shell Autocompletions (`mf completion`)
Generates native, fast autocompletion scripts for:
```bash
source <(mf completion bash)
source <(mf completion zsh)
mf completion fish | source
```

### 8.8 In-Process Self-Updater (`mf update`)
- **Direct GitHub Releases Streaming**: Checks for new versions and downloads updates directly via `net/http`.
- **SHA-256 Checksum Attestation**: Automatically downloads `checksums.txt`, computes the local binary hash, and cryptographically verifies the download.
- **Atomic Binary Swapping**: Replaces the executing binary in-place (utilizing a `.old` rename swap on Windows where open files are locked).

---

## 9. Zero-Dependency Engineering (Standard Library Matrix)

MayFly ships with an **empty `go.mod` require block**. Every system component was built from scratch using the Go standard library:

| # | Feature / Component | Replaced Third-Party Packages | Go Stdlib Replacement Primitives |
|---|---|---|---|
| **1** | **Terminal Raw Mode** | `x/term`, `muesli/termenv` | `syscall.SYS_IOCTL`, `syscall.TCGETS`/`TCSETS`, Windows Console API |
| **2** | **TUI Engine & Widgets** | `bubbletea`, `tview`, `termbox-go` | `bytes.Buffer`, 2D character canvas, ANSI state machine |
| **3** | **ANSI Key Parser** | `bubbletea/key`, `mattn/go-tty` | `unicode/utf8`, Streaming Finite-State Machine |
| **4** | **Key Derivation (KDF)** | `x/crypto/pbkdf2` | RFC 8018 `crypto/hmac`, `crypto/sha256` (600,000 iterations) |
| **5** | **Encrypted Vault** | `go-sqlite3`, `bbolt`, `go-keyring` | `crypto/aes`, `crypto/cipher` (AES-GCM), `crypto/rand` |
| **6** | **Process Injection** | `godotenv`, `gotenv`, `dotenvx` | `os.Environ`, `os/exec.CommandContext`, buffer zeroization |
| **7** | **Tamper-Evident Audit** | `logrus`, `zap`, SIEM databases | `crypto/sha256`, `encoding/hex`, `encoding/json` hash chain |
| **8** | **Project Identity** | `uuid`, `go-git` | `syscall.Stat_t` (`Dev`, `Ino`), Windows `FileIndex`, `EvalSymlinks` |
| **9** | **Leak Scanner** | `trufflehog`, `gitleaks` | `path/filepath.WalkDir`, `regexp`, `bufio.Scanner` |
| **10** | **Terminal Colors** | `fatih/color`, `chalk` | Custom ANSI SGR escape sequence builder, `NO_COLOR` |
| **11** | **Rune Widths** | `mattn/go-runewidth` | Custom Unicode East Asian width boundary calculator |
| **12** | **Clipboard Controller** | `atotto/clipboard`, `x/clipboard` | ANSI OSC 52 protocol escape sequences (`\x1b]52;c;...`) |
| **13** | **Self-Updater** | `go-selfupdate`, `go-github-selfupdate` | `net/http`, `crypto/sha256`, Windows `.old` atomic swap |

---

## 10. Comprehensive CLI Command Reference

Both `mayfly` and the short alias `mf` provide full access to all subcommands:

```text
Usage:
  mf <COMMAND> [ARGS...]          Inject secrets into RAM and execute child process directly
  mf                              Launch interactive Global TUI Dashboard (Project Grid)
  mf c, mf current                Launch TUI scoped directly to current project directory
```

| Command | Arguments / Flags | Description |
| :--- | :--- | :--- |
| **`mf <cmd>`** | `[args...]` | Transparent execution: decrypts secrets and executes process in RAM |
| **`mf init`** | `[-path DIR]` | Register current folder (or target path) as a MayFly project |
| **`mf set`** | `<KEY> [--clip] [VALUE]` | Add/update a secret (uses zero-history alt-screen if interactive) |
| **`mf get`** | `<KEY> [--clip] [--raw]` | View secret value in zero-history screen, copy to clip, or print raw |
| **`mf list`** | `[--json]` | List all registered secret keys for the current project |
| **`mf delete`** | `<KEY>` | Remove a secret from the vault |
| **`mf run`** | `<cmd> [args...]` | Explicit in-memory process execution wrapper |
| **`mf import`** | `[FILE] [--delete]` | Import `.env` file into vault (optionally shred original plaintext) |
| **`mf env`** | `[--shell bash\|fish\|json]` | Export secrets formatted for shell environment evaluation |
| **`mf template`**| `<FILE> [--output F]` | Render config template with `{{ SECRET_NAME }}` replaced |
| **`mf diff`** | `[PATH_A] [PATH_B]` | Compare secret key names between two projects |
| **`mf scan`** | `[DIR] [--json] [--severity CRITICAL\|WARNING]` | Scan directory for plaintext secret leaks and credentials |
| **`mf install-hook`** | — | Install pre-commit hook in `.git/hooks/` to run `mf scan` |
| **`mf uninstall-hook`** | — | Remove the MayFly git pre-commit hook |
| **`mf audit`** | `[verify] [--json] [--tail N]` | Inspect or cryptographically verify the SHA-256 audit chain |
| **`mf status`** | — | Display vault status, health, project count, and lock state |
| **`mf check`** | — | Validate vault, audit chain, and project registry integrity |
| **`mf rotate-password`** | — | Re-encrypt vault with a new master password and fresh salt |
| **`mf backup`** | `[FILE]` | Export password-protected encrypted vault backup snapshot |
| **`mf restore`** | `<FILE>` | Restore vault and projects from an encrypted backup snapshot |
| **`mf migrate`** | `<OLD_DIR> <NEW_DIR>` | Re-bind project hardware identity when directory moves |
| **`mf update`** | `[--check] [--yes]` | Check GitHub for releases, verify SHA-256, and update in-place |
| **`mf completion`** | `<bash\|zsh\|fish>` | Generate fast shell completion scripts |
| **`mf uninstall`** | — | Cleanly remove MayFly binaries and optionally delete data |
| **`mf version`** | — | Display MayFly version and build metadata |

---

## 11. Cross-Platform & Operational Matrix

### 11.1 Platform Support

| Operating System | Architecture | Terminal Line Discipline | Clipboard Integration | File Identity |
| :--- | :--- | :--- | :--- | :--- |
| **Linux** | `amd64`, `arm64`, `riscv64` | `syscall.SYS_IOCTL` (`termios`) | ANSI OSC 52 / `xclip` / `wl-copy` | `syscall.Stat_t` (`Dev`, `Ino`) |
| **macOS (Darwin)** | `Apple Silicon (arm64)`, `Intel (amd64)` | `syscall.TCGETS` / `TCSETS` | ANSI OSC 52 / `pbcopy` | `syscall.Stat_t` (`Dev`, `Ino`) |
| **Windows** | `x86_64 (amd64)`, `ARM64` | `SetConsoleMode` Console API | ANSI OSC 52 / `clip.exe` | `GetFileInformationByHandle` |

### 11.2 Build & Distribution Integrity
- **100% Deterministic Reproducible Builds**: Compiled with `-trimpath -ldflags="-s -w -buildid="` ensuring identical binary hashes from source anywhere.
- **Air-Gapped & Offline Ready**: 100% functional without internet connectivity. No telemetry, no background analytics, and no remote API calls.
- **One-Command Installers**: Single curl/PowerShell scripts with published SHA-256 checksum verification (`install.sh` and `install.ps1`).

---

<p align="center">
  <strong>MayFly</strong> — <em>Zero Disk Footprint. Zero Dependencies. Uncompromising Security.</em>
</p>
