package config

import "testing"

func TestParseHybridWidgets(t *testing.T) {
	in := `
[[line]]
widgets = ["model", { type = "dir", full = true }, "git"]

[[line]]
widgets = [{ type = "context", bar = true }, "cost"]
`
	cfg, err := ParseBytes([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(cfg.Lines))
	}

	l0 := cfg.Lines[0].Widgets
	if len(l0) != 3 || l0[0].Type != "model" || l0[1].Type != "dir" || l0[2].Type != "git" {
		t.Fatalf("line0 widgets = %+v", l0)
	}
	if full, _ := l0[1].Options["full"].(bool); !full {
		t.Errorf("dir.full = %v, want true", l0[1].Options["full"])
	}
	if l0[0].Options != nil {
		t.Errorf("bare string widget should have nil options, got %v", l0[0].Options)
	}
}

func TestParseThresholdOverride(t *testing.T) {
	cfg, err := ParseBytes([]byte("[threshold]\nwarn_pct = 60\ndanger_pct = 80\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Threshold.Warn != 60 || cfg.Threshold.Danger != 80 {
		t.Errorf("threshold = %v/%v, want 60/80", cfg.Threshold.Warn, cfg.Threshold.Danger)
	}

	if len(cfg.Lines) == 0 {
		t.Error("partial config blanked the lines")
	}
}

func TestParseMalformed(t *testing.T) {
	_, err := ParseBytes([]byte("this is = = not toml"))
	if err == nil {
		t.Error("malformed TOML should return an error")
	}
}

func TestWidgetTableMissingType(t *testing.T) {
	_, err := ParseBytes([]byte("[[line]]\nwidgets = [{ fg = \"red\" }]\n"))
	if err == nil {
		t.Error("widget table without 'type' should error")
	}
}
