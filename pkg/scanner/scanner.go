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

	ignorePatterns := loadIgnorePatterns(filepath.Join(absRoot, ".mayflyignore"))

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
			if defaultIgnoredDirs[base] || s.skipPaths[path] || isIgnored(relPath, base, true, ignorePatterns) {
				return filepath.SkipDir
			}
			return nil
		}

		if s.skipPaths[path] || isIgnored(relPath, base, false, ignorePatterns) {
			return nil
		}

// Check dangerous file names (.env, credentials, private keys), excluding template/example files
		if isDangerousConfigFile(base) {
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

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			continue // Skip pure comments and empty lines
		}

		for _, p := range tokenPatterns {
			loc := p.regex.FindStringIndex(line)
			if loc != nil {
				matchedStr := line[loc[0]:loc[1]]

				// For token-assignment heuristics, filter out safe placeholders / localhost URLs:
				if p.category == "token-assignment" {
					parts := strings.SplitN(matchedStr, "=", 2)
					if len(parts) < 2 {
						parts = strings.SplitN(matchedStr, ":", 2)
					}
					if len(parts) == 2 && isPlaceholderValue(parts[1]) {
						continue
					}
				}

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

// isTemplateFile identifies files intended as documentation templates or example schemas.
func isTemplateFile(filename string) bool {
	lower := strings.ToLower(filename)
	exts := []string{".example", ".sample", ".template", ".dist", ".default", ".tmpl"}
	for _, ext := range exts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// isDangerousConfigFile checks if a file name represents an active plaintext secret or key file.
func isDangerousConfigFile(filename string) bool {
	if isTemplateFile(filename) {
		return false
	}
	lower := strings.ToLower(filename)
	if strings.HasPrefix(lower, ".env") ||
		lower == "credentials.json" ||
		lower == "id_rsa" ||
		lower == "id_ed25519" ||
		lower == "id_ecdsa" ||
		lower == "id_dsa" ||
		lower == "service-account.json" ||
		lower == "service_account.json" {
		return true
	}
	return false
}

// isPlaceholderValue checks if an assigned value is a harmless placeholder, local URL, or dummy string.
func isPlaceholderValue(val string) bool {
	v := strings.Trim(strings.TrimSpace(val), "\"'`")
	if v == "" {
		return true
	}
	lower := strings.ToLower(v)

	// Localhost / development loopback URLs
	if strings.HasPrefix(lower, "http://localhost") ||
		strings.HasPrefix(lower, "https://localhost") ||
		strings.HasPrefix(lower, "http://127.0.0.1") ||
		strings.HasPrefix(lower, "https://127.0.0.1") ||
		strings.HasPrefix(lower, "localhost:") ||
		strings.HasPrefix(lower, "127.0.0.1:") ||
		strings.HasPrefix(lower, "http://0.0.0.0") ||
		strings.HasPrefix(lower, "0.0.0.0:") {
		return true
	}

	// Placeholder templates like <YOUR_KEY>, ${KEY}, {{KEY}}, %KEY%, $KEY
	if (strings.HasPrefix(v, "<") && strings.HasSuffix(v, ">")) ||
		(strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}")) ||
		(strings.HasPrefix(v, "{{") && strings.HasSuffix(v, "}}")) ||
		(strings.HasPrefix(v, "%") && strings.HasSuffix(v, "%")) ||
		strings.HasPrefix(v, "$") {
		return true
	}

	// Explicit dummy placeholder values
	exactDummies := map[string]bool{
		"your-api-key": true, "your_api_key": true, "your-secret": true, "your_secret": true,
		"your-token": true, "your_token": true, "your-password": true, "your_password": true,
		"your-key-here": true, "your_key_here": true, "your-api-key-here": true,
		"changeme": true, "change_me": true, "change-me": true, "replace_me": true, "replace-me": true,
		"dummy": true, "placeholder": true, "example": true, "xxx": true, "xxxx": true,
		"none": true, "null": true, "undefined": true, "true": true, "false": true, "0": true,
	}
	if exactDummies[lower] {
		return true
	}

	if strings.HasPrefix(lower, "your-") || strings.HasPrefix(lower, "your_") ||
		strings.HasPrefix(lower, "replace-") || strings.HasPrefix(lower, "replace_") ||
		strings.HasPrefix(lower, "insert-") || strings.HasPrefix(lower, "insert_") {
		return true
	}

	return false
}

func errorsIsDone(err error) bool {
	return err == context.Canceled || err == context.DeadlineExceeded
}

func loadIgnorePatterns(ignorePath string) []string {
	data, err := os.ReadFile(ignorePath)
	if err != nil {
		return nil
	}

	var patterns []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

func isIgnored(relPath, base string, isDir bool, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}

	cleanRel := filepath.ToSlash(relPath)
	cleanBase := base

	for _, pat := range patterns {
		pat = filepath.ToSlash(strings.TrimSpace(pat))
		if pat == "" {
			continue
		}

		// Directory-specific pattern (ends in /)
		if strings.HasSuffix(pat, "/") {
			dirPat := strings.TrimSuffix(pat, "/")
			if isDir && (cleanBase == dirPat || cleanRel == dirPat || strings.HasPrefix(cleanRel, dirPat+"/")) {
				return true
			}
			continue
		}

		// Direct name match
		if cleanBase == pat || cleanRel == pat {
			return true
		}

		// Path prefix match
		if strings.HasPrefix(cleanRel, pat+"/") {
			return true
		}

		// Glob pattern match against base name
		if matched, _ := filepath.Match(pat, cleanBase); matched {
			return true
		}

		// Glob pattern match against relative path
		if matched, _ := filepath.Match(pat, cleanRel); matched {
			return true
		}
	}

	return false
}

