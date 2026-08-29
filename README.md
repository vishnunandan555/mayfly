# 🦋 MayFly

**A local secrets manager that never writes a plaintext `.env` to disk.**

![Go](https://img.shields.io/badge/Go-1.27-00ADD8?style=flat&logo=go&logoColor=white)
![Zero Dependencies](https://img.shields.io/badge/dependencies-0-brightgreen)
![Track](https://img.shields.io/badge/track-E%20·%20Security%20%26%20Crypto-critical)
![Hackathon](https://img.shields.io/badge/Zero%20Dependency%202026-Hackathon%20Raptors-blueviolet)
[![License: AGPL-3.0](https://img.shields.io/badge/license-AGPL--3.0-blue)](LICENSE)

---

## The Problem

Every project has a `.env` file sitting on disk with real API keys in it, in plain text.

The moment you run `npm install` or `pip install`, you're trusting every package (and every package's dependencies) not to run malicious code. In 2025 alone, real supply-chain attacks (Shai-Hulud, the `chalk`/`debug` compromise, and others) shipped code that specifically scanned disks for `.env` files and stole what it found — before the developer even ran their app.

You can't fully stop a malicious `postinstall` script from running. But you *can* make sure there's nothing on disk worth stealing.

## What MayFly Does

MayFly removes `.env` files from your project entirely. Your secrets live in one encrypted vault on your machine. When you want to run your app, you run it *through* MayFly instead of directly:

```
mayfly run npm run dev
```

MayFly decrypts your secrets in memory only, injects them straight into the environment of that one process, and forgets them the moment the process stops. Nothing ever touches your project folder. Your code doesn't change — `process.env.API_KEY` still works exactly like before.

## How It Works

1. You save your secrets into MayFly's vault once, through a simple terminal menu.
2. Your secrets are encrypted and stored in one file, outside your project (`~/.mayfly/vault.enc`).
3. When you run `mayfly <your command>`, MayFly figures out which project you're in, decrypts only that project's secrets in RAM, and hands them to the process you're launching.
4. When the process exits, the secrets are gone. Nothing was ever written to your project folder.
5. Every time a secret is accessed, it's logged in a way that can't be quietly edited after the fact — so if anything looks off, there's a trail.

No `.env` file. Nothing for a malicious install script to find.

## Features

- 🔒 **Encrypted vault** — secrets are encrypted at rest, decrypted only in memory, only when needed
- ⚡ **Zero code changes** — your app still reads secrets the normal way
- 🖥️ **Simple terminal UI** — add, edit, and manage secrets without hand-editing files
- 📁 **Auto project detection** — MayFly knows which project's secrets to load based on where you run it
- 🧾 **Tamper-evident access log** — every secret access is recorded in a way that shows if the log itself was ever altered
- 🛡️ **Leak guard** — warns you if a plaintext secret or missing `.gitignore` rule is about to ship anyway

## Why Zero Dependency

This isn't a style choice, it's the point of the project. A secrets manager that pulls in third-party packages is asking you to trust exactly the kind of thing it's supposed to protect you from. Every single piece of MayFly — the encryption, the terminal interface, the process launching, the logging — is built using only Go's standard library. Nothing to audit but our own code.

## Tech Stack

- **Language:** Go (standard library only, no third-party packages)
- **Encryption:** `crypto/aes` + `crypto/cipher` (AES-256-GCM)
- **Key derivation:** hand-implemented from `crypto/hmac` + `crypto/sha256` (since the usual helper packages for this aren't part of true Go stdlib)
- **Process launching:** `os/exec`
- **Terminal UI:** built from raw terminal mode + ANSI escape codes, no UI library
- **CLI parsing:** `flag`

## Modules We're Rebuilding From Scratch

These are the pieces most projects would just `go get`. We're writing them ourselves using only Go's standard library.

| Module | What it normally is | What we're building |
|---|---|---|
| **TUI engine** | A library like `bubbletea` or `tview` | Our own terminal renderer — raw terminal mode, cursor control, key-event loop, and layout, all via hand-written ANSI escape codes |
| **Secrets loader** | `dotenv` / `godotenv` | In-memory environment injection — secrets go straight into a spawned process's environment via `os/exec`, no file ever written |
| **Key derivation function** | `golang.org/x/crypto/pbkdf2` or `scrypt` | Our own PBKDF2 implementation (RFC 8018), built from `crypto/hmac` + `crypto/sha256` — this isn't in true Go stdlib, so we're writing it ourselves |
| **Encrypted vault / storage format** | A local database lib or serialization package | Our own file format for the encrypted vault, built on `crypto/aes` (AES-256-GCM) and raw file I/O |
| **Tamper-evident audit log** | An audit-logging package | A hash-chained log we designed ourselves — each entry's hash includes the one before it, via `crypto/sha256` |
| **CLI argument parsing** | A framework like `cobra` | Go's built-in `flag` package |
| **Terminal styling/colour** | A package like `chalk` or `fatih/color` | Raw ANSI colour codes, written by hand |

## Architecture

MayFly is split into small layers with explicit dependencies:

```text
CLI / TUI
   |
   v
application services
   |-- project lookup
   |-- vault storage
   |-- secret operations
   |-- command execution
   |-- audit recording
   `-- optional leak scanner
   |
   v
filesystem / crypto / os/exec / terminal
```

The [`domain`](domain) package contains validated concepts such as projects,
secret metadata, execution requests, audit events, and scan findings. It has no
I/O or terminal behavior. The [`application`](application) package contains
use-case contracts and orchestration; implementations of storage, encryption,
process execution, and auditing can be supplied by the CLI or TUI without the
service layer knowing their internals. Secret values are absent from metadata
types and are loaded only through explicit value-bearing operations.

The [`screen`](screen) package remains the repository-owned TUI engine. The
application screens use presentation-safe boundaries and do not render
plaintext secret values. No layer shells out to `stty`, `tput`, or another
external executable.

The [`vault`](vault) package is the production encrypted storage boundary. It
uses AES-256-GCM with a fresh random salt at initialization, a fresh nonce for
each save, and the repository's PBKDF2-HMAC-SHA256 implementation. The
versioned header is authenticated as GCM associated data, while project and
secret payloads remain encrypted. Updates are written to a same-directory
unpredictable temporary file, synced, and atomically renamed where supported;
vault files are created with mode `0600` and newly created parent directories
with mode `0700`. MayFly minimizes plaintext copies but Go's garbage collector
does not provide a guarantee of cryptographic memory erasure.

Project discovery is handled by the [`project`](project) package and the
`mayfly init` command. MayFly resolves an existing directory to an absolute,
clean path with symlinks evaluated, then records its identity in an external
registry (`~/.mayfly/projects.json` by default). On Linux the deterministic
project ID is derived from the directory's filesystem device/inode pair, so a
rename or same-filesystem move keeps the project while deletion/recreation does
not inherit its secrets. A symlink invocation resolves to the target project;
copying a project to another filesystem creates a new identity. Commands from
nested directories discover the nearest initialized ancestor. The registry is
never created inside the project tree; use `-registry` for a safe external
location when initializing a home-directory or other boundary-case root.

Initialize the current project with the repository command:

```bash
go run ./cmd/mayfly init
```

Use `-path DIR` for another existing project directory and `-registry FILE`
when an explicit external metadata location is required.

### Secret management commands

After initializing a project, the application-level CRUD commands are:

```text
mayfly set <NAME>
mayfly get <NAME>
mayfly list
mayfly delete <NAME>
```

`set` prompts for the vault password and then the value. On first use, it
creates the encrypted default vault at `~/.mayfly/vault.enc`; later calls
unlock that vault. Setting an existing name overwrites it. Names are
case-sensitive, may contain valid Unicode and spaces, and are limited to 255
UTF-8 bytes; empty names, control characters, NUL, and `=` are rejected.
Empty values are allowed. `list` prints names only, `delete` asks for `y`, and
`get` is the only command that explicitly prints a secret value. Values are
never printed by status messages, audit events, or errors.

The strict standard-library-only rule means the CLI currently uses visible
line input for password and secret-value prompts: Go's standard library does
not provide a portable hidden terminal-input primitive. The TUI's raw input
layer is separate and is not used by these commands yet. Do not place a
password or secret in command arguments, logs, or debug output. This visible
prompt is a documented usability limitation, not a claim of secure display.

### Child-process execution

Run a command with the current project's secrets using:

```text
mayfly run <COMMAND> [ARGS...]
```

MayFly passes the command and argument slice directly to `os/exec`; it does
not invoke a shell, so spaces and Unicode remain argument data rather than
being reparsed. The child's stdin, stdout, and stderr are connected to
MayFly's corresponding process streams. The parent environment is inherited,
except that a project secret with the same name explicitly overrides the
inherited variable. No environment or `.env` file is written.

Security boundary: a command that receives a secret can read it, print it,
save it, or pass it to descendants. MayFly controls storage and injection,
not the behavior of the command or its child processes. Values are loaded for
the selected project in memory only and transient references are released
after execution; Go's garbage collector does not guarantee cryptographic
erasure of prior string storage. A child terminated by a Unix signal is
reported using the conventional `128+signal` status where available.

### Audit trail

MayFly records safe application metadata in `~/.mayfly/audit.log` and exposes:

```text
mayfly audit
mayfly audit verify
```

The log uses canonical newline-delimited JSON event records, a SHA-256
previous/current hash chain, and a checkpoint containing the expected event
count and head hash. Verification detects altered, removed, reordered, or
malformed records and broken links. Appends use a synced temporary file and
atomic rename where supported, so a normal write failure leaves the previous
complete log in place. The trail is tamper-evident, not cryptographically
immutable: someone able to rewrite the log and its checkpoint can rewrite its
history. It contains project IDs, secret names, command names, and exit
statuses only—never secret values, passwords, keys, or environment dumps.

### Local secret scan

Run the heuristic scanner with:

```text
mayfly scan
```

The scanner examines the discovered project tree using Go filesystem APIs. It
reports high-risk names such as `.env`, credentials files, and private-key
extensions, plus a small set of private-key, password-assignment, token, and
API-key-like content patterns. Findings contain only relative paths, 1-based
line/Unicode-scalar columns where available, categories, severities, and safe
explanations; matched values are never printed. Exit status is `0` for no
findings, `3` when findings are reported, and `1` for an operational error.
`.git`, common generated/build directories, binary or malformed-UTF-8 files,
and files over the configured size limit are skipped. This is a heuristic
warning tool and is not proof that a project contains no secrets.

*(Full list with one-line rationale for each goes in `STDLIB.md` — this table is the preview.)*

## Threat Model

- **Protects against:** malicious install-time scripts scanning for plaintext secrets on disk
- **Does not protect against:** a fully compromised machine, a hostile process running *while* MayFly is actively injecting secrets, or someone with your master password
- Threat model and its honest limits are documented in full in the project's `STDLIB.md` / security notes

## Built For

Zero Dependency 2026 - a 72-hour hackathon by Hackathon Raptors. Track E: Security & Crypto Utilities.

## License

 GNU Affero General Public License Version 3
