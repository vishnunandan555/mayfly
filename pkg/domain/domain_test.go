package domain

import (
	"strings"
	"testing"
)

func TestSecretNameValidate(t *testing.T) {
	valid := []string{
		"API_KEY",
		"_PRIVATE_KEY",
		"stripeKey123",
		"PORT",
		"A",
		"_123",
		"MY_SUPER_LONG_SECRET_NAME_WITH_NUMBERS_1234567890_AND_MORE_UNDERSCORES",
	}

	for _, name := range valid {
		sn := SecretName(name)
		if err := sn.Validate(); err != nil {
			t.Errorf("expected valid secret name %q, got error: %v", name, err)
		}
		if sn.String() != name {
			t.Errorf("expected String() to return %q, got %q", name, sn.String())
		}
	}

	invalid := []string{
		"",
		" ",
		"123_STARTS_WITH_DIGIT",
		"-STARTS_WITH_DASH",
		"CONTAINS-DASH",
		"CONTAINS.DOT",
		"CONTAINS SPACE",
		"KEY@VALUE",
		"KEY$NAME",
		strings.Repeat("A", 129), // Exceeds 128 characters
	}

	for _, name := range invalid {
		sn := SecretName(name)
		if err := sn.Validate(); err != ErrInvalidSecretName {
			t.Errorf("expected ErrInvalidSecretName for %q, got: %v", name, err)
		}
	}
}
