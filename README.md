# 🦋 MayFly

**A local secrets manager that never writes a plaintext `.env` to disk.**

![Go](https://img.shields.io/badge/Go-1.27-00ADD8?style=flat&logo=go&logoColor=white)
![Zero Dependencies](https://img.shields.io/badge/dependencies-0-brightgreen)
![Track](https://img.shields.io/badge/track-E%20·%20Security%20%26%20Crypto-critical)
![Hackathon](https://img.shields.io/badge/Zero%20Dependency%202026-Hackathon%20Raptors-blueviolet)
[![License: AGPL-3.0](https://img.shields.io/badge/license-AGPL--3.0-blue)](LICENSE)

---

## The Problem

Every software project typically has a `.env` or configuration file sitting on disk containing real API keys and database credentials in plaintext.

The moment you run `npm install`, `pip install`, or `cargo build`, you trust thousands of transitive third-party dependencies not to execute malicious install scripts. Real-world supply-chain attacks repeatedly scan disk trees for `.env` files and exfiltrate credentials before a developer even launches their application.

You cannot completely prevent untrusted install scripts from running on your machine. But you **can** ensure there are no plaintext secrets on disk for them to steal.

---

## What MayFly Does

MayFly eliminates plaintext `.env` files from your project folders. Secrets are encrypted at rest in an external vault (`~/.mayfly/vault.enc`) outside your code repository. When you launch your application, you run it **through** MayFly:

```bash
mayfly run npm run dev
```

MayFly:
1. Identifies the current project directory using filesystem identity.
2. Prompts for your master password (or uses your session).
3. Decrypts only that project's secrets in memory (RAM).
4. Injects the secrets directly into the target process's environment.
5. Launches the process directly via standard `os/exec` (never invoking a shell).
6. Releases transient memory references immediately upon process exit.
7. Records an authenticated, tamper-evident audit event.

Nothing is ever written to disk in your project folder. Your application code does not need to change: standard environment variable lookups like `process.env.API_KEY` or `os.Getenv("DATABASE_URL")` work seamlessly.

---

## Features

- 🔒 **Encrypted Vault at Rest**: AES-256-GCM authenticated encryption with PBKDF2-HMAC-SHA256 key derivation (600,000 iterations default) and fresh per-save random nonces.
- ⚡ **In-Memory Process Injection**: Secrets go straight into the launched process environment via `os/exec`; no temporary `.env` file is ever created.
- 🖥️ **Full-Featured Custom TUI**: Complete terminal interface built from scratch without UI libraries (Vault Unlock, Secret List with masked values, Editor, Delete Confirmation, Scan Results, and Audit Summary).
- 📁 **Project Isolation**: Deterministic Linux inode-based project identity ensures secrets for Project A are completely isolated from Project B.
- 🧾 **Tamper-Evident Access Trail**: Cryptographic SHA-256 hash-chained audit log with sequence checkpoints detects any historical alteration, truncation, or reordering.
- 🛡️ **Heuristic Leak Scanner**: Scans project files for accidental plaintext `.env` files, credentials, and private keys before committing code.
- 🚫 **Zero Third-Party Dependencies**: Built exclusively with the **Go 1.27 standard library** (`go.mod` has no `require` block).

---

## Quick Start

### 1. Build MayFly

MayFly requires Go 1.27+. Compile the binary using `go` or `make`:

```bash
# Using standard Go tooling
go build -o mayfly ./cmd/mayfly

# Or using Make
make build
```

Move `mayfly` to your `$PATH` (e.g. `/usr/local/bin/mayfly`).

### 2. Initialize a Project

Navigate to any project directory and initialize MayFly:

```bash
cd ~/code/my-project
mayfly init
```

This registers the project in MayFly's external registry (`~/.mayfly/projects.json`).

### 3. Add Secrets

Save secrets into your encrypted vault:

```bash
mayfly set OPENAI_API_KEY
# Prompts for your vault master password and secret value
```

List secret names stored for the current project:

```bash
mayfly list
```

Explicitly retrieve a secret value to stdout when needed:

```bash
mayfly get OPENAI_API_KEY
```

Delete a secret:

```bash
mayfly delete OPENAI_API_KEY
```

### 4. Run Applications with Injected Secrets

Execute your command with secrets injected directly into its runtime environment:

```bash
mayfly run npm run dev
mayfly run python main.py
mayfly run ./my-binary --config=prod
```

### 5. Launch the Interactive TUI

Launch MayFly's full-screen terminal interface:

```bash
mayfly tui
```

Or run the interactive demonstration harness against sample data:

```bash
go run ./cmd/tui-demo
```

### 6. Scan for Plaintext Leaks

Scan your project tree for accidentally committed plaintext `.env` files or API key assignments:

```bash
mayfly scan
```

Returns exit code `0` if clean, `3` if potential leaks are detected, and `1` on operational errors.

### 7. View & Verify the Audit Trail

Inspect the metadata access log:

```bash
mayfly audit
```

Verify that the cryptographic hash chain has not been tampered with:

```bash
mayfly audit verify
```

---

## Interactive TUI Screens

MayFly features 6 distinct presentation screens:

1. **Vault Unlock**: Centered master password prompt with bullet masking (`•`) and instant memory clearing upon submission.
2. **Secret List**: Displays the current project path in the header (`MayFly   ~/code/my-project`), formatted secret names with masked values (`OPENAI_API_KEY  ••••••••••`), safe status bar, and shortcut hints.
3. **Create / Edit Secret**: Modal form for secret name and masked value. Plaintext is loaded on demand only into the input field and cleared immediately upon save or cancellation.
4. **Delete Confirmation**: Modal dialog confirming secret removal.
5. **Scan Results**: Displays heuristic findings (relative path, line:col, severity `[CRITICAL]`/`[WARNING]`, category, and safe description) with `R` rescan and `Esc` return.
6. **Audit Summary**: Displays verified tamper-evident audit events (timestamps, actions, project IDs, secret names, commands, exit codes).

**TUI Navigation Shortcuts**:
- `↑ / ↓` or `j / k`: Navigate list items
- `Enter`: Edit selected secret / Submit form
- `N` / `n`: Create new secret
- `D` / `d`: Delete selected secret
- `S` / `s`: Run scanner and view results
- `A` / `a`: View audit log summary
- `Tab` / `Shift-Tab`: Toggle input field focus
- `Esc` / `Q` / `q`: Return to previous screen or quit

---

## Security Model & Honest Threat Boundaries

MayFly provides strong, pragmatic protection against modern development attack vectors, but security boundaries must be accurately stated:

### What MayFly Protects Against
- **Install-time supply chain exfiltration**: Malicious post-install scripts (`package.json`, `setup.py`, `build.rs`) scanning disk directories for `.env`, `.env.local`, or credentials files will find nothing.
- **Accidental git commits**: Plaintext `.env` files never exist on disk, preventing accidental staging or pushing to remote repositories.
- **Unauthorized local inspection at rest**: Vault files are encrypted with AES-256-GCM using authenticated headers and 600,000 PBKDF2 iterations; vault files are permissioned `0600` and directories `0700`.
- **Silent audit tampering**: The SHA-256 hash-chained log detects altered, reordered, or truncated event entries.

### What MayFly Does NOT Protect Against (Threat Limits)
- **Malicious code inside the running application**: A dependency executed *during* `mayfly run` has access to its own process environment and can read injected variables via `os.Environ()` or `process.env`. MayFly controls storage and injection, not the runtime behavior of the target process.
- **Child process inheritance**: Programs launched by the target application inherit the parent process environment according to OS conventions.
- **Full machine compromise / root access**: An attacker with root access or ptrace permissions can inspect process memory (`/proc/<pid>/mem`).
- **Cryptographic immutability**: The audit log is tamper-evident, not an immutable blockchain. An attacker with full write access to the filesystem who rewrites the entire log, recalculated hashes, and updated the checkpoint can forge history.
- **Go GC memory zeroization**: MayFly zeroes byte slices and releases transient references as soon as operations complete. However, Go's garbage-collected runtime does not guarantee immediate physical memory erasure of prior string allocations.
- **Heuristic scanner limits**: `mayfly scan` uses conservative regular expressions and filename checks; it is a heuristic detector of likely exposures, not mathematical proof that a codebase is free of secrets.
- **Terminal compatibility**: Raw mode requires ANSI/VT-compatible Linux TTYs (using termios ioctl). Unsupported platforms fallback to safe error handling.

---

## Zero-Dependency Architecture

MayFly achieves 100% of its capabilities using only Go 1.27 standard library packages:

```text
CLI / TUI Presentation Layer
  ↓ (application.ScreenService)
Application Orchestration Layer (application/)
  ↓
Domain Types & Validation (domain/)
  ↓
Core Subsystems (Standard Library Only):
  • Vault Storage & PBKDF2 KDF (vault/)        → crypto/aes, crypto/cipher, crypto/hmac, crypto/sha256
  • Project Identity & Isolation (project/)    → syscall, os, path/filepath, crypto/sha256
  • Process Executor (executor/)               → os/exec, context
  • Tamper-Evident Audit Log (audit/)          → crypto/sha256, encoding/json
  • Heuristic Leak Scanner (scanner/)          → path/filepath, regexp, unicode/utf8
  • Terminal UI Engine (screen/)               → syscall, os, io, unicode
```

See [STDLIB.md](STDLIB.md) for the detailed substitution breakdown and [deps-proof.txt](deps-proof.txt) for reproducible zero-dependency audit commands.

---

## Verification & Testing

Execute the complete verification suite:

```bash
# Run all unit, integration, and E2E tests
go test -v ./...

# Run the race detector
go test -race ./...

# Run static analysis
go vet ./...

# Verify zero dependencies
./zero-dep-audit.sh
```

---

## Built For

**Zero Dependency Hackathon 2026** — Organized by Hackathon Raptors.  
**Track:** Track E (Security & Crypto Utilities).

## License

[GNU Affero General Public License Version 3](LICENSE)
