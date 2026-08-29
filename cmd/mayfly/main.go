// mayfly is the application command-line entry point.
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"mayfly/application"
	"mayfly/domain"
	"mayfly/project"
	"mayfly/vault"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, input io.Reader, output, errorOutput io.Writer) int {
	if len(args) == 0 {
		usage(errorOutput)
		return 2
	}
	if args[0] == "init" {
		if err := runInit(args[1:], output, errorOutput); err != nil {
			_, _ = fmt.Fprintln(errorOutput, "mayfly init:", err)
			return 1
		}
		return 0
	}
	if args[0] != "set" && args[0] != "get" && args[0] != "list" && args[0] != "delete" {
		_, _ = fmt.Fprintf(errorOutput, "mayfly: unknown command %q\n", args[0])
		usage(errorOutput)
		return 2
	}
	runtime, err := newRuntime()
	if err != nil {
		_, _ = fmt.Fprintln(errorOutput, "mayfly:", err)
		return 1
	}
	if err := runtime.execute(context.Background(), args, input, output, errorOutput); err != nil {
		_, _ = fmt.Fprintln(errorOutput, "mayfly:", err)
		var usageErr usageError
		if errors.As(err, &usageErr) {
			usage(errorOutput)
			return 2
		}
		return 1
	}
	return 0
}

type commandRuntime struct {
	service *application.Service
	storage *vault.Storage
}

func newRuntime() (*commandRuntime, error) {
	registryPath, err := project.DefaultRegistryPath()
	if err != nil {
		return nil, err
	}
	registry, err := project.NewRegistry(registryPath)
	if err != nil {
		return nil, err
	}
	storage, err := vault.NewStorage(filepath.Join(filepath.Dir(registryPath), "vault.enc"), vault.Options{})
	if err != nil {
		return nil, err
	}
	return &commandRuntime{
		service: application.NewService(application.Dependencies{Projects: registry, Vault: storage}),
		storage: storage,
	}, nil
}

func (r *commandRuntime) execute(ctx context.Context, args []string, input io.Reader, output, errorOutput io.Writer) error {
	if r == nil || r.service == nil {
		return application.ErrMissingSecrets
	}
	if len(args) == 0 {
		return errorsWithUsage("command is required")
	}
	reader := bufio.NewReader(input)
	command := args[0]
	var name domain.SecretName
	if command == "list" {
		if len(args) != 1 {
			return errorsWithUsage("list takes no arguments")
		}
	} else {
		if len(args) != 2 || strings.TrimSpace(args[1]) == "" {
			return errorsWithUsage(command + " requires exactly one secret name")
		}
		name = domain.SecretName(args[1])
		if err := name.Validate(); err != nil {
			return application.ErrInvalidSecretName
		}
	}
	// Resolve the project before prompting or creating a vault. This prevents
	// an uninitialized directory from causing any vault-side state change.
	if _, err := r.service.CurrentProject(ctx); err != nil {
		return err
	}

	opened, err := r.open(ctx, reader, errorOutput, command == "set")
	if err != nil {
		return err
	}
	defer func() { _ = opened.Close() }()

	switch command {
	case "set":
		value, err := readLine(reader, errorOutput, "Secret value: ")
		if err != nil {
			return err
		}
		if err := opened.SetCurrentSecret(ctx, name, value); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(output, "Set %s\n", name)
		return nil
	case "get":
		material, err := opened.GetCurrentSecret(ctx, name)
		if err != nil {
			return err
		}
		// get is the explicit value-bearing command. Close the session before
		// writing the value, and do not include it in any status or error text.
		if err := opened.Close(); err != nil {
			return err
		}
		_, err = fmt.Fprintln(output, material.Value)
		return err
	case "list":
		secrets, err := opened.ListCurrentSecrets(ctx)
		if err != nil {
			return err
		}
		if err := opened.Close(); err != nil {
			return err
		}
		for _, secret := range secrets {
			if _, err := fmt.Fprintln(output, secret.Name); err != nil {
				return err
			}
		}
		return nil
	case "delete":
		answer, err := readLine(reader, errorOutput, "Delete secret? [y/N]: ")
		if err != nil {
			return err
		}
		if !strings.EqualFold(strings.TrimSpace(answer), "y") {
			_, _ = fmt.Fprintln(output, "Delete cancelled")
			return nil
		}
		if err := opened.DeleteCurrentSecret(ctx, name); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(output, "Deleted %s\n", name)
		return nil
	default:
		return errorsWithUsage("unknown command")
	}
}

func (r *commandRuntime) open(ctx context.Context, input *bufio.Reader, errorOutput io.Writer, allowInitialize bool) (*application.Service, error) {
	if r == nil || r.service == nil {
		return nil, application.ErrMissingVaultStorage
	}
	password, err := readLine(input, errorOutput, "Vault password: ")
	if err != nil {
		return nil, err
	}
	passwordBytes := []byte(password)
	defer clearBytes(passwordBytes)
	opened, err := r.service.OpenVault(ctx, passwordBytes)
	if err == nil {
		return opened, nil
	}
	if !allowInitialize || !errors.Is(err, application.ErrVaultMissing) {
		return nil, err
	}
	if r.storage == nil {
		return nil, application.ErrMissingVaultStorage
	}
	if err := r.storage.Initialize(passwordBytes); err != nil && !errors.Is(err, vault.ErrVaultExists) {
		return nil, err
	}
	return r.service.OpenVault(ctx, passwordBytes)
}

func runInit(args []string, output, errorOutput io.Writer) error {
	defaultRegistry, err := project.DefaultRegistryPath()
	if err != nil {
		return err
	}
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(errorOutput)
	root := flags.String("path", ".", "project directory to initialize")
	registryPath := flags.String("registry", defaultRegistry, "external MayFly project registry path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q", flags.Arg(0))
	}
	registry, err := project.NewRegistry(*registryPath)
	if err != nil {
		return err
	}
	identity, created, err := registry.Initialize(*root)
	if err != nil {
		return err
	}
	if created {
		_, _ = fmt.Fprintf(output, "Initialized project %s at %s\n", identity.ID, identity.Path)
	} else {
		_, _ = fmt.Fprintf(output, "Project %s is already initialized at %s\n", identity.ID, identity.Path)
	}
	return nil
}

func readLine(input *bufio.Reader, prompt io.Writer, text string) (string, error) {
	_, _ = io.WriteString(prompt, text)
	value, err := input.ReadString('\n')
	if err != nil && len(value) == 0 {
		return "", err
	}
	value = strings.TrimSuffix(value, "\n")
	value = strings.TrimSuffix(value, "\r")
	return value, nil
}

func clearBytes(value []byte) {
	for i := range value {
		value[i] = 0
	}
}

func errorsWithUsage(message string) error {
	return usageError(message)
}

type usageError string

func (e usageError) Error() string { return string(e) }

func usage(output io.Writer) {
	_, _ = fmt.Fprintln(output, "usage:")
	_, _ = fmt.Fprintln(output, "  mayfly init [-path DIR] [-registry FILE]")
	_, _ = fmt.Fprintln(output, "  mayfly set <NAME>")
	_, _ = fmt.Fprintln(output, "  mayfly get <NAME>")
	_, _ = fmt.Fprintln(output, "  mayfly list")
	_, _ = fmt.Fprintln(output, "  mayfly delete <NAME>")
}
