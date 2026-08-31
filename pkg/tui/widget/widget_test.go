package widget

import (
	"testing"

	"mayfly/pkg/tui/terminal"
)

func TestTextInputHandling(t *testing.T) {
	input := NewTextInput("Prompt: ", "enter text", false)
	input.Focused = true
	input.SetValue("default")

	if input.Value != "default" {
		t.Errorf("expected default value, got %q", input.Value)
	}

	// Test backspace
	input.HandleKey(terminal.KeyEvent{Type: terminal.KeyBackspace})
	if input.Value != "defaul" {
		t.Errorf("expected 'defaul' after backspace, got %q", input.Value)
	}

	// Test typing character
	input.HandleKey(terminal.KeyEvent{Type: terminal.KeyRune, Rune: 't'})
	if input.Value != "default" {
		t.Errorf("expected 'default' after typing 't', got %q", input.Value)
	}

	// Test home and cursor move
	input.HandleKey(terminal.KeyEvent{Type: terminal.KeyHome})
	input.HandleKey(terminal.KeyEvent{Type: terminal.KeyRight})

	// Test clear
	input.Clear()
	if input.Value != "" {
		t.Errorf("expected empty value after Clear(), got %q", input.Value)
	}

	// Test SetValue
	input.SetValue("hello")
	if input.Value != "hello" {
		t.Errorf("expected 'hello' after SetValue, got %q", input.Value)
	}
}

func TestConfirmDialogToggle(t *testing.T) {
	dialog := NewConfirmDialog("Confirm Delete", "Are you sure?")
	dialog.Active = true

	if dialog.Confirm {
		t.Errorf("expected initial confirm state to be false")
	}

	// Toggle with Tab
	handled := dialog.HandleKey(terminal.KeyEvent{Type: terminal.KeyTab})
	if !handled || !dialog.Confirm {
		t.Errorf("expected tab to toggle confirm to true")
	}

	// Toggle with Left/Right
	dialog.HandleKey(terminal.KeyEvent{Type: terminal.KeyRight})
	if dialog.Confirm {
		t.Errorf("expected right arrow to toggle confirm back to false")
	}
}

func TestStatusBarDraw(t *testing.T) {
	sb := NewStatusBar("Left", "Right")
	frame := terminal.NewFrame(terminal.Size{Rows: 5, Columns: 40})
	rect := terminal.NewRect(4, 0, 1, 40)
	sb.Draw(frame, rect)

	// Test error prominence
	sb.RightText = "Error: something failed"
	sb.Draw(frame, rect)
}
