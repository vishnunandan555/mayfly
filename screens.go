// Package mayfly contains the application-facing pieces of MayFly. The
// screen package remains responsible for terminal mechanics; this package
// composes its widgets into vault screens and knows nothing about encryption.
package mayfly

import (
	"io"
	"os"
	"strings"
	"unicode"

	"mayfly/screen"
)

// Secret is the vault record consumed by the screen layer. Value is retained
// only in memory for display masking and vault operations; it is never put in
// a status message, error text, or rendered frame as plaintext.
type Secret struct {
	Name  string
	Value string
}

// Vault is the small storage contract required by the MayFly screens. It is
// deliberately cryptography-free: a separate vault implementation owns
// encryption, persistence, and authentication details.
type Vault interface {
	Secrets() ([]Secret, error)
	SetSecret(name, value string) error
	DeleteSecret(name string) error
}

// VaultOpener unlocks or opens a vault. The password is passed directly to
// the vault implementation and is not retained by Screens after the attempt.
type VaultOpener interface {
	Unlock(password string) (Vault, error)
}

// ScreenMode identifies the currently visible application screen.
type ScreenMode uint8

const (
	ModeUnlock ScreenMode = iota
	ModeSecrets
	ModeEditor
	ModeDelete
	ModeError
)

// Screens is the stateful application screen controller. It exposes a
// screen.Widget view and ApplicationOptions so callers can use the existing
// terminal runtime without coupling this layer to process-global terminal
// state.
type Screens struct {
	view   *screenView
	opener VaultOpener
	vault  Vault

	mode       ScreenMode
	returnMode ScreenMode
	status     string

	secrets     []Secret
	editIndex   int
	deleteIndex int

	password  *screen.TextInput
	name      *screen.TextInput
	value     *screen.TextInput
	list      *screen.List
	confirm   *screen.ConfirmDialog
	errorBox  *screen.ConfirmDialog
	statusBar *screen.StatusBar

	app *screen.Application
}

// NewScreens creates a locked screen backed by opener. Unlock is performed
// only after the user submits the password field.
func NewScreens(opener VaultOpener) *Screens {
	s := newScreens(opener, nil)
	s.setMode(nil, ModeUnlock)
	return s
}

// NewScreensWithVault creates an already-unlocked screen. It loads records
// through vault immediately; a load failure is represented by the error modal
// without exposing the underlying error text.
func NewScreensWithVault(vault Vault) *Screens {
	s := newScreens(nil, vault)
	s.setMode(nil, ModeSecrets)
	if vault == nil {
		s.showError(nil, ModeSecrets, "Vault unavailable")
		return s
	}
	if !s.reload() {
		s.showError(nil, ModeSecrets, "Unable to load secrets")
	}
	return s
}

func newScreens(opener VaultOpener, vault Vault) *Screens {
	password := screen.NewTextInput()
	password.Password = true
	password.Placeholder = "Password"
	name := screen.NewTextInput()
	name.Placeholder = "Secret name"
	value := screen.NewTextInput()
	value.Password = true
	value.Placeholder = "Secret value"

	s := &Screens{
		opener:    opener,
		vault:     vault,
		password:  password,
		name:      name,
		value:     value,
		list:      screen.NewList(nil),
		confirm:   screen.NewConfirmDialog("Delete secret", "Delete this secret?"),
		errorBox:  screen.NewConfirmDialog("Error", "Unable to complete operation"),
		statusBar: screen.NewStatusBar(),
	}
	s.confirm.YesLabel = "Delete"
	s.confirm.NoLabel = "Keep"
	s.errorBox.YesLabel = "OK"
	s.errorBox.NoLabel = ""
	s.list.EmptyText = "No secrets. Press N to add one."
	s.view = &screenView{owner: s}
	return s
}

// View returns the root widget that renders the current application screen.
// Focusable child widgets are managed through the returned ApplicationOptions.
func (s *Screens) View() screen.Widget {
	if s == nil {
		return nil
	}
	return s.view
}

// Mode returns the currently visible screen mode.
func (s *Screens) Mode() ScreenMode {
	if s == nil {
		return ModeError
	}
	return s.mode
}

// Status returns the current safe, user-facing status message. It never
// contains a vault error string or secret value.
func (s *Screens) Status() string {
	if s == nil {
		return ""
	}
	return s.status
}

