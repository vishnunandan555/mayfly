package main

import (
	"fmt"
	"io"
	"strings"
)

func cmdCompletion(args []string, stdout, stderr io.Writer) int {
	shell := "bash"
	if len(args) >= 1 {
		shell = strings.ToLower(args[0])
	}
	switch shell {
	case "bash":
		printBashCompletion(stdout)
	case "zsh":
		printZshCompletion(stdout)
	case "fish":
		printFishCompletion(stdout)
	default:
		fmt.Fprintf(stderr, "mayfly: unsupported shell %q (supported: bash, zsh, fish)\n", shell)
		return 1
	}
	return 0
}

func printBashCompletion(w io.Writer) {
	fmt.Fprintln(w, `_mayfly() {
    local cur prev words cword
    _init_completion || return

    local commands="init set get list delete run scan audit backup restore migrate import update rotate-password completion uninstall version help c current"

    if [[ ${cword} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
        return 0
    fi

    case "${prev}" in
        get|delete|set)
            return 0
            ;;
        completion)
            COMPREPLY=( $(compgen -W "bash zsh fish" -- ${cur}) )
            return 0
            ;;
        audit)
            COMPREPLY=( $(compgen -W "verify" -- ${cur}) )
            return 0
            ;;
        update)
            COMPREPLY=( $(compgen -W "--check --yes" -- ${cur}) )
            return 0
            ;;
    esac
}
complete -F _mayfly mayfly mf`)
}

func printZshCompletion(w io.Writer) {
	fmt.Fprintln(w, `#compdef mayfly mf

_mayfly() {
    local -a commands
    commands=(
        'init:Initialize project in current directory'
        'set:Add or update an encrypted secret'
        'get:Output decrypted secret to stdout or clipboard'
        'list:List all secret keys for current project'
        'delete:Remove a secret from the vault'
        'import:Import secrets from .env file into vault'
        'update:Check for newer releases and update binary'
        'rotate-password:Re-encrypt vault with new master password'
        'run:Inject secrets in memory and execute process'
        'scan:Scan codebase for plaintext secret leaks'
        'audit:View or cryptographically verify audit log'
        'backup:Export encrypted vault backup snapshot'
        'restore:Restore vault and projects from backup snapshot'
        'migrate:Update project identity when directory moves'
        'completion:Generate shell autocompletion script'
        'uninstall:Uninstall mayfly and remove data'
        'version:Show version information'
        'help:Show help message'
        'c:Launch TUI scoped to current project'
    )

    if (( CURRENT == 2 )); then
        _describe -t commands 'mayfly commands' commands
    fi
}

compdef _mayfly mayfly mf`)
}

func printFishCompletion(w io.Writer) {
	fmt.Fprintln(w, `complete -c mayfly -f
complete -c mf -f

complete -c mayfly -n "__fish_use_subcommand" -a init -d "Initialize project in current directory"
complete -c mayfly -n "__fish_use_subcommand" -a set -d "Add or update an encrypted secret"
complete -c mayfly -n "__fish_use_subcommand" -a get -d "Output decrypted secret"
complete -c mayfly -n "__fish_use_subcommand" -a list -d "List secret keys"
complete -c mayfly -n "__fish_use_subcommand" -a delete -d "Remove a secret"
complete -c mayfly -n "__fish_use_subcommand" -a import -d "Import .env file into vault"
complete -c mayfly -n "__fish_use_subcommand" -a update -d "Check for updates and upgrade binary"
complete -c mayfly -n "__fish_use_subcommand" -a rotate-password -d "Re-encrypt vault with new password"
complete -c mayfly -n "__fish_use_subcommand" -a run -d "Inject secrets in memory and execute process"
complete -c mayfly -n "__fish_use_subcommand" -a scan -d "Scan codebase for plaintext secret leaks"
complete -c mayfly -n "__fish_use_subcommand" -a audit -d "View or verify audit log"
complete -c mayfly -n "__fish_use_subcommand" -a backup -d "Export encrypted backup"
complete -c mayfly -n "__fish_use_subcommand" -a restore -d "Restore vault backup"
complete -c mayfly -n "__fish_use_subcommand" -a migrate -d "Migrate project directory"
complete -c mayfly -n "__fish_use_subcommand" -a completion -d "Generate shell completion"
complete -c mayfly -n "__fish_use_subcommand" -a version -d "Show version"

complete -c mf -w mayfly`)
}
