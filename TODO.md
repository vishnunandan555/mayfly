# 🦋 MayFly — Project Roadmap & Completion Status

This document tracks all tasks, architectural milestones, and verification checks for **MayFly** (Zero-Dependency Ephemeral Secrets Workspace & In-Memory Process Injector).

---

## ✅ Completed Milestones

- [x] **Core Zero-Dependency Architecture**
  - Standard-library-only implementation with 0 external packages (`go.mod` has 0 `require` directives).
  - Multi-platform support across **Linux**, **macOS Darwin**, and **Windows**.
- [x] **RFC 8018 PBKDF2 Key Derivation Engine**
  - Hand-rolled PBKDF2-HMAC-SHA256 (600,000 iterations) with memory zeroization in [`pkg/vault/kdf.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/vault/kdf.go).
- [x] **Authenticated Binary Vault Storage**
  - 15-byte authenticated binary header with AES-256-GCM AEAD encryption in [`pkg/vault/vault.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/vault/vault.go).
- [x] **In-Memory Process Injection & RAM Zeroization**
  - Direct execution via `os/exec.CommandContext` with volatile RAM environment overlay in [`pkg/executor/process.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/executor/process.go).
- [x] **Filesystem Inode Project Identity**
  - Physical `(Dev, Ino)` resolution for Linux & macOS and Volume Serial Number for Windows in [`pkg/project/`](file:///home/vishnunandan555/Projects/mayfly/pkg/project/).
- [x] **Tamper-Evident Cryptographic Audit Trail**
  - Immutable SHA-256 hash-chained access log with verification in [`pkg/audit/log.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/audit/log.go).
- [x] **High-Performance Plaintext Leak Scanner**
  - Recursive crawler with regex heuristics and fast directory pruning in [`pkg/scanner/scanner.go`](file:///home/vishnunandan555/Projects/mayfly/pkg/scanner/scanner.go).
- [x] **Two-Tier TUI Engine & Custom Widget Framework**
  - **Global Mode (`mayfly` / `mf`):** 2D interactive Project Directory Cards Grid with arrow navigation and dynamic vertical scrolling.
  - **Project Scoped Mode (`mf c`):** Instant project secrets dashboard.
  - **Clipboard Integration:** Single-keypress `C` raw secret copy via OSC 52 and platform utilities.
  - **First-Run Onboarding:** Interactive master password creation modal.
- [x] **All-in-One Global Installer (`install.sh` & `install.ps1`)**
  - Interactive alias selection (`mayfly`, `mf`, or both), shell `$PATH` prompt, `--update`, and single `[y/N]` uninstallation.
- [x] **Key Management Suite**
  - `mf init`, `mf set`, `mf get`, `mf list`, `mf delete`, `mf run`, `mf scan`, `mf audit`, `mf backup`, `mf restore`, `mf migrate`, `mf uninstall`.
- [x] **Comprehensive Standard Library Matrix (`STDLIB.md`)**
  - 11 real, non-trivial stdlib-for-package substitutions documented with architectural rationale (+3 Bonus Points).

---

## 🎯 Next Steps / Remaining Actions

- [ ] **Record 5-Minute Demo Video**
  - Screen capture with audio/captions showing:
    1. Zero dependencies in `go.mod`.
    2. One-line installation via `./install.sh` (with alias and PATH selection).
    3. Global TUI Project Grid navigation (`mayfly`).
    4. Project initialization and secret setting (`mf set API_KEY`).
    5. In-memory execution showing process sees secret while `.env` file does NOT exist on disk (`mf run node server.js`).
    6. Clipboard copy (`C` in TUI).
    7. Plaintext leak scanner (`mf scan`) and audit verification (`mf audit verify`).
- [ ] **Submit Final Hackathon Entry**
  - Form: [https://tally.so/r/2EY7zD](https://tally.so/r/2EY7zD)
  - Track: `Track E — Security & Crypto Utilities`
