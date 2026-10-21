package tui

// system.go — what the app installs around itself: the binary in ~/.local/bin,
// the shell alias, and its log file. Every piece is visible on the Config
// screen and driven by a config key (standard v2: "all reflected in the config
// file").

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// aliasesSupported: shell aliases exist on Linux/macOS (zsh, bash).
func aliasesSupported() bool { return runtime.GOOS != "windows" }

var rcFiles = []string{".zshrc", ".bashrc"}

// aliasWhere returns where `alias <AppName>=` is defined ("~/.zshrc:12").
func aliasWhere() []string {
	home, _ := os.UserHomeDir()
	var where []string
	for _, rc := range rcFiles {
		data, err := os.ReadFile(filepath.Join(home, rc))
		if err != nil {
			continue
		}
		for i, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "alias "+AppName+"=") {
				where = append(where, fmt.Sprintf("~/%s:%d", rc, i+1))
			}
		}
	}
	return where
}

// installedBinary is ~/.local/bin/<AppName>.
func installedBinary() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "bin", AppName)
}

// syncAliases applies SHELL_ALIASES: add the alias where it is missing, or
// remove it. Only existing rc files are touched. Returns a summary or "".
func syncAliases(cfg *Config) string {
	if !aliasesSupported() {
		return ""
	}
	have := len(aliasWhere()) > 0
	home, _ := os.UserHomeDir()
	switch {
	case cfg.ShellAliases && !have:
		line := fmt.Sprintf("\nalias %s='%s'\n", AppName, installedBinary())
		for _, rc := range rcFiles {
			p := filepath.Join(home, rc)
			if f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
				_, _ = f.WriteString(line)
				f.Close()
			}
		}
		return "Shell alias installed (" + AppName + ")"
	case !cfg.ShellAliases && have:
		for _, rc := range rcFiles {
			p := filepath.Join(home, rc)
			data, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			var keep []string
			for _, l := range strings.Split(string(data), "\n") {
				if !strings.HasPrefix(l, "alias "+AppName+"=") {
					keep = append(keep, l)
				}
			}
			_ = os.WriteFile(p, []byte(strings.Join(keep, "\n")), 0o644)
		}
		return "Shell alias removed"
	}
	return ""
}

// logPath is ~/.local/state/<AppName>/<AppName>.log (XDG state dir).
func logPath() string {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(dir, AppName, AppName+".log")
}

// logf appends one line to the app log (best effort). Never log secrets.
func logf(format string, a ...any) {
	p := logPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, format+"\n", a...)
}

// tailLines returns up to n last lines of path.
func tailLines(path string, n int) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	ring := make([]string, 0, n)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if len(ring) == n {
			ring = ring[1:]
		}
		ring = append(ring, sc.Text())
	}
	return ring
}

// install copies the running binary to ~/.local/bin/<AppName>, applies
// SHELL_ALIASES and rewrites the config file in the current format.
func install(w io.Writer) error {
	cfg, _, err := Load()
	if err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	self, _ = filepath.EvalSymlinks(self)
	dst := installedBinary()
	if self != dst {
		data, err := os.ReadFile(self)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := atomicWrite(dst, data); err != nil {
			return err
		}
		if err := os.Chmod(dst, 0o755); err != nil {
			return err
		}
		fmt.Fprintln(w, "Installed", displayPath(dst))
	} else {
		fmt.Fprintln(w, "Already installed at", displayPath(dst))
	}
	if msg := syncAliases(cfg); msg != "" {
		fmt.Fprintln(w, msg)
	}
	if err := SaveFresh(cfg); err != nil {
		return err
	}
	fmt.Fprintln(w, "Config file written in the current format:", displayPath(ConfigPath()))
	return nil
}

// uninstall removes the binary and the alias; the config file stays.
func uninstall() {
	_ = os.Remove(installedBinary())
	cfg := Default()
	cfg.ShellAliases = false
	syncAliases(cfg)
}
