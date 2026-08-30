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
