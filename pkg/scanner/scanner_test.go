package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScannerDetectsPlaintextLeaks(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// 1. Create a dangerous .env file
	envFile := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envFile, []byte("DATABASE_PASSWORD=\"super_secret_db_password_12345\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. Create a source code file with hardcoded token assignment
	srcFile := filepath.Join(tmpDir, "server.js")
	if err := os.WriteFile(srcFile, []byte("const secret_key = \"sample_test_hardcoded_credential_98765\";\n"), 0644); err != nil {
		t.Fatal(err)
	}

	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}

	findings, err := s.Scan(ctx, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(findings) < 2 {
		t.Fatalf("expected at least 2 findings, got %d", len(findings))
	}
}

func TestMayflyIgnoreSupport(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Create a test fixture with leak
	testDir := filepath.Join(tmpDir, "fixtures")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcFile := filepath.Join(testDir, "mock_test.js")
	if err := os.WriteFile(srcFile, []byte("const secret_key = \"sample_test_hardcoded_credential_98765\";\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create .mayflyignore ignoring fixtures/
	ignoreFile := filepath.Join(tmpDir, ".mayflyignore")
	if err := os.WriteFile(ignoreFile, []byte("# ignore test fixtures\nfixtures/\n"), 0644); err != nil {
		t.Fatal(err)
	}

	s, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}

	findings, err := s.Scan(ctx, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(findings) != 0 {
		t.Fatalf("expected 0 findings due to .mayflyignore, got %d: %+v", len(findings), findings)
	}
}

