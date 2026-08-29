// tui-demo runs MayFly's application screens against an in-memory demo vault.
// It is intentionally a display/testing harness, not a persistence layer.
package main

import (
	"fmt"
	"os"

	"mayfly"
)

type demoVault struct {
	items []mayfly.Secret
}

func (v *demoVault) Secrets() ([]mayfly.Secret, error) {
	return append([]mayfly.Secret(nil), v.items...), nil
}

func (v *demoVault) SetSecret(name, value string) error {
	for index := range v.items {
		if v.items[index].Name == name {
			v.items[index].Value = value
			return nil
		}
	}
	v.items = append(v.items, mayfly.Secret{Name: name, Value: value})
	return nil
}

func (v *demoVault) DeleteSecret(name string) error {
	for index := range v.items {
		if v.items[index].Name == name {
			v.items = append(v.items[:index], v.items[index+1:]...)
			return nil
		}
	}
	return fmt.Errorf("demo secret not found")
}

type demoOpener struct{ vault mayfly.Vault }

func (o demoOpener) Unlock(string) (mayfly.Vault, error) { return o.vault, nil }

func main() {
	vault := &demoVault{items: []mayfly.Secret{
		{Name: "OPENAI_API_KEY", Value: "demo-openai-value"},
		{Name: "DATABASE_URL", Value: "demo-database-value"},
		{Name: "STRIPE_SECRET_KEY", Value: "demo-stripe-value"},
	}}
	screens := mayfly.NewScreens(demoOpener{vault: vault})
	if err := screens.RunIO(os.Stdin, os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "tui-demo: TUI stopped:", err)
	}
}
