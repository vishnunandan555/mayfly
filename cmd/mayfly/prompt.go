package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"mayfly/pkg/application"
	"mayfly/pkg/tui/terminal"
)

// passwordFromStdin, when true, causes getMasterPassword to read the vault
// password from the first line of stdin instead of prompting interactively.
// Safer than MAYFLY_VAULT_PASSWORD env var for CI pipelines.
var passwordFromStdin bool

func getMasterPassword(svc *application.Service, in io.Reader, out, errOut io.Writer) ([]byte, error) {
	// Priority 1: --password-stdin flag (safer than env var for CI)
	if passwordFromStdin {
		p, err := readLine(in)
		if err != nil {
			return nil, fmt.Errorf("--password-stdin: failed to read password: %w", err)
		}
		if p == "" {
			return nil, errors.New("--password-stdin: password cannot be empty")
		}
		if !svc.VaultExists() {
			if err := svc.InitializeVault(context.Background(), []byte(p)); err != nil {
				return nil, err
			}
		}
		return []byte(p), nil
	}

	// Priority 2: MAYFLY_VAULT_PASSWORD env var (legacy CI support)
	if envPass := os.Getenv("MAYFLY_VAULT_PASSWORD"); envPass != "" {
		if !svc.VaultExists() {
			if err := svc.InitializeVault(context.Background(), []byte(envPass)); err != nil {
				return nil, err
			}
		}
		return []byte(envPass), nil
	}


	f, isFile := in.(*os.File)
	isTerm := isFile && terminal.IsTerminal(f)

	if !svc.VaultExists() {
		fmt.Fprint(errOut, "Create Master Password: ")
		var p1 string
		var err error
		if isTerm {
			passBytes, rErr := terminal.ReadPassword(f)
			if rErr != nil {
				return nil, rErr
			}
			p1 = string(passBytes)
			fmt.Fprintln(errOut)
		} else {
			p1, err = readLine(in)
			if err != nil {
				return nil, err
			}
		}
		if p1 == "" {
			return nil, errors.New("password cannot be empty")
		}

		if isTerm {
			fmt.Fprint(errOut, "Confirm Master Password: ")
			passBytes2, rErr := terminal.ReadPassword(f)
			if rErr != nil {
				return nil, rErr
			}
			fmt.Fprintln(errOut)
			p2 := string(passBytes2)
			if p1 != p2 {
				return nil, errors.New("passwords do not match")
			}
		}

		if err := svc.InitializeVault(context.Background(), []byte(p1)); err != nil {
			return nil, err
		}
		return []byte(p1), nil
	}

	fmt.Fprint(errOut, "Vault password: ")
	var p string
	var err error
	if isTerm {
		passBytes, rErr := terminal.ReadPassword(f)
		if rErr != nil {
			return nil, rErr
		}
		fmt.Fprintln(errOut)
		p = string(passBytes)
	} else {
		p, err = readLine(in)
		if err != nil {
			return nil, err
		}
	}
	return []byte(p), nil
}

func isTesting() bool {
	return os.Getenv("GO_TESTING") == "1" || flag.Lookup("test.v") != nil
}

func readLine(in io.Reader) (string, error) {
	var buf [1]byte
	var line []byte
	for {
		n, err := in.Read(buf[:])
		if n > 0 {
			b := buf[0]
			if b == '\n' {
				break
			}
			if b != '\r' {
				line = append(line, b)
			}
		}
		if err != nil {
			if len(line) > 0 && err == io.EOF {
				return string(line), nil
			}
			if len(line) == 0 && err == io.EOF {
				return "", io.EOF
			}
			return string(line), err
		}
	}
	return string(line), nil
}
