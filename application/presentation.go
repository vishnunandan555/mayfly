package application

// ScreenSecret is the transitional, value-bearing record contract used by
// the existing application screens. It is intentionally separate from
// domain.Secret, whose metadata-only shape is safe for general listings. A
// future screen adapter can obtain these records from SecretService without
// exposing storage or crypto details to the TUI.
type ScreenSecret struct {
	Name  string
	Value string
}

// ScreenVault is the narrow application-facing contract currently consumed by
// the MayFly screens. It represents an already-open project vault; persistence
// and cryptography remain behind the implementation.
type ScreenVault interface {
	Secrets() ([]ScreenSecret, error)
	SetSecret(name, value string) error
	DeleteSecret(name string) error
}

// ScreenVaultOpener is the unlock boundary used by the current screens. The
// password is passed through and must not be retained or included in errors.
type ScreenVaultOpener interface {
	Unlock(password string) (ScreenVault, error)
}
