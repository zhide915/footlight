package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"
	"github.com/zhide915/footlight/internal/color"
)

type Config struct {
	Lines     []Line
	Threshold color.Threshold
	Color     *bool
}

type Line struct {
	Widgets []WidgetSpec
	Sep     string
}

type WidgetSpec struct {
	Type    string
	Options map[string]any
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".zhide915", "footlight.toml"), nil
}

func Path() (string, error) { return configPath() }

// Load reads ~/.zhide915/footlight.toml. A missing file returns defaults with no error; a
// present-but-unreadable or malformed file returns defaults plus the error, which
// the caller surfaces as a ⚠ cfg marker.
func Load() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Default(), err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), nil
		}
		return Default(), err
	}
	return parse(data)
}

func parse(data []byte) (Config, error) {
	var fc fileConfig
	if err := toml.Unmarshal(data, &fc); err != nil {
		return Default(), fmt.Errorf("parse config: %w", err)
	}
	cfg, err := fromFile(fc)
	if err != nil {
		return Default(), fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

func ParseBytes(data []byte) (Config, error) { return parse(data) }
