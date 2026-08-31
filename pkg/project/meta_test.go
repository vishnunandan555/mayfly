package project

import (
	"path/filepath"
	"testing"
	"time"
)

func TestMetaStoreLockoutWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	metaPath := filepath.Join(tmpDir, "meta.json")

	store, err := NewMetaStore(metaPath)
	if err != nil {
		t.Fatalf("NewMetaStore failed: %v", err)
	}

	// 1. Initial state: not locked
	locked, remaining := store.IsLocked()
	if locked || remaining > 0 {
		t.Fatalf("expected initial state not locked, got locked=%v, remaining=%v", locked, remaining)
	}

	// 2. Record 4 failed attempts (threshold is 5)
	for i := 1; i <= 4; i++ {
		if err := store.RecordFailedAttempt(); err != nil {
			t.Fatalf("RecordFailedAttempt %d failed: %v", i, err)
		}
		meta, err := store.Read()
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
		if meta.FailedAttempts != i {
			t.Errorf("expected FailedAttempts=%d, got %d", i, meta.FailedAttempts)
		}
		locked, _ := store.IsLocked()
		if locked {
			t.Errorf("expected not locked at %d attempts", i)
		}
	}

	// 3. 5th failed attempt should trigger lockout
	if err := store.RecordFailedAttempt(); err != nil {
		t.Fatalf("RecordFailedAttempt 5 failed: %v", err)
	}

	locked, remaining = store.IsLocked()
	if !locked {
		t.Fatalf("expected locked after 5 failed attempts")
	}
	if remaining <= 0 || remaining > 31*time.Second {
		t.Errorf("expected remaining lockout time ~30s, got %v", remaining)
	}

	// 4. Successful login resets counter and lockout
	if err := store.RecordSuccess(); err != nil {
		t.Fatalf("RecordSuccess failed: %v", err)
	}

	locked, remaining = store.IsLocked()
	if locked || remaining > 0 {
		t.Errorf("expected lockout cleared after RecordSuccess, got locked=%v", locked)
	}

	meta, err := store.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if meta.FailedAttempts != 0 {
		t.Errorf("expected FailedAttempts=0 after RecordSuccess, got %d", meta.FailedAttempts)
	}
}
