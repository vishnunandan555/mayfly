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
	"time"

	"mayfly"
	"mayfly/application"
	"mayfly/audit"
	"mayfly/domain"
	"mayfly/executor"
	"mayfly/project"
	"mayfly/scanner"
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
	if args[0] != "set" && args[0] != "get" && args[0] != "list" && args[0] != "delete" && args[0] != "run" && args[0] != "audit" && args[0] != "scan" && args[0] != "tui" {
		_, _ = fmt.Fprintf(errorOutput, "mayfly: unknown command %q\n", args[0])
		usage(errorOutput)
		return 2
	}
	runtime, err := newRuntime()
	if err != nil {
		_, _ = fmt.Fprintln(errorOutput, "mayfly:", err)
		return 1
	}
	result, err := runtime.execute(context.Background(), args, input, output, errorOutput)
	if err != nil {
		_, _ = fmt.Fprintln(errorOutput, "mayfly:", err)
		var usageErr usageError
		if errors.As(err, &usageErr) {
			usage(errorOutput)
			return 2
		}
		if args[0] == "run" && result.ExitCode > 0 {
			return result.ExitCode
		}
		return 1
	}
	return result.ExitCode
}

type commandRuntime struct {
	service *application.Service
	storage *vault.Storage
	audit   *audit.Log
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
	auditPath, err := audit.DefaultPath()
	if err != nil {
		return nil, err
	}
	auditLog, err := audit.New(auditPath)
	if err != nil {
		return nil, err
	}
	secretScanner, err := scanner.New(scanner.Options{SkipPaths: []string{storage.Path(), auditLog.Path()}})
	if err != nil {
		return nil, err
	}
	return &commandRuntime{
		service: application.NewService(application.Dependencies{
			Projects: registry,
			Vault:    storage,
			Executor: executor.NewProcessExecutor(nil, nil, nil),
			Auditor:  auditLog,
			Scanner:  secretScanner,
		}),
		storage: storage,
		audit:   auditLog,
	}, nil
}

func (r *commandRuntime) execute(ctx context.Context, args []string, input io.Reader, output, errorOutput io.Writer) (application.ExecutionResult, error) {
	if r == nil || r.service == nil {
		return application.ExecutionResult{}, application.ErrMissingSecrets
	}
	if len(args) == 0 {
		return application.ExecutionResult{}, errorsWithUsage("command is required")
	}
	reader := bufio.NewReader(input)
	command := args[0]
	if command == "tui" {
		if len(args) != 1 {
			return application.ExecutionResult{}, errorsWithUsage("tui takes no arguments")
		}
		screenService := application.NewScreenService(r.service)
		screens := mayfly.NewScreens(screenService)
		if inputFile, ok := input.(*os.File); ok {
			if err := screens.RunIO(inputFile, output); err != nil {
				return application.ExecutionResult{}, err
			}
			return application.ExecutionResult{}, nil
		}
		return application.ExecutionResult{}, errors.New("tui requires an interactive terminal input")
	}
	if command == "audit" {
		return r.executeAudit(ctx, args, output)
	}
	if command == "scan" {
		if len(args) != 1 {
			return application.ExecutionResult{}, errorsWithUsage("scan takes no arguments")
		}
		findings, err := r.service.ScanCurrentProject(ctx)
		if err != nil {
			return application.ExecutionResult{}, err
		}
		for _, finding := range findings {
			if _, err := fmt.Fprintf(output, "%s", finding.Path); err != nil {
				return application.ExecutionResult{}, err
			}
			if finding.Line > 0 {
				if _, err := fmt.Fprintf(output, ":%d:%d", finding.Line, finding.Column); err != nil {
					return application.ExecutionResult{}, err
				}
			}
			if _, err := fmt.Fprintf(output, ": %s [%s] %s\n", finding.Severity, finding.Category, finding.Message); err != nil {
				return application.ExecutionResult{}, err
			}
		}
		if len(findings) > 0 {
			return application.ExecutionResult{ExitCode: 3}, nil
		}
		return application.ExecutionResult{}, nil
	}
	var name domain.SecretName
	if command == "list" {
		if len(args) != 1 {
			return application.ExecutionResult{}, errorsWithUsage("list takes no arguments")
		}
	} else if command == "run" {
		if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
			return application.ExecutionResult{}, errorsWithUsage("run requires a command")
		}
	} else {
		if len(args) != 2 || strings.TrimSpace(args[1]) == "" {
			return application.ExecutionResult{}, errorsWithUsage(command + " requires exactly one secret name")
		}
		name = domain.SecretName(args[1])
		if err := name.Validate(); err != nil {
			return application.ExecutionResult{}, application.ErrInvalidSecretName
		}
	}
	// Resolve the project before prompting or creating a vault. This prevents
	// an uninitialized directory from causing any vault-side state change.
	if _, err := r.service.CurrentProject(ctx); err != nil {
		return application.ExecutionResult{}, err
	}

	opened, err := r.open(ctx, reader, errorOutput, command == "set")
	if err != nil {
		return application.ExecutionResult{}, err
	}
	defer func() { _ = opened.Close() }()

	switch command {
	case "set":
		value, err := readLine(reader, errorOutput, "Secret value: ")
		if err != nil {
			return application.ExecutionResult{}, err
		}
		if err := opened.SetCurrentSecret(ctx, name, value); err != nil {
			return application.ExecutionResult{}, err
		}
		_, _ = fmt.Fprintf(output, "Set %s\n", name)
		return application.ExecutionResult{}, nil
	case "get":
		material, err := opened.GetCurrentSecret(ctx, name)
		if err != nil {
			return application.ExecutionResult{}, err
		}
		// get is the explicit value-bearing command. Close the session before
		// writing the value, and do not include it in any status or error text.
		if err := opened.Close(); err != nil {
			return application.ExecutionResult{}, err
		}
		_, err = fmt.Fprintln(output, material.Value)
		return application.ExecutionResult{}, err
	case "list":
		secrets, err := opened.ListCurrentSecrets(ctx)
		if err != nil {
			return application.ExecutionResult{}, err
		}
		if err := opened.Close(); err != nil {
			return application.ExecutionResult{}, err
		}
		for _, secret := range secrets {
			if _, err := fmt.Fprintln(output, secret.Name); err != nil {
				return application.ExecutionResult{}, err
			}
		}
		return application.ExecutionResult{}, nil
	case "delete":
		answer, err := readLine(reader, errorOutput, "Delete secret? [y/N]: ")
		if err != nil {
			return application.ExecutionResult{}, err
		}
		if !strings.EqualFold(strings.TrimSpace(answer), "y") {
			_, _ = fmt.Fprintln(output, "Delete cancelled")
			return application.ExecutionResult{}, nil
		}
		if err := opened.DeleteCurrentSecret(ctx, name); err != nil {
			return application.ExecutionResult{}, err
		}
		_, _ = fmt.Fprintf(output, "Deleted %s\n", name)
		return application.ExecutionResult{}, nil
	case "run":
		project, err := opened.CurrentProject(ctx)
		if err != nil {
			return application.ExecutionResult{}, err
		}
		return opened.Run(ctx, domain.ExecutionRequest{ProjectID: project.ID, Command: append([]string(nil), args[1:]...)})
	default:
		return application.ExecutionResult{}, errorsWithUsage("unknown command")
	}
}

