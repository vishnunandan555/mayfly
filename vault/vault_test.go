package vault

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"mayfly/application"
	"mayfly/domain"
)

type counterReader struct{ next byte }

func (r *counterReader) Read(buffer []byte) (int, error) {
	for index := range buffer {
		buffer[index] = r.next
		r.next++
	}
	return len(buffer), nil
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("random source failed") }

func newTestStorage(t *testing.T, random interface{ Read([]byte) (int, error) }) (*Storage, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nested", "vault.enc")
	storage, err := NewStorage(path, Options{Iterations: minimumIterations, Random: random})
	if err != nil {
		t.Fatal(err)
	}
	return storage, path
}

func initializeTestVault(t *testing.T) (*Storage, *Vault, string, []byte) {
	t.Helper()
	password := []byte("correct horse battery")
	storage, path := newTestStorage(t, &counterReader{})
	if err := storage.Initialize(password); err != nil {
		t.Fatal(err)
	}
	opened, err := storage.Unlock(context.Background(), password)
	if err != nil {
		t.Fatal(err)
	}
	return storage, opened, path, password
}

func TestPBKDF2HMACSHA256KnownVector(t *testing.T) {
	got := deriveKey([]byte("password"), []byte("salt"), 1, 32)
	want, err := hex.DecodeString("120fb6cffcf8b32c43e7225256c4f837a86548c92ccc35480805987cb70be17b")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("derived key = %x, want %x", got, want)
	}
}

func TestInitializeUnlockAndWrongPassword(t *testing.T) {
	storage, opened, _, password := initializeTestVault(t)
	defer opened.Close()
	if _, err := storage.Unlock(context.Background(), []byte("wrong password")); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("wrong password error = %v, want ErrWrongPassword", err)
	}
	if _, err := storage.Unlock(context.Background(), nil); !errors.Is(err, ErrPasswordRequired) {
		t.Fatalf("empty password error = %v, want ErrPasswordRequired", err)
	}
	if err := storage.Initialize(password); !errors.Is(err, ErrVaultExists) {
		t.Fatalf("second initialize error = %v, want ErrVaultExists", err)
	}
}

func TestSaveReloadProcessRestartMultipleProjectsAndUnicode(t *testing.T) {
	_, opened, path, password := initializeTestVault(t)
	projectOne := domain.ProjectID("project-one")
	projectTwo := domain.ProjectID("project-two")
	if err := opened.PutProject(context.Background(), domain.Project{ID: projectOne, Name: "One", Path: "/tmp/one"}); err != nil {
		t.Fatal(err)
	}
	if err := opened.PutProject(context.Background(), domain.Project{ID: projectTwo, Name: "二", Path: "/tmp/two"}); err != nil {
		t.Fatal(err)
	}
	inputs := []domain.SecretInput{
		{ProjectID: projectOne, Name: "ASCII_KEY", Value: "value-one"},
		{ProjectID: projectOne, Name: "空の値", Value: ""},
		{ProjectID: projectTwo, Name: "ключ🔐", Value: "秘密値"},
		{ProjectID: projectTwo, Name: "LARGE", Value: strings.Repeat("large-value-", 10000)},
	}
	for _, input := range inputs {
		if err := opened.Put(context.Background(), input); err != nil {
			t.Fatal(err)
		}
	}
	list, err := opened.List(context.Background(), projectOne)
	if err != nil || len(list) != 2 {
		t.Fatalf("project one list = %#v, %v", list, err)
	}
	if _, err := opened.Get(context.Background(), projectTwo, "ключ🔐"); err != nil {
		t.Fatal(err)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}

	// A new Storage and Vault simulate a process restart; no in-memory session
	// is reused.
	restartedStorage, err := NewStorage(path, Options{Iterations: minimumIterations, Random: &counterReader{next: 99}})
	if err != nil {
		t.Fatal(err)
	}
	restarted, err := restartedStorage.Unlock(context.Background(), password)
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	material, err := restarted.Get(context.Background(), projectTwo, "ключ🔐")
	if err != nil || material.Value != "秘密値" {
		t.Fatalf("reloaded Unicode value = %q, %v", material.Value, err)
	}
	projects, err := restarted.ListProjects(context.Background())
	if err != nil || len(projects) != 2 {
		t.Fatalf("reloaded projects = %#v, %v", projects, err)
	}
}

func TestSerializedVaultDoesNotContainControlledPlaintext(t *testing.T) {
	_, opened, path, _ := initializeTestVault(t)
	defer opened.Close()
	secret := "controlled-test-secret-not-on-disk"
	if err := opened.Put(context.Background(), domain.SecretInput{ProjectID: "project", Name: "TOKEN", Value: secret}); err != nil {
		t.Fatal(err)
	}
	serialized, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(serialized, []byte(secret)) {
		t.Fatal("secret plaintext appeared in serialized vault")
	}
}

