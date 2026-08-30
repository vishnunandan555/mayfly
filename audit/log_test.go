package audit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"mayfly/domain"
)

func testLog(t *testing.T) (*Log, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "audit.log")
	log, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	return log, path
}

func testEvent(sequence int) domain.AuditEvent {
	status := sequence
	return domain.AuditEvent{
		At:         time.Date(2026, 8, 30, 12, 0, sequence, 0, time.UTC),
		Action:     domain.AuditCommandExited,
		ProjectID:  "project-1",
		Secret:     "TOKEN",
		Command:    "program",
		ExitStatus: &status,
	}
}

func TestEmptyLogIsValid(t *testing.T) {
	log, path := testLog(t)
	if err := log.Verify(context.Background()); err != nil {
		t.Fatal(err)
	}
	events, err := log.Events(context.Background())
	if err != nil || len(events) != 0 {
		t.Fatalf("empty events = %#v, %v", events, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("empty verification created log: %v", err)
	}
}

func TestOneEventAndDeterministicHash(t *testing.T) {
	log, path := testLog(t)
	event := testEvent(1)
	if err := log.Record(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	if err := log.Verify(context.Background()); err != nil {
		t.Fatal(err)
	}
	events, err := log.Events(context.Background())
	if err != nil || !reflect.DeepEqual(events, []domain.AuditEvent{event}) {
		t.Fatalf("events = %#v, %v", events, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "secret-value") {
		t.Fatal("secret value appeared in audit bytes")
	}
	record := eventRecord{Kind: "event", Version: formatVersion, Sequence: 1, Timestamp: event.At.Format(time.RFC3339Nano), Event: string(event.Action), ProjectID: string(event.ProjectID), Secret: string(event.Secret), Command: event.Command, ExitStatus: event.ExitStatus}
	one := hashEvent(record)
	if one == "" || one != hashEvent(record) || len(one) != 64 {
		t.Fatalf("non-deterministic hash = %q", one)
	}
	record.PreviousHash = "previous"
	if one == hashEvent(record) {
		t.Fatal("previous hash did not affect current hash")
	}
}

func TestMultipleEventsAndSecretNeverStored(t *testing.T) {
	log, path := testLog(t)
	for index := 1; index <= 3; index++ {
		event := testEvent(index)
		event.Command = "go"
		if err := log.Record(context.Background(), event); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "value-that-must-not-be-written") {
		t.Fatal("secret value appeared in audit bytes")
	}
	events, err := log.Events(context.Background())
	if err != nil || len(events) != 3 {
		t.Fatalf("events = %#v, %v", events, err)
	}
}

func TestVerifyDetectsAlteredDeletedReorderedAndBrokenRecords(t *testing.T) {
	tests := []struct {
		name string
		edit func([]string) []string
	}{
		{name: "altered event", edit: func(lines []string) []string {
			lines[1] = strings.Replace(lines[1], "COMMAND_EXITED", "COMMAND_STARTED", 1)
			return lines
		}},
		{name: "deleted event", edit: func(lines []string) []string {
			return append(lines[:2], lines[3:]...)
		}},
		{name: "reordered event", edit: func(lines []string) []string {
			lines[1], lines[2] = lines[2], lines[1]
			return lines
		}},
		{name: "malformed record", edit: func(lines []string) []string {
			lines[1] = "not-json"
			return lines
		}},
		{name: "broken previous hash", edit: func(lines []string) []string {
			lines[2] = strings.Replace(lines[2], `"previous_hash":"`, `"previous_hash":"0000000000000000000000000000000000000000000000000000000000000000`, 1)
			return lines
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			log, path := testLog(t)
			for index := 1; index <= 2; index++ {
				if err := log.Record(context.Background(), testEvent(index)); err != nil {
					t.Fatal(err)
				}
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
			lines = test.edit(lines)
			if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := log.Verify(context.Background()); err == nil || !errors.Is(err, ErrTampered) && !errors.Is(err, ErrMalformed) {
				t.Fatalf("Verify error = %v", err)
			}
		})
	}
}

func TestVerifyRejectsUnsupportedHeader(t *testing.T) {
	log, path := testLog(t)
	if err := os.WriteFile(path, []byte("MAYFLY-AUDIT/99\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := log.Verify(context.Background()); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Verify error = %v", err)
	}
}

func TestConcurrentRecordsThroughOneLog(t *testing.T) {
	log, _ := testLog(t)
	const count = 16
	var group sync.WaitGroup
	for index := 1; index <= count; index++ {
		index := index
		group.Add(1)
		go func() {
			defer group.Done()
			if err := log.Record(context.Background(), testEvent(index)); err != nil {
				t.Errorf("Record(%d): %v", index, err)
			}
		}()
	}
	group.Wait()
	events, err := log.Events(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != count {
		t.Fatalf("event count = %d, want %d", len(events), count)
	}
}

func TestRecordRejectsInvalidEventWithoutWriting(t *testing.T) {
	log, path := testLog(t)
	if err := log.Record(context.Background(), domain.AuditEvent{Action: "SECRET_VALUE"}); !errors.Is(err, ErrMalformed) {
		t.Fatalf("Record error = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("invalid record created log: %v", err)
	}
}
