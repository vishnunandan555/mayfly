# Changelog

All notable changes to MayFly are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [0.0.5] — 2026-08-31

### Added
- **Zero-History Ephemeral TUI Secret Viewer (`mf get`)**: When running in an interactive terminal, `mf get <KEY>` now launches an isolated alt-screen ephemeral viewer. Secrets are never printed to terminal scrollback or bash history; pressing `C` copies the secret to the clipboard and `Q`/`Esc` immediately zeroes memory and restores the terminal canvas. Plain stdout output is preserved when piped or invoked in non-interactive / CI scripts.
- **Comprehensive Features Matrix (`FEATURES.md`)**: Complete architectural specification document covering memory lifecycle diagrams, security guarantees, KDF iteration benchmarks, hardware inode identity mapping, and complete CLI command references.
- **Unified Build System & Versioning**: Integrated `-ldflags="-s -w -buildid= -X mayfly/pkg/domain.Version=..."` consistently across `Makefile`, `install.sh`, `install.ps1`, and GitHub Actions release pipelines.
- **Multi-Platform Release Artifacts**: Added `make release-artifacts` target with automated cryptographic `checksums.txt` generation and verification.
- **Comprehensive Documentation Hub**: Complete interactive Fumadocs / Next.js documentation portal with OpenGraph cards, LLM reference guides (`llms.txt`), and universal framework integration walkthroughs.

### Security
- Cryptographic SHA-256 binary verification enforced across both `install.sh` and `install.ps1`.
- Verified bit-for-bit reproducible build determinism (`make reproducible`).
- Confirmed 0 external runtime dependencies (`make deps-proof`).

---

## [0.0.4] — 2026-08-31

### Fixed
- **Windows UTF-8 BOM Handling**: Strip UTF-8 Byte Order Marks (`\xef\xbb\xbf`) emitted by Windows editors/PowerShell when reading audit logs and project registry metadata.
- **Audit Log Sequence Synchronization**: Scan latest state before write to eliminate sequence mismatch errors in audit trail verification.
- **Windows Binary Cleanup**: Support removing `.exe` binaries (`mayfly.exe`, `mf.exe`) in uninstaller.
- **Project Hygiene**: Added submission and test artifacts to `.gitignore`.

---

## [0.0.3] — 2026-08-31

### Added

- `mf env [--shell bash|fish|json]` — export all project secrets as shell environment variables (`eval $(mf env)`)
- `mf status` / `mf doctor` — vault health summary: file, lock state, project count, secret count
- `mf check` — integrity verification of vault header, audit log hash chain, and stale registry entries
- `mf template <FILE> [--output F]` — render config templates with `{{ SECRET_NAME }}` placeholders replaced by decrypted values
- `mf diff <OTHER_PATH>` — compare secret key sets between two projects (keys only, no values exposed)
- `mf install-hook` / `mf uninstall-hook` — install/remove a git pre-commit hook that runs `mf scan` before every commit
- `--password-stdin` global flag — read vault password from stdin pipe (safer than `MAYFLY_VAULT_PASSWORD` env var for CI)
- `mf set --clip` flag — copy value to clipboard after saving (symmetric with `mf get --clip`)
- `mf audit --json` — JSON output for audit trail
- `mf audit --tail N` — show last N audit events
- `mf audit` aligned column output for human readability
- `mf scan --json` — JSON output for scan findings
- `mf scan --severity CRITICAL|WARNING` — filter scan output by severity
- `mf scan` now exits `1` on CRITICAL findings and `2` on WARNING-only (previously always `0` — broke CI usage)
- Soft brute-force lockout: after 5 consecutive wrong passwords, vault locks for 30 seconds (tracked in `~/.mayfly/meta.json`)
- 15 new scanner patterns: Slack bot/user tokens, OpenAI, Anthropic, Twilio, SendGrid, Mailgun, JWT, Bearer tokens, npm auth tokens, DB connection strings, Dockerfile ENV secrets, pip.conf credentials
- Extended dangerous filename detection: `.pem`, `.key`, `.p12`, `.pfx`, `id_ecdsa`, `id_dsa`, `pip.conf`, `.npmrc`
- In-memory registry cache: `projects.json` now read from disk once per process lifetime, invalidated on write
- `docs/package.json` — added `clean` and `dev:clean` scripts to clear corrupt `.next/` cache

### Fixed
- Failing updater test: mock release tag bumped to `v0.0.3` so it is genuinely newer than current `v0.0.2`
- `mf update` replaced `curl | bash` (ironic security hole) with in-process download → SHA-256 verify → atomic binary replace
- Audit log `Events()` now warns to stderr on corrupt JSON lines instead of silently skipping them
- Removed dead `Secret.UpdatedAt` field that was never populated (always zero value)

### Removed
- `HACKATHON_WINNING_STRATEGY.md` — internal planning doc (moved to private archive)
- `WRITEUP.md` — hackathon writeup (converted to blog post)
- `test_playground/` — unrelated Gemini chatbot demo (dead code)
- `guides/` — stale hackathon judging context

---

## [0.0.2] — 2026-08-31

### Added
- `mf update [--check] [--yes]` — check for and apply newer releases
- Shell autocompletions: `mf completion bash|zsh|fish`
- `mf backup` / `mf restore` — encrypted vault snapshot export/import
- `mf migrate <OLD> <NEW>` — re-point project when directory moves
- `mf rotate-password` — re-encrypt vault with new master password + fresh salt
- `mf import [FILE] [--delete]` — bulk-import from `.env` file into vault
- Auto-lock: vault clears from memory after 15 minutes of inactivity
- Global TUI dashboard with project grid (arrow key navigation)
- `mf c` / `mf current` — TUI scoped to current project directory
- Cross-platform: Linux, macOS (Intel + Apple Silicon), Windows

### Security
- AES-256-GCM encryption with random 12-byte nonces
- PBKDF2-HMAC-SHA256 KDF at 600,000 iterations with random 16-byte salt
- SHA-256 hash-chained tamper-evident audit log
- Inode + device-based project identity (prevents path collision)
- Alt-screen ephemeral input for `mf set` (secrets not in shell history)
- Memory zeroing with `runtime.KeepAlive` after vault operations
- Atomic file writes (temp + rename) for vault and registry
- `0600` file permissions auto-enforced on vault file

---

## [0.0.1] — Initial Release

- Zero-dependency design (stdlib only)
- `mf set` / `mf get` / `mf list` / `mf delete` core CRUD
- `mf run <COMMAND>` — in-memory process injection
- `mf scan` — plaintext credential leak scanner with `.mayflyignore` support
- `mf init` — register project directory
- Reproducible builds (`-trimpath -ldflags="-s -w -buildid="`)
- Cross-compilation verified: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
