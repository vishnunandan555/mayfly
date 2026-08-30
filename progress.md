# MayFly — Build Progress & Phase Track

**Project:** MayFly — a zero-dependency, Go-only local secrets manager that keeps secrets out of `.env` files by storing them in an encrypted vault and injecting them straight into a launched process's environment.  
**Source:** https://github.com/vishnunandan555/mayfly  
**Branch:** `main` (Fully unified and up to date)

---

## Status Legend

- ✅ Done — implemented, tested, and verified
- 🟡 In Progress / Polish

---

## Completed Phases

### Phase 1 — Cryptographic Key Derivation (RFC 8018)
**Status: ✅ Done**
- Hand-written standard RFC 8018 **PBKDF2-HMAC-SHA256** implementation in [`pkg/vault/kdf.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/vault/kdf.go) (600,000 iterations default) using only `crypto/hmac` and `crypto/sha256`.

### Phase 2 — Authenticated Binary Vault Storage
**Status: ✅ Done**
- Versioned binary container with a 15-byte authenticated header (`MFVAUL` magic identifier, version, KDF ID, iteration count, salt/nonce lengths) used as AEAD associated data.
- **AES-256-GCM** encryption with atomic writes (`temp file → fsync → rename`) and strict `0600`/`0700` permissions in [`pkg/vault/vault.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/vault/vault.go).
- Encrypted snapshot export & restore (`mayfly backup` / `mayfly restore`).

### Phase 3 — Multi-Platform Project Identity & Registry
**Status: ✅ Done**
- Multi-platform filesystem identity binding:
  - Linux & macOS: Physical `(Device, Inode)` hashing via `syscall.Stat_t` + canonical symlink resolution in [`pkg/project/identity_linux.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/project/identity_linux.go) and [`identity_darwin.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/project/identity_darwin.go).
  - Windows: Volume Serial Number & File Index hashing in [`pkg/project/identity_windows.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/project/identity_windows.go).
- Central project registry with directory migration support in [`pkg/project/registry.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/project/registry.go).

### Phase 4 — Secure In-Memory Process Execution
**Status: ✅ Done**
- Direct child process execution via `os/exec.CommandContext` (no shell intermediate) with volatile memory environment overlay in [`pkg/executor/process.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/executor/process.go).
- Immediate memory buffer zeroization upon process exit.
- Multi-platform exit code propagation in [`pkg/executor/exit_unix.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/executor/exit_unix.go) and [`exit_windows.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/executor/exit_windows.go).

### Phase 5 — Tamper-Evident Cryptographic Audit Log
**Status: ✅ Done**
- Cryptographic SHA-256 hash-chained JSON log (`~/.mayfly/audit.log`) in [`pkg/audit/log.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/audit/log.go).
- Mathematical chain verification command (`mayfly audit verify`) to detect any modified, deleted, or reordered records.

### Phase 6 — Heuristic Plaintext Credential Scanner
**Status: ✅ Done**
- Bounded recursive filesystem crawler (skipping `.git`, build dirs, binary files) with regex heuristics detecting `.env` files and API key assignments without leaking secret values to logs in [`pkg/scanner/scanner.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/scanner/scanner.go).

### Phase 7 — Standalone Cross-Platform TUI Library
**Status: ✅ Done**
- Low-level raw mode controllers for Linux (`raw_linux.go`), macOS (`raw_darwin.go`), and Windows Console API (`raw_windows.go`).
- Streaming ANSI/CSI/SS3 key event parser finite-state machine in [`pkg/tui/terminal/parser.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/tui/terminal/parser.go).
- 2D double-buffered cell canvas with Unicode East Asian rune column calculations and 16-color ANSI builder in [`pkg/tui/terminal/terminal.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/tui/terminal/terminal.go).
- Deterministic flexbox layout engine in [`pkg/tui/layout/layout.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/tui/layout/layout.go).
- Widget toolkit (TextInput with masking & cursor, List with scrolling, ConfirmDialog, StatusBar, Label) in [`pkg/tui/widget/`](file:///home/vishnunandan555/Projects/mayfly/pkg/tui/widget/).
- Clipboard integration (OSC 52 + platform clipboard tools) in [`pkg/tui/terminal/clipboard.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/tui/terminal/clipboard.go).

### Phase 8 — Two-Tier TUI Dashboard & Views
**Status: ✅ Done**
- **Global TUI Mode (`mayfly` / `mf`):** 2D interactive Project Directory Cards Grid with arrow navigation (`←`, `→`, `↑`, `↓`) and secret drilldown.
- **Current Project Scoped TUI (`mf c`):** Instant project secrets dashboard.
- Single-keypress **`C`** secret value clipboard copy.
- First-run onboarding modal for master password initialization.

### Phase 9 — Package Reorganization & Code Cleanup
**Status: ✅ Done**
- Clean, decoupled `pkg/` package architecture.
- Removed dead prototype binaries and early milestone scratch tests.
- Replaced custom test children with standard in-process test executors.

### Phase 10 — All-in-One Installer & Complete Uninstaller
**Status: ✅ Done**
- POSIX [`install.sh`](file:///home/vishnunandan555/Projects/mayfly/install.sh) and Windows PowerShell [`install.ps1`](file:///home/vishnunandan555/Projects/mayfly/install.ps1).
- Automatically compiles and installs both `mayfly` and `mf` to user `$PATH`.
- Supports `--update` for in-place upgrades.
- Complete uninstallation with explicit confirmation prompt and automated shell `$PATH` cleanup.

---

## 🧪 Verification Matrix

| Target | Build Status | Test Status |
|---|---|---|
| **Linux (`linux/amd64`, `linux/arm64`)** | `PASS` | `PASS (100%)` |
| **macOS Darwin (`darwin/arm64`, `darwin/amd64`)** | `PASS` | `PASS (100%)` |
| **Windows (`windows/amd64`)** | `PASS` | `PASS (100%)` |
| **Zero-Dependency Check** | `PASS` (0 external packages in `go.mod`) | `PASS` |
