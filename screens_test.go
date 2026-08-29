package mayfly

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"mayfly/screen"
)

type memoryVault struct {
	items []Secret
	err   error
}

func (v *memoryVault) Secrets() ([]Secret, error) {
	if v.err != nil {
		return nil, v.err
	}
	return append([]Secret(nil), v.items...), nil
}

func (v *memoryVault) SetSecret(name, value string) error {
	if v.err != nil {
		return v.err
	}
	for index := range v.items {
		if v.items[index].Name == name {
			v.items[index].Value = value
			return nil
		}
	}
	v.items = append(v.items, Secret{Name: name, Value: value})
	return nil
}

func (v *memoryVault) DeleteSecret(name string) error {
	if v.err != nil {
		return v.err
	}
	for index := range v.items {
		if v.items[index].Name == name {
			v.items = append(v.items[:index], v.items[index+1:]...)
			return nil
		}
	}
	return errors.New("missing secret")
}

type testOpener struct {
	vault    Vault
	err      error
	password string
}

func (o *testOpener) Unlock(password string) (Vault, error) {
	o.password = password
	return o.vault, o.err
}

func runScreen(t *testing.T, screens *Screens, input string, size screen.Size) string {
	t.Helper()
	var output bytes.Buffer
	app := screen.NewApplication(screens.ApplicationOptions(&output, screen.NewInput(strings.NewReader(input)), size))
	if err := app.Run(); err != nil {
		t.Fatal(err)
	}
	return output.String()
}

func TestMainSecretsScreenMasksValuesAcrossSizes(t *testing.T) {
	vault := &memoryVault{items: []Secret{
		{Name: "OPENAI_API_KEY", Value: "openai-secret"},
		{Name: "DATABASE_URL", Value: "postgres://private"},
		{Name: "STRIPE_SECRET_KEY", Value: "stripe-secret"},
		{Name: "非常に長い名前とUnicode", Value: "unicode-secret"},
	}}

	for _, test := range []struct {
		name string
		size screen.Size
	}{
		{name: "normal", size: screen.Size{Rows: 24, Columns: 80}},
		{name: "narrow", size: screen.Size{Rows: 8, Columns: 20}},
		{name: "short", size: screen.Size{Rows: 3, Columns: 12}},
	} {
		t.Run(test.name, func(t *testing.T) {
			output := runScreen(t, NewScreensWithVault(vault), "\x03", test.size)
			if strings.Contains(output, "openai-secret") || strings.Contains(output, "postgres://private") || strings.Contains(output, "stripe-secret") {
				t.Fatalf("plaintext secret appeared in output: %q", output)
			}
			if !strings.Contains(output, "MayFly") {
				t.Fatalf("screen title missing from output: %q", output)
			}
		})
	}
}

func TestScreensHandleEmptyAndManyVaults(t *testing.T) {
	empty := NewScreensWithVault(&memoryVault{})
	output := runScreen(t, empty, "\x03", screen.Size{Rows: 10, Columns: 40})
	if !strings.Contains(output, "No secrets") {
		t.Fatalf("empty-vault message missing: %q", output)
	}

	items := make([]Secret, 50)
	for index := range items {
		items[index] = Secret{Name: "KEY_" + string(rune('A'+index%26)), Value: "hidden"}
	}
	many := NewScreensWithVault(&memoryVault{items: items})
	output = runScreen(t, many, "\x1b[B\x1b[B\x1b[6~\x1b[H\x1b[F\x03", screen.Size{Rows: 6, Columns: 30})
	if strings.Contains(output, "hidden") {
		t.Fatal("many-secret screen leaked a value")
	}
}

func TestScreensAddSecretAndKeyboardNavigation(t *testing.T) {
	vault := &memoryVault{}
	screens := NewScreensWithVault(vault)
	output := runScreen(t, screens, "nMY_KEY\tmy-secret\r\x03", screen.Size{Rows: 12, Columns: 50})
	if len(vault.items) != 1 || vault.items[0].Name != "MY_KEY" || vault.items[0].Value != "my-secret" {
		t.Fatalf("vault after add = %#v", vault.items)
	}
	if screens.Status() != "Secret saved" {
		t.Fatalf("status = %q, want save confirmation", screens.Status())
	}
	if strings.Contains(output, "my-secret") {
		t.Fatal("new secret appeared in rendered output")
	}
	if !strings.Contains(output, "Secret saved") {
		t.Fatalf("save status was not rendered: %q", output)
	}
}