// ApplicationOptions adapts the screens to the existing in-memory-testable
// application runtime. The runtime replaces Output, Input, and Size when
// Run is called through RunTerminal.
func (s *Screens) ApplicationOptions(output io.Writer, input screen.Input, size screen.Size) screen.ApplicationOptions {
	if s == nil {
		return screen.ApplicationOptions{Output: output, Input: input, Size: size}
	}
	focusOrder := []screen.Widget{s.password, s.list, s.name, s.value, s.confirm, s.errorBox}
	return screen.ApplicationOptions{
		Output:     output,
		Input:      input,
		Size:       size,
		Widgets:    []screen.Widget{s.view},
		FocusOrder: focusOrder,
		Arrange: func(size screen.Size) []screen.WidgetPlacement {
			return []screen.WidgetPlacement{{Widget: s.view, Bounds: screen.NewRect(0, 0, size.Rows, size.Columns)}}
		},
		GlobalHandler: s.handleGlobal,
	}
}

// Run starts the real terminal application. Raw mode and terminal sizing are
// acquired only by screen.RunTerminal and are restored there on every exit.
func (s *Screens) Run(file *os.File) error {
	if s == nil {
		return screen.RunTerminal(file, screen.ApplicationOptions{})
	}
	defer s.clearSensitiveInputs()
	return screen.RunTerminal(file, s.ApplicationOptions(nil, nil, screen.Size{}))
}

// RunIO starts the screens with separate input and output streams. This is
// the preferred form for normal terminal processes: input is usually
// os.Stdin and output is usually os.Stdout.
func (s *Screens) RunIO(input *os.File, output io.Writer) error {
	if s == nil {
		return screen.RunTerminalIO(input, output, screen.ApplicationOptions{})
	}
	defer s.clearSensitiveInputs()
	return screen.RunTerminalIO(input, output, s.ApplicationOptions(nil, nil, screen.Size{}))
}

type screenView struct {
	screen.WidgetState
	owner *Screens
}

func (*screenView) Focusable() bool          { return false }
func (*screenView) Handle(screen.Event) bool { return false }

func (v *screenView) Render(frame *screen.Frame) {
	if v == nil || v.owner == nil || frame == nil {
		return
	}
	area := v.Bounds().Intersect(frame.Bounds())
	if area.Empty() {
		return
	}
	frame.ClearRegion(area)
	s := v.owner
	s.renderStatus(area.Size().Columns)
	switch s.mode {
	case ModeUnlock:
		s.renderUnlock(frame, area)
	case ModeSecrets:
		s.renderSecrets(frame, area)
	case ModeEditor:
		s.renderSecrets(frame, area)
		s.renderEditor(frame, area)
	case ModeDelete:
		s.renderSecrets(frame, area)
		s.renderDelete(frame, area)
	case ModeError:
		if s.vault != nil {
			s.renderSecrets(frame, area)
		} else {
			s.renderUnlock(frame, area)
		}
		s.renderError(frame, area)
	}
}

func (s *Screens) renderStatus(width int) {
	s.statusBar.Message = s.status
	switch s.mode {
	case ModeUnlock:
		s.statusBar.Hints = "Enter Unlock   Esc Quit"
	case ModeSecrets:
		s.statusBar.Hints = "↑↓ Select   Enter Edit   N New   D Delete   Esc Quit"
	case ModeEditor:
		s.statusBar.Hints = "Tab Next   Enter Save   Esc Cancel"
	case ModeDelete:
		s.statusBar.Hints = "←→ Choose   Enter Confirm   Esc Cancel"
	default:
		s.statusBar.Hints = "Enter Continue   Esc Back"
	}
	if width < 1 {
		s.statusBar.Hints = ""
	}
	if s.status != "" && screen.TextWidth(s.status)+screen.TextWidth(s.statusBar.Hints)+1 > width {
		s.statusBar.Hints = "Esc Quit"
		if screen.TextWidth(s.status)+screen.TextWidth(s.statusBar.Hints)+1 > width {
			s.statusBar.Hints = ""
		}
	}
	if screen.TextWidth(s.statusBar.Hints) >= width {
		s.statusBar.Hints = ""
	}
}

