package vault

import (
	"bytes"
	"encoding/hex"
	"path/filepath"
	"testing"

	"mayfly/pkg/domain"
)

func TestPBKDF2RFC6070TestVectors(t *testing.T) {
	tests := []struct {
		name       string
		password   string
		salt       string
		iterations int
		keyLen     int
		expected   string
	}{
		{
			name:       "RFC-PBKDF2-HMAC-SHA256-Vector-1-iter",
			password:   "password",
			salt:       "salt",
			iterations: 1,
			keyLen:     32,
			expected:   "120fb6cffcf8b32c43e7225256c4f837a86548c92ccc35480805987cb70be17b",
		},
		{
			name:       "RFC-PBKDF2-HMAC-SHA256-Vector-2-iters",
			password:   "password",
			salt:       "salt",
			iterations: 2,
			keyLen:     32,
			expected:   "ae4d0c95af6b46d32d0adff928f06dd02a303f8ef3c251dfd6e2d85a95474c43",
		},
		{
			name:       "RFC-PBKDF2-HMAC-SHA256-Vector-4096-iters",
			password:   "password",
			salt:       "salt",
			iterations: 4096,
			keyLen:     32,
			expected:   "c5e478d59288c841aa530db6845c4c8d962893a001ce4e11a4963873aa98134a",
		},
		{
			name:       "RFC-PBKDF2-HMAC-SHA256-MultiBlock-40-bytes",
			password:   "passwordPASSWORDpassword",
			salt:       "saltSALTsaltSALTsaltSALTsaltSALTsalt",
			iterations: 4096,
			keyLen:     40,
			expected:   "348c89dbcbd32b2f32d814b8116e84cf2b17347ebc1800181c4e2a1fb8dd53e1c635518c7dac47e9",
		},
		{
			name:       "RFC-PBKDF2-HMAC-SHA256-NullCharacter",
			password:   "pass\x00word",
			salt:       "sa\x00lt",
			iterations: 4096,
			keyLen:     16,
			expected:   "89b69d0516f829893c696226650a8687",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			derived, err := DeriveKey([]byte(tc.password), []byte(tc.salt), tc.iterations, tc.keyLen)
			if err != nil {
				t.Fatalf("DeriveKey failed: %v", err)
			}
			gotHex := hex.EncodeToString(derived)
			if gotHex != tc.expected {
				t.Fatalf("mismatch for %s:\n  got:  %s\n  want: %s", tc.name, gotHex, tc.expected)
			}
		})
	}
}

func TestPBKDF2KeyDerivation(t *testing.T) {
	key1, err := DeriveKey([]byte("password"), []byte("salt123456789012"), 1000, 32)
	if err != nil {
		t.Fatal(err)
	}
	if len(key1) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(key1))
	}

	key2, err := DeriveKey([]byte("password"), []byte("salt123456789012"), 1000, 32)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(key1, key2) {
		t.Fatal("deterministic KDF produced different keys for same input")
	}

	key3, _ := DeriveKey([]byte("otherpass"), []byte("salt123456789012"), 1000, 32)
	if bytes.Equal(key1, key3) {
		t.Fatal("different passwords produced same key")
	}
}

func TestVaultStorageLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	vaultPath := filepath.Join(tmpDir, "vault.enc")

	storage, err := NewStorage(vaultPath, 1000)
	if err != nil {
		t.Fatal(err)
	}

	pass := []byte("secret-master-pass")

	// 1. Initialize
	if err := storage.Initialize(pass); err != nil {
		t.Fatal(err)
	}

	// 2. Open
	record, err := storage.Open(pass)
	if err != nil {
		t.Fatal(err)
	}

	if record.Projects == nil {
		t.Fatal("expected projects map to be initialized")
	}

	// 3. Save modified
	record.Projects["proj1"] = map[domain.SecretName]string{"API_KEY": "sk-123456"}
	if err := storage.Save(record, pass); err != nil {
		t.Fatal(err)
	}

	// 4. Open again
	reopened, err := storage.Open(pass)
	if err != nil {
		t.Fatal(err)
	}

	if reopened.Projects["proj1"]["API_KEY"] != "sk-123456" {
		t.Fatalf("unexpected secret value: %s", reopened.Projects["proj1"]["API_KEY"])
	}

	// 5. Wrong password
	_, err = storage.Open([]byte("wrongpass"))
	if err != ErrWrongPassword {
		t.Fatalf("expected ErrWrongPassword, got %v", err)
	}
}
