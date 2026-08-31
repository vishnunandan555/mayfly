package domain

import (
	"errors"
	"strings"
	"time"
	"unicode"
)

var (
	ErrInvalidSecretName   = errors.New("domain: invalid secret name (must be 1-128 alphanumeric/underscore characters, starting with a letter or underscore)")
	ErrInvalidProjectID    = errors.New("domain: invalid project ID")
	ErrInvalidProjectPath  = errors.New("domain: invalid project path")
	ErrInvalidBackupFile   = errors.New("domain: invalid or corrupted backup file")
	ErrSecretNotFound      = errors.New("domain: secret not found")
	ErrProjectNotFound     = errors.New("domain: project not found")
	ErrProjectExists       = errors.New("domain: project already initialized")
	ErrVaultLocked         = errors.New("domain: vault is locked")
	ErrWrongPassword       = errors.New("domain: wrong vault master password")
)

// SecretName represents a validated environment variable name.
type SecretName string

func (n SecretName) Validate() error {
	s := strings.TrimSpace(string(n))
	if len(s) == 0 || len(s) > 128 {
		return ErrInvalidSecretName
	}
	for i, r := range s {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return ErrInvalidSecretName
			}
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return ErrInvalidSecretName
		}
	}
	return nil
}

func (n SecretName) String() string {
	return string(n)
}

// Secret represents a secret associated with a project.
type Secret struct {
	Name  SecretName `json:"name"`
	Value string     `json:"value,omitempty"`
}


// Project represents a registered project folder.
type Project struct {
	ID            string    `json:"id"`
	CanonicalPath string    `json:"canonical_path"`
	Device        uint64    `json:"device"`
	Inode         uint64    `json:"inode"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AuditAction defines the operation recorded in the audit trail.
type AuditAction string

const (
	ActionVaultInitialized     AuditAction = "VAULT_INITIALIZED"
	ActionVaultUnlocked        AuditAction = "VAULT_UNLOCKED"
	ActionVaultPasswordRotated AuditAction = "VAULT_PASSWORD_ROTATED"
	ActionProjectInit          AuditAction = "PROJECT_INITIALIZED"
	ActionProjectMigrated      AuditAction = "PROJECT_MIGRATED"
	ActionProjectDeleted       AuditAction = "PROJECT_DELETED"
	ActionSecretSet            AuditAction = "SECRET_SET"
	ActionSecretGet            AuditAction = "SECRET_GET"
	ActionSecretDeleted        AuditAction = "SECRET_DELETED"
	ActionSecretImported       AuditAction = "SECRET_IMPORTED"
	ActionSecretInjected       AuditAction = "SECRET_INJECTED"
	ActionCommandExited        AuditAction = "COMMAND_EXITED"
	ActionScanCompleted        AuditAction = "SCAN_COMPLETED"
	ActionBackupCreated        AuditAction = "BACKUP_CREATED"
	ActionBackupRestored       AuditAction = "BACKUP_RESTORED"
)

// AuditEvent represents a tamper-evident audit record.
type AuditEvent struct {
	Sequence     uint64      `json:"seq"`
	At           time.Time   `json:"at"`
	Action       AuditAction `json:"action"`
	ProjectID    string      `json:"project_id,omitempty"`
	Secret       string      `json:"secret,omitempty"`
	Command      string      `json:"command,omitempty"`
	ExitStatus   *int        `json:"exit_status,omitempty"`
	PreviousHash string      `json:"prev_hash"`
	Hash         string      `json:"hash"`
}

// ScanSeverity represents the threat level of a detected leak finding.
type ScanSeverity string

const (
	SeverityCritical ScanSeverity = "CRITICAL"
	SeverityWarning  ScanSeverity = "WARNING"
	SeverityInfo     ScanSeverity = "INFO"
)

// ScanFinding represents a detected plaintext credential or configuration leak.
type ScanFinding struct {
	Path     string       `json:"path"`
	Line     int          `json:"line"`
	Column   int          `json:"column"`
	Severity ScanSeverity `json:"severity"`
	Category string       `json:"category"`
	Message  string       `json:"message"`
}

// ExecutionRequest specifies a child process launch with in-memory injection.
type ExecutionRequest struct {
	ProjectID string
	Command   []string
	Dir       string
}

// ExecutionResult captures the exit status of an executed command.
type ExecutionResult struct {
	ExitCode int
}

// BackupSnapshot represents an exportable encrypted snapshot of the vault.
type BackupSnapshot struct {
	Version   int                  `json:"version"`
	CreatedAt time.Time            `json:"created_at"`
	Projects  map[string]Project   `json:"projects"`
	Payload   []byte               `json:"payload"` // Encrypted vault bytes
}
