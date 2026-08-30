package terminal

import (
	"encoding/base64"
	"fmt"
	"io"
	"os/exec"
	"runtime"
)

// CopyToClipboard copies a text string to the system clipboard using OSC 52 and platform utilities.
func CopyToClipboard(text string, out io.Writer) error {
	if text == "" {
		return nil
	}

	// 1. ANSI OSC 52 (Works in iTerm2, Alacritty, Kitty, Windows Terminal, WezTerm, VS Code)
	if out != nil {
		encoded := base64.StdEncoding.EncodeToString([]byte(text))
		osc52 := fmt.Sprintf("\x1b]52;c;%s\x07", encoded)
		_, _ = io.WriteString(out, osc52)
	}

	// 2. Native OS clipboard tool fallback
	switch runtime.GOOS {
	case "darwin":
		if cmdPath, err := exec.LookPath("pbcopy"); err == nil {
			cmd := exec.Command(cmdPath)
			in, _ := cmd.StdinPipe()
			if err := cmd.Start(); err == nil {
				_, _ = io.WriteString(in, text)
				_ = in.Close()
				_ = cmd.Wait()
			}
		}
	case "windows":
		if cmdPath, err := exec.LookPath("clip"); err == nil {
			cmd := exec.Command(cmdPath)
			in, _ := cmd.StdinPipe()
			if err := cmd.Start(); err == nil {
				_, _ = io.WriteString(in, text)
				_ = in.Close()
				_ = cmd.Wait()
			}
		}
	case "linux":
		// Try wl-copy (Wayland) or xclip / xsel (X11)
		for _, tool := range []string{"wl-copy", "xclip", "xsel"} {
			if cmdPath, err := exec.LookPath(tool); err == nil {
				var cmd *exec.Cmd
				if tool == "xclip" {
					cmd = exec.Command(cmdPath, "-selection", "clipboard")
				} else if tool == "xsel" {
					cmd = exec.Command(cmdPath, "--clipboard", "--input")
				} else {
					cmd = exec.Command(cmdPath)
				}
				in, _ := cmd.StdinPipe()
				if err := cmd.Start(); err == nil {
					_, _ = io.WriteString(in, text)
					_ = in.Close()
					_ = cmd.Wait()
					break
				}
			}
		}
	}

	return nil
}
