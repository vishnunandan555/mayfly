package tui

import (
	"strings"
	"testing"

	"mayfly/pkg/tui/terminal"
)

func TestDrawSecretViewerFrame(t *testing.T) {
	sz := terminal.Size{Rows: 24, Columns: 80}
	frame := terminal.NewFrame(sz)

	DrawSecretViewerFrame(frame, sz, "GitChat", "GITHUB_TOKEN", "ghp_mocktoken12345", "")

	// Check if secret key and project are present in rendered text
	rendered := frame.String()
	if !strings.Contains(rendered, "MAYFLY SECRET VIEWER") {
		t.Errorf("expected frame to contain title")
	}
	if !strings.Contains(rendered, "GitChat") {
		t.Errorf("expected frame to contain project name 'GitChat'")
	}
	if !strings.Contains(rendered, "GITHUB_TOKEN") {
		t.Errorf("expected frame to contain secret name 'GITHUB_TOKEN'")
	}
	if !strings.Contains(rendered, "ghp_mocktoken12345") {
		t.Errorf("expected frame to contain secret value 'ghp_mocktoken12345'")
	}
	if !strings.Contains(rendered, "[C]") || !strings.Contains(rendered, "[Q / Esc / Enter]") {
		t.Errorf("expected frame to contain action helpers")
	}

	// Test status badge rendering
	frameWithStatus := terminal.NewFrame(sz)
	DrawSecretViewerFrame(frameWithStatus, sz, "GitChat", "GITHUB_TOKEN", "ghp_mocktoken12345", "✓ Copied to clipboard!")
	if !strings.Contains(frameWithStatus.String(), "✓ Copied to clipboard!") {
		t.Errorf("expected frame to contain status badge")
	}
}
