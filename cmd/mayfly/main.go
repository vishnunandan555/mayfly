package main

import (
	"context"
	"fmt"
	"io"
	"os"


	"mayfly/pkg/application"
	"mayfly/pkg/audit"
	"mayfly/pkg/domain"
	"mayfly/pkg/executor"
	"mayfly/pkg/project"
	"mayfly/pkg/scanner"
	"mayfly/pkg/tui"
	"mayfly/pkg/vault"
)


func main() {
	code := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	ctx := context.Background()
	passwordFromStdin = false

	// Extract --password-stdin anywhere in args before subcommand dispatch.
	// This ensures commands like `mf --password-stdin npm start` or `mf set KEY --password-stdin` work seamlessly.
	var cleanArgs []string
	for _, arg := range args {
		if arg == "--password-stdin" {
			passwordFromStdin = true
		} else {
			cleanArgs = append(cleanArgs, arg)
		}
	}
	args = cleanArgs

	// Initialize project workspace registry

	reg, err := project.NewRegistry("")
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: failed to initialize project registry: %v\n", err)
		return 1
	}

	// Initialize encrypted vault storage
	storage, err := vault.NewStorage("", 0)
	if err != nil {
		fmt.Fprintf(stderr, "mayfly: failed to initialize vault storage: %v\n", err)
		return 1
	}

	// Initialize process executor and optional telemetry/scanning subsystems
	execEngine := executor.NewProcessExecutor(stdin, stdout, stderr)
	auditLog, aErr := audit.New("")
	if aErr != nil {
		fmt.Fprintf(stderr, "mayfly: warning: audit log initialization failed: %v\n", aErr)
	}
	leakScanner, sErr := scanner.New(scanner.Options{})
	if sErr != nil {
		fmt.Fprintf(stderr, "mayfly: warning: scanner initialization failed: %v\n", sErr)
	}
	metaStore, mErr := project.NewMetaStore("")
	if mErr != nil {
		fmt.Fprintf(stderr, "mayfly: warning: vault meta store initialization failed: %v\n", mErr)
	}

	svc := application.NewService(application.Dependencies{
		Projects:  reg,
		Vault:     storage,
		Executor:  execEngine,
		Auditor:   auditLog,
		Scanner:   leakScanner,
		MetaStore: metaStore,
	})

	if len(args) == 0 {

		if err := tui.Run(svc, tui.Options{}); err != nil {
			fmt.Fprintf(stderr, "mayfly: tui error: %v\n", err)
			return 1
		}
		return 0
	}

	subcmd := args[0]
	subArgs := args[1:]

	switch subcmd {
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0

	case "version", "-v", "--version":
		fmt.Fprintf(stdout, "mayfly v%s (zero-dependency)\n", domain.Version)
		return 0

	case "c", "current", "tui":
		cwd, _ := os.Getwd()
		proj, err := svc.ResolveCurrentProject(cwd)
		opts := tui.Options{CurrentDir: cwd}
		if err == nil {
			opts.ProjectScoped = &proj
		}
		if err := tui.Run(svc, opts); err != nil {
			fmt.Fprintf(stderr, "mayfly: tui error: %v\n", err)
			return 1
		}
		return 0

	case "init":
		return cmdInit(ctx, svc, subArgs, stdout, stderr)

	case "set":
		return cmdSet(ctx, svc, subArgs, stdin, stdout, stderr)

	case "get":
		return cmdGet(ctx, svc, subArgs, stdin, stdout, stderr)

	case "list":
		return cmdList(ctx, svc, subArgs, stdin, stdout, stderr)

	case "delete":
		return cmdDelete(ctx, svc, subArgs, stdin, stdout, stderr)

	case "scan":
		return cmdScan(ctx, svc, subArgs, stdout, stderr)

	case "audit":
		return cmdAudit(ctx, svc, subArgs, stdout, stderr)

	case "env":
		return cmdEnv(ctx, svc, subArgs, stdin, stdout, stderr)

	case "status", "doctor":
		return cmdStatus(ctx, svc, stdout, stderr)

	case "check":
		return cmdCheck(ctx, svc, stdout, stderr)

	case "template":
		return cmdTemplate(ctx, svc, subArgs, stdin, stdout, stderr)

	case "diff":
		return cmdDiff(ctx, svc, subArgs, stdin, stdout, stderr)

	case "install-hook":
		return cmdInstallHook(subArgs, stdout, stderr)

	case "uninstall-hook":
		return cmdUninstallHook(stdout, stderr)

	case "backup":
		return cmdBackup(ctx, svc, subArgs, stdout, stderr)

	case "restore":
		return cmdRestore(ctx, svc, subArgs, stdout, stderr)

	case "migrate":
		return cmdMigrate(ctx, svc, subArgs, stdout, stderr)

	case "import":
		return cmdImport(ctx, svc, subArgs, stdin, stdout, stderr)

	case "rotate-password":
		return cmdRotatePassword(ctx, svc, stdin, stdout, stderr)

	case "completion":
		return cmdCompletion(subArgs, stdout, stderr)

	case "update":
		return cmdUpdate(ctx, subArgs, stdin, stdout, stderr)

	case "uninstall":
		return cmdUninstall(stdin, stdout)

	default:
		// Transparent execution: 'mayfly npm run dev' or 'mf npm run dev'
		return executeTargetProcess(ctx, svc, args, stdin, stdout, stderr)
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, `MayFly: Zero-Dependency Secrets Workspace & Process Injector

Usage:
  mayfly <COMMAND> [ARGS...]          Inject secrets in memory and execute process directly
  mayfly                              Launch interactive Global TUI Dashboard (Project Grid)
  mayfly c, mayfly current            Launch TUI scoped directly to current project
  mayfly init [-path DIR]             Initialize project in current directory or target path
  mayfly set <NAME> [--clip] [VALUE]  Add or update an encrypted secret (alt-screen if interactive)
  mayfly get <NAME> [--clip]          Output decrypted secret or copy to clipboard (-c)
  mayfly list [--json]                List secret keys for the current project
  mayfly delete <NAME>                Remove a secret from the vault
  mayfly import [FILE] [--delete]     Import secrets from .env file into vault (default: .env)
  mayfly rotate-password              Re-encrypt vault with a new master password
  mayfly env [--shell bash|fish|json] Export secrets as shell environment variables
  mayfly status                       Show vault health, project count, and audit summary
  mayfly check                        Verify vault, audit log, and project registry integrity
  mayfly scan [DIR] [--json] [--severity CRITICAL|WARNING]
                                      Scan codebase for plaintext secret leaks
  mayfly audit [verify] [--json] [--tail N]
                                      View or cryptographically verify audit log
  mayfly install-hook                 Install git pre-commit hook to run mf scan on commit
  mayfly uninstall-hook               Remove the mayfly pre-commit hook
  mayfly template <FILE> [--output F] Render a config template with secrets injected
  mayfly diff [PATH_A] [PATH_B]       Compare secret keys between two projects
  mayfly backup [FILE]                Export encrypted vault backup snapshot
  mayfly restore <FILE>               Restore vault and projects from backup snapshot
  mayfly migrate <OLD> <NEW>          Update project identity when directory moves
  mayfly update [--check] [--yes]     Check for newer releases and prompt to update
  mayfly completion <SHELL>           Generate autocompletion script (bash, zsh, fish)
  mayfly uninstall                    Cleanly uninstall binaries and remove data
  mayfly version                      Show version information
  mayfly help                         Show this help message

Global Flags:
  --password-stdin                    Read vault password from stdin (safer than env var for CI)

CI / Automation:
  echo "$VAULT_PASS" | mf --password-stdin npm start
  # Preferred over: MAYFLY_VAULT_PASSWORD=$VAULT_PASS mf npm start
  # (env vars are readable via /proc/<pid>/environ by same-user processes)

Short alias:
  All commands work with 'mf' as well:
    mf npm run dev                    Run dev server with injected secrets
    mf set STRIPE_KEY                 Enter secret value in clean ephemeral alt-screen
    mf get STRIPE_KEY --clip          Copy secret directly to clipboard
    mf import .env                    Migrate existing .env file into vault
    mf update                         Check for latest version and upgrade
    mf rotate-password                Rotate vault encryption master password
    mf                                Open TUI dashboard`)
}