func (s *Screens) renderUnlock(frame *screen.Frame, area screen.Rect) {
	box := modalRect(area, 8, 54)
	drawBoxTitle(frame, box, "Unlock MayFly", screen.Style{Foreground: screen.ColorCyan})
	frame.DrawTextIn(screen.NewRect(box.Min.Row+2, box.Min.Column+2, 1, max0(box.Size().Columns-4)), "Vault password", screen.TextOptions{Style: screen.Style{Attributes: screen.AttrBold}})
	field := screen.NewRect(box.Min.Row+3, box.Min.Column+2, 1, max0(box.Size().Columns-4))
	s.password.SetBounds(field)
	s.password.Render(frame)
	status := screen.NewRect(box.Max.Row-2, box.Min.Column+2, 1, max0(box.Size().Columns-4))
	s.statusBar.SetBounds(status)
	s.statusBar.Render(frame)
}

func (s *Screens) renderSecrets(frame *screen.Frame, area screen.Rect) {
	drawBoxTitle(frame, area, "MayFly", screen.Style{Foreground: screen.ColorCyan, Attributes: screen.AttrBold})
	inner := screen.Padding{Top: 1, Right: 1, Bottom: 1, Left: 1}.Inset(area)
	if inner.Empty() {
		return
	}
	statusRows := 1
	listRows := max0(inner.Size().Rows - statusRows)
	listArea := screen.NewRect(inner.Min.Row, inner.Min.Column, listRows, inner.Size().Columns)
	statusArea := screen.NewRect(inner.Max.Row-1, inner.Min.Column, statusRows, inner.Size().Columns)
	s.list.SetItems(s.displayLines(listArea.Size().Columns))
	s.list.SetBounds(listArea)
	s.list.Render(frame)
	s.statusBar.SetBounds(statusArea)
	s.statusBar.Render(frame)
}

func (s *Screens) renderEditor(frame *screen.Frame, area screen.Rect) {
	box := modalRect(area, 10, 66)
	title := "Add secret"
	if s.editIndex >= 0 {
		title = "Edit secret"
	}
	drawBoxTitle(frame, box, title, screen.Style{Foreground: screen.ColorYellow, Attributes: screen.AttrBold})
	width := max0(box.Size().Columns - 4)
	nameLabel := screen.NewRect(box.Min.Row+2, box.Min.Column+2, 1, width)
	valueLabel := screen.NewRect(box.Min.Row+5, box.Min.Column+2, 1, width)
	frame.DrawTextIn(nameLabel, "Name", screen.TextOptions{Style: screen.Style{Attributes: screen.AttrBold}})
	frame.DrawTextIn(valueLabel, "Value", screen.TextOptions{Style: screen.Style{Attributes: screen.AttrBold}})
	s.name.SetBounds(screen.NewRect(box.Min.Row+3, box.Min.Column+2, 1, width))
	s.value.SetBounds(screen.NewRect(box.Min.Row+6, box.Min.Column+2, 1, width))
	s.name.Render(frame)
	s.value.Render(frame)
	frame.DrawTextIn(screen.NewRect(box.Max.Row-2, box.Min.Column+2, 1, width), "Values are masked on screen", screen.TextOptions{Style: screen.Style{Attributes: screen.AttrDim}})
}

func (s *Screens) renderDelete(frame *screen.Frame, area screen.Rect) {
	box := modalRect(area, 8, 58)
	s.confirm.SetBounds(box)
	s.confirm.Render(frame)
}

func (s *Screens) renderError(frame *screen.Frame, area screen.Rect) {
	box := modalRect(area, 8, 58)
	s.errorBox.SetBounds(box)
	s.errorBox.Render(frame)
}

func drawBoxTitle(frame *screen.Frame, area screen.Rect, title string, style screen.Style) {
	frame.DrawBox(area, style)
	if title == "" || area.Size().Rows == 0 || area.Size().Columns < 3 {
		return
	}
	frame.DrawTextIn(screen.NewRect(area.Min.Row, area.Min.Column+1, 1, max0(area.Size().Columns-2)), " "+title+" ", screen.TextOptions{Style: style})
}

func modalRect(area screen.Rect, preferredRows, preferredColumns int) screen.Rect {
	available := area.Size()
	rows := minInt(preferredRows, available.Rows)
	columns := minInt(preferredColumns, available.Columns)
	if rows < 1 {
		rows = 0
	}
	if columns < 1 {
		columns = 0
	}
	return screen.NewRect(area.Min.Row+(available.Rows-rows)/2, area.Min.Column+(available.Columns-columns)/2, rows, columns)
}

