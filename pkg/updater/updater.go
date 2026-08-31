package updater

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"mayfly/pkg/domain"
)

// ReleaseAsset represents an attached binary or checksum asset in a GitHub release.
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// ReleaseInfo holds the metadata of a remote GitHub release.
type ReleaseInfo struct {
	TagName     string         `json:"tag_name"`
	Name        string         `json:"name"`
	Body        string         `json:"body"`
	PublishedAt string         `json:"published_at"`
	HTMLURL     string         `json:"html_url"`
	Assets      []ReleaseAsset `json:"assets"`
}

// SemVer represents a parsed semantic version (major.minor.patch).
type SemVer struct {
	Major int
	Minor int
	Patch int
}

// ParseSemVer parses version strings like "0.0.1", "v0.0.2", "1.2.3-alpha" into numeric components.
func ParseSemVer(v string) (SemVer, error) {
	clean := strings.TrimPrefix(strings.TrimSpace(v), "v")
	if idx := strings.IndexAny(clean, "-+"); idx != -1 {
		clean = clean[:idx]
	}

	parts := strings.Split(clean, ".")
	if len(parts) < 1 {
		return SemVer{}, fmt.Errorf("invalid version string: %q", v)
	}

	var sv SemVer
	var err error

	if len(parts) >= 1 {
		sv.Major, err = strconv.Atoi(parts[0])
		if err != nil {
			return sv, err
		}
	}
	if len(parts) >= 2 {
		sv.Minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return sv, err
		}
	}
	if len(parts) >= 3 {
		sv.Patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return sv, err
		}
	}

	return sv, nil
}

// Compare returns:
//
//	-1 if v < other
//	 0 if v == other
//	 1 if v > other
func (v SemVer) Compare(other SemVer) int {
	if v.Major != other.Major {
		if v.Major > other.Major {
			return 1
		}
		return -1
	}
	if v.Minor != other.Minor {
		if v.Minor > other.Minor {
			return 1
		}
		return -1
	}
	if v.Patch != other.Patch {
		if v.Patch > other.Patch {
			return 1
		}
		return -1
	}
	return 0
}

// CheckForUpdates queries the GitHub API for the latest release and determines if a newer version exists.
func CheckForUpdates(ctx context.Context, customEndpoint string) (ReleaseInfo, bool, error) {
	url := "https://api.github.com/repos/vishnunandan555/mayfly/releases/latest"
	if customEndpoint != "" {
		url = customEndpoint
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ReleaseInfo{}, false, err
	}

	req.Header.Set("User-Agent", "mayfly-cli/v"+domain.Version)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{
		Timeout: 6 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return ReleaseInfo{}, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ReleaseInfo{}, false, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ReleaseInfo{}, false, err
	}

	var rel ReleaseInfo
	if err := json.Unmarshal(body, &rel); err != nil {
		return ReleaseInfo{}, false, err
	}

	remoteVer, rErr := ParseSemVer(rel.TagName)
	if rErr != nil {
		return rel, false, nil
	}

	currVer, cErr := ParseSemVer(domain.Version)
	if cErr != nil {
		return rel, false, nil
	}

	isNewer := remoteVer.Compare(currVer) > 0
	return rel, isNewer, nil
}

// TargetBinaryName returns the asset filename for the current OS and architecture.
func TargetBinaryName() string {
	arch := runtime.GOARCH
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("mayfly-%s-%s%s", runtime.GOOS, arch, ext)
}

