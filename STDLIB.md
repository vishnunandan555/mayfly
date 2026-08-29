# Standard Library Substitutions in MayFly

MayFly is built entirely with the **Go 1.27 standard library** with zero runtime third-party dependencies (`go.mod` contains no `require` block).

Every major component that typically relies on external packages has been designed and implemented from first principles using standard library primitives.

---

## Component Substitution Matrix

| System Component | Conventional Third-Party Package | Standard Library Replacement in MayFly | Implementation Location | Standard Library Packages Used |
|---|---|---|---|---|
| **TUI Renderer & Runtime** | `github.com/charmbracelet/bubbletea`<br>`github.com/rivo/tview`<br>`github.com/nsf/termbox-go` | Repository-owned terminal engine: raw termios mode, ANSI/VT escape sequences, streaming key event parser, 2D cell canvas, deterministic layout engine, and full widget set. | `screen/` | `os`, `syscall`, `io`, `sync`, `unicode`, `unicode/utf8`, `bytes`, `strings`, `fmt`, `time` |
| **Key Derivation Function** | `golang.org/x/crypto/pbkdf2`<br>`golang.org/x/crypto/scrypt` | RFC 8018 PBKDF2-HMAC-SHA256 implementation with bounded work factor (600,000 default iterations). | `vault/kdf.go` | `crypto/hmac`, `crypto/sha256`, `encoding/binary` |
| **Encrypted Vault Storage** | `github.com/mattn/go-sqlite3`<br>`go.etcd.io/bbolt`<br>`github.com/zalando/go-keyring` | Versioned binary container: authenticated header, AES-256-GCM encryption with fresh per-save nonces, authenticated associated data, and atomic filesystem persistence. | `vault/` | `crypto/aes`, `crypto/cipher`, `crypto/rand`, `encoding/json`, `os`, `path/filepath`, `sync` |
| **Secret Injection & Execution** | `github.com/joho/godotenv`<br>`github.com/subosito/gotenv` | In-memory environment injection: parent environment inheritance with explicit secret override precedence and transient memory reference clearing upon child process termination. | `executor/` | `os`, `os/exec`, `context`, `strings`, `unicode/utf8`, `errors`, `fmt` |
| **Tamper-Evident Audit Log** | `github.com/sirupsen/logrus`<br>`go.uber.org/zap`<br>External SIEM / Log DB | SHA-256 hash-chained canonical JSON event records with previous-hash links, sequence numbers, and checkpoint integrity verification. | `audit/` | `crypto/sha256`, `encoding/hex`, `encoding/json`, `os`, `path/filepath`, `sync`, `time` |
| **Project Identity & Registry** | `github.com/go-git/go-git`<br>`github.com/google/uuid` | Deterministic Linux filesystem `(device, inode)` hashing, absolute path canonicalization via `filepath.EvalSymlinks`, and external JSON registry. | `project/` | `os`, `syscall`, `crypto/sha256`, `encoding/hex`, `encoding/json`, `path/filepath`, `strings` |
| **Heuristic Secret Scanner** | `github.com/trufflesecurity/trufflehog`<br>`github.com/zricethezav/gitleaks`<br>`ripgrep` / `grep` | Heuristic AST/pattern scanner walking project directories with size boundaries (1 MiB default), `.git`/build skips, UTF-8 validation, and location reporting without printing matched secrets. | `scanner/` | `path/filepath`, `regexp`, `os`, `io`, `unicode/utf8`, `strings`, `sort` |
| **CLI Command Parsing** | `github.com/spf13/cobra`<br>`github.com/urfave/cli` | Standard `flag` package with subcommand dispatch, clear usage formatting, stdout/stderr separation, and standard exit codes. | `cmd/mayfly/` | `flag`, `os`, `bufio`, `strings`, `fmt`, `errors` |
| **Terminal Styling & Color** | `github.com/fatih/color`<br>`github.com/mgutz/ansi` | Hand-crafted ANSI SGR sequence builder supporting 16-color foregrounds/backgrounds, bold, dim, reverse, underline, reset, and `NO_COLOR` compliance. | `screen/terminal.go` | `strings`, `strconv`, `os` |

---

## Detailed Substitution Architecture

### 1. Custom Terminal UI Engine (`screen/`)
- **Raw Mode & Terminal Sizing**: Direct Linux `termios` ioctl manipulation through standard `syscall.Syscall` (`TCGETS`/`TCSETS`) and window size ioctl (`TIOCGWINSZ`). Restores terminal state cleanly on normal exit, error return, panics, and OS signals (`SIGINT`, `SIGTERM`, `SIGWINCH`).
- **ANSI Key Parser**: Incremental byte-by-byte finite state machine resolving single escapes, CSI sequences (`\x1b[...]`), SS3 sequences (`\x1bO...`), UTF-8 multibyte characters, and function keys with configurable ambiguity timeouts.
- **2D Frame Canvas**: Cell grid supporting Unicode display widths (`RuneWidth`), text alignment, bordered boxes, clipping regions, and diff-based dirty row overwrites.

### 2. PBKDF2 Key Derivation (`vault/kdf.go`)
- RFC 8018 specification implemented directly using `crypto/hmac` and `crypto/sha256`.
- Persists iteration counts in authenticated headers to allow work factor evolution without breaking older vaults.

### 3. Encrypted Vault Container (`vault/`)
- Header structure: 6-byte magic (`MFVAUL`), 1-byte version, 1-byte KDF ID, 4-byte iteration count, 2-byte salt length, 1-byte nonce length, salt bytes, and nonce bytes.
- Header is passed as authenticated associated data to `cipher.AEAD.Seal` / `cipher.AEAD.Open`.
- Atomic persistence: complete encrypted payload written to same-directory temporary file (`.mayfly-vault-*`), chmodded `0600`, synced, and atomically renamed.

### 4. Process Injection (`executor/`)
- Direct invocation of `os/exec.CommandContext` without spawning intermediate shells (`sh`, `bash`).
- Preserves discrete command argument slices exactly.
- Environment merging: inherits `os.Environ()`, removes overridden names, and appends project secrets. Clears environment slice memory upon return.

### 5. Tamper-Evident Audit Trail (`audit/`)
- Canonical newline-delimited JSON records.
- Each event record includes: sequence, timestamp, action, project ID, secret name, command, exit status, previous record's SHA-256 hash, and current record's SHA-256 hash.
- Verified on startup and via `mayfly audit verify`.
