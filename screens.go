// Package mayfly contains the application-facing screens of MayFly.
// The screen package is the presentation/input layer; this package composes
// widgets into MayFly application screens and coordinates use cases via
// application service interfaces.
package mayfly

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"mayfly/application"
	"mayfly/domain"
	"mayfly/screen"
)

// Secret is the legacy vault record type retained for compatibility.
type Secret = application.ScreenSecret

// Vault is the legacy storage contract retained for compatibility.
type Vault = application.ScreenVault

// VaultOpener is the legacy unlock boundary retained for compatibility.
type VaultOpener = application.ScreenVaultOpener

// ScreenService is the application service boundary used by MayFly screens.
type ScreenService = application.ScreenService

// ScreenMode identifies the currently visible application screen.
type ScreenMode uint8

const (
	ModeUnlock ScreenMode = iota
	ModeSecrets
	ModeEditor
	ModeDelete
	ModeScan
	ModeAudit
	ModeError
)

// Screens is the stateful application screen controller. It exposes a
// screen.Widget view and ApplicationOptions so callers can use the existing
// terminal runtime without coupling this layer to backend internals or
// process-global terminal state.
type Screens struct {
	view    *screenView
	service ScreenService

	mode        ScreenMode
	returnMode  ScreenMode
	status      string
	projectPath string

	secrets       []domain.Secret
	scanFindings  []domain.ScanFinding
	auditEvents   []domain.AuditEvent
	editIndex     int
	editOrigName  domain.SecretName
	deleteIndex   int

	password  *screen.TextInput
	name      *screen.TextInput
	value     *screen.TextInput
	list      *screen.List
	scanList  *screen.List
	auditList *screen.List
	confirm   *screen.ConfirmDialog
	errorBox  *screen.ConfirmDialog
	statusBar *screen.StatusBar

	app *screen.Application
}

// NewScreens creates an application screen controller backed by service.
// If the service is not yet unlocked, it starts in ModeUnlock. If already
// unlocked, it loads secrets and starts in ModeSecrets.
func NewScreens(service ScreenService) *Screens {
	s := newScreens(service)
	if service != nil && service.IsUnlocked() {
		s.loadProjectInfo()
		s.setMode(nil, ModeSecrets)
		if !s.reloadSecrets() {
			s.showError(nil, ModeSecrets, "Unable to load secrets")
		}
	} else {
		s.setMode(nil, ModeUnlock)
	}
	return s
}

// NewScreensWithService creates an application screen controller backed by service.
func NewScreensWithService(service ScreenService) *Screens {
	return NewScreens(service)
}

// NewScreensWithVault creates an unlocked screen controller from a legacy Vault.
func NewScreensWithVault(vault Vault) *Screens {
	service := application.ScreenServiceFromVault(vault)
	return NewScreens(service)
}

// NewScreensWithOpener creates a locked screen controller from a legacy VaultOpener.
func NewScreensWithOpener(opener VaultOpener) *Screens {
	service := application.ScreenServiceFromOpener(opener)
	return NewScreens(service)
}

func newScreens(service ScreenService) *Screens {
	password := screen.NewTextInput()
	password.Password = true
	password.Placeholder = "Master password"

	name := screen.NewTextInput()
	name.Placeholder = "Secret name"

	value := screen.NewTextInput()
	value.Password = true
	value.Placeholder = "Secret value"

	list := screen.NewList(nil)
	list.EmptyText = "No secrets in project. Press N to add one."

	scanList := screen.NewList(nil)
	scanList.EmptyText = "Project scan clean. No plaintext secrets found."

	auditList := screen.NewList(nil)
	auditList.EmptyText = "No audit events recorded yet."

	confirm := screen.NewConfirmDialog("Delete secret", "Delete this secret?")
	confirm.YesLabel = "Delete"
	confirm.NoLabel = "Keep"

	errorBox := screen.NewConfirmDialog("Error", "Unable to complete operation")
	errorBox.YesLabel = "OK"
	errorBox.NoLabel = ""

	statusBar := screen.NewStatusBar()

	s := &Screens{
		service:   service,
		password:  password,
		name:      name,
		value:     value,
		list:      list,
		scanList:  scanList,
		auditList: auditList,
		confirm:   confirm,
		errorBox:  errorBox,
		statusBar: statusBar,
	}
	s.view = &screenView{owner: s}
	return s
}

