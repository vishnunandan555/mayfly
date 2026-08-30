// Package scanner provides a conservative heuristic scanner for plaintext
// secrets in a project tree. It is a detector of likely exposures, not proof
// that a project is free of secrets.
package scanner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"mayfly/domain"
)

const (
	defaultMaxFileSize = 1 << 20
)

var (
	ErrInvalidProject = errors.New("scanner: invalid project")
	ErrWalkFailed     = errors.New("scanner: project walk failed")
	ErrReadFailed     = errors.New("scanner: file read failed")
)

// Options controls safe scanner limits and directories/files excluded from
// content inspection. SkipPaths are absolute paths; relative paths are
// rejected by New so caller mistakes cannot silently broaden the scan.
type Options struct {
	MaxFileSize int64
	SkipPaths   []string
}

// Scanner walks a project without following symlinks. It has no persistent
// state and is safe to reuse after construction.
type Scanner struct {
	maxFileSize int64
	skipPaths   []string
}

// New constructs a scanner without reading the filesystem.
func New(options Options) (*Scanner, error) {
	if options.MaxFileSize == 0 {
		options.MaxFileSize = defaultMaxFileSize
	}
	if options.MaxFileSize < 1 || options.MaxFileSize > 64<<20 {
		return nil, errors.New("scanner: invalid maximum file size")
	}
	skipPaths := make([]string, 0, len(options.SkipPaths))
	for _, path := range options.SkipPaths {
		if strings.TrimSpace(path) == "" || strings.ContainsRune(path, '\x00') {
			return nil, errors.New("scanner: invalid skip path")
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, errors.New("scanner: invalid skip path")
		}
		skipPaths = append(skipPaths, filepath.Clean(absolute))
	}
	sort.Strings(skipPaths)
	return &Scanner{maxFileSize: options.MaxFileSize, skipPaths: skipPaths}, nil
}

// Scan reports safe locations and explanations only. It never includes the
// matching text in a finding. Paths are project-relative and use slash
// separators; line and column are one-based Unicode scalar positions. A
// filename-only finding has line and column zero.
func (s *Scanner) Scan(ctx context.Context, project domain.Project) ([]domain.ScanFinding, error) {
	if s == nil {
		return nil, ErrInvalidProject
	}
	if err := project.Validate(); err != nil || strings.TrimSpace(project.Path) == "" {
		return nil, ErrInvalidProject
	}
	root, err := filepath.Abs(project.Path)
	if err != nil {
		return nil, ErrInvalidProject
	}
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, ErrInvalidProject
	}

	findings := make([]domain.ScanFinding, 0)
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("%w: %v", ErrWalkFailed, walkErr)
		}
		if err := contextError(ctx); err != nil {
			return err
		}
		if path != root && s.shouldSkip(path, entry.IsDir()) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path == root || entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		fileInfo, err := entry.Info()
		if err != nil {
			return fmt.Errorf("%w: %v", ErrReadFailed, err)
		}
		if !fileInfo.Mode().IsRegular() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("%w: relative path", ErrWalkFailed)
		}
		relative = filepath.ToSlash(relative)
		findings = append(findings, filenameFindings(relative)...)
		if fileInfo.Size() > s.maxFileSize {
			return nil
		}
		content, err := readTextCandidate(path, s.maxFileSize)
		if err != nil {
			if errors.Is(err, errBinaryOrInvalidText) || errors.Is(err, errTooLarge) {
				return nil
			}
			return fmt.Errorf("%w: %v", ErrReadFailed, err)
		}
		findings = append(findings, contentFindings(relative, content)...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(findings, func(i, j int) bool {
		left, right := findings[i], findings[j]
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.Line != right.Line {
			return left.Line < right.Line
		}
		if left.Column != right.Column {
			return left.Column < right.Column
		}
		return left.Category < right.Category
	})
	return findings, nil
}

var (
	errBinaryOrInvalidText = errors.New("scanner: binary or invalid text")
	errTooLarge            = errors.New("scanner: file too large")
)

