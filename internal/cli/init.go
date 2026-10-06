package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zhide915/footlight/internal/config"
)

func runInit(args []string) int {
	force := false
	for _, a := range args {
		if a == "--force" || a == "-f" {
			force = true
		}
	}

	cfgPath, err := config.Path()
	if err != nil {
		fmt.Fprintln(os.Stderr, "init: locating config:", err)
		return 1
	}
	created, err := writeDefaultConfig(cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "init: writing config:", err)
		return 1
	}
	if created {
		fmt.Println("config:     created", cfgPath)
	} else {
		fmt.Println("config:     kept existing", cfgPath)
	}

	cmd := commandString()
	settingsPath := claudeSettingsPath()
	status, err := wireSettings(settingsPath, cmd, force)
	if err != nil {
		fmt.Fprintln(os.Stderr, "init:", err)
		return 1
	}
	switch status {
	case statusCreated:
		fmt.Println("statusLine: wired into", settingsPath, "(command: "+cmd+")")
	case statusUpdated:
		fmt.Println("statusLine: updated", settingsPath, "(command: "+cmd+"; backup at "+settingsPath+".bak)")
	case statusNoop:
		fmt.Println("statusLine: already points to footlight — nothing to do")
	case statusRefused:
		fmt.Fprintln(os.Stderr, "statusLine: an existing statusLine points elsewhere in")
		fmt.Fprintln(os.Stderr, "            "+settingsPath)
		fmt.Fprintln(os.Stderr, "            re-run with --force to replace it (a .bak is kept)")
		return 1
	}

	if shadow := shadowingFile(); shadow != "" {
		fmt.Printf("warning:    %s has its own statusLine, which overrides the user-global one\n", shadow)
	}

	fmt.Println("done — your next message refreshes the status line.")
	return 0
}

func writeDefaultConfig(path string) (created bool, err error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(path, []byte(config.DefaultTOML), 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// Claude Code runs the command through a shell, which is Git Bash on Windows, so
// the path uses forward slashes and single quotes to keep \, $, and backticks
// literal.
func commandString() string {
	exe, err := os.Executable()
	if err != nil {
		return "footlight"
	}
	if found, err := exec.LookPath("footlight"); err == nil && sameFile(found, exe) {
		return "footlight"
	}
	return "'" + strings.ReplaceAll(filepath.ToSlash(exe), "'", `'\''`) + "'"
}

func sameFile(a, b string) bool {
	ai, err := os.Stat(a)
	if err != nil {
		return false
	}
	bi, err := os.Stat(b)
	return err == nil && os.SameFile(ai, bi)
}

func claudeSettingsPath() string {
	dir := os.Getenv("CLAUDE_CONFIG_DIR")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".claude")
	}
	return filepath.Join(dir, "settings.json")
}

type wireStatus int

const (
	statusCreated wireStatus = iota
	statusUpdated
	statusNoop
	statusRefused
)

// wireSettings leaves every member other than statusLine byte for byte as it
// was, so the user's key order and formatting survive.
func wireSettings(path, cmd string, force bool) (wireStatus, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return 0, err
		}
		out := defaultLayout.settings(cmd) + "\n"
		return statusCreated, os.WriteFile(path, []byte(out), 0o644)
	}
	if err != nil {
		return 0, err
	}

	members, open, closing, err := scanObject(data)
	if err != nil {
		return 0, fmt.Errorf("existing %s is not valid JSON (refusing to overwrite): %w", path, err)
	}

	l := layoutOf(data, open)
	var out []byte
	if sl, ok := lastMember(members, "statusLine"); !ok {
		out = insertStatusLine(data, members, open, closing, l, cmd)
	} else {
		if !force {
			var existing any
			if err := json.Unmarshal(data[sl.start:sl.end], &existing); err != nil {
				return 0, err
			}
			if m, ok := existing.(map[string]any); ok && isStatusLine(m) {
				return statusNoop, nil
			}
			return statusRefused, nil
		}
		out = replaceStatusLine(data, sl, l, cmd)
	}

	if err := os.WriteFile(path+".bak", data, 0o644); err != nil {
		return 0, fmt.Errorf("writing backup: %w", err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return 0, err
	}
	return statusUpdated, nil
}

func isStatusLine(statusLine map[string]any) bool {
	cmd, _ := statusLine["command"].(string)
	return strings.Contains(strings.ToLower(cmd), "footlight")
}

// span is the byte range [start, end) of a JSON value.
type span struct{ start, end int }

type member struct {
	key   string
	value span
}

