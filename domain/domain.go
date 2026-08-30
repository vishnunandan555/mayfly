// Package domain contains MayFly's application-neutral concepts and their
// validation rules. It deliberately contains no filesystem, crypto, process,
// or terminal code.
package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrInvalidProjectID   = errors.New("domain: invalid project id")
	ErrInvalidProject     = errors.New("domain: invalid project")
	ErrInvalidSecretName  = errors.New("domain: invalid secret name")
	ErrInvalidSecretInput = errors.New("domain: invalid secret input")
	ErrInvalidExecution   = errors.New("domain: invalid execution request")
	ErrInvalidScanFinding = errors.New("domain: invalid scan finding")
)

// ProjectID identifies a project without encoding how IDs are generated.
// Generation belongs to a project service and is intentionally not part of
// this domain package.
type ProjectID string

func (id ProjectID) Validate() error {
	if strings.TrimSpace(string(id)) == "" || !utf8.ValidString(string(id)) || containsControl(string(id)) {
		return fmt.Errorf("%w: empty or malformed value", ErrInvalidProjectID)
	}
	return nil
}

// Project is non-sensitive project identity and location metadata. Path is a
// hint for project lookup; it is not used by the domain package itself.
type Project struct {
	ID   ProjectID
	Name string
	Path string
}

func (p Project) Validate() error {
	if err := p.ID.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(p.Name) == "" || !utf8.ValidString(p.Name) || containsControl(p.Name) {
		return fmt.Errorf("%w: project name", ErrInvalidProject)
	}
	if !utf8.ValidString(p.Path) || strings.ContainsRune(p.Path, '\x00') || containsControl(p.Path) {
		return fmt.Errorf("%w: project path", ErrInvalidProject)
	}
	return nil
}

// SecretName is the user-visible and environment-facing name of a secret.
// It accepts Unicode and spaces, but rejects control characters, NUL, and '='
// because those characters cannot safely cross an environment boundary.
type SecretName string

func (name SecretName) Validate() error {
	value := string(name)
	if strings.TrimSpace(value) == "" || !utf8.ValidString(value) {
		return fmt.Errorf("%w: empty or malformed value", ErrInvalidSecretName)
	}
	if len(value) > 255 || containsControl(value) || strings.ContainsRune(value, '=') {
		return fmt.Errorf("%w: unsupported character or length", ErrInvalidSecretName)
	}
	return nil
}

// SecretMetadata contains non-secret information about a stored secret.
type SecretMetadata struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Secret is safe to use in listings and general UI state. Secret values are
// intentionally not included here.
type Secret struct {
	ProjectID ProjectID
	Name      SecretName
	Metadata  SecretMetadata
}

func (s Secret) Validate() error {
	if err := s.ProjectID.Validate(); err != nil {
		return err
	}
	return s.Name.Validate()
}

// SecretInput is used only at an explicit write boundary. Callers should
// clear Value when practical after the storage operation completes.
type SecretInput struct {
	ProjectID ProjectID
	Name      SecretName
	Value     string
}

func (s SecretInput) Validate() error {
	if err := s.ProjectID.Validate(); err != nil {
		return fmt.Errorf("%w: project: %v", ErrInvalidSecretInput, err)
	}
	if err := s.Name.Validate(); err != nil {
		return fmt.Errorf("%w: name: %v", ErrInvalidSecretInput, err)
	}
	if !utf8.ValidString(s.Value) || strings.ContainsRune(s.Value, '\x00') {
		return fmt.Errorf("%w: malformed value", ErrInvalidSecretInput)
	}
	return nil
}

// SecretMaterial is returned only by an explicit read operation. Its String
// method is deliberately redacted so accidental formatting does not expose
// the value. This is a safety aid, not a substitute for careful handling.
type SecretMaterial struct {
	Name  SecretName
	Value string
}

func (SecretMaterial) String() string { return "[REDACTED SECRET]" }

// AuditAction describes a security-relevant application action.
type AuditAction string

const (
	AuditProjectInitialized AuditAction = "PROJECT_INITIALIZED"
	AuditVaultUnlocked      AuditAction = "VAULT_UNLOCKED"
	AuditSecretCreated      AuditAction = "SECRET_CREATED"
	AuditSecretUpdated      AuditAction = "SECRET_UPDATED"
	AuditSecretDeleted      AuditAction = "SECRET_DELETED"
	AuditSecretInjected     AuditAction = "SECRET_INJECTED"
	AuditCommandStarted     AuditAction = "COMMAND_STARTED"
	AuditCommandExited      AuditAction = "COMMAND_EXITED"
	AuditScanCompleted      AuditAction = "SCAN_COMPLETED"

	// These names remain for compatibility with the earlier application API.
	AuditVaultOpened     AuditAction = AuditVaultUnlocked
	AuditSecretListed    AuditAction = "SECRET_LISTED"
	AuditSecretRead      AuditAction = "SECRET_READ"
	AuditSecretWritten   AuditAction = AuditSecretUpdated
	AuditCommandExecuted AuditAction = "COMMAND_EXECUTED"
)

