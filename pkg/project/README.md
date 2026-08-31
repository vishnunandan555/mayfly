# 📁 MayFly Workspace Project Registry (`pkg/project`)

[![Zero Dependencies](https://img.shields.io/badge/Dependencies-0%20(Pure%20Stdlib)-brightgreen)](../../STDLIB.md)
[![Package Killer](https://img.shields.io/badge/Replaces-google%2Fuuid%20%7C%20afero-red)](../../STDLIB.md#8-filesystem-project-identity)
[![Storage](https://img.shields.io/badge/Registry-Atomic%20JSON%20Persistence-blue)](registry.go)


`pkg/project` is a **zero-dependency directory registry and canonical path resolution engine** that links local development workspaces with their corresponding MayFly project vaults.

---

## 📦 Packages Replaced (Package Killer)

| Third-Party Package | Typical Downloads | Standard Library Replacement in `pkg/project` |
|---|---|---|
| `github.com/google/uuid` | **20M+/week** | Cryptographic random UUIDv4 generation using standard library `crypto/rand`. |
| `github.com/spf13/afero` | **5M+/week** | Standard library `filepath.EvalSymlinks`, `filepath.Clean`, and atomic file swaps. |

---

## 🧭 Architectural Features

1. **Canonical Symlink Resolution:** Normalizes complex symlinks and relative path variations (`./`, `../`, symlinked directories) into unambiguous absolute canonical paths using `filepath.EvalSymlinks` and `filepath.Abs`.
2. **Deterministic & Random UUIDv4 Engine:** Custom RFC 4122 compliant UUIDv4 generator powered strictly by Go's `crypto/rand`, ensuring unguessable project identifiers without third-party libraries.
3. **Thread-Safe Atomic Registry:** Protects registry mutations with `sync.RWMutex` and atomic file write patterns (`temp file → fsync → rename`).

---

## 🚀 API Reference & Usage

```go
package main

import (
	"fmt"
	"mayfly/pkg/project"
)

func main() {
	// Initialize registry at ~/.mayfly/projects.json
	reg := project.NewRegistry("~/.mayfly/projects.json")

	// Register a workspace directory
	proj, err := reg.Register("/home/user/my-app")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Registered Project ID: %s for Path: %s\n", proj.ID, proj.CanonicalPath)

	// Resolve project for current working directory
	found, err := reg.ResolveCurrent("/home/user/my-app/src")
	if err == nil {
		fmt.Printf("Resolved Parent Project: %s\n", found.ID)
	}
}
```

---

## 🧪 Testing & Verification

Run project registry unit tests:
```bash
go test -race -v ./pkg/project
```
