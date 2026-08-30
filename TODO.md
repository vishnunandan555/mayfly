# 🦋 MayFly — Zero Dependency Hackathon 2026: Master TODO

This document tracks all remaining tasks required to polish, verify, maximize scoring, and submit **MayFly** for **Track E (Security & Crypto Utilities)** in the **Zero Dependency Hackathon 2026**.

**Deadline:** August 31, 2026 · 18:00 UTC  
**Submission Portal:** [https://tally.so/r/2EY7zD](https://tally.so/r/2EY7zD)

---

## 🚨 Phase 1: Critical Submission Blockers (Must Do First)

- [x] **1.1 Merge `application-layer` into `main`**
  - **Status:** Completed. All subsystems, tests, documentation, and tooling from `origin/application-layer` are merged cleanly into `main`.
- [ ] **1.2 Verify `go.mod` is 100% Dependency-Free**
  - Ensure `go.mod` contains no `require` block, no `golang.org/x/` packages, and no vendored code.
  - Run `./zero-dep-audit.sh` and ensure it outputs `ZERO-DEPENDENCY AUDIT: 100% COMPLIANT`.
- [ ] **1.3 Record the 5-Minute Demo Video**
  - **Required Deliverable:** A ≤5 minute screen capture with voiceover/captions demonstrating:
    1. `go.mod` contents and zero-dependency audit proof (`./zero-dep-audit.sh`).
    2. Project initialization (`mayfly init`).
    3. Adding & retrieving encrypted secrets (`mayfly set API_KEY`, `mayfly list`).
    4. Running an app with in-memory injected secrets (`mayfly run python/node/bash-script`) showing process environment has the secret while `.env` file does NOT exist on disk.
    5. Running the plaintext heuristic leak scanner (`mayfly scan`).
    6. Verifying the tamper-evident audit trail (`mayfly audit`, `mayfly audit verify`).
    7. Interactive TUI navigation (`mayfly tui`).
  - Upload video (YouTube unlisted, Loom, or MP4 in release/submission).
- [ ] **1.4 Complete Final Hackathon Submission on Tally**
  - Fill out [https://tally.so/r/2EY7zD](https://tally.so/r/2EY7zD) with:
    - Repository URL (public GitHub).
    - Track: `Track E — Security & Crypto Utilities`.
    - Demo video link.
    - Summary pitch from `.zero-dep.toml`.

---

## 🏆 Phase 2: Bonus Points & Scoring Maximization (+11 to +16 Pts)

### [ ] **2.1 STDLIB Log (+3 Points — Medium)**
- **Requirement:** Submit a `STDLIB.md` with at least **10 real, non-trivial stdlib-for-package substitutions**, each with a clear one-line rationale.
- **Action:** Current `STDLIB.md` has 9 entries. Add entries 10 & 11:
  - `unicode/utf8` + `unicode` replacing `go-runewidth` (calculating terminal column width for multibyte / wide East Asian / emoji runes).
  - Raw ANSI SGR sequence formatting (`screen/terminal.go`) replacing `fatih/color` / `chalk`.

### [ ] **2.2 Reproducible Build (+5 Points — Hard)**
- **Requirement:** Build the binary twice independently and produce byte-identical SHA-256 hashes. Publish the hashes in `deps-proof.txt` / `README.md`.
- **Action:** Add reproducible build target to `Makefile`:
  ```bash
  CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w" -o bin/mayfly ./cmd/mayfly
  ```
  Run build twice, compute `sha256sum bin/mayfly`, and document matching hashes.

### [ ] **2.3 Package Killer (+3 Points — Medium)**
- **Requirement:** Cleanly reimplement a widely installed package with millions of downloads.
- **Action:** Highlight in `README.md` and `STDLIB.md` how MayFly completely replaces:
  - `dotenv` / `godotenv` (10M+ weekly downloads in Node/Go ecosystems) by eliminating the need to ever parse or store `.env` files.
  - `golang.org/x/crypto/pbkdf2` with hand-rolled RFC 8018 PBKDF2-HMAC-SHA256.

### [ ] **2.4 Side Quest: The Write-Up ($300 Prize / 3 Winners)**
- **Requirement:** Publish a public post (e.g. dev.to, Medium, X/Twitter thread, or Substack) explaining:
  - What was reimplemented (raw termios TUI engine, PBKDF2, hash-chained log).
  - What the standard library made painful.
  - The attack vector MayFly eliminates (install-time script scanning disk for `.env`).
  - Tag **Hackathon Raptors**.

---

## ⚙️ Phase 3: Engineering Polish & CI

- [ ] **3.1 Add GitHub Actions CI Workflow (`.github/workflows/ci.yml`)**
  - Automatically test and audit every push/PR on GitHub:
    - `go build ./...`
    - `go vet ./...`
    - `go test -race -v ./...`
    - `./zero-dep-audit.sh`
- [ ] **3.2 Master Password / Environment Ergonomics**
  - Add optional support for `MAYFLY_VAULT_PASSWORD` or standard input pipe for non-interactive / CI automation scripting without breaking secure TTY interactive prompts.
- [ ] **3.3 Clarify Platform Support in Documentation**
  - Document that raw terminal mode and inode-based project identity use Linux `termios` & `stat_t` primitives, with clean fallback stubs for other OSes.
- [ ] **3.4 Inspect `project_test.go` Inode Reuse Edge Case**
  - Review `TestMovedProjectKeepsIdentityAndRecreatedProjectDoesNot` to ensure robust behavior across different container / tempfs file systems.

---

## 🎬 5-Minute Demo Video Checklist

| Minute | Topic | Action to Show |
|---|---|---|
| **0:00 - 0:45** | The Threat & Zero-Dep Proof | Show `go.mod` (0 deps), run `./zero-dep-audit.sh`, explain why secrets managers must have zero supply-chain risk. |
| **0:45 - 1:45** | Quick Start & Vault Storage | Run `mayfly init`, `mayfly set STRIPE_KEY`, show that secrets are saved to encrypted `~/.mayfly/vault.enc` (AES-256-GCM) with 0 files in project directory. |
| **1:45 - 2:45** | In-Memory Process Injection | Run `mayfly run ./demo-app`. Show that `demo-app` sees `STRIPE_KEY` in memory, but `ls -a` proves `.env` does not exist on disk. |
| **2:45 - 3:30** | Plaintext Scanner & Audit Log | Run `mayfly scan` (detects accidental uncommitted keys) and `mayfly audit verify` (proves cryptographic SHA-256 hash chain is intact). |
| **3:30 - 4:45** | Full Interactive TUI | Launch `mayfly tui`. Navigate Secret List, Unlock screen, Edit Secret modal with masked inputs, Scan results screen, and Audit summary. |
| **4:45 - 5:00** | Conclusion & Track Pitch | Summarize Track E fit, standard library craft, and submission details. |

---

## 📅 Submission Summary Checklist

- [ ] Repository is public on GitHub
- [ ] `main` branch contains all code, tests, and documentation
- [ ] `deps-proof.txt` updated with reproducible build hashes
- [ ] `.zero-dep.toml` present in root
- [ ] `README.md` and `STDLIB.md` up to date
- [ ] Demo video uploaded and link tested
- [ ] Tally submission form submitted before **Aug 31, 18:00 UTC**
