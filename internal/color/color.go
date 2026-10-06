package color

import (
	"fmt"
	"strconv"
	"strings"
)

type Mode int

const (
	Off Mode = iota
	On
)

func ResolveMode(noColor bool, cfg *bool) Mode {
	if noColor {
		return Off
	}
	if cfg != nil && !*cfg {
		return Off
	}
	return On
}

// Color is either a 256-palette index or a truecolor RGB. The zero value is
// "unset": Set reports false and Apply emits no escape for it.
type Color struct {
	set     bool
	rgb     bool
	idx     byte
	r, g, b byte
}

func (c Color) Set() bool { return c.set }

var named = map[string]byte{
	"black": 0, "red": 1, "green": 2, "yellow": 3, "blue": 4, "magenta": 5,
	"cyan": 6, "white": 7, "gray": 8, "grey": 8,
	"bright_black": 8, "bright_red": 9, "bright_green": 10, "bright_yellow": 11,
	"bright_blue": 12, "bright_magenta": 13, "bright_cyan": 14, "bright_white": 15,
}

func Parse(s string) (Color, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return Color{}, nil
	}
	if idx, ok := named[s]; ok {
		return Color{idx: idx, set: true}, nil
	}
	if strings.HasPrefix(s, "#") {
		return parseHex(s)
	}
	if n, err := strconv.Atoi(s); err == nil && n >= 0 && n <= 255 {
		return Color{idx: byte(n), set: true}, nil
	}
	return Color{}, fmt.Errorf("invalid color %q", s)
}

func parseHex(s string) (Color, error) {
	h := strings.TrimPrefix(s, "#")
	var rr, gg, bb string
	switch len(h) {
	case 3:
		rr, gg, bb = h[0:1]+h[0:1], h[1:2]+h[1:2], h[2:3]+h[2:3]
	case 6:
		rr, gg, bb = h[0:2], h[2:4], h[4:6]
	default:
		return Color{}, fmt.Errorf("invalid hex color %q", s)
	}
	r, e1 := strconv.ParseUint(rr, 16, 8)
	g, e2 := strconv.ParseUint(gg, 16, 8)
	b, e3 := strconv.ParseUint(bb, 16, 8)
	if e1 != nil || e2 != nil || e3 != nil {
		return Color{}, fmt.Errorf("invalid hex color %q", s)
	}
	return Color{set: true, rgb: true, r: byte(r), g: byte(g), b: byte(b)}, nil
}

func MustParse(s string) Color {
	c, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return c
}

func (c Color) fg() string {
	if c.rgb {
		return fmt.Sprintf("38;2;%d;%d;%d", c.r, c.g, c.b)
	}
	return "38;5;" + strconv.Itoa(int(c.idx))
}

func (c Color) bg() string {
	if c.rgb {
		return fmt.Sprintf("48;2;%d;%d;%d", c.r, c.g, c.b)
	}
	return "48;5;" + strconv.Itoa(int(c.idx))
}

type Style struct{ Fg, Bg Color }

func FG(c Color) Style { return Style{Fg: c} }

func (s Style) Apply(mode Mode, text string) string {
	if mode == Off {
		return text
	}
	var p []string
	if s.Fg.set {
		p = append(p, s.Fg.fg())
	}
	if s.Bg.set {
		p = append(p, s.Bg.bg())
	}
	if len(p) == 0 {
		return text
	}
	return "\x1b[" + strings.Join(p, ";") + "m" + text + "\x1b[0m"
}
