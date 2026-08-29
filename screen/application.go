package screen

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
)

// ErrApplicationRunning indicates that Run was called while an application
// was already running. An Application is intentionally single-threaded.
var ErrApplicationRunning = errors.New("screen: application already running")

// ExitReason describes why an Application event loop stopped.
type ExitReason uint8

const (
	ExitNone ExitReason = iota
	ExitCtrlC
	ExitEscape
	ExitEOF
	ExitRequested
	ExitSignal
)

// WidgetPlacement assigns a widget a rectangle for one render pass. Entries
// are rendered in order, so later entries may intentionally overlay earlier
// entries.
type WidgetPlacement struct {
	Widget Widget
	Bounds Rect
}

// ApplicationOptions configures an Application. Output, Input, and Size are
// injectable, which makes the event loop testable without a physical TTY.
// Arrange is optional; when absent, each widget is assigned the full frame in
// slice order. A custom arrangement is the normal way to express screen
// layout without putting terminal control sequences in widgets.
type ApplicationOptions struct {
	Output io.Writer
	Input  Input
	Size   Size

	Widgets    []Widget
	FocusOrder []Widget
	Arrange    func(Size) []WidgetPlacement

	// GlobalHandler runs after built-in Ctrl-C and focus-navigation shortcuts,
	// and before the focused widget. Return true when the event was consumed.
	GlobalHandler func(*Application, Event) bool

	// StyleConfig is optional. A nil value uses NewTerminal's conservative
	// default, including NO_COLOR handling.
	StyleConfig *StyleConfig

	// SizeProvider is used after a supported resize signal. RunTerminal fills
	// this from TerminalSize when the caller leaves it nil.
	SizeProvider func() (Size, error)
}

// Application owns the logical frame, focus traversal, event dispatch, and
// terminal output for one TUI run. It does not acquire raw mode itself;
// RunTerminal composes it with RawInput, while NewApplication supports
// injected non-terminal inputs for tests and alternate front ends.
type Application struct {
	terminal *Terminal
	input    Input

	widgets    []Widget
	focusOrder []Widget
	arrange    func(Size) []WidgetPlacement
	global     func(*Application, Event) bool
	resize     func() (Size, error)

	size       Size
	lastFrame  *Frame
	focused    Widget
	exitReason ExitReason
	quit       bool
	running    bool
}

// NewApplication constructs an inert application. It does not clear a
// terminal, acquire raw mode, install signal handlers, or write output.
func NewApplication(options ApplicationOptions) *Application {
	config := DefaultStyleConfig()
	if options.StyleConfig != nil {
		config = *options.StyleConfig
	}
	terminal := NewTerminalWithConfig(options.Output, options.Size, config)
	widgets := append([]Widget(nil), options.Widgets...)
	focusOrder := append([]Widget(nil), options.FocusOrder...)
	if len(focusOrder) == 0 {
		focusOrder = append([]Widget(nil), widgets...)
	}

	app := &Application{
		terminal:   terminal,
		input:      options.Input,
		widgets:    widgets,
		focusOrder: focusOrder,
		arrange:    options.Arrange,
		global:     options.GlobalHandler,
		resize:     options.SizeProvider,
		size:       options.Size.normalized(),
	}
	app.terminal.SetViewport(app.size)
	app.ensureFocus()
	return app
}

