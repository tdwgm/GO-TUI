package tui

// cli.go — the command line: the TUI by default, plus the standard commands
// help/info, version, install, doctor and uninstall (TUI Scaffold standard).

import (
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/charmbracelet/x/term"
)

// Main dispatches the subcommand and returns the exit code.
func Main(args []string, stdout, stderr io.Writer) int {
	cmd := ""
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "", "tui":
		return Run(stdout, stderr)
	case "help", "info", "-h", "--help":
		usage(stderr)
	case "version", "-v", "--version":
		if f, ok := stdout.(*os.File); ok && term.IsTerminal(f.Fd()) {
			fmt.Fprintln(stdout, renderBanner()+"\n")
		}
		fmt.Fprintf(stdout, "%s v%s (%s)\n", AppName, Version, runtime.Version())
	case "install":
		fmt.Fprintln(stderr, renderBanner()+"\n")
		if err := install(stdout); err != nil {
			fmt.Fprintln(stderr, "install:", err)
			return 1
		}
	case "uninstall":
		uninstall()
		fmt.Fprintln(stdout, "Removed", installedBinary(), "and the shell alias (config kept).")
	case "doctor":
		return doctor(stdout)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", cmd)
		usage(stderr)
		return 1
	}
	return 0
}

func usage(w io.Writer) {
	fmt.Fprintln(w, renderBanner()+"\n")
	fmt.Fprintf(w, "Usage: %s [command]   (no command opens the TUI)\n\n", AppName)
	for _, c := range [][2]string{
		{"tui", "the interactive UI (default)"},
		{"install", "binary to ~/.local/bin, shell alias, config file in the current format"},
		{"doctor", "check config, log, alias, install"},
		{"uninstall", "remove the binary and the alias (config kept)"},
		{"version", "print the version"},
		{"help", "this text"},
	} {
		fmt.Fprintf(w, "  %-10s %s\n", c[0], c[1])
	}
	fmt.Fprintf(w, "\nConfig: %s\nLog:    %s\n", displayPath(ConfigPath()), displayPath(logPath()))
}

func doctor(w io.Writer) int {
	bad := 0
	row := func(ok bool, what, detail string) {
		mark := "✓"
		if !ok {
			mark = "✗"
			bad++
		}
		fmt.Fprintf(w, "  %s  %-14s %s\n", mark, what, detail)
	}
	cfg, found, err := Load()
	row(err == nil, "config", fmt.Sprintf("%s (found=%v)", displayPath(ConfigPath()), found))
	if cfg == nil {
		cfg = Default()
	}
	_, ierr := os.Stat(installedBinary())
	row(ierr == nil, "installed", displayPath(installedBinary()))
	if aliasesSupported() {
		where := aliasWhere()
		row(cfg.ShellAliases == (len(where) > 0), "shell alias", fmt.Sprintf("want=%v at %v", cfg.ShellAliases, where))
	}
	_, lerr := os.Stat(logPath())
	row(lerr == nil || os.IsNotExist(lerr), "log", displayPath(logPath()))
	if bad > 0 {
		return 1
	}
	return 0
}
