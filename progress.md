# MayFly — Build Progress & Phase Track

**Project:** MayFly — a zero-dependency, Go-only local secrets manager that keeps
secrets out of `.env` files by storing them in an encrypted vault and injecting
them straight into a launched process's environment.
**Source:** https://github.com/vishnunandan555/mayfly

> **Branch note:** GitHub's default `main` branch currently only contains the
> early TUI-engine work (the `screen/` package, `screens.go`, two demo
> binaries). The full application — vault, project identity, executor, audit
> log, scanner, and the real `mayfly` CLI — exists on the **`application-layer`**
> branch, which is 21 commits ahead of `main` and not yet merged. That branch
> is what actually matches the README's feature list, so this progress track
> (and the code in this zip) is based on `application-layer`. Merging it into
> `main` is called out explicitly below as outstanding work.

---

## Status legend

- ✅ Done — implemented, tested, present in this snapshot
- 🟡 Partial — implemented but with known gaps
- ⬜ Not started

---

## Phase 0 — Project Scaffolding
**Status: ✅ Done** (`dcaa999`, `32648cc`)
- Repo initialized, AGPL-3.0 `LICENSE` added.
- README pitch, problem statement, and hackathon track (Track E — Security &
  Crypto Utilities) written.

## Phase 1 — Terminal Output Abstraction
**Status: ✅ Done** (`1ce72c8`)
- `screen/terminal.go`, `screen/frame.go`: base terminal writer + frame buffer.
- First demo entrypoint: `cmd/phase1-demo/main.go`.

## Phase 2 — Raw Mode & Input Parsing
**Status: ✅ Done** (`acadce2`)
- `screen/raw.go` (+ `raw_linux.go`, `raw_unsupported.go`): termios raw mode via
  `syscall`, restored cleanly on exit/signal.
- `screen/parser.go` + `screen/input.go`: byte-level ANSI/CSI/SS3 key parser.

## Phase 3 — Text / Cell Rendering Model
**Status: ✅ Done** (`6cad403`)
- `screen/frame.go` extended into a full 2D cell grid (Unicode width aware,
  dirty-row diffing).

## Phase 4 — Styles, Color, Masking
**Status: ✅ Done** (`a6da8ab`)
- `screen/masking.go`: masked-value rendering (`••••••••••`) for secret values.
- Hand-rolled ANSI SGR color/style builder in `screen/terminal.go`.

## Phase 5 — Layout Engine
**Status: ✅ Done** (`1bfbe63`)
- `screen/layout.go`: deterministic box/flex-style layout with no external
  layout library.

## Phase 6 — Widget System
**Status: ✅ Done** (`574058b`)
- `screen/widget.go`, `label.go`, `panel.go`, `list.go`, `text_input.go`,
  `confirm_dialog.go`, `status_bar.go`.

## Phase 7 — Event Dispatch, Focus, App Loop
**Status: ✅ Done** (`30985dc`)
- `screen/application.go`: the runtime loop tying input → focus → widgets →
  render together. `screen/size*.go` for terminal size detection (SIGWINCH).

## Phase 8 — MayFly Screens (TUI Layer)
**Status: ✅ Done** (`05339b1`, `cf89604`)
- `screens.go`: the 6 MayFly-specific screens (Unlock, Secret List, Editor,
  Delete Confirm, Scan Results, Audit Summary) composed from the widget kit.
- At this point the TUI worked against fake/in-memory data only — no real
  crypto or process execution yet.

## Phase 9 — App Architecture & Domain Model
**Status: ✅ Done** (`07fbaf5`)
- `domain/domain.go`: core types (Secret, Project, AuditEvent, ScanFinding,
  errors) shared by every subsystem.
- `application/application.go`, `application/presentation.go`: the
  orchestration layer between CLI/TUI and the subsystems below.

## Phase 10 — Encrypted Vault Backend
**Status: ✅ Done** (`7898dea`)
- `vault/vault.go`: versioned binary container, atomic write (`temp file →
  fsync → rename`), file permissions `0600`/`0700`.
- `vault/format.go`: header layout (magic `MFVAUL`, version, KDF id, iteration
  count, salt/nonce lengths) used as AEAD associated data.
- `vault/kdf.go`: hand-written PBKDF2-HMAC-SHA256 (RFC 8018), 600,000
  iterations default — `crypto/hmac` + `crypto/sha256` only.
- Encryption: AES-256-GCM via `crypto/aes` + `crypto/cipher`, fresh nonce per
  save.

## Phase 11 — Project Discovery & Identity
**Status: 🟡 Partial — Linux only** (`a8b7f7c`)
- `project/identity.go` + `identity_linux.go`: deterministic `(device,
  inode)`-based project identity so Project A's secrets can never leak into
  Project B.
- `project/identity_unsupported.go`: **stub that returns an empty identity on
  every non-Linux OS** — this is the biggest concrete gap in the project.
- `project/registry.go`: external JSON registry at `~/.mayfly/projects.json`.

## Phase 12 — Application-Level Secret Management
**Status: ✅ Done** (`47d5b3a`)
- `mayfly init / set / get / list / delete` fully wired through
  `application/application.go` into the vault.
- `cmd/mayfly/main.go`: real CLI entrypoint (this phase replaces the demo
  binaries as the "real" program).

## Phase 13 — Secure Child Process Execution
**Status: ✅ Done** (`f4b5c54`)
- `executor/process.go`: `mayfly run <cmd>` — direct `os/exec.CommandContext`
  (never a shell), environment inherited + secrets overlaid, memory cleared
  after exit.