// View returns the root widget that renders the current application screen.
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

// Status returns the current safe, user-facing status message.
func (s *Screens) Status() string {
	if s == nil {
		return ""
	}
	return s.status
}

// ApplicationOptions adapts the screens to the existing in-memory application runtime.
func (s *Screens) ApplicationOptions(output io.Writer, input screen.Input, size screen.Size) screen.ApplicationOptions {
	if s == nil {
		return screen.ApplicationOptions{Output: output, Input: input, Size: size}
	}
	focusOrder := []screen.Widget{s.password, s.list, s.name, s.value, s.confirm, s.scanList, s.auditList, s.errorBox}
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

// Run starts the real terminal application.
func (s *Screens) Run(file *os.File) error {
	if s == nil {
		return screen.RunTerminal(file, screen.ApplicationOptions{})
	}
	defer s.clearSensitiveInputs()
	return screen.RunTerminal(file, s.ApplicationOptions(nil, nil, screen.Size{}))
}

// RunIO starts the screens with separate input and output streams.
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
	case ModeScan:
		s.renderScan(frame, area)
	case ModeAudit:
		s.renderAudit(frame, area)
	case ModeError:
		if s.service != nil && s.service.IsUnlocked() {
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
		s.statusBar.Hints = "N New   Enter Edit   D Delete   S Scan   A Audit   Q Quit"
	case ModeEditor:
		s.statusBar.Hints = "Tab Next   Enter Save   Esc Cancel"
	case ModeDelete:
		s.statusBar.Hints = "←→ Choose   Enter Confirm   Esc Cancel"
	case ModeScan:
		s.statusBar.Hints = "↑↓ Scroll   R Rescan   Esc Back"
	case ModeAudit:
		s.statusBar.Hints = "↑↓ Scroll   Esc Back"
	default:
		s.statusBar.Hints = "Enter OK   Esc Back"
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
	box := modalRect(area, 10, 50)
	drawBoxTitle(frame, box, "MayFly", screen.Style{Foreground: screen.ColorCyan, Attributes: screen.AttrBold})
	if box.Size().Rows < 6 || box.Size().Columns < 10 {
		return
	}
	width := max0(box.Size().Columns - 4)
	frame.DrawTextIn(screen.NewRect(box.Min.Row+2, box.Min.Column+2, 1, width), "Unlock vault", screen.TextOptions{Style: screen.Style{Attributes: screen.AttrBold}})
	frame.DrawTextIn(screen.NewRect(box.Min.Row+4, box.Min.Column+2, 1, width), "Master password:", screen.TextOptions{})
	field := screen.NewRect(box.Min.Row+5, box.Min.Column+2, 1, width)
	s.password.SetBounds(field)
	s.password.Render(frame)

	buttonText := "[ Enter ]"
	buttonCol := box.Min.Column + (box.Size().Columns-screen.TextWidth(buttonText))/2
	if box.Min.Row+7 < box.Max.Row-1 {
		frame.DrawTextIn(screen.NewRect(box.Min.Row+7, buttonCol, 1, width), buttonText, screen.TextOptions{Style: screen.Style{Foreground: screen.ColorCyan, Attributes: screen.AttrBold}})
	}

	status := screen.NewRect(box.Max.Row-2, box.Min.Column+2, 1, width)
	s.statusBar.SetBounds(status)
	s.statusBar.Render(frame)
}

func (s *Screens) renderSecrets(frame *screen.Frame, area screen.Rect) {
	title := "MayFly"
	if s.projectPath != "" {
		title = fmt.Sprintf("MayFly   %s", s.projectPath)
	}
	drawBoxTitle(frame, area, title, screen.Style{Foreground: screen.ColorCyan, Attributes: screen.AttrBold})
	inner := screen.Padding{Top: 1, Right: 1, Bottom: 1, Left: 1}.Inset(area)
	if inner.Empty() {
		return
	}
	statusRows := 1
	listRows := max0(inner.Size().Rows - statusRows)
	listArea := screen.NewRect(inner.Min.Row, inner.Min.Column, listRows, inner.Size().Columns)
	statusArea := screen.NewRect(inner.Max.Row-1, inner.Min.Column, statusRows, inner.Size().Columns)
	s.list.SetItems(s.displaySecretLines(listArea.Size().Columns))
	s.list.SetBounds(listArea)
	s.list.Render(frame)
	s.statusBar.SetBounds(statusArea)
	s.statusBar.Render(frame)
}

func (s *Screens) renderEditor(frame *screen.Frame, area screen.Rect) {
	box := modalRect(area, 11, 66)
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

func (s *Screens) renderScan(frame *screen.Frame, area screen.Rect) {
	drawBoxTitle(frame, area, "MayFly — Scan Results", screen.Style{Foreground: screen.ColorYellow, Attributes: screen.AttrBold})
	inner := screen.Padding{Top: 1, Right: 1, Bottom: 1, Left: 1}.Inset(area)
	if inner.Empty() {
		return
	}
	headerRows := 2
	statusRows := 1
	listRows := max0(inner.Size().Rows - headerRows - statusRows)

	headerArea := screen.NewRect(inner.Min.Row, inner.Min.Column, headerRows, inner.Size().Columns)
	listArea := screen.NewRect(inner.Min.Row+headerRows, inner.Min.Column, listRows, inner.Size().Columns)
	statusArea := screen.NewRect(inner.Max.Row-1, inner.Min.Column, statusRows, inner.Size().Columns)

	if len(s.scanFindings) == 0 {
		frame.DrawTextIn(headerArea, "✓ No plaintext secrets detected in project.", screen.TextOptions{Style: screen.Style{Foreground: screen.ColorGreen, Attributes: screen.AttrBold}})
	} else {
		frame.DrawTextIn(headerArea, fmt.Sprintf("⚠ Found %d potential secret leak(s) in project:", len(s.scanFindings)), screen.TextOptions{Style: screen.Style{Foreground: screen.ColorYellow, Attributes: screen.AttrBold}})
	}

	lines := make([]string, len(s.scanFindings))
	for i, f := range s.scanFindings {
		lines[i] = formatScanFindingLine(f, listArea.Size().Columns)
	}
	s.scanList.SetItems(lines)
	s.scanList.SetBounds(listArea)
	s.scanList.Render(frame)

	s.statusBar.SetBounds(statusArea)
	s.statusBar.Render(frame)
}

func (s *Screens) renderAudit(frame *screen.Frame, area screen.Rect) {
	drawBoxTitle(frame, area, "MayFly — Audit Summary", screen.Style{Foreground: screen.ColorCyan, Attributes: screen.AttrBold})
	inner := screen.Padding{Top: 1, Right: 1, Bottom: 1, Left: 1}.Inset(area)
	if inner.Empty() {
		return
	}
	headerRows := 2
	statusRows := 1
	listRows := max0(inner.Size().Rows - headerRows - statusRows)

	headerArea := screen.NewRect(inner.Min.Row, inner.Min.Column, headerRows, inner.Size().Columns)
	listArea := screen.NewRect(inner.Min.Row+headerRows, inner.Min.Column, listRows, inner.Size().Columns)
	statusArea := screen.NewRect(inner.Max.Row-1, inner.Min.Column, statusRows, inner.Size().Columns)

	summaryText := fmt.Sprintf("✓ Tamper-evident audit trail: %d event(s) recorded", len(s.auditEvents))
	frame.DrawTextIn(headerArea, summaryText, screen.TextOptions{Style: screen.Style{Foreground: screen.ColorCyan, Attributes: screen.AttrBold}})

	lines := make([]string, len(s.auditEvents))
	for i, e := range s.auditEvents {
		lines[i] = formatAuditEventLine(e, listArea.Size().Columns)
	}
	s.auditList.SetItems(lines)
	s.auditList.SetBounds(listArea)
	s.auditList.Render(frame)

	s.statusBar.SetBounds(statusArea)
	s.statusBar.Render(frame)
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

func (s *Screens) displaySecretLines(width int) []string {
	lines := make([]string, len(s.secrets))
	for i, secret := range s.secrets {
		lines[i] = formatSecretLine(string(secret.Name), width)
	}
	return lines
}

func formatSecretLine(name string, width int) string {
	if width <= 0 {
		return ""
	}
	name = displayName(name)
	masked := "••••••••••"
	maskWidth := screen.TextWidth(masked)
	if maskWidth >= width {
		return clipDisplay(masked, width)
	}
	nameWidth := width - maskWidth - 2
	if nameWidth < 1 {
		return clipDisplay(name, width)
	}
	return clipDisplay(name, nameWidth) + strings.Repeat(" ", width-nameWidth-maskWidth) + masked
}

func formatScanFindingLine(finding domain.ScanFinding, width int) string {
	if width <= 0 {
		return ""
	}
	loc := finding.Path
	if finding.Line > 0 {
		loc = fmt.Sprintf("%s:%d:%d", finding.Path, finding.Line, finding.Column)
	}
	sev := strings.ToUpper(string(finding.Severity))
	line := fmt.Sprintf("[%s] %s — %s", sev, loc, finding.Message)
	return clipDisplay(displayName(line), width)
}

func formatAuditEventLine(event domain.AuditEvent, width int) string {
	if width <= 0 {
		return ""
	}
	timeStr := event.At.UTC().Format("2006-01-02 15:04:05")
	var extra []string
	if event.Secret != "" {
		extra = append(extra, fmt.Sprintf("secret=%s", event.Secret))
	}
	if event.Command != "" {
		extra = append(extra, fmt.Sprintf("cmd=%s", event.Command))
	}
	if event.ExitStatus != nil {
		extra = append(extra, fmt.Sprintf("exit=%d", *event.ExitStatus))
	}
	line := fmt.Sprintf("%s  %-18s", timeStr, string(event.Action))
	if len(extra) > 0 {
		line += "  " + strings.Join(extra, " ")
	}
	return clipDisplay(displayName(line), width)
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
		case screen.EventEscape, screen.EventCtrlD:
			app.RequestQuit(screen.ExitEscape)
			return true
		}
	case ModeSecrets:
		if event.Type == screen.EventEscape || event.Type == screen.EventCtrlD {
			app.RequestQuit(screen.ExitEscape)
			return true
		}
		if event.Type == screen.EventEnter && s.list.SelectedIndex() >= 0 {
			s.openEditor(true)
			return true
		}
		if event.Type == screen.EventRune {
			switch event.Rune {
			case 'n', 'N':
				s.openEditor(false)
				return true
			case 'd', 'D':
				s.openDelete()
				return true
			case 's', 'S':
				s.openScan()
				return true
			case 'a', 'A':
				s.openAudit()
				return true
			case 'q', 'Q':
				app.RequestQuit(screen.ExitRequested)
				return true
			}
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
	case ModeScan:
		if event.Type == screen.EventEscape || event.Type == screen.EventEnter || (event.Type == screen.EventRune && (event.Rune == 'q' || event.Rune == 'Q')) {
			s.closeToSecrets("Scan completed")
			return true
		}
		if event.Type == screen.EventRune && (event.Rune == 'r' || event.Rune == 'R') {
			s.openScan()
			return true
		}
	case ModeAudit:
		if event.Type == screen.EventEscape || event.Type == screen.EventEnter || (event.Type == screen.EventRune && (event.Rune == 'q' || event.Rune == 'Q')) {
			s.closeToSecrets("")
			return true
		}
	case ModeError:
		if event.Type == screen.EventEnter || event.Type == screen.EventEscape || event.Type == screen.EventCtrlD {
			s.dismissError()
			return true
		}
		if event.Type == screen.EventRune && (event.Rune == ' ' || event.Rune == 'q' || event.Rune == 'Q' || event.Rune == 'y' || event.Rune == 'Y' || event.Rune == 'n' || event.Rune == 'N') {
			s.dismissError()
			return true
		}
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
	if s.service == nil {
		s.showError(s.app, ModeUnlock, "Unable to unlock vault")
		return
	}
	if err := s.service.Unlock(context.Background(), password); err != nil {
		s.showError(s.app, ModeUnlock, "Unable to unlock vault")
		return
	}
	s.loadProjectInfo()
	if !s.reloadSecrets() {
		s.showError(s.app, ModeUnlock, "Unable to load secrets")
		return
	}
	s.status = "Vault unlocked"
	s.setMode(s.app, ModeSecrets)
}

func (s *Screens) loadProjectInfo() {
	if s.service == nil {
		return
	}
	if path, err := s.service.ProjectPath(context.Background()); err == nil && path != "" {
		s.projectPath = path
	}
}

func (s *Screens) reloadSecrets() bool {
	if s.service == nil {
		return false
	}
	secrets, err := s.service.ListSecrets(context.Background())
	if err != nil {
		return false
	}
	s.secrets = append([]domain.Secret(nil), secrets...)
	return true
}

func (s *Screens) openEditor(edit bool) {
	s.editIndex = -1
	s.editOrigName = ""
	if edit {
		index := s.list.SelectedIndex()
		if index < 0 || index >= len(s.secrets) {
			return
		}
		secret := s.secrets[index]
		material, err := s.service.GetSecret(context.Background(), secret.Name)
		if err != nil {
			s.showError(s.app, ModeSecrets, "Unable to read secret")
			return
		}
		s.editIndex = index
		s.editOrigName = secret.Name
		s.name.SetValue(string(secret.Name))
		s.value.SetValue(material.Value)
	} else {
		s.name.SetValue("")
		s.value.SetValue("")
	}
	s.status = ""
	s.setMode(s.app, ModeEditor)
}

func (s *Screens) saveEditor() {
	name := strings.TrimSpace(s.name.Value())
	value := s.value.Value()
	if name == "" {
		s.showError(s.app, ModeEditor, "Secret name is required")
		return
	}
	if s.service == nil {
		s.showError(s.app, ModeEditor, "Vault unavailable")
		return
	}
	secretName := domain.SecretName(name)
	if err := secretName.Validate(); err != nil {
		s.showError(s.app, ModeEditor, "Invalid secret name")
		return
	}

	if s.editIndex >= 0 && s.editOrigName != "" && s.editOrigName != secretName {
		if err := s.service.SetSecret(context.Background(), secretName, value); err != nil {
			s.showError(s.app, ModeEditor, "Unable to save secret")
			return
		}
		if err := s.service.DeleteSecret(context.Background(), s.editOrigName); err != nil {
			s.showError(s.app, ModeEditor, "Unable to save secret")
			return
		}
	} else {
		if err := s.service.SetSecret(context.Background(), secretName, value); err != nil {
			s.showError(s.app, ModeEditor, "Unable to save secret")
			return
		}
	}

	s.clearSensitiveInputs()
	if !s.reloadSecrets() {
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
	s.confirm.Message = fmt.Sprintf("Delete secret %q?", s.secrets[index].Name)
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
	if result != screen.DialogYes || s.service == nil || s.deleteIndex < 0 || s.deleteIndex >= len(s.secrets) {
		s.closeToSecrets("")
		return
	}
	if err := s.service.DeleteSecret(context.Background(), s.secrets[s.deleteIndex].Name); err != nil {
		s.showError(s.app, ModeDelete, "Unable to delete secret")
		return
	}
	if !s.reloadSecrets() {
		s.showError(s.app, ModeDelete, "Unable to load secrets")
		return
	}
	s.closeToSecrets("Secret deleted")
}

func (s *Screens) openScan() {
	if s.service == nil {
		s.showError(s.app, ModeSecrets, "Scanner unavailable")
		return
	}
	findings, err := s.service.Scan(context.Background())
	if err != nil {
		s.showError(s.app, ModeSecrets, "Unable to scan project")
		return
	}
	s.scanFindings = append([]domain.ScanFinding(nil), findings...)
	s.status = ""
	s.setMode(s.app, ModeScan)
}

func (s *Screens) openAudit() {
	if s.service == nil {
		s.showError(s.app, ModeSecrets, "Audit log unavailable")
		return
	}
	events, err := s.service.AuditEvents(context.Background())
	if err != nil {
		s.showError(s.app, ModeSecrets, "Unable to read audit log")
		return
	}
	s.auditEvents = append([]domain.AuditEvent(nil), events...)
	s.status = ""
	s.setMode(s.app, ModeAudit)
}

func (s *Screens) showError(app *screen.Application, returnMode ScreenMode, message string) {
	if app == nil {
		app = s.app
	}
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
	s.name.SetValue("")
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
	case ModeScan:
		target = s.scanList
		targets = []screen.Widget{s.scanList}
	case ModeAudit:
		target = s.auditList
		targets = []screen.Widget{s.auditList}
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
	return []screen.Widget{s.password, s.list, s.name, s.value, s.confirm, s.scanList, s.auditList, s.errorBox}
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
