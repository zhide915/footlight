package widget

import (
	"fmt"
	"time"

	"github.com/zhide915/footlight/internal/color"
	"github.com/zhide915/footlight/internal/config"
	"github.com/zhide915/footlight/internal/git"
	"github.com/zhide915/footlight/internal/session"
)

type Ctx struct {
	Session *session.Session
	Git     *git.GitInfo
	Color   color.Mode
	Thresh  color.Threshold
	Now     time.Time
}

type Widget interface {
	Render(c Ctx) (text string, ok bool)
}

type Constructor func(opts map[string]any) (Widget, error)

type BuiltLine struct {
	Widgets []Widget
	Sep     string
}

func Build(spec config.WidgetSpec) (Widget, error) {
	c, ok := registry[spec.Type]
	if !ok {
		return nil, fmt.Errorf("unknown widget %q", spec.Type)
	}
	return c(spec.Options)
}

// Resolve builds every widget in cfg, collecting errors rather than failing on
// them: a bad widget is dropped and its error returned, so the caller can show
// a ⚠ cfg marker instead of an empty status line.
func Resolve(cfg config.Config) ([]BuiltLine, []error) {
	lines := make([]BuiltLine, 0, len(cfg.Lines))
	var errs []error
	for _, ln := range cfg.Lines {
		bl := BuiltLine{Sep: ln.Sep}
		for _, spec := range ln.Widgets {
			w, err := Build(spec)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bl.Widgets = append(bl.Widgets, w)
		}
		lines = append(lines, bl)
	}
	return lines, errs
}

func NeedsGit(lines []BuiltLine) bool {
	for _, bl := range lines {
		for _, w := range bl.Widgets {
			if _, ok := w.(gitWidget); ok {
				return true
			}
		}
	}
	return false
}

type baseStyle struct{ style color.Style }

func parseBase(o map[string]any) (baseStyle, error) {
	var b baseStyle
	if v, ok := o["fg"].(string); ok {
		c, err := color.Parse(v)
		if err != nil {
			return b, err
		}
		b.style.Fg = c
	}
	if v, ok := o["bg"].(string); ok {
		c, err := color.Parse(v)
		if err != nil {
			return b, err
		}
		b.style.Bg = c
	}
	return b, nil
}

func (b baseStyle) paint(c Ctx, text string) string { return b.style.Apply(c.Color, text) }

func defFg(b baseStyle, def color.Color) baseStyle {
	if !b.style.Fg.Set() {
		b.style.Fg = def
	}
	return b
}
