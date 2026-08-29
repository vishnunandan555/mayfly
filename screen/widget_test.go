package screen

import "testing"

func TestLabelAlignmentAndFocusContract(t *testing.T) {
	label := NewLabel("hi")
	label.SetBounds(NewRect(0, 0, 1, 6))
	label.Horizontal = AlignRight
	frame := NewFrame(Size{Rows: 1, Columns: 6})
	label.Render(frame)
	if got, want := frameRow(frame, 0), "    hi"; got != want {
		t.Fatalf("label row = %q, want %q", got, want)
	}
	if label.Focusable() || label.Focused() || label.Handle(Event{Type: EventRune, Rune: 'x'}) {
		t.Fatal("label incorrectly participates in focus or input")
	}
}

func TestTextInputInsertionDeletionAndCursorMovement(t *testing.T) {
	input := NewTextInput()
	input.SetFocused(true)
	input.SetBounds(NewRect(0, 0, 1, 12))
	for _, event := range []Event{{Type: EventRune, Rune: 'a'}, {Type: EventRune, Rune: 'é'}, {Type: EventRune, Rune: 'b'}} {
		if !input.Handle(event) {
			t.Fatalf("input did not consume %#v", event)
		}
	}
	if got, want := input.Value(), "aéb"; got != want {
		t.Fatalf("input value = %q, want %q", got, want)
	}
	if got, want := input.Cursor(), 3; got != want {
		t.Fatalf("cursor = %d, want %d", got, want)
	}

	input.Handle(Event{Type: EventArrowLeft})
	input.Handle(Event{Type: EventBackspace})
	if got, want := input.Value(), "ab"; got != want {
		t.Fatalf("backspace value = %q, want %q", got, want)
	}
	input.Handle(Event{Type: EventHome})
	input.Handle(Event{Type: EventDelete})
	if got, want := input.Value(), "b"; got != want {
		t.Fatalf("delete value = %q, want %q", got, want)
	}
	input.Handle(Event{Type: EventEnd})
	if got, want := input.Cursor(), 1; got != want {
		t.Fatalf("end cursor = %d, want %d", got, want)
	}
	if input.Handle(Event{Type: EventEnter}) {
		t.Fatal("TextInput consumed parent-level Enter")
	}
}

func TestTextInputCtrlUCtrlWAndFocusBoundary(t *testing.T) {
	input := NewTextInput()
	input.SetValue("one two three")
	input.SetBounds(NewRect(0, 0, 1, 8))
	if input.Handle(Event{Type: EventArrowLeft}) {
		t.Fatal("unfocused input accepted input")
	}
	input.SetFocused(true)
	input.Handle(Event{Type: EventCtrlW})
	if got, want := input.Value(), "one two "; got != want {
		t.Fatalf("Ctrl-W value = %q, want %q", got, want)
	}
	input.Handle(Event{Type: EventCtrlU})
	if got, want := input.Value(), ""; got != want {
		t.Fatalf("Ctrl-U value = %q, want empty", got)
	}
	input.Handle(Event{Type: EventBackspace})
	input.Handle(Event{Type: EventDelete})
	if got := input.Cursor(); got != 0 {
		t.Fatalf("empty input cursor = %d, want 0", got)
	}
}

func TestTextInputPlaceholderPasswordAndHorizontalScroll(t *testing.T) {
	placeholder := NewTextInput()
	placeholder.Placeholder = "enter value"
	placeholder.SetBounds(NewRect(0, 0, 1, 6))
	frame := NewFrame(Size{Rows: 1, Columns: 6})
	placeholder.Render(frame)
	if got := frameRow(frame, 0); got != "enter " {
		t.Fatalf("placeholder row = %q, want %q", got, "enter ")
	}

	password := NewTextInput()
	password.SetValue("TOP-SECRET")
	password.SetFocused(true)
	password.SetBounds(NewRect(0, 0, 1, 4))
	frame = NewFrame(Size{Rows: 1, Columns: 4})
	password.Render(frame)
	password.Password = true
	password.Render(frame)
	if got := frameRow(frame, 0); got != "••• " {
		t.Fatalf("password row = %q, want masked value with cursor", got)
	}
	if got := frameRow(frame, 0); got == password.Value() || containsRuneString(got, []rune(password.Value())) {
		t.Fatalf("password rendered plaintext: %q", got)
	}
	if cell, _ := frame.Cell(0, 3); cell.Style.Attributes&AttrReverse == 0 {
		t.Fatal("focused password cursor is not styled")
	}
}

