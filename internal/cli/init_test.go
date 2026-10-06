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

func TestWireSettings_BacksUpOriginal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	const orig = `{"theme":"dark","permissions":{"allow":["Bash"]}}`
	os.WriteFile(path, []byte(orig), 0o644)

	st, err := wireSettings(path, `"footlight"`, false)
	if err != nil || st != statusUpdated {
		t.Fatalf("status=%v err=%v, want updated", st, err)
	}
	if data, _ := os.ReadFile(path + ".bak"); string(data) != orig {
		t.Errorf(".bak = %q, want the original %q", data, orig)
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

func TestWireSettings_EditsInPlace(t *testing.T) {
	const fl = `"command": "footlight"`
	cases := []struct {
		name   string
		in     string
		cmd    string // "footlight" when empty
		force  bool
		status wireStatus
		want   string
	}{
		{
			name:   "insert keeps order and indentation",
			in:     "{\n    \"zeta\": 1,\n    \"alpha\": {\n        \"x\": [1, 2]\n    }\n}\n",
			status: statusUpdated,
			want: "{\n    \"zeta\": 1,\n    \"alpha\": {\n        \"x\": [1, 2]\n    },\n" +
				"    \"statusLine\": {\n        \"type\": \"command\",\n        " + fl + "\n    }\n}\n",
		},
		{
			name:   "insert keeps CRLF line endings",
			in:     "{\r\n  \"a\": 1\r\n}\r\n",
			status: statusUpdated,
			want:   "{\r\n  \"a\": 1,\r\n  \"statusLine\": {\r\n    \"type\": \"command\",\r\n    " + fl + "\r\n  }\r\n}\r\n",
		},
		{
			name:   "insert into a compact object",
			in:     `{"theme":"dark","permissions":{"allow":["Bash"]}}`,
			status: statusUpdated,
			want:   "{\"theme\":\"dark\",\"permissions\":{\"allow\":[\"Bash\"]},\n  \"statusLine\": {\n    \"type\": \"command\",\n    " + fl + "\n  }}",
		},
		{
			name:   "insert leaves & unescaped",
			in:     "{}",
			cmd:    "'C:/A&B/footlight.exe'",
			status: statusUpdated,
			want:   "{\n  \"statusLine\": {\n    \"type\": \"command\",\n    \"command\": \"'C:/A&B/footlight.exe'\"\n  }\n}",
		},
		{
			name:   "insert into empty object",
			in:     "{}\n",
			status: statusUpdated,
			want:   "{\n  \"statusLine\": {\n    \"type\": \"command\",\n    " + fl + "\n  }\n}\n",
		},
		{
			name:   "force replaces command, fixes type, keeps padding",
			in:     "{\n  \"statusLine\": {\"type\": \"static\", \"command\": \"other\", \"padding\": 2},\n  \"b\": true\n}\n",
			force:  true,
			status: statusUpdated,
			want:   "{\n  \"statusLine\": {\"type\": \"command\", \"command\": \"footlight\", \"padding\": 2},\n  \"b\": true\n}\n",
		},
		{
			name:   "force repairs a stale footlight command",
			in:     "{\n  \"statusLine\": {\"type\": \"command\", \"command\": \"\\\"$HOME/go/bin/footlight.exe\\\"\"}\n}\n",
			force:  true,
			status: statusUpdated,
			want:   "{\n  \"statusLine\": {\"type\": \"command\", \"command\": \"footlight\"}\n}\n",
		},
		{
			name:   "force replaces an object without type whole",
			in:     "{\n  \"statusLine\": {\"command\": \"other\"}\n}\n",
			force:  true,
			status: statusUpdated,
			want:   "{\n  \"statusLine\": {\n    \"type\": \"command\",\n    " + fl + "\n  }\n}\n",
		},
		{
			name:   "non-object statusLine is refused",
			in:     `{"statusLine": null}`,
			status: statusRefused,
			want:   `{"statusLine": null}`,
		},
		{
			name:   "force replaces a non-object statusLine",
			in:     `{"statusLine": null}`,
			force:  true,
			status: statusUpdated,
			want:   "{\"statusLine\": {\n    \"type\": \"command\",\n    " + fl + "\n  }}",
		},
		{
			name:   "repeated key: the last value decides",
			in:     `{"statusLine": {"type": "command", "command": "footlight"}, "statusLine": {"type": "command", "command": "other"}}`,
			status: statusRefused,
			want:   `{"statusLine": {"type": "command", "command": "footlight"}, "statusLine": {"type": "command", "command": "other"}}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			os.WriteFile(path, []byte(tc.in), 0o644)

			cmd := tc.cmd
			if cmd == "" {
				cmd = "footlight"
			}
			st, err := wireSettings(path, cmd, tc.force)
			if err != nil || st != tc.status {
				t.Fatalf("status=%v err=%v, want %v", st, err, tc.status)
			}
			if got, _ := os.ReadFile(path); string(got) != tc.want {
				t.Errorf("file =\n%s\nwant\n%s", got, tc.want)
			}
		})
	}
}
