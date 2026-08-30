package audit

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"mayfly/pkg/domain"
)

const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

var (
	ErrAuditCorrupt = errors.New("audit: log file integrity check failed (broken hash chain or corrupt record)")
	ErrAuditFailed  = errors.New("audit: failed to record audit event")
)

type Log struct {
	mu       sync.Mutex
	path     string
	lastHash string
	sequence uint64
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mayfly", "audit.log"), nil
}

func New(path string) (*Log, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}

	l := &Log{
		path:     path,
		lastHash: GenesisHash,
		sequence: 0,
	}

	if err := l.initOrVerify(); err != nil {
		return nil, err
	}

	return l, nil
}

func (l *Log) Path() string {
	return l.path
}

func (l *Log) initOrVerify() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	dir := filepath.Dir(l.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	lastHash, seq, err := l.scanAndVerifyLocked()
	if err != nil {
		return err
	}

	l.lastHash = lastHash
	l.sequence = seq
	return nil
}

func (l *Log) scanAndVerifyLocked() (string, uint64, error) {
	file, err := os.Open(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return GenesisHash, 0, nil
		}
		return "", 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	prevHash := GenesisHash
	var seq uint64 = 0

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event domain.AuditEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return "", 0, fmt.Errorf("%w: invalid JSON format", ErrAuditCorrupt)
		}

		seq++
		if event.Sequence != seq {
			return "", 0, fmt.Errorf("%w: sequence mismatch at %d (expected %d, got %d)", ErrAuditCorrupt, seq, seq, event.Sequence)
		}

		if event.PreviousHash != prevHash {
			return "", 0, fmt.Errorf("%w: broken previous hash link at sequence %d", ErrAuditCorrupt, seq)
		}

		computedHash := computeHash(event)
		if computedHash != event.Hash {
			return "", 0, fmt.Errorf("%w: invalid hash checksum at sequence %d", ErrAuditCorrupt, seq)
		}

		prevHash = event.Hash
	}

	if err := scanner.Err(); err != nil {
		return "", 0, err
	}

	return prevHash, seq, nil
}

func computeHash(event domain.AuditEvent) string {
	exitStr := ""
	if event.ExitStatus != nil {
		exitStr = fmt.Sprintf("%d", *event.ExitStatus)
	}

	raw := fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s|%s",
		event.Sequence,
		event.At.UTC().Format(time.RFC3339Nano),
		event.Action,
		event.ProjectID,
		event.Secret,
		event.Command,
		exitStr,
		event.PreviousHash,
	)

	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// Record appends a tamper-evident audit event to the log.
func (l *Log) Record(ctx context.Context, action domain.AuditAction, projectID, secret, command string, exitStatus *int) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.sequence++
	now := time.Now().UTC()

	event := domain.AuditEvent{
		Sequence:     l.sequence,
		At:           now,
		Action:       action,
		ProjectID:    projectID,
		Secret:       secret,
		Command:      command,
		ExitStatus:   exitStatus,
		PreviousHash: l.lastHash,
	}

	event.Hash = computeHash(event)
	l.lastHash = event.Hash

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}

	return file.Sync()
}

// Verify mathematically validates the entire audit log hash chain.
func (l *Log) Verify(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, _, err := l.scanAndVerifyLocked()
	return err
}

// Events returns all recorded audit events in chronological order.
func (l *Log) Events(ctx context.Context) ([]domain.AuditEvent, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	file, err := os.Open(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var events []domain.AuditEvent
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event domain.AuditEvent
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}
		events = append(events, event)
	}

	return events, scanner.Err()
}