func readTextCandidate(path string, limit int64) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return "", err
	}
	if int64(len(data)) > limit {
		return "", errTooLarge
	}
	if bytesLookBinary(data) || !utf8.Valid(data) {
		return "", errBinaryOrInvalidText
	}
	return string(data), nil
}

func bytesLookBinary(data []byte) bool {
	for _, value := range data {
		if value == 0 {
			return true
		}
	}
	return false
}

func (s *Scanner) shouldSkip(path string, directory bool) bool {
	base := filepath.Base(path)
	if directory && isGeneratedDirectory(base) {
		return true
	}
	for _, skip := range s.skipPaths {
		if path == skip || pathWithin(skip, path) {
			return true
		}
	}
	return false
}

func pathWithin(parent, path string) bool {
	relative, err := filepath.Rel(parent, path)
	return err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func isGeneratedDirectory(name string) bool {
	switch strings.ToLower(name) {
	case ".git", ".hg", ".svn", "node_modules", "vendor", "dist", "build", "target", ".next", ".cache", "coverage", "tmp", "out":
		return true
	default:
		return false
	}
}

func filenameFindings(path string) []domain.ScanFinding {
	base := strings.ToLower(filepath.Base(path))
	category, severity, explanation := "", domain.SeverityInfo, ""
	switch {
	case base == ".env" || strings.HasPrefix(base, ".env."):
		category, severity, explanation = "high-risk-filename", domain.SeverityWarning, "environment-style file may contain plaintext secrets"
	case base == "credentials" || strings.HasPrefix(base, "credentials."):
		category, severity, explanation = "high-risk-filename", domain.SeverityWarning, "credentials-style file may contain plaintext secrets"
	case strings.HasSuffix(base, ".pem") || strings.HasSuffix(base, ".key") || base == "id_rsa" || base == "id_ed25519":
		category, severity, explanation = "private-key-filename", domain.SeverityCritical, "private-key-style filename requires review"
	case strings.HasSuffix(base, ".p12") || strings.HasSuffix(base, ".pfx"):
		category, severity, explanation = "credential-file-filename", domain.SeverityCritical, "credential-container filename requires review"
	}
	if category == "" {
		return nil
	}
	return []domain.ScanFinding{{Path: path, Severity: severity, Category: category, Message: explanation}}
}

type contentRule struct {
	pattern     *regexp.Regexp
	category    string
	severity    domain.ScanSeverity
	explanation string
}

var contentRules = []contentRule{
	{regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`), "private-key-marker", domain.SeverityCritical, "private key marker found"},
	{regexp.MustCompile(`(?i)\b(password|passwd|pwd)\b\s*[:=]\s*["']?[^\s"']+`), "password-assignment", domain.SeverityWarning, "password-like assignment found"},
	{regexp.MustCompile(`(?i)\b(token|api[_-]?key|secret|client[_-]?secret)\b\s*[:=]\s*["']?[^\s"']+`), "token-assignment", domain.SeverityWarning, "token-like assignment found"},
	{regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`), "api-key-pattern", domain.SeverityCritical, "cloud access-key-like pattern found"},
	{regexp.MustCompile(`\b(?:ghp|github_pat|xox[baprs])_[A-Za-z0-9_\-]{16,}\b`), "api-token-pattern", domain.SeverityCritical, "service-token-like pattern found"},
}

func contentFindings(path, content string) []domain.ScanFinding {
	findings := make([]domain.ScanFinding, 0)
	lineNumber := 0
	for _, line := range strings.Split(content, "\n") {
		lineNumber++
		for _, rule := range contentRules {
			for _, location := range rule.pattern.FindAllStringIndex(line, -1) {
				column := utf8.RuneCountInString(line[:location[0]]) + 1
				findings = append(findings, domain.ScanFinding{
					Path: path, Line: lineNumber, Column: column,
					Category: rule.category, Severity: rule.severity, Message: rule.explanation,
				})
			}
		}
	}
	return findings
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

var _ interface {
	Scan(context.Context, domain.Project) ([]domain.ScanFinding, error)
} = (*Scanner)(nil)