// Run executes the application loop. Terminal setup is limited to the
// injected output and input supplied at construction; raw mode is acquired
// explicitly by RunTerminal. Every rendering/error path still performs the
// output cleanup sequence and flushes it before returning.
func (a *Application) Run() (err error) {
	if a == nil {
		return errors.New("screen: nil application")
	}
	if a.running {
		return ErrApplicationRunning
	}
	a.running = true
	defer func() {
		a.running = false
		cleanupErr := a.cleanup()
		err = errors.Join(err, cleanupErr)
	}()

	if a.terminal == nil {
		return errors.New("screen: application was not constructed with NewApplication")
	}
	if a.input == nil {
		return errors.New("screen: application input is nil")
	}
	if err := a.terminal.ClearScreen(); err != nil {
		return err
	}
	if err := a.render(); err != nil {
		return err
	}

	resizeEvents := make(chan os.Signal, 1)
	resize := resizeSignal()
	if resize != nil && a.resize != nil {
		signal.Notify(resizeEvents, resize)
		defer signal.Stop(resizeEvents)
	}

	for !a.quit {
		resized, err := a.pollResize(resizeEvents)
		if err != nil {
			return err
		}
		event, readErr := a.input.ReadEvent()
		if errors.Is(readErr, ErrInputTimeout) {
			if resized {
				if err := a.render(); err != nil {
					return err
				}
			}
			continue
		}
		if errors.Is(readErr, io.EOF) {
			a.exitReason = ExitEOF
			return nil
		}
		if errors.Is(readErr, ErrInputClosed) {
			a.exitReason = ExitSignal
			return nil
		}
		if readErr != nil {
			return readErr
		}

		resizedAfter, err := a.pollResize(resizeEvents)
		if err != nil {
			return err
		}
		if resizedAfter {
			resized = true
		}
		changed := a.Dispatch(event)
		if resized || changed {
			if err := a.render(); err != nil {
				return err
			}
		}
	}
	return nil
}

// RunTerminal discovers the terminal size, enters raw mode, runs an
// Application, and restores the original terminal state on every return path.
// It is Linux-only in this zero-dependency implementation because raw mode
// and resize ioctls are platform-specific; unsupported builds return the
// relevant error without invoking the application.
func RunTerminal(file *os.File, options ApplicationOptions) error {
	size, err := TerminalSize(file)
	if err != nil {
		return err
	}
	return RunRaw(file, func(input Input) error {
		options.Output = file
		options.Input = input
		options.Size = size
		if options.SizeProvider == nil {
			options.SizeProvider = func() (Size, error) { return TerminalSize(file) }
		}
		return NewApplication(options).Run()
	})
}

// Run is a convenience entry point for a full-screen application using the
// supplied widgets in their slice order.
func Run(file *os.File, widgets ...Widget) error {
	return RunTerminal(file, ApplicationOptions{Widgets: widgets})
}

// Dispatch applies global shortcuts and then offers the event to the focused
// widget. It reports whether the event changed or was consumed by application
// state. Escape exits only when the focused widget does not consume it.
func (a *Application) Dispatch(event Event) bool {
	if a == nil || a.quit {
		return false
	}
	switch event.Type {
	case EventCtrlC:
		a.RequestQuit(ExitCtrlC)
		return true
	case EventTab:
		return a.FocusNext()
	case EventShiftTab:
		return a.FocusPrevious()
	}
	if a.global != nil && a.global(a, event) {
		return true
	}
	if a.focused != nil && a.focused.Handle(event) {
		return true
	}
	if event.Type == EventEscape {
		a.RequestQuit(ExitEscape)
		return true
	}
	return false
}

// FocusNext moves to the next enabled, focusable widget, wrapping around.
func (a *Application) FocusNext() bool { return a.moveFocus(1) }

// FocusPrevious moves to the previous enabled, focusable widget, wrapping
// around.
func (a *Application) FocusPrevious() bool { return a.moveFocus(-1) }

// SetFocus selects the focusable widget at index in FocusOrder. It returns
// false for an invalid or currently disabled/non-focusable entry.
func (a *Application) SetFocus(index int) bool {
	if a == nil || index < 0 || index >= len(a.focusOrder) || !a.canFocus(a.focusOrder[index]) {
		return false
	}
	a.setFocused(a.focusOrder[index])
	return true
}

// Focused returns the current focused widget, or nil when no widget can take
// focus.
func (a *Application) Focused() Widget {
	if a == nil {
		return nil
	}
	return a.focused
}

// FocusIndex returns the explicit FocusOrder index of the focused widget, or
// -1 when focus is empty.
func (a *Application) FocusIndex() int {
	if a == nil || a.focused == nil {
		return -1
	}
	for index, widget := range a.focusOrder {
		if widget == a.focused {
			return index
		}
	}
	return -1
}

