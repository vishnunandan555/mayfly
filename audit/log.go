// Package audit provides MayFly's tamper-evident application audit log.
//
// The log is tamper-evident, not cryptographically immutable: an attacker who
// can rewrite both the log and all of its verification material can alter it.
// The checkpoint in each complete file makes accidental edits, omissions,
// reordering, and truncation detectable by Verify.
package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mayfly/application"
	"mayfly/domain"
)

const (
	formatVersion = 1
	header        = "MAYFLY-AUDIT/1\n"
)

var (
	ErrInvalidPath       = errors.New("audit: invalid log path")
	ErrMalformed         = errors.New("audit: malformed log")
	ErrUnsupported       = errors.New("audit: unsupported log version")
	ErrTampered          = errors.New("audit: log verification failed")
	ErrPersistenceFailed = application.ErrPersistenceFailed
)

// Log is a process-local synchronized audit log. Separate Log values that
// point to the same path are not coordinated; callers should share one Log
// instance, and separate MayFly processes should use an OS-level coordination
// layer in a future phase if concurrent writers are required.
type Log struct {
	path string
	mu   sync.Mutex
}

// New constructs an inert log. It does not create files or directories.
func New(path string) (*Log, error) {
	if strings.TrimSpace(path) == "" || strings.ContainsRune(path, '\x00') {
		return nil, ErrInvalidPath
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, ErrInvalidPath
	}
	return &Log{path: filepath.Clean(abs)}, nil
}

// DefaultPath returns the external per-user audit path.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", ErrInvalidPath
	}
	return filepath.Join(home, ".mayfly", "audit.log"), nil
}

// Path returns the configured path without reading it.
func (l *Log) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

// Record appends one validated metadata-only event. It writes a complete new
// log to an unpredictable same-directory temporary file, syncs it, and then
// renames it over the old log where the platform permits atomic replacement.
func (l *Log) Record(ctx context.Context, event domain.AuditEvent) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := event.Validate(); err != nil {
		return ErrMalformed
	}
	if l == nil {
		return ErrInvalidPath
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	events, head, err := l.readAndVerifyLocked()
	if err != nil {
		return err
	}
	record := makeEventRecord(event, uint64(len(events)+1), head)
	record.Hash = hashEvent(record)
	events = append(events, record)
	data, err := encodeLog(events)
	if err != nil {
		return ErrMalformed
	}
	if err := writeAtomic(l.path, data); err != nil {
		return fmt.Errorf("%w: %v", ErrPersistenceFailed, err)
	}
	return nil
}

// Verify checks the complete chain and checkpoint. A missing log is an empty
// valid log, which makes `mayfly audit verify` useful before the first event.
func (l *Log) Verify(ctx context.Context) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if l == nil {
		return ErrInvalidPath
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _, err := l.readAndVerifyLocked()
	return err
}

