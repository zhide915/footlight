package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatal(err)
	}
	return obj
}

func statusLineCmd(t *testing.T, obj map[string]any) string {
	t.Helper()
	sl, ok := obj["statusLine"].(map[string]any)
	if !ok {
		t.Fatalf("no statusLine in %v", obj)
	}
	return sl["command"].(string)
}

func TestWriteDefaultConfigNoClobber(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".zhide915", "footlight.toml")

	created, err := writeDefaultConfig(path)
	if err != nil || !created {
		t.Fatalf("first write: created=%v err=%v", created, err)
	}

	if err := os.WriteFile(path, []byte("custom = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	created, err = writeDefaultConfig(path)
	if err != nil || created {
		t.Fatalf("second write: created=%v err=%v, want created=false", created, err)
	}
	if data, _ := os.ReadFile(path); string(data) != "custom = true\n" {
		t.Errorf("clobbered existing config: %q", data)
	}
}

func TestWireSettings_NoFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude", "settings.json")
	st, err := wireSettings(path, `"C:\footlight.exe"`, false)
	if err != nil || st != statusCreated {
		t.Fatalf("status=%v err=%v, want created", st, err)
	}
	if got := statusLineCmd(t, readJSON(t, path)); got != `"C:\footlight.exe"` {
		t.Errorf("command = %q", got)
	}
}

func TestWireSettings_PreservesOtherKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	os.WriteFile(path, []byte(`{"theme":"dark","permissions":{"allow":["Bash"]}}`), 0o644)

	st, err := wireSettings(path, `"footlight"`, false)
	if err != nil || st != statusUpdated {
		t.Fatalf("status=%v err=%v, want updated", st, err)
	}
	obj := readJSON(t, path)
	if obj["theme"] != "dark" {
		t.Error("theme key lost")
	}
	if _, ok := obj["permissions"]; !ok {
		t.Error("permissions key lost")
	}

	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Error("no .bak written")
	}
}

func TestWireSettings_RefuseAndForce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	os.WriteFile(path, []byte(`{"statusLine":{"type":"command","command":"npx some-other-tool"}}`), 0o644)

	st, err := wireSettings(path, `"footlight"`, false)
	if err != nil || st != statusRefused {
		t.Fatalf("status=%v err=%v, want refused", st, err)
	}
	if got := statusLineCmd(t, readJSON(t, path)); got != "npx some-other-tool" {
		t.Errorf("refused but file changed: %q", got)
	}

	st, err = wireSettings(path, `"footlight"`, true)
	if err != nil || st != statusUpdated {
		t.Fatalf("force status=%v err=%v, want updated", st, err)
	}
	if got := statusLineCmd(t, readJSON(t, path)); got != `"footlight"` {
		t.Errorf("force command = %q", got)
	}
}

func TestWireSettings_IdempotentForStatusLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	os.WriteFile(path, []byte(`{"statusLine":{"type":"command","command":"\"C:\\go\\bin\\footlight.exe\""}}`), 0o644)

	st, err := wireSettings(path, `"new"`, false)
	if err != nil || st != statusNoop {
		t.Fatalf("status=%v err=%v, want noop", st, err)
	}
}

func TestWireSettings_RefuseMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	os.WriteFile(path, []byte("{ this is not json"), 0o644)

	_, err := wireSettings(path, `"footlight"`, true)
	if err == nil {
		t.Error("malformed settings.json should refuse, not overwrite")
	}

	if data, _ := os.ReadFile(path); string(data) != "{ this is not json" {
		t.Errorf("malformed file was modified: %q", data)
	}
}
