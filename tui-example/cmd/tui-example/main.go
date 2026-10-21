package main

import (
	"os"

	"github.com/tdwgm/tui-example/internal/tui"
)

func main() {
	// App metadata flows from here into the TUI so the rest of the package
	// stays project-agnostic. These are the only lines that name the project.
	tui.AppName = "tui-example"
	tui.AppDescription = "Full standard Go TUI example demonstrating all features"
	// tui.Version is overridable at build time:
	//   go build -ldflags "-X github.com/tdwgm/tui-example/internal/tui.Version=1.2.3"

	// 3-line double-line banner logo (default when BANNER_FONT="default").
	tui.BannerArt = `╔╦╗╦ ╦╦   ╔═╗═╗ ╦╔═╗╔╦╗╔═╗╦  ╔═╗
 ║ ║ ║║   ╠═ ╔╩╦╝╠═╣║║║╠═╝║  ╠═ 
 ╩ ╚═╝╩   ╚═╝╩ ╚═╩ ╩╩ ╩╩  ╩═╝╚═╝`
	os.Exit(tui.Main(os.Args[1:], os.Stdout, os.Stderr))
}
