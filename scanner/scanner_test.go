package scanner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mayfly/domain"
)

func projectFor(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func scanProject(t *testing.T, root string, options Options) []domain.ScanFinding {
	t.Helper()
	scan, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	findings, err := scan.Scan(context.Background(), domain.Project{ID: "project-1", Name: "Demo", Path: root})
	if err != nil {
		t.Fatal(err)
	}
	return findings
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func categories(findings []domain.ScanFinding) map[string]domain.ScanFinding {
	result := make(map[string]domain.ScanFinding, len(findings))
	for _, finding := range findings {
		result[finding.Category+":"+finding.Path] = finding
	}
	return result
}

func TestScanFlagsHighRiskFilenamesAndSkipsGeneratedDirectories(t *testing.T) {
	root := projectFor(t)
	writeFile(t, filepath.Join(root, ".env"), "TOKEN=secret-value")
	writeFile(t, filepath.Join(root, ".env.local"), "TOKEN=secret-value")
	writeFile(t, filepath.Join(root, "credentials.json"), "{}")
	writeFile(t, filepath.Join(root, "private.pem"), "not-a-real-key")
	writeFile(t, filepath.Join(root, "build", ".env"), "TOKEN=secret-value")
	writeFile(t, filepath.Join(root, ".git", "config"), "password=secret-value")
	findings := scanProject(t, root, Options{})
	got := categories(findings)
	for _, expected := range []struct {
		category string
		path     string
	}{
		{"high-risk-filename", ".env"},
		{"high-risk-filename", ".env.local"},
		{"high-risk-filename", "credentials.json"},
		{"private-key-filename", "private.pem"},
	} {
		if _, ok := got[expected.category+":"+expected.path]; !ok {
			t.Fatalf("missing finding %s: %#v", expected.category, findings)
		}
	}
	for _, finding := range findings {
		if strings.HasPrefix(finding.Path, "build/") || strings.HasPrefix(finding.Path, ".git/") {
			t.Fatalf("ignored directory was scanned: %#v", finding)
		}
	}
}

func TestScanContentPatternsAndUnicodeLocations(t *testing.T) {
	root := projectFor(t)
	writeFile(t, filepath.Join(root, "src", "config.go"), "package main\n\néé password = \"super-secret\"\nTOKEN = 'token-value'\nAKIA1234567890ABCDEF\n-----BEGIN RSA PRIVATE KEY-----\n")
	findings := scanProject(t, root, Options{})
	var password, token, apiKey, privateKey *domain.ScanFinding
	for index := range findings {
		finding := &findings[index]
		switch finding.Category {
		case "password-assignment":
			password = finding
		case "token-assignment":
			token = finding
		case "api-key-pattern":
			apiKey = finding
		case "private-key-marker":
			privateKey = finding
		}
		if strings.Contains(finding.Message, "super-secret") || strings.Contains(finding.Message, "token-value") {
			t.Fatalf("finding exposed content: %#v", finding)
		}
		if err := finding.Validate(); err != nil {
			t.Fatalf("invalid finding: %v", err)
		}
	}
	if password == nil || password.Line != 3 || password.Column != 4 || password.Severity != domain.SeverityWarning {
		t.Fatalf("password finding = %#v", password)
	}
	if token == nil || token.Line != 4 {
		t.Fatalf("token finding = %#v", token)
	}
	if apiKey == nil || apiKey.Line != 5 || apiKey.Severity != domain.SeverityCritical {
		t.Fatalf("API key finding = %#v", apiKey)
	}
	if privateKey == nil || privateKey.Line != 6 || privateKey.Severity != domain.SeverityCritical {
		t.Fatalf("private key finding = %#v", privateKey)
	}
}

func TestScanFalsePositiveBinaryMalformedAndEmptyFiles(t *testing.T) {
	root := projectFor(t)
	writeFile(t, filepath.Join(root, "documentation.txt"), "The password field is described here; no assignment is present.\n")
	if err := os.WriteFile(filepath.Join(root, "binary.dat"), []byte("prefix\x00password=secret-value"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "malformed.txt"), []byte{0xff, 0xfe, 0x00}, 0600); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "empty.txt"), "")
	findings := scanProject(t, root, Options{})
	for _, finding := range findings {
		if finding.Path == "documentation.txt" || finding.Path == "binary.dat" || finding.Path == "malformed.txt" || finding.Path == "empty.txt" {
			t.Fatalf("false positive or binary scan: %#v", finding)
		}
	}
}

func TestScanSkipsOversizedContentButStillFlagsFilename(t *testing.T) {
	root := projectFor(t)
	writeFile(t, filepath.Join(root, "large.txt"), strings.Repeat("x", 64)+"password=secret-value")
	writeFile(t, filepath.Join(root, ".env.large"), strings.Repeat("TOKEN=secret-value\n", 64))
	findings := scanProject(t, root, Options{MaxFileSize: 64})
	for _, finding := range findings {
		if finding.Path == "large.txt" {
			t.Fatalf("oversized ordinary content was scanned: %#v", finding)
		}
	}
	if _, ok := categories(findings)["high-risk-filename:.env.large"]; !ok {
		t.Fatalf("oversized high-risk filename was not reported: %#v", findings)
	}
}

func TestScanUnicodePathAndDeterministicOrdering(t *testing.T) {
	root := projectFor(t)
	writeFile(t, filepath.Join(root, "é", "秘密.txt"), "password = value\n")
	writeFile(t, filepath.Join(root, "a.txt"), "token = value\n")
	first := scanProject(t, root, Options{})
	second := scanProject(t, root, Options{})
	if len(first) != 2 || !equalFindings(first, second) {
		t.Fatalf("non-deterministic findings: %#v / %#v", first, second)
	}
	if first[0].Path != "a.txt" || first[1].Path != "é/秘密.txt" {
		t.Fatalf("finding order = %#v", first)
	}
}

func TestScanCustomSkipPathAndCancellation(t *testing.T) {
	root := projectFor(t)
	skip := filepath.Join(root, "generated")
	writeFile(t, filepath.Join(skip, "config.txt"), "password=secret-value")
	scanner, err := New(Options{SkipPaths: []string{skip}})
	if err != nil {
		t.Fatal(err)
	}
	findings, err := scanner.Scan(context.Background(), domain.Project{ID: "project-1", Name: "Demo", Path: root})
	if err != nil || len(findings) != 0 {
		t.Fatalf("custom skip findings = %#v, %v", findings, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := scanner.Scan(ctx, domain.Project{ID: "project-1", Name: "Demo", Path: root}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled scan error = %v", err)
	}
}

func TestScanRejectsInvalidOptionsAndProject(t *testing.T) {
	if _, err := New(Options{MaxFileSize: -1}); err == nil {
		t.Fatal("negative max size was accepted")
	}
	scanner, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scanner.Scan(context.Background(), domain.Project{ID: "project-1", Name: "Demo", Path: filepath.Join(t.TempDir(), "missing")}); !errors.Is(err, ErrInvalidProject) {
		t.Fatalf("invalid project error = %v", err)
	}
}

func equalFindings(left, right []domain.ScanFinding) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
