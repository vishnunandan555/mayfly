package screen

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestApplicationConstructionHasNoTerminalSideEffects(t *testing.T) {
	var output bytes.Buffer
	input := NewTextInput()
	app := NewApplication(ApplicationOptions{
		Output:  &output,
		Input:   NewInput(strings.NewReader("")),
		Size:    Size{Rows: 2, Columns: 8},
		Widgets: []Widget{input},
	})
	if output.Len() != 0 {
		t.Fatalf("construction wrote %q", output.String())
	}
	if app.Focused() != input {
		t.Fatal("first focusable widget was not focused")
	}
}

func TestApplicationFocusTraversalSkipsDisabledAndNonFocusable(t *testing.T) {
	label := NewLabel("label")
	disabled := NewTextInput()
	disabled.SetEnabled(false)
	first := NewTextInput()
	second := NewList([]string{"one", "two"})
	app := NewApplication(ApplicationOptions{
		FocusOrder: []Widget{label, disabled, first, second},
		Size:       Size{Rows: 3, Columns: 20},
	})

	if app.Focused() != first || app.FocusIndex() != 2 {
		t.Fatalf("initial focus = %v/%d, want first/%d", app.Focused(), app.FocusIndex(), 2)
	}
	if !app.FocusNext() || app.Focused() != second {
		t.Fatal("FocusNext did not move to second input")
	}
	if !app.FocusPrevious() || app.Focused() != first {
		t.Fatal("FocusPrevious did not skip disabled/non-focusable widgets")
	}
	if disabled.Focused() {
		t.Fatal("disabled widget received focus")
	}
}

func TestApplicationHarnessDispatchesInputAndRendersChanges(t *testing.T) {
	var output bytes.Buffer
	input := NewTextInput()
	items := NewList([]string{"first", "second"})
	app := NewApplication(ApplicationOptions{
		Output:  &output,
		Input:   NewInput(strings.NewReader("abc\t\x1b[B\x03")),
		Size:    Size{Rows: 5, Columns: 24},
		Widgets: []Widget{input, items},
		Arrange: func(size Size) []WidgetPlacement {
			return []WidgetPlacement{
				{Widget: input, Bounds: NewRect(0, 0, 1, size.Columns)},
				{Widget: items, Bounds: NewRect(1, 0, size.Rows-1, size.Columns)},
			}
		},
	})

	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
	if got := input.Value(); got != "abc" {
		t.Fatalf("input value = %q, want abc", got)
	}
	if got := items.SelectedIndex(); got != 1 {
		t.Fatalf("selected index = %d, want 1", got)
	}
	if got := app.ExitReason(); got != ExitCtrlC {
		t.Fatalf("exit reason = %v, want Ctrl-C", got)
	}
	if renders := strings.Count(output.String(), "\x1b[?25l"); renders < 6 {
		t.Fatalf("render count = %d, want initial plus state changes", renders)
	}
	if !strings.Contains(output.String(), "\x1b[0m\x1b[?25h") {
		t.Fatal("cleanup did not reset attributes and show cursor")
	}
}

func TestApplicationGlobalHandlerAndEscapeBehavior(t *testing.T) {
	input := NewTextInput()
	called := false
	app := NewApplication(ApplicationOptions{
		Input: NewInput(strings.NewReader("")), Size: Size{Rows: 2, Columns: 8},
		Widgets: []Widget{input},
		GlobalHandler: func(app *Application, event Event) bool {
			called = true
			app.RequestQuit(ExitRequested)
			return true
		},
	})
	if !app.Dispatch(Event{Type: EventRune, Rune: 'x'}) || !called {
		t.Fatal("global handler did not consume the event")
	}
	if input.Value() != "" || app.ExitReason() != ExitRequested {
		t.Fatal("global handler was not applied before focused widget")
	}

	dialog := NewConfirmDialog("Confirm", "Proceed?")
	dialogApp := NewApplication(ApplicationOptions{FocusOrder: []Widget{dialog}, Size: Size{Rows: 4, Columns: 20}})
	if !dialogApp.Dispatch(Event{Type: EventEscape}) || dialog.Result != DialogCancelled {
		t.Fatal("focused dialog did not consume Escape")
	}
	if dialogApp.ExitReason() != ExitNone {
		t.Fatal("dialog Escape incorrectly exited the application")
	}

	labelApp := NewApplication(ApplicationOptions{Widgets: []Widget{NewLabel("x")}, Size: Size{Rows: 1, Columns: 4}})
	if !labelApp.Dispatch(Event{Type: EventEscape}) || labelApp.ExitReason() != ExitEscape {
		t.Fatal("unhandled Escape did not request application exit")
	}
}