// Events returns verified safe metadata in chain order. It never returns
// secret values because domain.AuditEvent has no value field.
func (l *Log) Events(ctx context.Context) ([]domain.AuditEvent, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	if l == nil {
		return nil, ErrInvalidPath
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	records, _, err := l.readAndVerifyLocked()
	if err != nil {
		return nil, err
	}
	events := make([]domain.AuditEvent, 0, len(records))
	for _, record := range records {
		event, err := record.event()
		if err != nil {
			return nil, ErrMalformed
		}
		events = append(events, event)
	}
	return events, nil
}

type eventRecord struct {
	Kind         string `json:"kind"`
	Version      int    `json:"version"`
	Sequence     uint64 `json:"sequence"`
	Timestamp    string `json:"timestamp"`
	Event        string `json:"event"`
	ProjectID    string `json:"project_id,omitempty"`
	Secret       string `json:"secret,omitempty"`
	Command      string `json:"command,omitempty"`
	ExitStatus   *int   `json:"exit_status,omitempty"`
	PreviousHash string `json:"previous_hash"`
	Hash         string `json:"hash"`
}

type checkpointRecord struct {
	Kind    string `json:"kind"`
	Version int    `json:"version"`
	Count   uint64 `json:"count"`
	Head    string `json:"head"`
}

func makeEventRecord(event domain.AuditEvent, sequence uint64, previous string) eventRecord {
	record := eventRecord{
		Kind:         "event",
		Version:      formatVersion,
		Sequence:     sequence,
		Timestamp:    event.At.UTC().Format(time.RFC3339Nano),
		Event:        string(event.Action),
		ProjectID:    string(event.ProjectID),
		Secret:       string(event.Secret),
		Command:      event.Command,
		PreviousHash: previous,
	}
	if event.ExitStatus != nil {
		status := *event.ExitStatus
		record.ExitStatus = &status
	}
	return record
}

func (r eventRecord) event() (domain.AuditEvent, error) {
	at, err := time.Parse(time.RFC3339Nano, r.Timestamp)
	if err != nil || at.UTC().Format(time.RFC3339Nano) != r.Timestamp {
		return domain.AuditEvent{}, ErrMalformed
	}
	event := domain.AuditEvent{
		At:         at,
		Action:     domain.AuditAction(r.Event),
		ProjectID:  domain.ProjectID(r.ProjectID),
		Secret:     domain.SecretName(r.Secret),
		Command:    r.Command,
		ExitStatus: r.ExitStatus,
	}
	if err := event.Validate(); err != nil {
		return domain.AuditEvent{}, ErrMalformed
	}
	return event, nil
}

func hashEvent(record eventRecord) string {
	record.Hash = ""
	data, _ := json.Marshal(record)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func encodeLog(events []eventRecord) ([]byte, error) {
	var output []byte
	output = append(output, header...)
	for _, event := range events {
		line, err := json.Marshal(event)
		if err != nil {
			return nil, err
		}
		output = append(output, line...)
		output = append(output, '\n')
	}
	checkpoint := checkpointRecord{Kind: "checkpoint", Version: formatVersion, Count: uint64(len(events))}
	if len(events) > 0 {
		checkpoint.Head = events[len(events)-1].Hash
	}
	line, err := json.Marshal(checkpoint)
	if err != nil {
		return nil, err
	}
	output = append(output, line...)
	output = append(output, '\n')
	return output, nil
}

func (l *Log) readAndVerifyLocked() ([]eventRecord, string, error) {
	data, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("audit: read log: %w", err)
	}
	if len(data) == 0 {
		return nil, "", ErrMalformed
	}
	if !strings.HasPrefix(string(data), header) {
		return nil, "", ErrUnsupported
	}
	body := data[len(header):]
	if len(body) == 0 {
		return nil, "", nil
	}
	if body[len(body)-1] != '\n' {
		return nil, "", ErrMalformed
	}
	lines := strings.Split(string(body[:len(body)-1]), "\n")
	if len(lines) == 0 || lines[len(lines)-1] == "" {
		return nil, "", ErrMalformed
	}
	events := make([]eventRecord, 0, len(lines)-1)
	previous := ""
	for index, line := range lines[:len(lines)-1] {
		var record eventRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, "", ErrMalformed
		}
		if !canonicalEqual(record, line) || record.Kind != "event" || record.Version != formatVersion || record.Sequence != uint64(index+1) || record.PreviousHash != previous || record.Hash == "" || !validHash(record.Hash) {
			return nil, "", ErrTampered
		}
		if _, err := record.event(); err != nil {
			return nil, "", ErrMalformed
		}
		if hashEvent(record) != record.Hash {
			return nil, "", ErrTampered
		}
		events = append(events, record)
		previous = record.Hash
	}
	var checkpoint checkpointRecord
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &checkpoint); err != nil {
		return nil, "", ErrMalformed
	}
	if !canonicalEqual(checkpoint, lines[len(lines)-1]) {
		return nil, "", ErrTampered
	}
	if checkpoint.Kind != "checkpoint" || checkpoint.Version != formatVersion || checkpoint.Count != uint64(len(events)) || checkpoint.Head != previous {
		return nil, "", ErrTampered
	}
	return events, previous, nil
}

func canonicalEqual(value any, line string) bool {
	data, err := json.Marshal(value)
	return err == nil && string(data) == line
}

func validHash(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func writeAtomic(path string, data []byte) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".audit-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	removeTemporary := true
	defer func() {
		_ = temporary.Close()
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0600); err != nil {
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	removeTemporary = false
	if directoryFile, err := os.Open(directory); err == nil {
		_ = directoryFile.Sync()
		_ = directoryFile.Close()
	}
	return nil
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

var _ application.AuditService = (*Log)(nil)
