package audit

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"mayfly/pkg/domain"
)

func TestAuditLogChainAndVerification(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	log, err := New(logPath)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Record events
	if err := log.Record(ctx, domain.ActionProjectInit, "proj1", "", "", nil); err != nil {
		t.Fatal(err)
	}
	if err := log.Record(ctx, domain.ActionSecretSet, "proj1", "API_KEY", "", nil); err != nil {
		t.Fatal(err)
	}

	// 2. Verify chain is intact
	if err := log.Verify(ctx); err != nil {
		t.Fatalf("expected valid chain, got: %v", err)
	}

	// 3. List events
	events, err := log.Events(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// 4. Test tampering detection
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	// Corrupt a byte in the middle of the log
	tampered := append([]byte(nil), data...)
	tampered[len(tampered)/2] ^= 0xFF
	_ = os.WriteFile(logPath, tampered, 0600)

	log2, _ := New(logPath)
	if log2 != nil {
		err = log2.Verify(ctx)
		if err == nil {
			t.Fatal("expected verification failure after tampering, but got success")
		}
	}
}

func TestAuditLogBOMHandling(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")

	log, err := New(logPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := log.Record(ctx, domain.ActionProjectInit, "proj1", "", "", nil); err != nil {
		t.Fatal(err)
	}

	// Prepend UTF-8 BOM as Windows Notepad or PowerShell Out-File might do
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	bomData := append([]byte("\xef\xbb\xbf"), data...)
	if err := os.WriteFile(logPath, bomData, 0600); err != nil {
		t.Fatal(err)
	}

	// Should successfully initialize and verify despite BOM
	logBOM, err := New(logPath)
	if err != nil {
		t.Fatalf("expected New() to succeed with UTF-8 BOM, got: %v", err)
	}

	if err := logBOM.Verify(ctx); err != nil {
		t.Fatalf("expected Verify() to succeed with UTF-8 BOM, got: %v", err)
	}

	events, err := logBOM.Events(ctx)
	if err != nil {
		t.Fatalf("expected Events() to succeed with UTF-8 BOM, got: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}