func TestApplicationIgnoredInputAndRepeatedInput(t *testing.T) {
	var output bytes.Buffer
	label := NewLabel("static")
	app := NewApplication(ApplicationOptions{
		Output: &output, Input: NewInput(strings.NewReader("abc")),
		Size: Size{Rows: 1, Columns: 10}, Widgets: []Widget{label},
	})
	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
	if renders := strings.Count(output.String(), "\x1b[?25l"); renders != 1 {
		t.Fatalf("ignored input caused %d renders, want 1", renders)
	}
}

func TestApplicationResizeRearrangesAndClips(t *testing.T) {
	label := NewLabel("resize")
	var arranged []Size
	app := NewApplication(ApplicationOptions{
		Size: Size{Rows: 4, Columns: 12}, Widgets: []Widget{label},
		Arrange: func(size Size) []WidgetPlacement {
			arranged = append(arranged, size)
			return []WidgetPlacement{{Widget: label, Bounds: NewRect(0, 0, size.Rows+4, size.Columns+4)}}
		},
	})
	if err := app.Render(); err != nil {
		t.Fatal(err)
	}
	app.Resize(Size{Rows: 1, Columns: 3})
	if err := app.Render(); err != nil {
		t.Fatal(err)
	}
	if got := app.Size(); got != (Size{Rows: 1, Columns: 3}) {
		t.Fatalf("size = %#v, want 1x3", got)
	}
	if got := label.Bounds(); got != NewRect(0, 0, 5, 7) {
		t.Fatalf("assigned bounds = %#v, want unclipped logical placement", got)
	}
	if len(arranged) != 2 || arranged[1] != (Size{Rows: 1, Columns: 3}) {
		t.Fatalf("arrangement sizes = %#v, want resize pass", arranged)
	}
}

type applicationResizeSignal string

func (s applicationResizeSignal) Signal()        {}
func (s applicationResizeSignal) String() string { return string(s) }

func TestApplicationResizeSignalUsesProvider(t *testing.T) {
	app := NewApplication(ApplicationOptions{Size: Size{Rows: 5, Columns: 10}})
	app.resize = func() (Size, error) { return Size{Rows: 2, Columns: 7}, nil }
	events := make(chan os.Signal, 1)
	events <- applicationResizeSignal("resize")
	changed, err := app.pollResize(events)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || app.Size() != (Size{Rows: 2, Columns: 7}) {
		t.Fatalf("resize result = changed:%v size:%#v", changed, app.Size())
	}
}

type applicationErrorInput struct{ err error }

func (i applicationErrorInput) ReadEvent() (Event, error) { return Event{}, i.err }

func TestApplicationCleanupOnInputError(t *testing.T) {
	var output bytes.Buffer
	wantErr := errors.New("input failed")
	app := NewApplication(ApplicationOptions{
		Output: &output, Input: applicationErrorInput{err: wantErr},
		Size: Size{Rows: 2, Columns: 8}, Widgets: []Widget{NewLabel("x")},
	})
	err := app.Run()
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run error = %v, want input error", err)
	}
	if !strings.HasSuffix(output.String(), "\x1b[0m\x1b[?25h") {
		t.Fatalf("cleanup output suffix = %q", output.String()[max(0, len(output.String())-20):])
	}
}

func TestApplicationNilInputStillCleansOutput(t *testing.T) {
	var output bytes.Buffer
	app := NewApplication(ApplicationOptions{Output: &output, Size: Size{Rows: 1, Columns: 1}})
	if err := app.Run(); err == nil {
		t.Fatal("Run with nil input returned nil")
	}
	if !strings.HasSuffix(output.String(), "\x1b[0m\x1b[?25h") {
		t.Fatal("nil-input path did not flush cleanup")
	}
}

func TestApplicationDispatchDoesNotExposePasswordValue(t *testing.T) {
	var output bytes.Buffer
	secret := "do-not-leak"
	input := NewTextInput()
	input.Password = true
	app := NewApplication(ApplicationOptions{
		Output: &output, Input: NewInput(strings.NewReader(secret + "\x03")),
		Size: Size{Rows: 1, Columns: 20}, Widgets: []Widget{input},
	})
	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), secret) {
		t.Fatal("password value appeared in rendered output")
	}
}

func TestApplicationEOFIsCleanExit(t *testing.T) {
	app := NewApplication(ApplicationOptions{
		Output: io.Discard, Input: NewInput(strings.NewReader("")), Size: Size{Rows: 1, Columns: 1},
	})
	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
	if app.ExitReason() != ExitEOF {
		t.Fatalf("exit reason = %v, want EOF", app.ExitReason())
	}
}
