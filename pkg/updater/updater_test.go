package updater

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mayfly/pkg/domain"
)

func TestParseSemVerAndCompare(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"0.0.1", "0.0.2", -1},
		{"0.1.0", "0.0.9", 1},
		{"1.0.0", "1.0.0", 0},
		{"v0.0.1", "0.0.1", 0},
		{"v1.2.3", "v1.2.4", -1},
		{"v2.0.0", "v1.9.9", 1},
		{"0.0.1-beta", "0.0.1", 0},
	}

	for _, tt := range tests {
		sv1, err1 := ParseSemVer(tt.v1)
		if err1 != nil {
			t.Fatalf("ParseSemVer(%q) failed: %v", tt.v1, err1)
		}
		sv2, err2 := ParseSemVer(tt.v2)
		if err2 != nil {
			t.Fatalf("ParseSemVer(%q) failed: %v", tt.v2, err2)
		}

		res := sv1.Compare(sv2)
		if res != tt.expected {
			t.Errorf("Compare(%q, %q) = %d; want %d", tt.v1, tt.v2, res, tt.expected)
		}
	}
}

func TestCheckForUpdatesWithMockServer(t *testing.T) {
	origVersion := domain.Version
	domain.Version = "0.0.1"
	defer func() { domain.Version = origVersion }()

	mockRelease := ReleaseInfo{
		TagName:     "v0.0.2",
		Name:        "MayFly v0.0.2",
		Body:        "- Security updates\n- Performance improvements",
		PublishedAt: "2026-08-31T12:00:00Z",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockRelease)
	}))
	defer server.Close()

	ctx := context.Background()
	rel, isNewer, err := CheckForUpdates(ctx, server.URL)
	if err != nil {
		t.Fatalf("CheckForUpdates failed: %v", err)
	}

	if rel.TagName != "v0.0.2" {
		t.Errorf("expected tag v0.0.2, got %q", rel.TagName)
	}

	if !isNewer {
		t.Errorf("expected v0.0.2 to be newer than current v0.0.1")
	}
}

func TestParseExpectedHash(t *testing.T) {
	manifest := `# Official SHA-256 Checksums
a1b2c3d4e5f60718293a4b5c6d7e8f90123456789abcdef0123456789abcdef0  mayfly-linux-amd64
b2c3d4e5f60718293a4b5c6d7e8f90123456789abcdef0123456789abcdef01  mayfly-darwin-arm64
c3d4e5f60718293a4b5c6d7e8f90123456789abcdef0123456789abcdef012  mayfly-windows-amd64.exe
`
	hash := ParseExpectedHash(manifest, "mayfly-linux-amd64")
	if hash != "a1b2c3d4e5f60718293a4b5c6d7e8f90123456789abcdef0123456789abcdef0" {
		t.Fatalf("unexpected hash: %q", hash)
	}

	hashWin := ParseExpectedHash(manifest, "mayfly-windows-amd64.exe")
	if hashWin != "c3d4e5f60718293a4b5c6d7e8f90123456789abcdef0123456789abcdef012" {
		t.Fatalf("unexpected win hash: %q", hashWin)
	}

	hashMissing := ParseExpectedHash(manifest, "nonexistent-bin")
	if hashMissing != "" {
		t.Fatalf("expected empty for missing bin, got: %q", hashMissing)
	}
}

func TestTargetBinaryName(t *testing.T) {
	name := TargetBinaryName()
	if name == "" {
		t.Fatal("expected non-empty target binary name")
	}
}
