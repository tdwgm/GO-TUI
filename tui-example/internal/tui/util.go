package tui

// Project-agnostic helpers shared across screens: clipboard, atomic file
// write, and the list-scrolling window. Reuse as-is.

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// copyToClipboard shells out to wl-copy (Wayland), pbcopy (macOS), or xclip.
func copyToClipboard(s string) (string, error) {
	for _, candidate := range []string{"wl-copy", "pbcopy", "xclip"} {
		if path, err := exec.LookPath(candidate); err == nil {
			c := exec.Command(path)
			if candidate == "xclip" {
				c = exec.Command(path, "-selection", "clipboard")
			}
			c.Stdin = strings.NewReader(s)
			if err := c.Run(); err == nil {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("no clipboard helper found (install wl-copy/xclip)")
}

// atomicWrite writes data to path via a temp file + rename, so a crash never
// leaves a half-written file.
func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	_ = os.Remove(tmp)
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// listWindow returns the [start,end) row range to render so idx stays visible.
// height is the terminal height; ~reserve lines cover the banner/footer/flash.
// When height is unknown (0) or everything fits, the full range is returned.
func listWindow(idx, total, height, reserve int) (int, int) {
	visible := height - reserve
	if height <= 0 || visible < 3 {
		visible = total // no/short size info → render all (terminal scrolls)
	}
	if total <= visible {
		return 0, total
	}
	start := idx - visible/2
	if start < 0 {
		start = 0
	}
	if start > total-visible {
		start = total - visible
	}
	return start, start + visible
}
