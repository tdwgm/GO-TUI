package main

import (
	"os"

	"github.com/tdwgm/tui-example/internal/tui"
)

// App identity variables (matching the TUI standard from gu-vpn).
const (
	appName        = "GO-TUI"
	appDescription = "Standardized TUI Framework & Reference Implementation"
)

var (
	appVersion = "0.1.0"
	repoURL    = "https://github.com/tdwgm/GO-TUI.git"
)

// bannerArt is the 3-line double-line logo for GO-TUI.
const bannerArt = `╔═╗ ╔═╗     ╔╦╗ ╦ ╦ ╦
║ ╦ ║ ║ ───  ║  ║ ║ ║
╚═╝ ╚═╝      ╩  ╚═╝ ╩`

func main() {
	tui.AppName = appName
	tui.AppDescription = appDescription
	tui.Version = appVersion
	tui.RepoURL = repoURL
	tui.BinaryName = "go-tui"
	tui.BannerArt = bannerArt

	os.Exit(tui.Main(os.Args[1:], os.Stdout, os.Stderr))
}
