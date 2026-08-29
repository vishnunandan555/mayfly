// mayfly is the application command-line entry point.
package main

import (
	"flag"
	"fmt"
	"os"

	"mayfly/project"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "init":
		if err := runInit(os.Args[2:]); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "mayfly init:", err)
			os.Exit(1)
		}
	default:
		_, _ = fmt.Fprintf(os.Stderr, "mayfly: unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func runInit(args []string) error {
	defaultRegistry, err := project.DefaultRegistryPath()
	if err != nil {
		return err
	}
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
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
		_, _ = fmt.Fprintf(os.Stdout, "Initialized project %s at %s\n", identity.ID, identity.Path)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "Project %s is already initialized at %s\n", identity.ID, identity.Path)
	}
	return nil
}

func usage() {
	_, _ = fmt.Fprintln(os.Stderr, "usage: mayfly init [-path DIR] [-registry FILE]")
}
