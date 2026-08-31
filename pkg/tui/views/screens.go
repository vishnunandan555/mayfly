package views

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
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
	ModeInitProject
	ModeDeleteProjectConfirm
	ModeDeleteProjectPassword
)

// Screens coordinates the interactive terminal user interface views and modals.
type Screens struct {
	svc             *application.Service
	mode            ScreenMode
	prevMode        ScreenMode
	status          string
	statusTimer     time.Time
	currentDir      string
	selProject      domain.Project
	secrets         []domain.Secret
	findings        []domain.ScanFinding
	auditEvents     []domain.AuditEvent

	// Widgets
	passInput       *widget.TextInput
	confirmPass     *widget.TextInput
	secretName      *widget.TextInput
	secretValue     *widget.TextInput
	backupPath      *widget.TextInput
	customInitPath  *widget.TextInput
	deletePassInput *widget.TextInput
	projectGrid     *widget.ProjectCardGrid
	secretsList     *widget.List
	scanList        *widget.List
	auditList       *widget.List
	confirmDlg      *widget.ConfirmDialog
	editOrigName    domain.SecretName
	revealValue     bool
	revealTimer     time.Time

	// Init & Delete state
	initChoice      int
	deleteProjectID string
	deleteProjName  string
}

// NewScreens initializes all screen widgets and determines the initial display mode.
func NewScreens(svc *application.Service, currentDir string) *Screens {
	s := &Screens{
		svc:             svc,
		currentDir:      currentDir,
		passInput:       widget.NewTextInput("Master Password", "Enter vault password...", true),
		confirmPass:     widget.NewTextInput("Confirm Password", "Re-enter password...", true),
		secretName:      widget.NewTextInput("Secret Name", "e.g. STRIPE_API_KEY", false),
		secretValue:     widget.NewTextInput("Secret Value", "Enter raw secret value...", true),
		backupPath:      widget.NewTextInput("Backup File Path", "~/.mayfly/backup.json", false),
		customInitPath:  widget.NewTextInput("Custom Directory Path", "/path/to/project", false),
		deletePassInput: widget.NewTextInput("Master Password", "Enter master password to delete project...", true),
		projectGrid:     widget.NewProjectCardGrid("Project Directories"),
		secretsList:     widget.NewList("Project Secrets"),
		scanList:        widget.NewList("Plaintext Leak Scan"),
		auditList:       widget.NewList("Audit Log"),
		confirmDlg:      widget.NewConfirmDialog("Confirm Delete", "Are you sure you want to delete this secret?"),
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

// SetProjectScoped focuses the TUI directly on a specific project workspace.
func (s *Screens) SetProjectScoped(proj domain.Project) {
	s.selProject = proj
	s.mode = ModeProjectSecrets
	s.reloadSecrets()
}

// SetStatus displays a temporary status message in the status bar.
func (s *Screens) SetStatus(msg string) {
	s.status = msg
	s.statusTimer = time.Now()
}

// Tick handles periodic timer updates such as 10-second secret auto-masking.
func (s *Screens) Tick() {
	if s.revealValue && !s.revealTimer.IsZero() {
		elapsed := time.Since(s.revealTimer)
		if elapsed >= 10*time.Second {
			s.revealValue = false
			s.revealTimer = time.Time{}
			s.SetStatus("[MASKED] Secrets auto-masked after 10s timeout")
			s.reloadSecrets()
		} else {
			remaining := 10 - int(elapsed.Seconds())
			s.status = fmt.Sprintf("[REVEALED] Secret visible (%ds remaining before auto-mask)", remaining)
		}
	} else if !s.statusTimer.IsZero() && time.Since(s.statusTimer) > 4*time.Second && !s.revealValue {
		s.status = ""
	}
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
			return true
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
			return true
		}
		s.passInput.HandleKey(event)

	case ModeGlobalProjects:
		switch event.Type {
		case terminal.KeyEscape:
			return true
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
			case 'n', 'N':
				s.mode = ModeInitProject
				s.initChoice = 0
				s.customInitPath.Clear()
				s.customInitPath.SetFocused(false)
				s.SetStatus("Select initialization target: [1] Current Directory or [2] Custom Path")
				return false
			case 'd', 'D':
				selected := s.projectGrid.SelectedCard()
				if selected == nil {
					s.SetStatus("Please select a project card first (use arrow keys).")
					return false
				}
				s.deleteProjectID = selected.Project.ID
				s.deleteProjName = filepath.Base(selected.Project.CanonicalPath)
				if s.deleteProjName == "/" || s.deleteProjName == "." {
					s.deleteProjName = selected.Project.CanonicalPath
				}
				s.confirmDlg.Title = "Delete Project Vault"
				s.confirmDlg.Message = fmt.Sprintf("Delete project '%s' and permanently wipe all its encrypted secrets?", s.deleteProjName)
				s.confirmDlg.Confirm = false
				s.confirmDlg.Active = true
				s.mode = ModeDeleteProjectConfirm
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
			case 'b', 'B':
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
		case terminal.KeyEnter:
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
			case 'c', 'C':
				sel := s.secretsList.SelectedItem()
				if sel != nil {
					sec := sel.Data.(domain.Secret)
					_ = terminal.CopyToClipboard(sec.Value, nil)
					s.SetStatus(fmt.Sprintf("[OK] Copied '%s' to clipboard", sec.Name))
				}
				return false
			case 'v', 'V':
				s.revealValue = !s.revealValue
				if s.revealValue {
					s.revealTimer = time.Now()
					s.status = "[REVEALED] Secret visible (10s remaining before auto-mask)"
				} else {
					s.revealTimer = time.Time{}
					s.SetStatus("[MASKED] Secrets masked")
				}
				s.reloadSecrets()
				return false
			case 'n', 'N':
				s.editOrigName = ""
				s.secretName.Clear()
				s.secretValue.Clear()
				s.secretName.Focused = true
				s.secretValue.Focused = false
				s.mode = ModeEditSecret
				return false
			case 'd', 'D':
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
			nameStr := strings.TrimSpace(s.secretName.Value)
			valStr := s.secretValue.Value

			// If user pasted KEY=VALUE into the name field, automatically split key and value:
			if k, v, ok := strings.Cut(nameStr, "="); ok {
				nameStr = strings.TrimSpace(k)
				if valStr == "" {
					valStr = v
					s.secretValue.SetValue(valStr)
				}
			}
			// Strip any accidental trailing '=' from secret name:
			nameStr = strings.TrimSuffix(nameStr, "=")
			s.secretName.SetValue(nameStr)

			// If user is on Name field and Value is still empty, advance focus to Value field:
			if s.secretName.Focused && valStr == "" && nameStr != "" {
				s.secretName.Focused = false
				s.secretValue.Focused = true
				return false
			}

			if nameStr == "" {
				s.SetStatus("Error: Secret name cannot be empty")
				return false
			}

			secName := domain.SecretName(nameStr)
			if err := secName.Validate(); err != nil {
				s.SetStatus(fmt.Sprintf("Error: %v", err))
				return false
			}

			if s.editOrigName != "" && s.editOrigName != secName {
				_ = s.svc.DeleteSecret(ctx, s.selProject.ID, s.editOrigName)
			}

			if err := s.svc.SetSecret(ctx, s.selProject.ID, secName, valStr); err != nil {
				s.SetStatus(fmt.Sprintf("Save error: %v", err))
				return false
			}

			s.secretName.Clear()
			s.secretValue.Clear()
			s.mode = ModeProjectSecrets
			s.reloadSecrets()
			s.SetStatus(fmt.Sprintf("[OK] Saved secret '%s'", nameStr))
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
					s.SetStatus(fmt.Sprintf("[OK] Deleted secret '%s'", sec.Name))
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
				s.SetStatus(fmt.Sprintf("[OK] Backup exported to %s", target))
			}
			s.mode = s.prevMode
			return false
		}
		s.backupPath.HandleKey(event)

	case ModeDeleteProjectConfirm:
		if event.Type == terminal.KeyEscape {
			s.confirmDlg.Active = false
			s.mode = ModeGlobalProjects
			return false
		}
		if event.Type == terminal.KeyEnter {
			if s.confirmDlg.Confirm {
				s.confirmDlg.Active = false
				s.deletePassInput.Clear()
				s.deletePassInput.SetFocused(true)
				s.mode = ModeDeleteProjectPassword
				s.SetStatus(fmt.Sprintf("Enter master password to authorize deleting '%s'.", s.deleteProjName))
				return false
			}
			s.confirmDlg.Active = false
			s.mode = ModeGlobalProjects
			return false
		}
		s.confirmDlg.HandleKey(event)

	case ModeDeleteProjectPassword:
		if event.Type == terminal.KeyEscape {
			s.deletePassInput.Clear()
			s.mode = ModeGlobalProjects
			return false
		}
		if event.Type == terminal.KeyEnter {
			pass := s.deletePassInput.Value
			if pass == "" {
				s.SetStatus("Password cannot be empty.")
				return false
			}
			if err := s.svc.UnlockVault(ctx, []byte(pass)); err != nil {
				s.SetStatus("Incorrect master password! Deletion aborted.")
				s.deletePassInput.Clear()
				s.mode = ModeGlobalProjects
				return false
			}
			if err := s.svc.DeleteProject(ctx, s.deleteProjectID); err != nil {
				s.SetStatus(fmt.Sprintf("Delete failed: %v", err))
			} else {
				s.SetStatus(fmt.Sprintf("[OK] Project '%s' and its secrets were deleted.", s.deleteProjName))
			}
			s.deletePassInput.Clear()
			s.deleteProjectID = ""
			s.mode = ModeGlobalProjects
			s.reloadProjects()
			return false
		}
		s.deletePassInput.HandleKey(event)

	case ModeInitProject:
		if event.Type == terminal.KeyEscape {
			s.customInitPath.Clear()
			s.mode = ModeGlobalProjects
			return false
		}
		if event.Type == terminal.KeyTab || event.Type == terminal.KeyUp || event.Type == terminal.KeyDown {
			s.initChoice = 1 - s.initChoice
			s.customInitPath.SetFocused(s.initChoice == 1)
			return false
		}
		if event.Type == terminal.KeyRune {
			if event.Rune == '1' && !s.customInitPath.Focused {
				s.initChoice = 0
				s.customInitPath.SetFocused(false)
				return false
			}
			if event.Rune == '2' && !s.customInitPath.Focused {
				s.initChoice = 1
				s.customInitPath.SetFocused(true)
				return false
			}
		}
		if event.Type == terminal.KeyEnter {
			targetDir := s.currentDir
			if s.initChoice == 1 {
				targetDir = strings.TrimSpace(s.customInitPath.Value)
				if targetDir == "" {
					s.SetStatus("Please enter a valid directory path.")
					return false
				}
			}
			proj, err := s.svc.RegisterProject(ctx, targetDir)
			if err != nil {
				s.SetStatus(fmt.Sprintf("Init error: %v", err))
			} else {
				s.SetStatus(fmt.Sprintf("[OK] Initialized project: %s", proj.CanonicalPath))
			}
			s.customInitPath.Clear()
			s.mode = ModeGlobalProjects
			s.reloadProjects()
			return false
		}
		if s.initChoice == 1 {
			s.customInitPath.HandleKey(event)
		}
	}

	return false
}

