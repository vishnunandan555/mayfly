# 🔐 MayFly Vault Package (`pkg/vault`)

[![Zero Dependencies](https://img.shields.io/badge/Dependencies-0%20(Pure%20Stdlib)-brightgreen)](../../STDLIB.md)
[![Go Version](https://img.shields.io/badge/Go-1.22+-blue)](../../go.mod)
[![Security Standard](https://img.shields.io/badge/Crypto-AES--256--GCM%20%2B%20PBKDF2-orange)](vault.go)


`pkg/vault` is a **zero-dependency, production-grade cryptographic secrets storage engine** built entirely with the Go standard library. It replaces heavy third-party cryptography and database dependencies with a clean, authenticated, tamper-resistant binary storage container.

---

## 📦 Packages Replaced (Package Killer)

| Third-Party Package | Typical Downloads | Standard Library Replacement in `pkg/vault` |
|---|---|---|
| `golang.org/x/crypto/pbkdf2` | **5M+/week** | Hand-rolled **RFC 8018 PBKDF2-HMAC-SHA256** using standard `crypto/hmac` and `crypto/sha256`. |
| `github.com/mattn/go-sqlite3` | **3M+/week** | Atomic binary authenticated container with AES-256-GCM AEAD encryption. |
| `github.com/zalando/go-keyring` | **1M+/week** | Pure stdlib cryptographic master-key encrypted vault file (`~/.mayfly/vault.enc`). |

---

## 🛡️ Cryptographic Architecture

```
[ Master Password (bytes) ] + [ 32-Byte Cryptographic Salt (crypto/rand) ]
                              │
                              ▼
            RFC 8018 PBKDF2-HMAC-SHA256 (600,000 Iterations)
                              │
                              ▼
                   [ 256-Bit Master Key ]
                              │
                              ▼
            AES-256-GCM AEAD Authenticated Encryption
                              │
  ┌───────────────────────────┴───────────────────────────┐
  │ Vault Binary Format (15-Byte Magic + Header + GCM)   │
  │   - Magic Header: "MAYFLY_VAULT_V1"                   │
  │   - 32-Byte Salt                                      │
  │   - 12-Byte Nonce                                     │
  │   - AES-GCM Ciphertext + 16-Byte Poly1305 Auth Tag    │
  └───────────────────────────────────────────────────────┘
```

### Key Security Properties:
1. **PBKDF2-HMAC-SHA256 (600,000 Iterations):** Exceeds OWASP 2024 recommendations for PBKDF2 key stretching, resisting GPU/ASIC brute-force dictionary attacks.
2. **AES-256-GCM AEAD:** Ensures both confidentiality and cryptographic integrity. Any tampering with encrypted bits causes immediate authentication failure.
3. **Atomic File Write (`temp file → fsync → rename`):** Guarantees zero vault corruption even if the machine suddenly loses power mid-write.

---

## 🚀 API Reference & Usage

### 1. Derive 256-Bit Key with RFC 8018 PBKDF2
```go
package main

import (
	"crypto/rand"
	"fmt"
	"mayfly/pkg/vault"
)

func main() {
	password := []byte("super-secure-master-password")
	salt := make([]byte, vault.SaltSize) // 32 bytes
	_, _ = rand.Read(salt)

	// Derives a 32-byte (256-bit) AES key using 600,000 HMAC-SHA256 rounds
	key := vault.DeriveKey(password, salt)
	fmt.Printf("Derived Key: %x\n", key)
}
```

### 2. Save and Load Encrypted Vault Storage
```go
package main

import (
	"fmt"
	"mayfly/pkg/domain"
	"mayfly/pkg/vault"
)

func main() {
	storage := vault.NewFileStorage("~/.mayfly/vault.enc")
	password := []byte("master-password-123")

	// Create project secrets payload
	data := domain.VaultData{
		Projects: map[string]domain.ProjectVault{
			"proj-123": {
				Secrets: map[domain.SecretName]string{
					"STRIPE_API_KEY": "sk_live_938472918237192",
				},
			},
		},
	}

	// Encrypt and atomically write to disk
	if err := storage.Save(data, password); err != nil {
		panic(err)
	}

	// Decrypt and load back into memory
	loaded, err := storage.Load(password)
	if err != nil {
		panic("Invalid password or corrupted vault!")
	}

	fmt.Println("Loaded Secret:", loaded.Projects["proj-123"].Secrets["STRIPE_API_KEY"])
}
```

---

## 🧪 Testing & Verification

Run the comprehensive vault test suite with the Go race detector:
```bash
go test -race -v ./pkg/vault
```
