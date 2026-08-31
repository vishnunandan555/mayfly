# 🔍 MayFly Credential Scanner (`pkg/scanner`)

[![Zero Dependencies](https://img.shields.io/badge/Dependencies-0%20(Pure%20Stdlib)-brightgreen)](../../STDLIB.md)
[![Package Killer](https://img.shields.io/badge/Replaces-trufflehog%20%7C%20gitleaks-red)](../../STDLIB.md#9-plaintext-credential-scanner)
[![Detection](https://img.shields.io/badge/Engine-Entropy%20%2B%20Regex%20Signatures-blue)](scanner.go)


`pkg/scanner` is a **high-performance, zero-dependency filesystem crawler and credential leak detector** that inspects codebases for plaintext secrets before git commits occur.

---

## 📦 Packages Replaced (Package Killer)

| Third-Party Package | Typical Downloads | Standard Library Replacement in `pkg/scanner` |
|---|---|---|
| `github.com/trufflesecurity/trufflehog` | **500K+/week** | Bounded stdlib `filepath.WalkDir` with Shannon entropy analysis. |
| `github.com/zricethezav/gitleaks` | **1M+/week** | Standard library `regexp` credential pattern matchers + custom `.mayflyignore` parser. |

---

## 🛡️ Detection Engine Features

1. **Shannon Entropy Analysis:** Computes character set information density to catch high-entropy random strings (e.g. Base64 API tokens, hex hashes, private keys) without false-positive dictionary words.
2. **Regex Signature Matrix:**
   * AWS Access & Secret Keys (`AKIA[0-9A-Z]{16}`)
   * Stripe Live & Test API Keys (`sk_live_[0-9a-zA-Z]{24}`)
   * GitHub Personal & OAuth Tokens (`ghp_[0-9a-zA-Z]{36}`, `github_pat_`)
   * Google Gemini & Cloud API Keys (`AIza[0-9A-Za-z\\-_]{35}`)
   * Generic Private Keys (`-----BEGIN RSA PRIVATE KEY-----`)
   * Plaintext `.env` key-value pairs (`DATABASE_URL=postgres://...`)
3. **`.mayflyignore` Parser:** Supports custom ignore rules, comments (`#`), and glob syntax to exclude vendor directories (`node_modules/`, `dist/`, `.git/`).

---

## 🚀 API Reference & Usage

```go
package main

import (
	"context"
	"fmt"
	"mayfly/pkg/scanner"
)

func main() {
	ctx := context.Background()

	// Initialize scanner with default ignore patterns
	sc := scanner.New()

	// Scan directory for plaintext secret leaks
	findings, err := sc.ScanDir(ctx, "/path/to/project")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Scan complete! Found %d potential leak(s):\n", len(findings))
	for _, f := range findings {
		fmt.Printf("  [%s] %s:%d - %s\n", f.Severity, f.Path, f.Line, f.Message)
	}
}
```

---

## 🧪 Testing & Verification

Run scanner unit tests:
```bash
go test -race -v ./pkg/scanner
```