func (r *commandRuntime) executeAudit(ctx context.Context, args []string, output io.Writer) (application.ExecutionResult, error) {
	if r == nil || r.audit == nil {
		return application.ExecutionResult{}, application.ErrAuditFailed
	}
	if len(args) == 2 && args[1] == "verify" {
		if err := r.audit.Verify(ctx); err != nil {
			return application.ExecutionResult{}, err
		}
		_, err := fmt.Fprintln(output, "Audit verified")
		return application.ExecutionResult{}, err
	}
	if len(args) != 1 {
		return application.ExecutionResult{}, errorsWithUsage("audit accepts only the optional verify subcommand")
	}
	events, err := r.audit.Events(ctx)
	if err != nil {
		return application.ExecutionResult{}, err
	}
	for _, event := range events {
		if _, err := fmt.Fprintf(output, "%s %s project=%s", event.At.UTC().Format(time.RFC3339Nano), event.Action, event.ProjectID); err != nil {
			return application.ExecutionResult{}, err
		}
		if event.Secret != "" {
			if _, err := fmt.Fprintf(output, " secret=%s", event.Secret); err != nil {
				return application.ExecutionResult{}, err
			}
		}
		if event.Command != "" {
			if _, err := fmt.Fprintf(output, " command=%s", event.Command); err != nil {
				return application.ExecutionResult{}, err
			}
		}
		if event.ExitStatus != nil {
			if _, err := fmt.Fprintf(output, " exit=%d", *event.ExitStatus); err != nil {
				return application.ExecutionResult{}, err
			}
		}
		if _, err := fmt.Fprintln(output); err != nil {
			return application.ExecutionResult{}, err
		}
	}
	return application.ExecutionResult{}, nil
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
		auditPath, err := audit.DefaultPath()
		if err != nil {
			return err
		}
		auditLog, err := audit.New(auditPath)
		if err != nil {
			return err
		}
		if err := auditLog.Record(context.Background(), domain.AuditEvent{
			At: time.Now(), Action: domain.AuditProjectInitialized, ProjectID: identity.ID,
		}); err != nil {
			return err
		}
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
	_, _ = fmt.Fprintln(output, "  mayfly tui")
	_, _ = fmt.Fprintln(output, "  mayfly set <NAME>")
	_, _ = fmt.Fprintln(output, "  mayfly get <NAME>")
	_, _ = fmt.Fprintln(output, "  mayfly list")
	_, _ = fmt.Fprintln(output, "  mayfly delete <NAME>")
	_, _ = fmt.Fprintln(output, "  mayfly run <COMMAND> [ARGS...]")
	_, _ = fmt.Fprintln(output, "  mayfly scan")
	_, _ = fmt.Fprintln(output, "  mayfly audit [verify]")
}
