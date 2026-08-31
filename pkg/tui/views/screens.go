package views

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"mayfly/pkg/application"
	"mayfly/pkg/domain"
	"mayfly/pkg/tui/layout"
	"mayfly/pkg/tui/terminal"
	"mayfly/pkg/tui/widget"
)

type ScreenMode int

const (
	ModeUnlock ScreenMode = iota
	ModeFirstRunSetup
	ModeGlobalProjects
	ModeProjectSecrets
	ModeEditSecret
	ModeDeleteConfirm
	ModeScan
	ModeAudit
	ModeBackup
)

type Screens struct {
	svc          *application.Service
	mode         ScreenMode
	prevMode     ScreenMode
	status       string
	statusTimer  time.Time
	currentDir   string
	selProject   domain.Project
	secrets      []domain.Secret
	findings     []domain.ScanFinding
	auditEvents  []domain.AuditEvent

	// Widgets
	passInput    *widget.TextInput
	confirmPass  *widget.TextInput
	secretName   *widget.TextInput
	secretValue  *widget.TextInput
	backupPath   *widget.TextInput
	projectGrid  *widget.ProjectCardGrid
	secretsList  *widget.List
	scanList     *widget.List
	auditList    *widget.List
	confirmDlg   *widget.ConfirmDialog
	editOrigName domain.SecretName
	revealValue  bool
}

func NewScreens(svc *application.Service, currentDir string) *Screens {
	s := &Screens{
		svc:         svc,
		currentDir:  currentDir,
		passInput:   widget.NewTextInput("Master Password", "Enter vault password...", true),
		confirmPass: widget.NewTextInput("Confirm Password", "Re-enter password...", true),
		secretName:  widget.NewTextInput("Secret Name", "e.g. STRIPE_API_KEY", false),
		secretValue: widget.NewTextInput("Secret Value", "Enter raw secret value...", true),
		backupPath:  widget.NewTextInput("Backup File Path", "~/.mayfly/backup.json", false),
		projectGrid: widget.NewProjectCardGrid("Project Directories"),
		secretsList: widget.NewList("Project Secrets"),
		scanList:    widget.NewList("Plaintext Leak Scan"),
		auditList:   widget.NewList("Audit Log"),
		confirmDlg:  widget.NewConfirmDialog("Confirm Delete", "Are you sure you want to delete this secret?"),
	}

	s.passInput.SetFocused(true)

	if !svc.VaultExists() {
		s.mode = ModeFirstRunSetup
		s.status = "Welcome to MayFly! Set up your vault master password to begin."
	} else if !svc.IsUnlocked() {
		s.mode = ModeUnlock
		s.status = "Vault locked. Enter master password to unlock."
	} else {
		s.mode = ModeGlobalProjects
		s.reloadProjects()
	}

	return s
}

func (s *Screens) SetProjectScoped(proj domain.Project) {
	s.selProject = proj
	s.mode = ModeProjectSecrets
	s.reloadSecrets()
}

func (s *Screens) SetStatus(msg string) {
	s.status = msg
	s.statusTimer = time.Now()
}

func (s *Screens) reloadProjects() {
	projects, err := s.svc.Projects()
	if err != nil {
		s.SetStatus(fmt.Sprintf("Failed to load projects: %v", err))
		return
	}

	var currentProjID string
	if cur, err := s.svc.ResolveCurrentProject(s.currentDir); err == nil {
		currentProjID = cur.ID
	}

	var cards []widget.ProjectCard
	for _, p := range projects {
		secCount := 0
		if s.svc.IsUnlocked() {
			if list, err := s.svc.ListSecrets(p.ID); err == nil {
				secCount = len(list)
			}
		}
		cards = append(cards, widget.ProjectCard{
			Project:     p,
			SecretCount: secCount,
			IsCurrent:   p.ID == currentProjID,
		})
	}

	s.projectGrid.SetCards(cards)
}

