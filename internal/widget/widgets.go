package widget

import (
	"fmt"
	"strings"
	"time"

	"github.com/zhide915/footlight/internal/color"
	"github.com/zhide915/footlight/internal/session"
)

// green and red are shared by the lines widget; the cX values are per-widget
// foreground defaults, applied only when the config sets no explicit fg.
var (
	green = color.MustParse("#98C379")
	red   = color.MustParse("#E06C75")

	cModel    = color.MustParse("#E0A188")
	cDir      = color.MustParse("#E8B339")
	cGit      = color.MustParse("#5BC0BE")
	cCost     = color.MustParse("#5FC59B")
	cDuration = color.MustParse("#8DB4E2")
	cEffort   = color.MustParse("#C678DD")
	cSession  = color.MustParse("#D19FB4")
)

// registry maps a config widget type to its constructor. The order here mirrors
// the order the widget definitions appear in below.
var registry = map[string]Constructor{
	"model":    simpleC(model, cModel),
	"cost":     simpleC(cost, cCost),
	"duration": simpleC(duration, cDuration),
	"effort":   simpleC(effort, cEffort),
	"session":  simpleC(sessionName, cSession),
	"dir":      dirC,
	"context":  contextC,
	"lines":    baseC(func(b baseStyle) Widget { return linesWidget{b} }),
	"git":      gitC,
	"limit_5h": limitC("5h", fiveHour),
	"limit_7d": limitC("7d", sevenDay),
}

// --- constructor helpers ---

// simple adapts a plain session accessor into a Widget with a styled foreground.
type simple struct {
	base baseStyle
	fn   func(*session.Session) (string, bool)
}

func simpleC(fn func(*session.Session) (string, bool), def color.Color) Constructor {
	return func(o map[string]any) (Widget, error) {
		b, err := parseBase(o)
		if err != nil {
			return nil, err
		}
		return simple{defFg(b, def), fn}, nil
	}
}

func (w simple) Render(c Ctx) (string, bool) {
	t, ok := w.fn(c.Session)
	if !ok {
		return "", false
	}
	return w.base.paint(c, t), true
}

func baseC(make func(baseStyle) Widget) Constructor {
	return func(o map[string]any) (Widget, error) {
		b, err := parseBase(o)
		if err != nil {
			return nil, err
		}
		return make(b), nil
	}
}

// --- session-field widgets (built via simpleC) ---

func model(s *session.Session) (string, bool) {
	if s.Model.DisplayName == "" {
		return "", false
	}
	return s.Model.DisplayName, true
}

func cost(s *session.Session) (string, bool) {
	if s.Cost == nil {
		return "", false
	}
	return fmt.Sprintf("$%.2f", s.Cost.TotalCostUSD), true
}

func duration(s *session.Session) (string, bool) {
	if s.Cost == nil {
		return "", false
	}
	return humanizeMS(s.Cost.TotalDurationMS), true
}

func effort(s *session.Session) (string, bool) {
	if s.Effort == nil || s.Effort.Level == "" {
		return "", false
	}
	out := s.Effort.Level
	if s.Thinking != nil && s.Thinking.Enabled {
		out += " + thinking"
	}
	return out, true
}

func sessionName(s *session.Session) (string, bool) {
	if s.SessionName == nil || *s.SessionName == "" {
		return "", false
	}
	return *s.SessionName, true
}

// --- dir ---

type dirWidget struct {
	base baseStyle
	full bool
}

func dirC(o map[string]any) (Widget, error) {
	b, err := parseBase(o)
	if err != nil {
		return nil, err
	}
	full, _ := o["full"].(bool)
	return dirWidget{defFg(b, cDir), full}, nil
}

func (w dirWidget) Render(c Ctx) (string, bool) {
	d := c.Session.Workspace.CurrentDir
	if d == "" {
		return "", false
	}
	if !w.full {
		d = baseName(d)
	}
	return w.base.paint(c, d), true
}

// --- context ---

type contextWidget struct {
	base  baseStyle
	bar   bool
	width int
}

func contextC(o map[string]any) (Widget, error) {
	b, err := parseBase(o)
	if err != nil {
		return nil, err
	}
	bar, _ := o["bar"].(bool)
	return contextWidget{b, bar, optInt(o, "width", 10)}, nil
}

func (w contextWidget) Render(c Ctx) (string, bool) {
	cw := c.Session.ContextWindow
	if cw == nil || cw.UsedPercentage == nil {
		return "", false
	}
	pct := *cw.UsedPercentage
	var text string
	if w.bar {
		text = renderBar(pct, w.width)
	} else {
		text = fmt.Sprintf("%d%% ctx", int(pct))
	}
	// Color comes from the usage threshold, not w.base; parseBase still ran so a
	// bad fg/bg in the config is rejected rather than silently ignored.
	return color.FG(c.Thresh.Pick(pct)).Apply(c.Color, text), true
}

