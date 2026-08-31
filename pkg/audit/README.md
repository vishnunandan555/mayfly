# 📜 MayFly Cryptographic Audit Log (`pkg/audit`)

[![Zero Dependencies](https://img.shields.io/badge/Dependencies-0%20(Pure%20Stdlib)-brightgreen)](../../STDLIB.md)
[![Package Killer](https://img.shields.io/badge/Replaces-logrus%20%7C%20zap%20%7C%20SIEM-red)](../../STDLIB.md#7-tamper-evident-audit-trail)
[![Integrity](https://img.shields.io/badge/Blockchain%20Grade-SHA--256%20Hash%20Chained-orange)](log.go)


`pkg/audit` is a **zero-dependency, tamper-evident cryptographic access logger**. It records every secret access, injection, and password rotation event into a mathematically verifiable SHA-256 hash chain (`~/.mayfly/audit.log`).

---

## 📦 Packages Replaced (Package Killer)

| Third-Party Package | Typical Downloads | Standard Library Replacement in `pkg/audit` |
|---|---|---|
| `github.com/sirupsen/logrus` | **10M+/week** | Append-only structured JSON log recorder with timestamp precision. |
| `go.uber.org/zap` | **5M+/week** | Fast, lock-guarded buffered stdlib file writer with zero external dependencies. |
| SIEM / Blockchain Ledgers | Enterprise | Cryptographic SHA-256 hash-chaining where each event incorporates the previous event's hash. |

---

## ⛓️ Cryptographic Hash-Chain Architecture

```
Genesis Event (Seq: 1, PrevHash: "0000000000000000000000000000000000000000000000000000000000000000")
   │
   ▼
Hash_1 = SHA256(Seq:1 + Time + PrevHash + Action + Secret + Data)
   │
   ▼
Event 2 (Seq: 2, PrevHash: Hash_1)
   │
   ▼
Hash_2 = SHA256(Seq:2 + Time + PrevHash:Hash_1 + Action + Secret + Data)
   │
   ▼
Event 3 (Seq: 3, PrevHash: Hash_2) ──► Mathematical proof against log tampering! 🛡️
```

### Tamper-Evidence Properties:
* **Cannot modify past events:** Changing any field in a past entry invalidates all subsequent hash links.
* **Cannot delete past events:** Removing an entry breaks the parent hash pointer of the next entry.
* **Cannot reorder events:** Sequence numbers and timestamps are cryptographically bound into each node's SHA-256 digest.

---

## 🚀 API Reference & Usage

### 1. Record Secret Access Events
```go
package main

import (
	"context"
	"mayfly/pkg/audit"
	"mayfly/pkg/domain"
)

func main() {
	ctx := context.Background()

	// Initialize log engine at ~/.mayfly/audit.log
	logger := audit.NewFileLogger("~/.mayfly/audit.log")

	// Record a secret injection event
	_ = logger.Record(
		ctx,
		domain.ActionSecretInjected,
		"project-123",
		"STRIPE_API_KEY",
		"npm run start",
		map[string]any{"pid": 48192},
	)
}
```

### 2. Cryptographic Verification
```go
package main

import (
	"context"
	"fmt"
	"mayfly/pkg/audit"
)

func main() {
	ctx := context.Background()
	logger := audit.NewFileLogger("~/.mayfly/audit.log")

	// Verifies the entire mathematical hash chain from genesis to tail
	valid, count, err := logger.Verify(ctx)
	if err != nil || !valid {
		fmt.Printf("SECURITY ALERT: Audit log tampering detected!\n")
	} else {
		fmt.Printf("Audit log verified: %d authentic events with 100%% cryptographic integrity.\n", count)
	}
}
```

---

## 🧪 Testing & Verification

Run the cryptographic audit log test suite:
```bash
go test -race -v ./pkg/audit
```