func (s *Screens) reloadSecrets() {
	if s.selProject.ID == "" {
		return
	}

	list, err := s.svc.ListSecrets(s.selProject.ID)
	if err != nil {
		s.SetStatus(fmt.Sprintf("Error: %v", err))
		return
	}
	s.secrets = list

	var items []widget.ListItem
	for _, sec := range list {
		maskedVal := "••••••••••••••••"
		if s.revealValue {
			maskedVal = sec.Value
		}
		items = append(items, widget.ListItem{
			Primary:   string(sec.Name),
			Secondary: maskedVal,
			Extra:     "[C: Copy]",
			Data:      sec,
		})
	}

	projName := filepath.Base(s.selProject.CanonicalPath)
	s.secretsList.Title = fmt.Sprintf("Secrets for: %s", projName)
	s.secretsList.SetItems(items)
}

// HandleKey processes keyboard input and transitions between screens.
func (s *Screens) HandleKey(event terminal.KeyEvent) (shouldQuit bool) {
	ctx := context.Background()

	// Clear temporary status message after 4 seconds
	if !s.statusTimer.IsZero() && time.Since(s.statusTimer) > 4*time.Second {
		s.status = ""
	}

	switch s.mode {
	case ModeFirstRunSetup:
		if event.Type == terminal.KeyTab || event.Type == terminal.KeyShiftTab || event.Type == terminal.KeyDown || event.Type == terminal.KeyUp {
			s.passInput.Focused = !s.passInput.Focused
			s.confirmPass.Focused = !s.confirmPass.Focused
			return false
		}
		if event.Type == terminal.KeyEnter {
			if s.passInput.Focused && s.confirmPass.Value == "" {
				if s.passInput.Value == "" {
					s.SetStatus("Password cannot be empty.")
					return false
				}
				s.passInput.SetFocused(false)
				s.confirmPass.SetFocused(true)
				s.SetStatus("Please re-enter password to confirm.")
				return false
			}
			if s.passInput.Value == "" {
				s.SetStatus("Password cannot be empty.")
				return false
			}
			if s.passInput.Value != s.confirmPass.Value {
				s.SetStatus("Passwords do not match! Please check and try again.")
				return false
			}
			if err := s.svc.InitializeVault(ctx, []byte(s.passInput.Value)); err != nil {
				s.SetStatus(fmt.Sprintf("Setup failed: %v", err))
				return false
			}
			s.passInput.Clear()
			s.confirmPass.Clear()
			s.mode = ModeGlobalProjects
			s.reloadProjects()
			s.SetStatus("Vault initialized successfully! Welcome to MayFly.")
			return false
		}
		if event.Type == terminal.KeyEscape {
			return true // Exit
		}
		if s.passInput.Focused {
			s.passInput.HandleKey(event)
		} else {
			s.confirmPass.HandleKey(event)
		}

	case ModeUnlock:
		if event.Type == terminal.KeyEnter {
			if err := s.svc.UnlockVault(ctx, []byte(s.passInput.Value)); err != nil {
				s.SetStatus("Invalid password. Try again.")
				return false
			}
			s.passInput.Clear()
			if s.selProject.ID != "" {
				s.mode = ModeProjectSecrets
				s.reloadSecrets()
			} else {
				s.mode = ModeGlobalProjects
				s.reloadProjects()
			}
			s.SetStatus("Vault unlocked.")
			return false
		}
		if event.Type == terminal.KeyEscape {
			return true // Exit
		}
		s.passInput.HandleKey(event)

	case ModeGlobalProjects:
		switch event.Type {
		case terminal.KeyEscape:
			return true // Exit TUI
		case terminal.KeyEnter:
			selected := s.projectGrid.SelectedCard()
			if selected != nil {
				s.selProject = selected.Project
				s.mode = ModeProjectSecrets
				s.reloadSecrets()
			}
			return false
		case terminal.KeyRune:
			switch event.Rune {
			case 'q', 'Q':
				return true
			case 'n', 'N': // Initialize current folder as project
				proj, err := s.svc.RegisterProject(ctx, s.currentDir)
				if err != nil {
					s.SetStatus(fmt.Sprintf("Init error: %v", err))
				} else {
					s.selProject = proj
					s.reloadProjects()
					s.SetStatus(fmt.Sprintf("Initialized project: %s", proj.CanonicalPath))
				}
				return false
			case 's', 'S': // Scanner
				s.prevMode = s.mode
				s.mode = ModeScan
				s.runScan()
				return false
			case 'a', 'A': // Audit Log
				s.prevMode = s.mode
				s.mode = ModeAudit
				s.loadAudit()
				return false
			case 'b', 'B': // Backup
				s.prevMode = s.mode
				s.mode = ModeBackup
				s.backupPath.SetValue(filepath.Join(s.currentDir, "mayfly-backup.json"))
				s.backupPath.SetFocused(true)
				return false
			}
		}
		s.projectGrid.HandleKey(event)

	case ModeProjectSecrets:
		switch event.Type {
		case terminal.KeyEscape:
			s.mode = ModeGlobalProjects
			s.reloadProjects()
			return false
		case terminal.KeyEnter: // Edit secret
			sel := s.secretsList.SelectedItem()
			if sel != nil {
				sec := sel.Data.(domain.Secret)
				s.editOrigName = sec.Name
				s.secretName.SetValue(string(sec.Name))
				s.secretValue.SetValue(sec.Value)
				s.secretName.Focused = false
				s.secretValue.Focused = true
				s.mode = ModeEditSecret
			}
			return false
		case terminal.KeyRune:
			switch event.Rune {
			case 'q', 'Q':
				s.mode = ModeGlobalProjects
				s.reloadProjects()
				return false
			case 'c', 'C': // Copy to clipboard
				sel := s.secretsList.SelectedItem()
				if sel != nil {
					sec := sel.Data.(domain.Secret)
					_ = terminal.CopyToClipboard(sec.Value, nil)
					s.SetStatus(fmt.Sprintf("✓ Copied '%s' to clipboard!", sec.Name))
				}
				return false
			case 'v', 'V': // Toggle reveal
				s.revealValue = !s.revealValue
				s.reloadSecrets()
				return false
			case 'n', 'N': // Add new secret
				s.editOrigName = ""
				s.secretName.Clear()
				s.secretValue.Clear()
				s.secretName.Focused = true
				s.secretValue.Focused = false
				s.mode = ModeEditSecret
				return false
			case 'd', 'D': // Delete secret
				sel := s.secretsList.SelectedItem()
				if sel != nil {
					sec := sel.Data.(domain.Secret)
					s.confirmDlg.Title = "Delete Secret"
					s.confirmDlg.Message = fmt.Sprintf("Delete '%s' from project vault?", sec.Name)
					s.confirmDlg.Confirm = false
					s.confirmDlg.Active = true
					s.mode = ModeDeleteConfirm
				}
				return false
			case 's', 'S':
				s.prevMode = s.mode
				s.mode = ModeScan
				s.runScan()
				return false
			case 'a', 'A':
				s.prevMode = s.mode
				s.mode = ModeAudit
				s.loadAudit()
				return false
			}
		}
		s.secretsList.HandleKey(event)

	case ModeEditSecret:
		if event.Type == terminal.KeyTab {
			s.secretName.Focused = !s.secretName.Focused
			s.secretValue.Focused = !s.secretValue.Focused
			return false
		}
		if event.Type == terminal.KeyEscape {
			s.mode = ModeProjectSecrets
			return false
		}
		if event.Type == terminal.KeyEnter {
			nameStr := s.secretName.Value
			valStr := s.secretValue.Value

			if s.editOrigName != "" && s.editOrigName != domain.SecretName(nameStr) {
				_ = s.svc.DeleteSecret(ctx, s.selProject.ID, s.editOrigName)
			}

			if err := s.svc.SetSecret(ctx, s.selProject.ID, domain.SecretName(nameStr), valStr); err != nil {
				s.SetStatus(fmt.Sprintf("Save error: %v", err))
				return false
			}

			s.mode = ModeProjectSecrets
			s.reloadSecrets()
			s.SetStatus(fmt.Sprintf("✓ Saved secret '%s'", nameStr))
			return false
		}
		if s.secretName.Focused {
			s.secretName.HandleKey(event)
		} else {
			s.secretValue.HandleKey(event)
		}

	case ModeDeleteConfirm:
		if event.Type == terminal.KeyEscape {
			s.confirmDlg.Active = false
			s.mode = ModeProjectSecrets
			return false
		}
		if event.Type == terminal.KeyEnter {
			if s.confirmDlg.Confirm {
				sel := s.secretsList.SelectedItem()
				if sel != nil {
					sec := sel.Data.(domain.Secret)
					_ = s.svc.DeleteSecret(ctx, s.selProject.ID, sec.Name)
					s.SetStatus(fmt.Sprintf("✓ Deleted secret '%s'", sec.Name))
				}
			}
			s.confirmDlg.Active = false
			s.mode = ModeProjectSecrets
			s.reloadSecrets()
			return false
		}
		s.confirmDlg.HandleKey(event)

	case ModeScan:
		if event.Type == terminal.KeyEscape || event.Type == terminal.KeyEnter {
			s.mode = s.prevMode
			return false
		}
		s.scanList.HandleKey(event)

	case ModeAudit:
		if event.Type == terminal.KeyEscape || event.Type == terminal.KeyEnter {
			s.mode = s.prevMode
			return false
		}
		s.auditList.HandleKey(event)

	case ModeBackup:
		if event.Type == terminal.KeyEscape {
			s.mode = s.prevMode
			return false
		}
		if event.Type == terminal.KeyEnter {
			target := s.backupPath.Value
			if err := s.svc.ExportBackup(ctx, target); err != nil {
				s.SetStatus(fmt.Sprintf("Backup error: %v", err))
			} else {
				s.SetStatus(fmt.Sprintf("✓ Backup exported to %s", target))
			}
			s.mode = s.prevMode
			return false
		}
		s.backupPath.HandleKey(event)
	}

	return false
}