// Draw renders the current view or active modal dialog onto the frame.
func (s *Screens) Draw(frame *terminal.Frame) {
	bounds := frame.Bounds()
	frame.Clear(terminal.Style{})

	// Layout: Header (3 rows), Body (Flexible), Status/Footer (1 row)
	split := layout.Split(layout.DirVertical, bounds, []layout.Constraint{
		layout.Fixed(3),
		layout.Flexible(),
		layout.Fixed(1),
	})

	headerRect := split[0]
	bodyRect := split[1]
	footerRect := split[2]

	// Draw Header bar
	frame.DrawBox(headerRect, terminal.Style{Foreground: terminal.ColorBrightCyan}, fmt.Sprintf("MayFly v%s: Zero-Dependency Secrets Workspace", domain.Version))
	headerText := "Secure RAM Environment Injection · AES-256-GCM Vault"
	frame.DrawText(headerRect.Min.Row+1, headerRect.Min.Column+3, terminal.Style{Foreground: terminal.ColorBrightWhite}, headerText)

	// Draw Body based on active mode
	switch s.mode {
	case ModeFirstRunSetup:
		s.drawFirstRunSetup(frame, bodyRect)
	case ModeUnlock:
		s.drawUnlock(frame, bodyRect)
	case ModeGlobalProjects:
		s.drawDashboard(frame, bodyRect)
	case ModeProjectSecrets:
		s.drawProjectDetail(frame, bodyRect)
	case ModeEditSecret:
		s.drawEditSecret(frame, bodyRect)
	case ModeDeleteConfirm:
		s.drawDeleteConfirm(frame, bodyRect)
	case ModeScan:
		s.drawScan(frame, bodyRect)
	case ModeAudit:
		s.drawAudit(frame, bodyRect)
	case ModeBackup:
		s.drawBackup(frame, bodyRect)
	case ModeInitProject:
		s.drawInitProject(frame, bodyRect, bounds)
	case ModeDeleteProjectConfirm:
		s.drawDeleteProjectConfirm(frame, bodyRect)
	case ModeDeleteProjectPassword:
		s.drawDeleteProjectPassword(frame, bodyRect, bounds)
	}

	// Draw Footer with context-sensitive key shortcuts
	footerText := ""
	switch s.mode {
	case ModeGlobalProjects:
		footerText = " [Enter] Open  [N] Init Project  [D] Delete Project  [S] Scan  [A] Audit  [B] Backup  [Q/Esc] Exit "
	case ModeProjectSecrets:
		footerText = " [C] Copy  [V] Reveal  [Enter] Edit  [N] Add  [D] Del  [S] Scan  [A] Audit  [Esc] Back "
	case ModeFirstRunSetup, ModeEditSecret:
		footerText = " [Tab] Switch Field  [Enter] Save  [Esc] Cancel "
	case ModeUnlock:
		footerText = " [Enter] Unlock  [Esc] Exit "
	case ModeInitProject:
		footerText = " [1/2 or Tab] Toggle Option  [Enter] Confirm Init  [Esc] Cancel "
	case ModeDeleteProjectConfirm:
		footerText = " [Tab/Left/Right] Select  [Enter] Confirm  [Esc] Cancel "
	case ModeDeleteProjectPassword:
		footerText = " [Enter] Authorize Delete  [Esc] Cancel "
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