- `executor/exit_unix.go` / `exit_other.go`: correct exit-code propagation.
- `cmd/mayfly-child/main.go` + `internal/childdemo/`: a tiny helper binary used
  to test injection end-to-end.

## Phase 14 — Tamper-Evident Audit Log
**Status: ✅ Done** (`564dac7`)
- `audit/log.go`: SHA-256 hash-chained, newline-delimited JSON log; each
  record links to the previous record's hash. `mayfly audit` /
  `mayfly audit verify`.

## Phase 15 — Heuristic Plaintext Secret Scanner
**Status: ✅ Done** (`d58d9a6`)
- `scanner/scanner.go`: walks the project tree (skips `.git`, size-bounded at
  1 MiB/file), regex + filename heuristics for `.env` files / API-key-shaped
  strings, reports `[CRITICAL]` / `[WARNING]` findings without ever printing
  the matched secret itself. `mayfly scan`.

## Phase 16 — Wire the Real Application Into the TUI
**Status: ✅ Done** (`0b2f7f6`)
- `application/presentation.go` grew into the real adapter between the six
  screens (Phase 8) and the real vault/executor/audit/scanner (Phases 10–15).
  Before this phase the TUI was a shell; after it, `mayfly tui` is fully live.

## Phase 17 — Security Docs & Zero-Dependency Proof
**Status: ✅ Done** (`f245bc2`)
- `STDLIB.md`: full substitution table (what a normal project would `go get`
  vs. what MayFly hand-built instead).
- `zero-dep-audit.sh`, `deps-proof.txt`, `.zero-dep.toml`: reproducible,
  scripted proof that `go.mod` has no `require` block and nothing under
  `github.com/`, `golang.org/x/`, etc. is imported anywhere.
- `Makefile`: `make build / test / test-race / vet / audit / demo / clean`.
- `cmd/mayfly/e2e_test.go`: black-box end-to-end tests against the built
  binary.
- README rewritten into its current, accurate form (Quick Start, Security
  Model & Honest Threat Boundaries, Architecture diagram).

## Phase 18 — Post-Integration Bug Fixes
**Status: ✅ Done** (`854d297`, `ed48bac`, `58bef20`)
- Fixed a lock-lifecycle bug in the multi-screen service adapter.
- Fixed a TUI login/unlock UI issue.
- Removed a `bin/` folder that had been accidentally committed with built
  binaries (now correctly gitignored).

---

## Phase 19 — Outstanding Work (not yet done)

This is the real remaining track — what's actually left, in priority order.

1. **⬜ Merge `application-layer` → `main`.** This is the single most
   important next step: the branch GitHub shows by default is stale and
   badly undersells the project. Open a PR, fast-forward or merge, delete the
   stale README-only state from `main`.
2. **⬜ Cross-platform support.** `raw_unsupported.go`, `size_unsupported.go`,
   and `identity_unsupported.go` are all real no-op/stub implementations —
   MayFly's raw-mode TUI, terminal sizing, and project identity **only work
   on Linux today**. macOS (BSD termios/ioctl numbers differ) and Windows
   (no termios at all — needs the Console API) are explicitly unimplemented.
3. **⬜ CI pipeline.** `Makefile` and `zero-dep-audit.sh` exist locally but
   there's no `.github/workflows/` yet to run `go build`, `go vet`,
   `go test -race`, and the zero-dep audit on every push/PR.
4. **🟡 Fix/investigate the flaky project-identity test.** In this sandbox,
   `TestMovedProjectKeepsIdentityAndRecreatedProjectDoesNot` fails because a
   recreated directory reused the same inode instead of getting a fresh one
   (an overlay-filesystem quirk of the container, not necessarily a real
   bug) — worth a real-disk re-check before relying on it.
5. **⬜ Session/agent for repeated unlocks.** README mentions "or uses your
   session" for the master password but there's no session cache — every
   command currently re-prompts.
6. **⬜ Release packaging.** No tagged release / prebuilt binaries yet; only
   `go build` from source is documented.
7. **⬜ Key rotation / vault re-encryption.** No documented path for rotating
   the master password or bumping PBKDF2 iterations on an existing vault.

---

## Verification performed on this snapshot

Run in a clean container (Go 1.22 toolchain, project declares `go 1.27` in
`go.mod` — bump your local toolchain accordingly; only used a lower version
here to confirm the code itself has no version-specific dependency):

```
go build ./...      → PASS (all packages compile)
go vet ./...         → PASS (no warnings)
go test ./...         → PASS, except:
                         mayfly/project: TestMovedProjectKeepsIdentityAndRecreatedProjectDoesNot FAILS
                         (see Phase 19, item 4 — container filesystem inode reuse)
```

Every other package (`mayfly`, `application`, `audit`, `cmd/mayfly`, `domain`,
`executor`, `scanner`, `screen`, `vault`) passed its test suite cleanly.

---

## How to pick this up

```bash
cd mayfly
go build -o bin/mayfly ./cmd/mayfly
go build -o bin/tui-demo ./cmd/tui-demo
go build -o bin/mayfly-child ./cmd/mayfly-child

cd your-other-project
mayfly init
mayfly set OPENAI_API_KEY
mayfly run npm run dev
mayfly tui
```

Start Phase 19 work on a new branch off `application-layer` (or off `main`
after the merge). Item 1 (merging the branches) should happen first so future
work isn't split across two divergent histories again.
