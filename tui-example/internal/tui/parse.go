package tui

// This file is the round-trip-aware parser/writer for the shell-style
// config file (KEY="value", # comments). Comments, blank lines, and unknown
// keys are preserved across Load → Save so the file stays hand-editable in
// $EDITOR without the TUI clobbering layout. It is project-agnostic — reuse
// as-is.

import (
	"bytes"
	"fmt"
	"strings"
)

type entryKind int

const (
	kindBlank entryKind = iota
	kindComment
	kindKV
	kindRaw
)

type entry struct {
	kind  entryKind
	text  string
	key   string
	value string
}

// Doc is a parsed config-file representation that preserves layout.
type Doc struct {
	entries []entry
}

// NewDoc returns an empty Doc.
func NewDoc() *Doc { return &Doc{} }

// Parse parses raw bytes into a Doc.
func Parse(raw []byte) (*Doc, error) {
	d := &Doc{}
	if len(raw) == 0 {
		return d, nil
	}
	lines := strings.Split(string(raw), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			d.entries = append(d.entries, entry{kind: kindBlank, text: ""})
		case strings.HasPrefix(trimmed, "#"):
			d.entries = append(d.entries, entry{kind: kindComment, text: line})
		default:
			k, v, ok := parseKV(line)
			if !ok {
				d.entries = append(d.entries, entry{kind: kindRaw, text: line})
				continue
			}
			d.entries = append(d.entries, entry{kind: kindKV, text: line, key: k, value: v})
		}
	}
	return d, nil
}

func parseKV(line string) (string, string, bool) {
	eq := strings.Index(line, "=")
	if eq <= 0 {
		return "", "", false
	}
	rawKey := strings.TrimSpace(line[:eq])
	if !validKey(rawKey) {
		return "", "", false
	}
	rawVal := strings.TrimSpace(line[eq+1:])
	if len(rawVal) >= 2 && rawVal[0] == '"' && rawVal[len(rawVal)-1] == '"' {
		rawVal = rawVal[1 : len(rawVal)-1]
	}
	return rawKey, rawVal, true
}

func validKey(k string) bool {
	if k == "" {
		return false
	}
	for i, c := range k {
		switch {
		case c >= 'A' && c <= 'Z':
		case c >= 'a' && c <= 'z':
		case c == '_':
		case i > 0 && c >= '0' && c <= '9':
		default:
			return false
		}
	}
	return true
}

// Get returns the value for key and whether it was set in the doc.
func (d *Doc) Get(key string) (string, bool) {
	for _, e := range d.entries {
		if e.kind == kindKV && e.key == key {
			return e.value, true
		}
	}
	return "", false
}

// Set updates the value for key, preserving comments and position. A new key
// is appended at the end of the doc.
func (d *Doc) Set(key, value string) {
	for i, e := range d.entries {
		if e.kind == kindKV && e.key == key {
			d.entries[i].value = value
			d.entries[i].text = fmt.Sprintf(`%s="%s"`, key, escape(value))
			return
		}
	}
	d.entries = append(d.entries, entry{
		kind:  kindKV,
		text:  fmt.Sprintf(`%s="%s"`, key, escape(value)),
		key:   key,
		value: value,
	})
}

// AppendComment adds a comment line at the end of the doc.
func (d *Doc) AppendComment(text string) {
	if !strings.HasPrefix(text, "#") {
		text = "# " + text
	}
	d.entries = append(d.entries, entry{kind: kindComment, text: text})
}

// AppendBlank adds a blank line at the end of the doc.
func (d *Doc) AppendBlank() {
	d.entries = append(d.entries, entry{kind: kindBlank, text: ""})
}

// Bytes renders the doc back to bytes for writing to disk.
func (d *Doc) Bytes() []byte {
	var buf bytes.Buffer
	for _, e := range d.entries {
		buf.WriteString(e.text)
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

func escape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
