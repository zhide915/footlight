package config

import (
	"fmt"

	"github.com/zhide915/footlight/internal/color"
)

type fileConfig struct {
	Color     *bool         `toml:"color"`
	Threshold thresholdFile `toml:"threshold"`
	Line      []lineFile    `toml:"line"`
}

type lineFile struct {
	Widgets []any  `toml:"widgets"`
	Sep     string `toml:"sep"`
}

type thresholdFile struct {
	Warn      *float64 `toml:"warn_pct"`
	Danger    *float64 `toml:"danger_pct"`
	OK        string   `toml:"ok_color"`
	WarnCol   string   `toml:"warn_color"`
	DangerCol string   `toml:"danger_color"`
}

func Default() Config {
	return Config{
		Threshold: color.DefaultThreshold(),
		Lines: []Line{
			{Widgets: []WidgetSpec{
				{Type: "dir"},
				{Type: "git"},
				{Type: "lines"},
				{Type: "model"},
				{Type: "effort"},
				{Type: "context", Options: map[string]any{"bar": true}},
				{Type: "duration"},
				{Type: "cost"},
				{Type: "limit_5h"},
				{Type: "limit_7d"},
			}},
		},
	}
}

func fromFile(fc fileConfig) (Config, error) {
	c := Config{Color: fc.Color}

	th := color.DefaultThreshold()
	if fc.Threshold.Warn != nil {
		th.Warn = *fc.Threshold.Warn
	}
	if fc.Threshold.Danger != nil {
		th.Danger = *fc.Threshold.Danger
	}
	if col, err := color.Parse(fc.Threshold.OK); err == nil && col.Set() {
		th.OK = col
	}
	if col, err := color.Parse(fc.Threshold.WarnCol); err == nil && col.Set() {
		th.WarnC = col
	}
	if col, err := color.Parse(fc.Threshold.DangerCol); err == nil && col.Set() {
		th.DangerC = col
	}
	c.Threshold = th

	if len(fc.Line) == 0 {
		c.Lines = Default().Lines
		return c, nil
	}
	c.Lines = make([]Line, len(fc.Line))
	for i, lf := range fc.Line {
		specs := make([]WidgetSpec, 0, len(lf.Widgets))
		for _, raw := range lf.Widgets {
			ws, err := toWidgetSpec(raw)
			if err != nil {
				return Config{}, err
			}
			specs = append(specs, ws)
		}
		c.Lines[i] = Line{Widgets: specs, Sep: lf.Sep}
	}
	return c, nil
}

func toWidgetSpec(v any) (WidgetSpec, error) {
	switch x := v.(type) {
	case string:
		return WidgetSpec{Type: x}, nil
	case map[string]any:
		t, _ := x["type"].(string)
		if t == "" {
			return WidgetSpec{}, fmt.Errorf("widget table missing 'type'")
		}
		opts := make(map[string]any, len(x))
		for k, val := range x {
			if k != "type" {
				opts[k] = val
			}
		}
		return WidgetSpec{Type: t, Options: opts}, nil
	default:
		return WidgetSpec{}, fmt.Errorf("widget must be a string or table, got %T", v)
	}
}

const DefaultTOML = `# footlight config — https://github.com/zhide915/footlight
# Each [[line]] is one status-line row. A widget is a bare name or a table
# { type = "name", fg = "cyan", ... }. Colors: named (red, bright_blue), a 256
# index (0-255), or hex truecolor (#ff8800).

[[line]]
widgets = ["dir", "git", "lines", "model", "effort", { type = "context", bar = true }, "duration", "cost", "limit_5h", "limit_7d"]

[threshold]
warn_pct = 50    # > 50% -> yellow
danger_pct = 70  # > 70% -> red
`