// AuditEvent contains metadata only. It must never carry secret values.
type AuditEvent struct {
	At         time.Time
	Action     AuditAction
	ProjectID  ProjectID
	Secret     SecretName
	Command    string
	ExitStatus *int
}

func (e AuditEvent) Validate() error {
	if e.At.IsZero() {
		return errors.New("domain: audit event time is required")
	}
	if e.Action == "" {
		return errors.New("domain: audit event action is required")
	}
	if !validAuditAction(e.Action) {
		return errors.New("domain: unsupported audit event action")
	}
	if e.ProjectID != "" {
		if err := e.ProjectID.Validate(); err != nil {
			return err
		}
	}
	if e.Secret != "" {
		if err := e.Secret.Validate(); err != nil {
			return err
		}
	}
	if !utf8.ValidString(e.Command) || containsControl(e.Command) {
		return errors.New("domain: audit command is malformed")
	}
	if e.ExitStatus != nil && *e.ExitStatus < 0 {
		return errors.New("domain: audit exit status is malformed")
	}
	return nil
}

func validAuditAction(action AuditAction) bool {
	switch action {
	case AuditProjectInitialized, AuditVaultUnlocked, AuditSecretCreated,
		AuditSecretUpdated, AuditSecretDeleted, AuditSecretInjected,
		AuditCommandStarted, AuditCommandExited, AuditScanCompleted,
		AuditSecretListed, AuditSecretRead, AuditCommandExecuted:
		return true
	default:
		return false
	}
}

// ExecutionRequest describes a child process request without containing the
// resolved secret values that will be passed to it.
type ExecutionRequest struct {
	ProjectID   ProjectID
	Command     []string
	SecretNames []SecretName
}

func (r ExecutionRequest) Validate() error {
	if err := r.ProjectID.Validate(); err != nil {
		return fmt.Errorf("%w: project: %v", ErrInvalidExecution, err)
	}
	if len(r.Command) == 0 || strings.TrimSpace(r.Command[0]) == "" {
		return fmt.Errorf("%w: command is required", ErrInvalidExecution)
	}
	for _, part := range r.Command {
		if !utf8.ValidString(part) || containsControl(part) {
			return fmt.Errorf("%w: malformed command argument", ErrInvalidExecution)
		}
	}
	seen := make(map[SecretName]struct{}, len(r.SecretNames))
	for _, name := range r.SecretNames {
		if err := name.Validate(); err != nil {
			return fmt.Errorf("%w: secret name: %v", ErrInvalidExecution, err)
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("%w: duplicate secret name", ErrInvalidExecution)
		}
		seen[name] = struct{}{}
	}
	return nil
}

// ScanSeverity classifies a scanner result without embedding scanner logic in
// the domain model.
type ScanSeverity string

const (
	SeverityInfo     ScanSeverity = "info"
	SeverityWarning  ScanSeverity = "warning"
	SeverityCritical ScanSeverity = "critical"
)

// ScanFinding contains a safe, location-oriented scanner result. Message
// must describe the finding without including a matching secret value.
type ScanFinding struct {
	Severity ScanSeverity
	Path     string
	Line     int
	Column   int
	Category string
	Message  string
}

func (f ScanFinding) Validate() error {
	if f.Severity != SeverityInfo && f.Severity != SeverityWarning && f.Severity != SeverityCritical {
		return fmt.Errorf("%w: severity", ErrInvalidScanFinding)
	}
	if !utf8.ValidString(f.Path) || strings.ContainsRune(f.Path, '\x00') || !utf8.ValidString(f.Category) || containsControl(f.Category) || strings.TrimSpace(f.Category) == "" || !utf8.ValidString(f.Message) || containsControl(f.Message) {
		return fmt.Errorf("%w: malformed text", ErrInvalidScanFinding)
	}
	if f.Line < 0 || f.Column < 0 {
		return fmt.Errorf("%w: negative location", ErrInvalidScanFinding)
	}
	return nil
}

func containsControl(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
