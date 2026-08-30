package scanner

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"mayfly/pkg/domain"
)

const MaxFileSizeBytes = 1024 * 1024 // 1 MiB boundary

var defaultIgnoredDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	".next":        true,
	".nuxt":        true,
	"dist":         true,
	"build":        true,
	"vendor":       true,
	"target":       true,
	"bin":          true,
	"obj":          true,
	".turbo":       true,
	".cache":       true,
	".venv":        true,
	"venv":         true,
	"__pycache__":  true,
	".mayfly":      true,
}

var tokenPatterns = []struct {
	category string
	severity domain.ScanSeverity
	message  string
	regex    *regexp.Regexp
}{
	{
		category: "github-token",
		severity: domain.SeverityCritical,
		message:  "GitHub Personal Access / Secret Token detected",
		regex:    regexp.MustCompile(`\b(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{36,255}\b`),
	},
	{
		category: "stripe-key",
		severity: domain.SeverityCritical,
		message:  "Stripe Secret / Restricted Key detected",
		regex:    regexp.MustCompile(`\b(sk_live|rk_live)_[0-9a-zA-Z]{24,99}\b`),
	},
	{
		category: "google-api-key",
		severity: domain.SeverityCritical,
		message:  "Google / Gemini API Key detected",
		regex:    regexp.MustCompile(`\bAIzaSy[0-9A-Za-z\-_]{33}\b`),
	},
	{
		category: "aws-key",
		severity: domain.SeverityCritical,
		message:  "AWS Access Key ID detected",
		regex:    regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
	},
	{
		category: "private-key",
		severity: domain.SeverityCritical,
		message:  "Private Encryption Key Block detected",
		regex:    regexp.MustCompile(`-----BEGIN (RSA|EC|DSA|OPENSSH|PRIVATE) KEY-----`),
	},
	{
		category: "token-assignment",
		severity: domain.SeverityWarning,
		message:  "Hardcoded credential / secret variable assignment detected",
		regex:    regexp.MustCompile(`(?i)(api[_-]?key|secret[_-]?key|auth[_-]?token|db[_-]?pass|password)\s*[:=]\s*["'][^"'\s]{8,}["']`),
	},
}

type Scanner struct {
	skipPaths map[string]bool
}

type Options struct {
	SkipPaths []string
}

func New(opts Options) (*Scanner, error) {
	skip := make(map[string]bool)
	for _, p := range opts.SkipPaths {
		if abs, err := filepath.Abs(p); err == nil {
			skip[abs] = true
		}
	}
	return &Scanner{skipPaths: skip}, nil
}

func (s *Scanner) Scan(ctx context.Context, rootDir string) ([]domain.ScanFinding, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	var findings []domain.ScanFinding

	err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		relPath, _ := filepath.Rel(absRoot, path)
		base := d.Name()

		if d.IsDir() {
			if defaultIgnoredDirs[base] || s.skipPaths[path] {
				return filepath.SkipDir
			}
			return nil
		}

		if s.skipPaths[path] {
			return nil
		}

		// Check dangerous file names (.env, credentials)
		if strings.HasPrefix(base, ".env") || base == "credentials.json" || base == "id_rsa" || base == "id_ed25519" {
			findings = append(findings, domain.ScanFinding{
				Path:     relPath,
				Line:     1,
				Column:   1,
				Severity: domain.SeverityCritical,
				Category: "plaintext-env-file",
				Message:  "Plaintext secrets configuration file found on disk",
			})
		}

		// Inspect file content for token patterns
		info, err := d.Info()
		if err != nil || info.Size() > MaxFileSizeBytes || info.Size() == 0 {
			return nil
		}

		fileFindings, err := s.scanFile(path, relPath)
		if err == nil && len(fileFindings) > 0 {
			findings = append(findings, fileFindings...)
		}

		return nil
	})

	if err != nil && !errorsIsDone(err) {
		return nil, err
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path == findings[j].Path {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].Path < findings[j].Path
	})

	return findings, nil
}

func (s *Scanner) scanFile(fullPath, relPath string) ([]domain.ScanFinding, error) {
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var findings []domain.ScanFinding
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if !utf8.ValidString(line) {
			return nil, nil // Skip binary files
		}

		for _, p := range tokenPatterns {
			loc := p.regex.FindStringIndex(line)
			if loc != nil {
				findings = append(findings, domain.ScanFinding{
					Path:     relPath,
					Line:     lineNum,
					Column:   loc[0] + 1,
					Severity: p.severity,
					Category: p.category,
					Message:  p.message,
				})
			}
		}
	}

	return findings, scanner.Err()
}

func errorsIsDone(err error) bool {
	return err == context.Canceled || err == context.DeadlineExceeded
}