// Resize updates the logical frame size and terminal clipping viewport. It
// does not render immediately; Run renders the next frame after a resize.
func (a *Application) Resize(size Size) {
	if a == nil {
		return
	}
	a.size = size.normalized()
	a.terminal.SetViewport(a.size)
}

// Size returns the current logical terminal size.
func (a *Application) Size() Size {
	if a == nil {
		return Size{}
	}
	return a.size
}

// LastFrame returns the most recently rendered logical frame. It is nil until
// the first render. The returned frame is for inspection; widgets should draw
// only during their Render call.
func (a *Application) LastFrame() *Frame {
	if a == nil {
		return nil
	}
	return a.lastFrame
}

// RequestQuit stops the loop after the current dispatch. The cleanup sequence
// still runs before Run returns.
func (a *Application) RequestQuit(reason ExitReason) {
	if a == nil {
		return
	}
	a.quit = true
	if reason != ExitNone {
		a.exitReason = reason
	} else if a.exitReason == ExitNone {
		a.exitReason = ExitRequested
	}
}

// ExitReason reports the last requested loop exit reason.
func (a *Application) ExitReason() ExitReason {
	if a == nil {
		return ExitNone
	}
	return a.exitReason
}

// Render creates and writes one complete logical frame. It is useful for
// deterministic harnesses and does not acquire raw mode or install signals.
func (a *Application) Render() error {
	if a == nil {
		return errors.New("screen: nil application")
	}
	return a.render()
}

func (a *Application) render() error {
	frame := NewFrame(a.size)
	if a.arrange != nil {
		for _, placement := range a.arrange(a.size) {
			if placement.Widget == nil {
				continue
			}
			placement.Widget.SetBounds(placement.Bounds)
			placement.Widget.Render(frame)
		}
	} else {
		bounds := frame.Bounds()
		for _, widget := range a.widgets {
			if widget == nil {
				continue
			}
			widget.SetBounds(bounds)
			widget.Render(frame)
		}
	}
	if err := a.terminal.Render(frame); err != nil {
		return err
	}
	if err := a.terminal.Flush(); err != nil {
		return fmt.Errorf("screen: render flush: %w", err)
	}
	a.lastFrame = frame
	return nil
}

func (a *Application) cleanup() error {
	if a.terminal == nil {
		return nil
	}
	return errors.Join(a.terminal.Reset(), a.terminal.Flush())
}

func (a *Application) pollResize(events <-chan os.Signal) (bool, error) {
	if !a.pollResizeNow(events) {
		return false, nil
	}
	if a.resize == nil {
		return false, nil
	}
	size, err := a.resize()
	if err != nil {
		return false, err
	}
	if size.normalized() == a.size {
		return false, nil
	}
	a.Resize(size)
	return true, nil
}

func (a *Application) pollResizeNow(events <-chan os.Signal) bool {
	if events == nil {
		return false
	}
	select {
	case <-events:
		return true
	default:
		return false
	}
}

func (a *Application) ensureFocus() {
	if a.focused != nil && a.canFocus(a.focused) {
		return
	}
	for _, widget := range a.focusOrder {
		if a.canFocus(widget) {
			a.setFocused(widget)
			return
		}
	}
	a.focused = nil
}

func (a *Application) moveFocus(direction int) bool {
	if a == nil || len(a.focusOrder) == 0 {
		return false
	}
	current := a.FocusIndex()
	if current < 0 {
		if direction < 0 {
			current = len(a.focusOrder)
		} else {
			current = -1
		}
	}
	for count := 0; count < len(a.focusOrder); count++ {
		current = (current + direction + len(a.focusOrder)) % len(a.focusOrder)
		if a.canFocus(a.focusOrder[current]) {
			a.setFocused(a.focusOrder[current])
			return true
		}
	}
	return false
}

func (a *Application) canFocus(widget Widget) bool {
	if widget == nil || !widget.Focusable() {
		return false
	}
	if enabled, ok := widget.(interface{ Enabled() bool }); ok && !enabled.Enabled() {
		return false
	}
	return true
}

func (a *Application) setFocused(widget Widget) {
	for _, candidate := range a.focusOrder {
		if candidate != nil {
			candidate.SetFocused(candidate == widget)
		}
	}
	a.focused = widget
}
