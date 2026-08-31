package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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

// PerformUpdate downloads and executes the latest installation script or replaces the binary.
func PerformUpdate(ctx context.Context) error {
	if runtime.GOOS == "windows" {
		cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", "irm https://raw.githubusercontent.com/vishnunandan555/mayfly/main/install.ps1 | iex")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Unix / macOS: Run the official installer updater script
	cmd := exec.CommandContext(ctx, "bash", "-c", "curl -fsSL https://raw.githubusercontent.com/vishnunandan555/mayfly/main/install.sh | bash -s -- --update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