func TestScreensEditAndDeleteConfirmation(t *testing.T) {
	vault := &memoryVault{items: []Secret{{Name: "EDIT_ME", Value: "before"}, {Name: "KEEP", Value: "also-hidden"}}}
	screens := NewScreensWithVault(vault)
	// Enter opens editing, Tab selects the masked value field, the new value is
	// submitted, then D opens the delete dialog and Enter confirms it.
	output := runScreen(t, screens, "\r\t\x15after\rD\r\x03", screen.Size{Rows: 14, Columns: 60})
	if len(vault.items) != 1 || vault.items[0].Name != "KEEP" {
		t.Fatalf("vault after edit/delete = %#v", vault.items)
	}
	if vault.items[0].Value != "also-hidden" {
		t.Fatal("unrelated secret changed")
	}
	if strings.Contains(output, "before") || strings.Contains(output, "after") || strings.Contains(output, "also-hidden") {
		t.Fatal("edit/delete flow leaked a secret value")
	}
}

func TestScreensUnlockAndNeverEchoPassword(t *testing.T) {
	vault := &memoryVault{items: []Secret{{Name: "KEY", Value: "vault-value"}}}
	opener := &testOpener{vault: vault}
	screens := NewScreens(opener)
	output := runScreen(t, screens, "unlock-password\r\x03", screen.Size{Rows: 12, Columns: 50})
	if opener.password != "unlock-password" {
		t.Fatalf("opener password = %q", opener.password)
	}
	if screens.Mode() != ModeSecrets {
		t.Fatalf("mode after unlock = %v, want secrets", screens.Mode())
	}
	if strings.Contains(output, "unlock-password") || strings.Contains(output, "vault-value") {
		t.Fatal("unlock password or vault value appeared in output")
	}
}

func TestScreensSanitizeVaultErrorsBeforeDisplaying(t *testing.T) {
	secretInError := "backend failed with secret=do-not-display"
	opener := &testOpener{err: errors.New(secretInError)}
	screens := NewScreens(opener)
	output := runScreen(t, screens, "pw\r\r\x03", screen.Size{Rows: 10, Columns: 50})
	if strings.Contains(output, secretInError) || strings.Contains(output, "do-not-display") {
		t.Fatal("vault error text leaked into UI")
	}
	if !strings.Contains(output, "Unable to unlock vault") {
		t.Fatal("safe generic unlock error was not displayed")
	}
}

func TestScreensKeepStatusMessagesNonSensitive(t *testing.T) {
	vault := &memoryVault{items: []Secret{{Name: "KEY", Value: "hidden-status-value"}}}
	screens := NewScreensWithVault(vault)
	if !screens.reload() {
		t.Fatal("reload failed")
	}
	screens.status = "Secret saved"
	if strings.Contains(screens.Status(), "hidden-status-value") {
		t.Fatal("status contains secret value")
	}
	output := runScreen(t, screens, "\x03", screen.Size{Rows: 2, Columns: 10})
	if strings.Contains(output, "hidden-status-value") {
		t.Fatal("tiny screen leaked secret value")
	}
}

func TestFormatSecretLineClipsNamesAndMasksUnicode(t *testing.T) {
	line := formatSecretLine("very-long-名前", "é界", 12)
	if screen.TextWidth(line) > 12 {
		t.Fatalf("line width = %d, want <= 12: %q", screen.TextWidth(line), line)
	}
	if strings.Contains(line, "é") || strings.Contains(line, "界") {
		t.Fatalf("line contains secret runes: %q", line)
	}
	if got := formatSecretLine("\x1b[31mKEY", "value", 20); strings.Contains(got, "\x1b") {
		t.Fatalf("ANSI-looking name was not sanitized: %q", got)
	}
}
