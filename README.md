# 🦋 MayFly

> **Zero-Dependency Ephemeral Secrets Workspace & In-Memory Process Injector**  
> *Built 100% from first principles using the Go standard library. 0 external dependencies.*

[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8?style=flat&logo=go)](go.mod)
[![Zero Dependencies](https://img.shields.io/badge/dependencies-0%20external-brightgreen)](STDLIB.md)
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

# 2. Add an encrypted secret
mf set STRIPE_KEY sk_live_123456

# 3. Read a secret (clean stdout for piping)
API_KEY=$(mf get STRIPE_KEY)

# 4. List secret keys
mf list

# 5. Run your app with secrets injected into RAM (no .env on disk)
mf run npm start
mf run python app.py
mf run go run main.go

# 6. Scan codebase for accidental plaintext leaks
mf scan

# 7. Verify cryptographic audit trail
mf audit verify

# 8. Export/Import encrypted backups
mf backup my-backup.json
mf restore my-backup.json

# 9. Migrate a project if its directory moves
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
│   ├── scanner/                # Plaintext credential leak crawler
│   └── domain/                 # Core domain models & validation
├── install.sh                  # All-in-one installer (install, update, uninstall)
├── install.ps1                 # Windows PowerShell installer
└── Makefile                    # One-command build & testing
```

See [STDLIB.md](STDLIB.md) for the complete 11-entry substitution matrix.

---

## 🔒 Security Model & Limits

| Security Property | Implementation Details |
|---|---|
| **Encryption at Rest** | `AES-256-GCM` with random 12-byte nonces and 15-byte authenticated binary header AAD. |
| **Key Derivation** | `RFC 8018 PBKDF2-HMAC-SHA256` running **600,000 iterations** with random 16-byte cryptographic salt. |
| **Memory Isolation** | Decrypted values reside only in RAM during process execution. Memory buffers are zeroed on exit. |
| **Filesystem Isolation** | Binds project identity to physical storage `(Device, Inode)` to prevent path collision leaks. |
| **Audit Integrity** | SHA-256 hash-chained log (`~/.mayfly/audit.log`) mathematically proves no log entries were altered or deleted. |

---

## 🧪 Testing & Verification

Run the full automated test suite:
```bash
make test
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