func (s *Screens) displayLines(width int) []string {
	lines := make([]string, len(s.secrets))
	for i, secret := range s.secrets {
		lines[i] = formatSecretLine(secret.Name, secret.Value, width)
	}
	return lines
}

func formatSecretLine(name, value string, width int) string {
	if width <= 0 {
		return ""
	}
	name = displayName(name)
	masked := screen.MaskSecret(value)
	if screen.TextWidth(masked) >= width {
		return masked
	}
	maskWidth := screen.TextWidth(masked)
	if maskWidth == 0 {
		return clipDisplay(name, width)
	}
	nameWidth := width - maskWidth - 2
	if nameWidth < 1 {
		return masked
	}
	return clipDisplay(name, nameWidth) + strings.Repeat(" ", width-nameWidth-maskWidth) + masked
}

func displayName(name string) string {
	var builder strings.Builder
	for _, r := range name {
		if unicode.IsControl(r) {
			builder.WriteRune(' ')
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func clipDisplay(text string, width int) string {
	if width <= 0 {
		return ""
	}
	var builder strings.Builder
	used := 0
	for _, r := range text {
		runeWidth := screen.RuneWidth(r)
		if runeWidth == 0 {
			continue
		}
		if used+runeWidth > width {
			break
		}
		builder.WriteRune(r)
		used += runeWidth
	}
	return builder.String()
}

func (s *Screens) handleGlobal(app *screen.Application, event screen.Event) bool {
	if s == nil {
		return false
	}
	s.app = app
	switch s.mode {
	case ModeUnlock:
		switch event.Type {
		case screen.EventEnter:
			s.unlock()
			return true
		case screen.EventEscape:
			app.RequestQuit(screen.ExitEscape)
			return true
		}
	case ModeSecrets:
		if event.Type == screen.EventEscape || event.Type == screen.EventCtrlD {
			app.RequestQuit(screen.ExitEscape)
			return true
		}
		if event.Type != screen.EventRune {
			if event.Type == screen.EventEnter && s.list.SelectedIndex() >= 0 {
				s.openEditor(true)
				return true
			}
			return false
		}
		switch event.Rune {
		case 'n', 'N':
			s.openEditor(false)
			return true
		case 'd', 'D':
			s.openDelete()
			return true
		case 'q', 'Q':
			app.RequestQuit(screen.ExitRequested)
			return true
		}
	case ModeEditor:
		switch event.Type {
		case screen.EventEscape:
			s.clearSensitiveInputs()
			s.closeToSecrets("Edit cancelled")
			return true
		case screen.EventEnter:
			s.saveEditor()
			return true
		}
	case ModeDelete:
		if s.confirm.Handle(event) {
			s.resolveDelete()
			return true
		}
	case ModeError:
		if s.errorBox.Handle(event) {
			if s.errorBox.Result != screen.DialogPending {
				s.dismissError()
			}
			return true
		}
	}
	return false
}

func (s *Screens) unlock() {
	password := s.password.Value()
	s.password.SetValue("")
	if s.opener == nil {
		s.showError(nil, ModeUnlock, "Unable to unlock vault")
		return
	}
	vault, err := s.opener.Unlock(password)
	if err != nil || vault == nil {
		s.showError(nil, ModeUnlock, "Unable to unlock vault")
		return
	}
	s.vault = vault
	if !s.reload() {
		s.showError(nil, ModeUnlock, "Unable to load secrets")
		return
	}
	s.status = "Vault unlocked"
	s.setMode(s.app, ModeSecrets)
}

func (s *Screens) reload() bool {
	if s.vault == nil {
		return false
	}
	secrets, err := s.vault.Secrets()
	if err != nil {
		return false
	}
	s.secrets = append([]Secret(nil), secrets...)
	return true
}

func (s *Screens) openEditor(edit bool) {
	s.editIndex = -1
	if edit {
		index := s.list.SelectedIndex()
		if index < 0 || index >= len(s.secrets) {
			return
		}
		s.editIndex = index
		s.name.SetValue(s.secrets[index].Name)
		s.value.SetValue(s.secrets[index].Value)
	} else {
		s.name.SetValue("")
		s.value.SetValue("")
	}
	s.status = ""
	s.setMode(s.app, ModeEditor)
}

func (s *Screens) saveEditor() {
	name := s.name.Value()
	value := s.value.Value()
	if strings.TrimSpace(name) == "" {
		s.showError(s.app, ModeEditor, "Secret name is required")
		return
	}
	if s.vault == nil {
		s.showError(s.app, ModeEditor, "Vault unavailable")
		return
	}
	if s.editIndex >= 0 && s.editIndex < len(s.secrets) && s.secrets[s.editIndex].Name != name {
		if err := s.vault.SetSecret(name, value); err != nil {
			s.showError(s.app, ModeEditor, "Unable to save secret")
			return
		}
		if err := s.vault.DeleteSecret(s.secrets[s.editIndex].Name); err != nil {
			s.showError(s.app, ModeEditor, "Unable to save secret")
			return
		}
	} else if err := s.vault.SetSecret(name, value); err != nil {
		s.showError(s.app, ModeEditor, "Unable to save secret")
		return
	}
	s.name.SetValue("")
	s.value.SetValue("")
	if !s.reload() {
		s.showError(s.app, ModeEditor, "Unable to load secrets")
		return
	}
	s.status = "Secret saved"
	s.setMode(s.app, ModeSecrets)
}

func (s *Screens) openDelete() {
	index := s.list.SelectedIndex()
	if index < 0 || index >= len(s.secrets) {
		return
	}
	s.deleteIndex = index
	s.confirm.Message = "Delete selected secret?"
	s.confirm.Reset()
	s.status = ""
	s.setMode(s.app, ModeDelete)
}

func (s *Screens) resolveDelete() {
	result := s.confirm.Result
	if result == screen.DialogCancelled || result == screen.DialogNo {
		s.closeToSecrets("Delete cancelled")
		return
	}
	if result != screen.DialogYes || s.vault == nil || s.deleteIndex < 0 || s.deleteIndex >= len(s.secrets) {
		s.closeToSecrets("")
		return
	}
	if err := s.vault.DeleteSecret(s.secrets[s.deleteIndex].Name); err != nil {
		s.showError(s.app, ModeDelete, "Unable to delete secret")
		return
	}
	if !s.reload() {
		s.showError(s.app, ModeDelete, "Unable to load secrets")
		return
	}
	s.closeToSecrets("Secret deleted")
}

func (s *Screens) showError(app *screen.Application, returnMode ScreenMode, message string) {
	s.returnMode = returnMode
	s.errorBox.Message = message
	s.errorBox.Reset()
	s.status = ""
	s.setMode(app, ModeError)
}

func (s *Screens) closeToSecrets(status string) {
	s.status = status
	s.setMode(s.app, ModeSecrets)
}

func (s *Screens) dismissError() {
	mode := s.returnMode
	if mode == ModeDelete {
		mode = ModeSecrets
	}
	s.status = ""
	s.setMode(s.app, mode)
}

func (s *Screens) clearSensitiveInputs() {
	if s == nil {
		return
	}
	s.password.SetValue("")
	s.value.SetValue("")
}

func (s *Screens) setMode(app *screen.Application, mode ScreenMode) {
	s.mode = mode
	for _, widget := range s.focusWidgets() {
		if enabled, ok := widget.(interface{ SetEnabled(bool) }); ok {
			enabled.SetEnabled(false)
		}
	}
	var target screen.Widget
	var targets []screen.Widget
	switch mode {
	case ModeUnlock:
		target = s.password
		targets = []screen.Widget{s.password}
	case ModeSecrets:
		target = s.list
		targets = []screen.Widget{s.list}
	case ModeEditor:
		target = s.name
		targets = []screen.Widget{s.name, s.value}
	case ModeDelete:
		target = s.confirm
		targets = []screen.Widget{s.confirm}
	case ModeError:
		target = s.errorBox
		targets = []screen.Widget{s.errorBox}
	}
	for _, widget := range targets {
		if enabled, ok := widget.(interface{ SetEnabled(bool) }); ok {
			enabled.SetEnabled(true)
		}
	}
	if app != nil {
		s.app = app
		s.focusWidget(app, target)
	}
}

func (s *Screens) focusWidgets() []screen.Widget {
	return []screen.Widget{s.password, s.list, s.name, s.value, s.confirm, s.errorBox}
}

func (s *Screens) focusWidget(app *screen.Application, target screen.Widget) {
	if app == nil || target == nil {
		return
	}
	for index, widget := range s.focusWidgets() {
		if widget == target {
			app.SetFocus(index)
			return
		}
	}
}

func max0(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