func TestListSelectionScrollingAndEmptyList(t *testing.T) {
	items := []string{"zero", "one", "two", "three", "four", "five", "six", "seven"}
	list := NewList(items)
	list.SetBounds(NewRect(0, 0, 3, 10))
	if list.Handle(Event{Type: EventArrowDown}) {
		t.Fatal("unfocused list accepted navigation")
	}
	list.SetFocused(true)
	for index := 0; index < 4; index++ {
		list.Handle(Event{Type: EventArrowDown})
	}
	if got, want := list.SelectedIndex(), 4; got != want {
		t.Fatalf("selection = %d, want %d", got, want)
	}
	if got, want := list.ScrollOffset(), 2; got != want {
		t.Fatalf("scroll offset = %d, want %d", got, want)
	}
	list.Handle(Event{Type: EventPageDown})
	if got, want := list.SelectedIndex(), 7; got != want {
		t.Fatalf("PageDown selection = %d, want %d", got, want)
	}
	list.Handle(Event{Type: EventHome})
	if list.SelectedIndex() != 0 || list.ScrollOffset() != 0 {
		t.Fatalf("Home did not reset list: selection=%d offset=%d", list.SelectedIndex(), list.ScrollOffset())
	}
	list.Handle(Event{Type: EventEnd})
	if list.SelectedIndex() != 7 || list.ScrollOffset() != 5 {
		t.Fatalf("End did not reach bottom: selection=%d offset=%d", list.SelectedIndex(), list.ScrollOffset())
	}
	frame := NewFrame(Size{Rows: 3, Columns: 10})
	list.Render(frame)
	if got := frameRow(frame, 0); got[:4] != "five" {
		t.Fatalf("scrolled first row = %q, want five...", got)
	}

	empty := NewList(nil)
	empty.SetBounds(NewRect(0, 0, 2, 10))
	empty.SetFocused(true)
	if _, ok := empty.Selected(); ok || empty.SelectedIndex() != -1 {
		t.Fatal("empty list has a selection")
	}
	if !empty.Handle(Event{Type: EventArrowDown}) {
		t.Fatal("empty list did not consume navigation")
	}
}

func TestPanelStatusBarAndChildBounds(t *testing.T) {
	child := NewLabel("inside")
	panel := NewPanel("Title", child)
	panel.Padding = Padding{Left: 1, Right: 1}
	panel.SetBounds(NewRect(0, 0, 5, 12))
	if got, want := child.Bounds(), NewRect(1, 2, 3, 8); got != want {
		t.Fatalf("child bounds = %#v, want %#v", got, want)
	}
	frame := NewFrame(Size{Rows: 5, Columns: 12})
	panel.Render(frame)
	if got := []rune(frameRow(frame, 0)); got[0] != '┌' || got[len(got)-1] != '┐' {
		t.Fatalf("panel top border = %q", got)
	}
	if got := []rune(frameRow(frame, 1)); string(got[2:8]) != "inside" {
		t.Fatalf("panel child row = %q", got)
	}

	status := NewStatusBar()
	status.Message = "Ready"
	status.Hints = "Enter: open"
	status.SetBounds(NewRect(0, 0, 1, 20))
	frame = NewFrame(Size{Rows: 1, Columns: 20})
	status.Render(frame)
	if got := frameRow(frame, 0); got[:5] != "Ready" || got[len(got)-11:] != "Enter: open" {
		t.Fatalf("status row = %q", got)
	}
}

func TestConfirmDialogAndTinyBounds(t *testing.T) {
	dialog := NewConfirmDialog("Delete", "Really delete?")
	dialog.SetFocused(true)
	if !dialog.Handle(Event{Type: EventArrowRight}) || dialog.YesSelected {
		t.Fatal("dialog did not select No")
	}
	if !dialog.Handle(Event{Type: EventEnter}) || dialog.Result != DialogNo {
		t.Fatalf("dialog Enter result = %v, want DialogNo", dialog.Result)
	}
	dialog.Reset()
	if !dialog.Handle(Event{Type: EventEscape}) || dialog.Result != DialogCancelled {
		t.Fatalf("dialog Escape result = %v, want cancelled", dialog.Result)
	}

	widgets := []Widget{
		NewLabel("label"), NewTextInput(), NewList([]string{"one", "two"}),
		NewPanel("panel", NewLabel("child")), NewStatusBar(), NewConfirmDialog("title", "message"),
	}
	for _, widget := range widgets {
		widget.SetBounds(NewRect(0, 0, 1, 1))
		widget.SetFocused(true)
		widget.Render(NewFrame(Size{Rows: 1, Columns: 1}))
		widget.Handle(Event{Type: EventArrowDown})
	}
}

func containsRuneString(haystack string, needle []rune) bool {
	return len(needle) > 0 && len(haystack) >= len(string(needle)) && string(needle) == haystack
}