// PerformDirectUpdate downloads the release binary directly via Go's built-in net/http client,
// cryptographically verifies the SHA-256 hash against checksums.txt, and replaces the current
// executable in-place with zero external shell dependencies.
func PerformDirectUpdate(ctx context.Context, tag string, baseURL string) error {
	if tag == "" {
		tag = "latest"
	}
	if baseURL == "" {
		if tag == "latest" {
			baseURL = "https://github.com/vishnunandan555/mayfly/releases/latest/download"
		} else {
			baseURL = "https://github.com/vishnunandan555/mayfly/releases/download/" + tag
		}
	}

	targetBinary := TargetBinaryName()
	binaryURL := fmt.Sprintf("%s/%s", baseURL, targetBinary)
	checksumURL := fmt.Sprintf("%s/checksums.txt", baseURL)

	client := &http.Client{Timeout: 60 * time.Second}

	// 1. Fetch official checksums.txt manifest
	reqCheck, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create checksum request: %w", err)
	}
	respCheck, err := client.Do(reqCheck)
	if err != nil {
		return fmt.Errorf("failed to fetch checksums.txt: %w", err)
	}
	defer respCheck.Body.Close()

	if respCheck.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download checksums manifest (HTTP %d)", respCheck.StatusCode)
	}

	checksumBytes, err := io.ReadAll(respCheck.Body)
	if err != nil {
		return fmt.Errorf("failed to read checksums manifest: %w", err)
	}

	expectedHash := ParseExpectedHash(string(checksumBytes), targetBinary)
	if expectedHash == "" {
		return fmt.Errorf("no matching SHA-256 entry found for %s in checksums.txt", targetBinary)
	}

	// 2. Download binary bytes
	reqBin, err := http.NewRequestWithContext(ctx, http.MethodGet, binaryURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create binary download request: %w", err)
	}
	respBin, err := client.Do(reqBin)
	if err != nil {
		return fmt.Errorf("failed to download release binary: %w", err)
	}
	defer respBin.Body.Close()

	if respBin.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s (HTTP %d)", targetBinary, respBin.StatusCode)
	}

	binData, err := io.ReadAll(respBin.Body)
	if err != nil {
		return fmt.Errorf("failed to read downloaded binary data: %w", err)
	}

	// 3. Cryptographic SHA-256 bit-for-bit verification
	computedHash := fmt.Sprintf("%x", sha256.Sum256(binData))
	if !strings.EqualFold(computedHash, expectedHash) {
		return fmt.Errorf("security alert: SHA-256 checksum mismatch (expected %s, got %s)", expectedHash, computedHash)
	}

	// 4. In-place atomic binary replacement
	execPath := GetExecutableLocation()
	execDir := filepath.Dir(execPath)

	tmpFile := filepath.Join(execDir, fmt.Sprintf(".mayfly-update-%d.tmp", time.Now().UnixNano()))
	if err := os.WriteFile(tmpFile, binData, 0755); err != nil {
		tmpFile = filepath.Join(os.TempDir(), fmt.Sprintf(".mayfly-update-%d.tmp", time.Now().UnixNano()))
		if wErr := os.WriteFile(tmpFile, binData, 0755); wErr != nil {
			return fmt.Errorf("failed to write temporary updated binary: %w", err)
		}
	}
	defer os.Remove(tmpFile)

	if runtime.GOOS == "windows" {
		oldExec := execPath + ".old"
		_ = os.Remove(oldExec)
		_ = os.Rename(execPath, oldExec)
	}

	if err := os.Rename(tmpFile, execPath); err != nil {
		if err := copyExecutable(tmpFile, execPath); err != nil {
			return fmt.Errorf("failed to replace binary at %s: %w", execPath, err)
		}
	}

	return nil
}

// ParseExpectedHash extracts the SHA-256 hash for a specific binary from a checksums.txt manifest.
func ParseExpectedHash(checksumContent, targetBinary string) string {
	scanner := bufio.NewScanner(strings.NewReader(checksumContent))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			filename := filepath.Base(parts[1])
			if filename == targetBinary {
				return parts[0]
			}
		}
	}
	return ""
}

func copyExecutable(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0755)
}

// PerformUpdate downloads and applies the latest verified release using pure standard library networking.
func PerformUpdate(ctx context.Context) error {
	return PerformDirectUpdate(ctx, "latest", "")
}

// GetExecutableLocation returns the active binary path on the filesystem.
func GetExecutableLocation() string {
	execPath, err := os.Executable()
	if err != nil {
		return "mayfly"
	}
	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return execPath
	}
	return realPath
}