func (s *Screens) runScan() {
	ctx := context.Background()
	dir := s.currentDir
	if s.selProject.CanonicalPath != "" {
		dir = s.selProject.CanonicalPath
	}

	findings, err := s.svc.Scan(ctx, dir)
	if err != nil {
		s.SetStatus(fmt.Sprintf("Scan error: %v", err))
		return
	}

	var items []widget.ListItem
	for _, f := range findings {
		items = append(items, widget.ListItem{
			Primary:   fmt.Sprintf("[%s] %s:%d", f.Severity, f.Path, f.Line),
			Secondary: f.Message,
		})
	}

	s.scanList.SetItems(items)
	if len(findings) == 0 {
		s.SetStatus("✓ No plaintext credential leaks detected!")
	}
}

func (s *Screens) loadAudit() {
	ctx := context.Background()
	events, err := s.svc.AuditTrail(ctx)
	if err != nil {
		s.SetStatus(fmt.Sprintf("Audit error: %v", err))
		return
	}

	var items []widget.ListItem
	for i := len(events) - 1; i >= 0; i-- { // Most recent first
		ev := events[i]
		items = append(items, widget.ListItem{
			Primary:   fmt.Sprintf("#%d %s", ev.Sequence, ev.Action),
			Secondary: ev.At.Format("15:04:05") + " " + ev.Secret + " " + ev.Command,
			Extra:     ev.Hash[:8] + "...",
		})
	}
	s.auditList.SetItems(items)
}