// scanObject returns the members of the JSON object that is all of data, in
// file order, with the offsets of its opening and closing braces.
func scanObject(data []byte) (members []member, open, closing int, err error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return nil, 0, 0, err
	}
	if tok != json.Delim('{') {
		return nil, 0, 0, errors.New("top level is not an object")
	}
	open = int(dec.InputOffset()) - 1
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, 0, 0, err
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, 0, 0, err
		}
		end := int(dec.InputOffset())
		members = append(members, member{key: tok.(string), value: span{end - len(raw), end}})
	}
	if _, err := dec.Token(); err != nil {
		return nil, 0, 0, err
	}
	closing = int(dec.InputOffset()) - 1
	if _, err := dec.Token(); err != io.EOF {
		return nil, 0, 0, errors.New("unexpected data after the top-level object")
	}
	return members, open, closing, nil
}

// lastMember finds key the way JSON.parse does: a repeated key's last value wins.
func lastMember(members []member, key string) (span, bool) {
	for i := len(members) - 1; i >= 0; i-- {
		if members[i].key == key {
			return members[i].value, true
		}
	}
	return span{}, false
}

// layout is how a settings file indents its keys and ends its lines.
type layout struct{ indent, newline string }

var defaultLayout = layout{indent: "  ", newline: "\n"}

// layoutOf copies the file's line ending and the indentation of the object's
// first key, keeping the default indent when there is none to copy.
func layoutOf(data []byte, open int) layout {
	l := defaultLayout
	if bytes.Contains(data, []byte("\r\n")) {
		l.newline = "\r\n"
	}
	gap := data[open+1:]
	gap = gap[:len(gap)-len(bytes.TrimLeft(gap, " \t\r\n"))]
	if i := bytes.LastIndexByte(gap, '\n'); i >= 0 && i < len(gap)-1 {
		l.indent = string(gap[i+1:])
	}
	return l
}

// statusLine renders the statusLine object for a top-level key.
func (l layout) statusLine(cmd string) string {
	in, nl := l.indent, l.newline
	return "{" + nl + in + in + `"type": "command",` + nl + in + in + `"command": ` + jsonString(cmd) + nl + in + "}"
}

// settings renders an object whose only member is statusLine.
func (l layout) settings(cmd string) string {
	return "{" + l.newline + l.indent + `"statusLine": ` + l.statusLine(cmd) + l.newline + "}"
}

// jsonString skips HTML escaping, so a path with & stays readable.
func jsonString(s string) string {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.Encode(s)
	return strings.TrimSuffix(b.String(), "\n")
}

func insertStatusLine(data []byte, members []member, open, closing int, l layout, cmd string) []byte {
	if len(members) == 0 {
		return splice(data, span{open, closing + 1}, l.settings(cmd))
	}
	at := members[len(members)-1].value.end
	return splice(data, span{at, at}, ","+l.newline+l.indent+`"statusLine": `+l.statusLine(cmd))
}

// replaceStatusLine rewrites only command and type, so fields such as padding
// survive. A value without both keys is replaced whole.
func replaceStatusLine(data []byte, sl span, l layout, cmd string) []byte {
	members, _, _, err := scanObject(data[sl.start:sl.end])
	if err == nil {
		cmdSpan, hasCmd := lastMember(members, "command")
		typeSpan, hasType := lastMember(members, "type")
		if hasCmd && hasType && data[sl.start+cmdSpan.start] == '"' {
			cmdSpan = span{sl.start + cmdSpan.start, sl.start + cmdSpan.end}
			typeSpan = span{sl.start + typeSpan.start, sl.start + typeSpan.end}
			// Splice the later range first so the earlier offsets stay valid.
			if cmdSpan.start > typeSpan.start {
				return splice(splice(data, cmdSpan, jsonString(cmd)), typeSpan, `"command"`)
			}
			return splice(splice(data, typeSpan, `"command"`), cmdSpan, jsonString(cmd))
		}
	}
	return splice(data, sl, l.statusLine(cmd))
}

func splice(data []byte, s span, text string) []byte {
	out := make([]byte, 0, len(data)-(s.end-s.start)+len(text))
	out = append(out, data[:s.start]...)
	out = append(out, text...)
	return append(out, data[s.end:]...)
}

func shadowingFile() string {
	for _, name := range []string{
		filepath.Join(".claude", "settings.json"),
		filepath.Join(".claude", "settings.local.json"),
	} {
		data, err := os.ReadFile(name)
		if err != nil {
			continue
		}
		var obj map[string]any
		if json.Unmarshal(data, &obj) == nil {
			if _, ok := obj["statusLine"]; ok {
				return name
			}
		}
	}
	return ""
}
