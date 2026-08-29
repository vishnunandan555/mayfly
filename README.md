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
mayfly npm run dev
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

*(Full list with one-line rationale for each goes in `STDLIB.md` — this table is the preview.)*

## Threat Model

- **Protects against:** malicious install-time scripts scanning for plaintext secrets on disk
- **Does not protect against:** a fully compromised machine, a hostile process running *while* MayFly is actively injecting secrets, or someone with your master password
- Threat model and its honest limits are documented in full in the project's `STDLIB.md` / security notes

## Built For

Zero Dependency 2026 - a 72-hour hackathon by Hackathon Raptors. Track E: Security & Crypto Utilities.

## License

 GNU Affero General Public License Version 3