func renderBar(pct float64, width int) string {
	if width <= 0 {
		width = 10
	}
	filled := int(pct/100*float64(width) + 0.5)
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + fmt.Sprintf(" %d%%", int(pct))
}

// --- lines ---

type linesWidget struct{ base baseStyle }

func (w linesWidget) Render(c Ctx) (string, bool) {
	if c.Session.Cost == nil {
		return "", false
	}
	add := color.FG(green).Apply(c.Color, fmt.Sprintf("+%d", c.Session.Cost.TotalLinesAdded))
	rem := color.FG(red).Apply(c.Color, fmt.Sprintf("-%d", c.Session.Cost.TotalLinesRemoved))
	return add + " " + rem, true
}

// --- limit ---

type limitWidget struct {
	label string
	pick  func(*session.Session) (session.RateWindow, bool)
}

func limitC(label string, pick func(*session.Session) (session.RateWindow, bool)) Constructor {
	return func(o map[string]any) (Widget, error) {
		// Like context, the limit color is threshold-driven; parseBase runs only
		// to validate any fg/bg the user set.
		if _, err := parseBase(o); err != nil {
			return nil, err
		}
		return limitWidget{label, pick}, nil
	}
}

func (w limitWidget) Render(c Ctx) (string, bool) {
	rw, ok := w.pick(c.Session)
	if !ok {
		return "", false
	}
	text := fmt.Sprintf("%s %d%%", w.label, int(rw.UsedPercentage))
	if rw.ResetsAt > 0 {
		if d := time.Unix(rw.ResetsAt, 0).Sub(c.Now); d > 0 {
			text += " (" + shortDur(d) + ")"
		}
	}
	return color.FG(c.Thresh.Pick(rw.UsedPercentage)).Apply(c.Color, text), true
}

func fiveHour(s *session.Session) (session.RateWindow, bool) {
	if s.RateLimits == nil {
		return session.RateWindow{}, false
	}
	return s.RateLimits.FiveHour, true
}

func sevenDay(s *session.Session) (session.RateWindow, bool) {
	if s.RateLimits == nil {
		return session.RateWindow{}, false
	}
	return s.RateLimits.SevenDay, true
}

// --- git ---

type gitWidget struct{ base baseStyle }

func gitC(o map[string]any) (Widget, error) {
	b, err := parseBase(o)
	if err != nil {
		return nil, err
	}
	return gitWidget{defFg(b, cGit)}, nil
}

func (w gitWidget) Render(c Ctx) (string, bool) {
	g := c.Git
	if g == nil || g.Branch == "" {
		return "", false
	}
	tokens := []string{g.Branch}
	if g.Ahead > 0 {
		tokens = append(tokens, fmt.Sprintf("↑%d", g.Ahead))
	}
	if g.Behind > 0 {
		tokens = append(tokens, fmt.Sprintf("↓%d", g.Behind))
	}
	if g.Staged > 0 {
		tokens = append(tokens, fmt.Sprintf("+%d", g.Staged))
	}
	if g.Unstaged > 0 {
		tokens = append(tokens, fmt.Sprintf("~%d", g.Unstaged))
	}
	if g.Untracked > 0 {
		tokens = append(tokens, fmt.Sprintf("?%d", g.Untracked))
	}
	return w.base.paint(c, strings.Join(tokens, " ")), true
}

// --- formatting helpers ---

func baseName(p string) string {
	p = strings.TrimRight(p, `/\`)
	if i := strings.LastIndexAny(p, `/\`); i >= 0 {
		return p[i+1:]
	}
	return p
}

func humanizeMS(ms int64) string {
	sec := ms / 1000
	h := sec / 3600
	m := (sec % 3600) / 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%dm", h, m)
	case m > 0:
		return fmt.Sprintf("%dm", m)
	default:
		return fmt.Sprintf("%ds", sec)
	}
}

func shortDur(d time.Duration) string {
	mins := int(d.Minutes())
	days := mins / (24 * 60)
	h := (mins / 60) % 24
	m := mins % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd%dh", days, h)
	case h > 0:
		return fmt.Sprintf("%dh%dm", h, m)
	default:
		return fmt.Sprintf("%dm", m)
	}
}

// optInt reads an integer option, accepting the int64 and float64 that a TOML
// decoder may produce as well as a plain int.
func optInt(o map[string]any, key string, def int) int {
	switch v := o[key].(type) {
	case int64:
		return int(v)
	case int:
		return v
	case float64:
		return int(v)
	}
	return def
}
