package tui

import (
	"os"
	"os/signal"

	"mayfly/pkg/application"
	"mayfly/pkg/domain"
	"mayfly/pkg/tui/terminal"
	"mayfly/pkg/tui/views"
)

// Options specifies runtime settings for the TUI session.
type Options struct {
	ProjectScoped *domain.Project
	CurrentDir    string
}

// Run launches the interactive full-screen Terminal UI.
func Run(svc *application.Service, opts Options) error {
	rawState, err := terminal.EnableRaw(os.Stdin)
	if err != nil {
		return err
	}
	defer func() {
		_ = terminal.Restore(os.Stdin, rawState)
	}()

	sz, err := terminal.GetSize(os.Stdout)
	if err != nil {
		sz = terminal.Size{Rows: 24, Columns: 80}
	}

	term := terminal.NewTerminal(os.Stdout, sz)
	term.EnterAltScreen()
	defer term.ExitAltScreen()

	dir := opts.CurrentDir
	if dir == "" {
		dir, _ = os.Getwd()
	}

	screens := views.NewScreens(svc, dir)
	if opts.ProjectScoped != nil {
		screens.SetProjectScoped(*opts.ProjectScoped)
	}

	frame := terminal.NewFrame(sz)
	screens.Draw(frame)
	_ = term.Render(frame)

	// Listen for window resize signals
	sigCh := make(chan os.Signal, 1)
	terminal.NotifyResize(sigCh)
	defer signal.Stop(sigCh)

	// Key reader
	keyCh := make(chan []terminal.KeyEvent, 16)
	errCh := make(chan error, 1)

	go func() {
		buf := make([]byte, 128)
		parser := terminal.NewParser()
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				errCh <- err
				return
			}
			if n > 0 {
				events := parser.Feed(buf[:n])
				if len(events) > 0 {
					keyCh <- events
				}
			}
		}
	}()

	for {
		select {
		case <-sigCh:
			newSz, err := terminal.GetSize(os.Stdout)
			if err == nil {
				sz = newSz
				frame = terminal.NewFrame(sz)
				screens.Draw(frame)
				_ = term.Render(frame)
			}

		case events := <-keyCh:
			quit := false
			for _, ev := range events {
				if screens.HandleKey(ev) {
					quit = true
					break
				}
			}
			if quit {
				return nil
			}
			screens.Draw(frame)
			_ = term.Render(frame)

		case err := <-errCh:
			return err
		}
	}
}