func TestRepeatedSavesUpdateAndUseFreshNonce(t *testing.T) {
	_, opened, path, _ := initializeTestVault(t)
	defer opened.Close()
	input := domain.SecretInput{ProjectID: "project", Name: "TOKEN", Value: "first"}
	if err := opened.Put(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	input.Value = "second"
	if err := opened.Put(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first, second) {
		t.Fatal("repeated save produced identical encrypted bytes")
	}
	material, err := opened.Get(context.Background(), "project", "TOKEN")
	if err != nil || material.Value != "second" {
		t.Fatalf("updated value = %q, %v", material.Value, err)
	}
}

func TestDeleteAndEnumerateMetadataWithoutValues(t *testing.T) {
	_, opened, _, _ := initializeTestVault(t)
	defer opened.Close()
	if err := opened.Put(context.Background(), domain.SecretInput{ProjectID: "project", Name: "TOKEN", Value: "private"}); err != nil {
		t.Fatal(err)
	}
	metadata, err := opened.List(context.Background(), "project")
	if err != nil || len(metadata) != 1 || metadata[0].Name != "TOKEN" {
		t.Fatalf("metadata = %#v, %v", metadata, err)
	}
	if err := opened.Delete(context.Background(), "project", "TOKEN"); err != nil {
		t.Fatal(err)
	}
	if _, err := opened.Get(context.Background(), "project", "TOKEN"); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("deleted secret error = %v", err)
	}
}

func TestMalformedUnsupportedTruncatedAndTamperedVaults(t *testing.T) {
	tests := []struct {
		name string
		edit func([]byte)
		want error
	}{
		{name: "malformed header", edit: func(data []byte) { copy(data, []byte("broken")) }, want: ErrInvalidFormat},
		{name: "unsupported version", edit: func(data []byte) { data[6] = 2 }, want: ErrUnsupportedVersion},
		{name: "truncated", edit: func(data []byte) { data = data[:len(data)-1] }, want: ErrWrongPassword},
		{name: "tampered metadata", edit: func(data []byte) { data[headerFixedSize] ^= 1 }, want: ErrWrongPassword},
		{name: "tampered ciphertext", edit: func(data []byte) { data[len(data)-1] ^= 1 }, want: ErrWrongPassword},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			storage, opened, path, password := initializeTestVault(t)
			if err := opened.Put(context.Background(), domain.SecretInput{ProjectID: "project", Name: "TOKEN", Value: "value"}); err != nil {
				t.Fatal(err)
			}
			opened.Close()
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if test.name == "truncated" {
				data = data[:len(data)-1]
			} else {
				test.edit(data)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := storage.Unlock(context.Background(), password); !errors.Is(err, test.want) {
				t.Fatalf("Unlock error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestFailedSaveLeavesMemoryAndFileUnchanged(t *testing.T) {
	storage, opened, path, password := initializeTestVault(t)
	defer opened.Close()
	if err := opened.Put(context.Background(), domain.SecretInput{ProjectID: "project", Name: "OLD", Value: "old-value"}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	failingStorage, err := NewStorage(path, Options{Iterations: minimumIterations, Random: failingReader{}})
	if err != nil {
		t.Fatal(err)
	}
	failingVault, err := failingStorage.Unlock(context.Background(), password)
	if err != nil {
		t.Fatal(err)
	}
	defer failingVault.Close()
	if err := failingVault.Put(context.Background(), domain.SecretInput{ProjectID: "project", Name: "NEW", Value: "new-value"}); err == nil {
		t.Fatal("Put unexpectedly succeeded with failing random source")
	}
	if _, err := failingVault.Get(context.Background(), "project", "NEW"); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("failed Put changed memory: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("failed Put changed the vault file")
	}
	_ = storage
}

func TestFilePermissionsAndApplicationIntegration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not portable to Windows")
	}
	storage, path := newTestStorage(t, &counterReader{})
	password := []byte("permission-test")
	if err := storage.Initialize(password); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("vault permissions = %o, want 600", got)
	}
	parentInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if got := parentInfo.Mode().Perm(); got != 0700 {
		t.Fatalf("new vault directory permissions = %o, want 700", got)
	}
	service := application.NewService(application.Dependencies{Vault: storage})
	opened, err := service.OpenVault(context.Background(), password)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := opened.ListSecrets(context.Background(), "project"); err != nil {
		t.Fatal(err)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestContextCancellationAndClosedVault(t *testing.T) {
	_, opened, _, _ := initializeTestVault(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := opened.List(ctx, "project"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled List error = %v", err)
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := opened.Get(context.Background(), "project", "TOKEN"); !errors.Is(err, ErrVaultClosed) {
		t.Fatalf("closed Get error = %v", err)
	}
}

func FuzzParseHeader(f *testing.F) {
	header := vaultHeader{version: formatVersion, kdf: kdfPBKDF2SHA256, iterations: 100000, salt: make([]byte, 32), nonce: make([]byte, 12)}
	f.Add(header.marshal())
	f.Add([]byte("MFVAUL\x01\x01\x00\x01\x86\xa0\x00\x20\x0c"))
	f.Add([]byte("random corrupt data"))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _, _ = parseHeader(data)
	})
}