// Draw renders the active screen layout onto the frame.
func (s *Screens) Draw(frame *terminal.Frame) {
	bounds := frame.Bounds()
	frame.Clear(terminal.Style{})

	// Layout: Header (3 rows), Body (Flexible), Status (1 row), Footer (1 row)
	split := layout.Split(layout.DirVertical, bounds, []layout.Constraint{
		layout.Fixed(3),
		layout.Flexible(),
		layout.Fixed(1),
	})

	headerRect := split[0]
	bodyRect := split[1]
	footerRect := split[2]

	// Draw Header
	frame.DrawBox(headerRect, terminal.Style{Foreground: terminal.ColorBrightCyan}, fmt.Sprintf("MayFly v%s — Zero-Dependency Secrets Workspace", domain.Version))
	headerText := "Secure RAM Environment Injection · AES-256-GCM Vault"
	frame.DrawText(headerRect.Min.Row+1, headerRect.Min.Column+3, terminal.Style{Foreground: terminal.ColorBrightWhite}, headerText)

	// Draw Body based on Mode
	switch s.mode {
	case ModeFirstRunSetup:
		inputRects := layout.Split(layout.DirVertical, bodyRect, []layout.Constraint{
			layout.Fixed(1),
			layout.Fixed(1),
			layout.Fixed(3),
			layout.Fixed(1),
			layout.Fixed(3),
			layout.Flexible(),
		})
		frame.DrawText(inputRects[0].Min.Row, inputRects[0].Min.Column+3, terminal.Style{Foreground: terminal.ColorBrightYellow, Attributes: terminal.AttrBold}, "FIRST-TIME SETUP: CREATE VAULT MASTER PASSWORD")
		frame.DrawText(inputRects[1].Min.Row, inputRects[1].Min.Column+3, terminal.Style{Foreground: terminal.ColorBrightBlack, Attributes: terminal.AttrDim}, "Choose a password to encrypt your local secrets vault (~/.mayfly/vault.enc)")
		s.passInput.Draw(frame, inputRects[2])
		s.confirmPass.Draw(frame, inputRects[4])

	case ModeUnlock:
		inputRects := layout.Split(layout.DirVertical, bodyRect, []layout.Constraint{
			layout.Fixed(3),
			layout.Fixed(3),
			layout.Flexible(),
		})
		s.passInput.Draw(frame, inputRects[1])

	case ModeGlobalProjects:
		s.projectGrid.Draw(frame, bodyRect)

	case ModeProjectSecrets:
		s.secretsList.Draw(frame, bodyRect)

	case ModeEditSecret:
		inputRects := layout.Split(layout.DirVertical, bodyRect, []layout.Constraint{
			layout.Fixed(2),
			layout.Fixed(3),
			layout.Fixed(1),
			layout.Fixed(3),
			layout.Flexible(),
		})
		title := "ADD NEW SECRET"
		if s.editOrigName != "" {
			title = fmt.Sprintf("EDIT SECRET '%s'", s.editOrigName)
		}
		frame.DrawText(inputRects[0].Min.Row+1, inputRects[0].Min.Column+3, terminal.Style{Foreground: terminal.ColorBrightCyan, Attributes: terminal.AttrBold}, title)
		s.secretName.Draw(frame, inputRects[1])
		s.secretValue.Draw(frame, inputRects[3])

	case ModeDeleteConfirm:
		s.secretsList.Draw(frame, bodyRect)
		s.confirmDlg.Draw(frame, bodyRect)

	case ModeScan:
		s.scanList.Draw(frame, bodyRect)

	case ModeAudit:
		s.auditList.Draw(frame, bodyRect)

	case ModeBackup:
		inputRects := layout.Split(layout.DirVertical, bodyRect, []layout.Constraint{
			layout.Fixed(2),
			layout.Fixed(3),
			layout.Flexible(),
		})
		frame.DrawText(inputRects[0].Min.Row+1, inputRects[0].Min.Column+3, terminal.Style{Foreground: terminal.ColorBrightCyan, Attributes: terminal.AttrBold}, "EXPORT ENCRYPTED VAULT BACKUP")
		s.backupPath.Draw(frame, inputRects[1])
	}

	// Draw Footer with context-sensitive key shortcuts
	footerText := ""
	switch s.mode {
	case ModeGlobalProjects:
		footerText = " [Enter] Open  [N] Init Dir  [S] Scan  [A] Audit  [B] Backup  [Q/Esc] Exit "
	case ModeProjectSecrets:
		footerText = " [C] Copy  [V] Reveal  [Enter] Edit  [N] Add  [D] Del  [S] Scan  [A] Audit  [Esc] Back "
	case ModeFirstRunSetup, ModeEditSecret:
		footerText = " [Tab] Switch Field  [Enter] Save  [Esc] Cancel "
	case ModeUnlock:
		footerText = " [Enter] Unlock  [Esc] Exit "
	case ModeScan, ModeAudit, ModeBackup:
		footerText = " [Esc] Return to Dashboard "
	default:
		footerText = " [Esc] Back "
	}

	statusMsg := s.status
	if statusMsg == "" {
		statusMsg = "Ready"
	}

	statusBar := widget.NewStatusBar(footerText, statusMsg)
	statusBar.Draw(frame, footerRect)
}
