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

func TestEnvExampleIgnoredWhenSafe(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	// Safe .env.example with placeholders, comments, and localhost URLs
	exampleContent := `# Supabase project
VITE_SUPABASE_URL=
VITE_SUPABASE_ANON_KEY=
VITE_API_URL=http://localhost:4000
DATABASE_URL=postgres://user:password@localhost:5432/mydb
STRIPE_KEY="your-api-key-here"
`
	if err := os.WriteFile(filepath.Join(tmpDir, ".env.example"), []byte(exampleContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Also test subfolder example: server/.env.sample
	serverDir := filepath.Join(tmpDir, "server")
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, ".env.sample"), []byte("PORT=3000\nNODE_ENV=development\n"), 0644); err != nil {
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
		t.Fatalf("expected 0 findings for safe .env.example files, got %d: %+v", len(findings), findings)
	}
}

func TestEnvExampleFlaggedWhenRealTokenPresent(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	mockStripe := "sk_" + "live_" + "51MzFakeKey998877665544332211"
	mockAWS := "AK" + "IA" + "IOSFODNN7EXAMPLE"

	// .env.example containing an accidental live Stripe key and AWS key
	leakedContent := "# Example file with leaked credentials\nSTRIPE_KEY=" + mockStripe + "\nAWS_KEY=" + mockAWS + "\n"
	if err := os.WriteFile(filepath.Join(tmpDir, ".env.example"), []byte(leakedContent), 0644); err != nil {
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

	if len(findings) != 2 {
		t.Fatalf("expected 2 token findings in .env.example, got %d: %+v", len(findings), findings)
	}

	if findings[0].Category != "stripe-key" && findings[1].Category != "stripe-key" {
		t.Errorf("expected stripe-key finding, got: %+v", findings)
	}
	if findings[0].Category != "aws-key" && findings[1].Category != "aws-key" {
		t.Errorf("expected aws-key finding, got: %+v", findings)
	}
}

func TestActiveEnvFileFlaggedAsCritical(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("PORT=3000\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".env.production"), []byte("SECRET=12345\n"), 0644); err != nil {
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

	if len(findings) != 2 {
		t.Fatalf("expected 2 critical findings for active .env files, got %d: %+v", len(findings), findings)
	}
	for _, f := range findings {
		if f.Category != "plaintext-env-file" || f.Severity != "CRITICAL" {
			t.Errorf("expected CRITICAL plaintext-env-file, got: %+v", f)
		}
	}
}

