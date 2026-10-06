package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
		fmt.Println("statusLine: wired into", settingsPath)
	case statusUpdated:
		fmt.Println("statusLine: updated", settingsPath, "(backup at "+settingsPath+".bak)")
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

func commandString() string {
	return `"$HOME/go/bin/footlight.exe"`
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

func wireSettings(path, cmd string, force bool) (wireStatus, error) {
	want := map[string]any{"type": "command", "command": cmd}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return statusCreated, writeJSON(path, map[string]any{"statusLine": want})
	}
	if err != nil {
		return 0, err
	}

	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return 0, fmt.Errorf("existing %s is not valid JSON (refusing to overwrite): %w", path, err)
	}

	if existing, ok := obj["statusLine"].(map[string]any); ok {
		if isStatusLine(existing) {
			return statusNoop, nil
		}
		if !force {
			return statusRefused, nil
		}
	}

	if err := os.WriteFile(path+".bak", data, 0o644); err != nil {
		return 0, fmt.Errorf("writing backup: %w", err)
	}
	obj["statusLine"] = want
	if err := writeJSON(path, obj); err != nil {
		return 0, err
	}
	return statusUpdated, nil
}

func isStatusLine(statusLine map[string]any) bool {
	cmd, _ := statusLine["command"].(string)
	return strings.Contains(strings.ToLower(cmd), "footlight")
}

func writeJSON(path string, obj any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
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
