# ⚡ MayFly Process Executor (`pkg/executor`)

[![Zero Dependencies](https://img.shields.io/badge/Dependencies-0%20(Pure%20Stdlib)-brightgreen)](../../STDLIB.md)
[![Package Killer](https://img.shields.io/badge/Replaces-godotenv%20%7C%20dotenv-red)](../../STDLIB.md#6-in-memory-environment-injection)
[![Memory Security](https://img.shields.io/badge/RAM%20Boundary-Zero--Disk%20I%2FO-blue)](process.go)


`pkg/executor` is a **zero-dependency, zero-disk in-memory process environment injection engine**. It allows applications to receive runtime secrets without ever writing a plaintext `.env` file to disk.

---

## 📦 Packages Replaced (Package Killer)

| Third-Party Package | Typical Downloads | Standard Library Replacement in `pkg/executor` |
|---|---|---|
| `github.com/joho/godotenv` | **12M+/week** | Direct in-memory `os.Environ` overlay into child process `exec.Cmd.Env`. |
| `github.com/subosito/gotenv` | **2M+/week** | Clean RAM table builder with zero disk writes. |
| `github.com/direnv/direnv` | **1M+/week** | Ephemeral process-scoped lifecycle with deferred RAM wiping and signal forwarding. |

---

## 🛡️ In-Memory Security Boundary

Modern supply-chain attacks (`postinstall` malware in `npm`, `pip`, `cargo`) systematically scan the working directory for `.env` files to exfiltrate database credentials and API keys.

```
Traditional Workflow (VULNERABLE):
[.env on Disk] ───────► Malicious npm/pip script reads file ───────► Credential Exfiltrated! 🚨

MayFly In-Memory Injection (SECURE):
[RAM Decryption] ─────► Injected into Child Process RAM Table ─────► NO DISK FILE EVER CREATED! ✅
                                  │
                                  ▼
                     Process Finishes (SIGINT/SIGTERM)
                                  │
                                  ▼
                Deferred Zeroization & runtime.GC() Wipe RAM
```

### Key Technical Mechanisms:
1. **Zero-Disk Boundary:** Decrypted credentials exist exclusively in volatile process heap memory. No temporary files (`/tmp`, `.env.tmp`) are ever created.
2. **Signal Forwarding:** Intercepts `os.Interrupt` (`SIGINT`) and `syscall.SIGTERM` to propagate termination directly to the child process tree, preventing orphaned or zombie background processes.
3. **Deferred Memory Zeroization:** Wipes internal environment slices and maps with zero-byte buffers and invokes `runtime.GC()` immediately when execution finishes.

---

## 🚀 API Reference & Usage

```go
package main

import (
	"context"
	"fmt"
	"os"

	"mayfly/pkg/domain"
	"mayfly/pkg/executor"
)

func main() {
	ctx := context.Background()

	// Initialize executor with standard OS streams
	exec := executor.NewProcessExecutor(nil, os.Stdout, os.Stderr)

	// Define runtime secrets to overlay in RAM
	secrets := map[domain.SecretName]string{
		"DATABASE_URL":   "postgres://user:pass@localhost:5432/app",
		"GEMINI_API_KEY": "AIzaSyB0xIf9lgC_FJPU...",
	}

	// Execution request
	req := domain.ExecutionRequest{
		Command: []string{"node", "server.js"},
		Dir:     "/home/user/project",
	}

	// Executes process with injected secrets in RAM
	result, err := exec.Execute(ctx, req, secrets)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Execution failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Process completed with exit code: %d\n", result.ExitCode)
}
```

---

## 🧪 Testing & Verification

Run the process executor test suite:
```bash
go test -race -v ./pkg/executor
```
