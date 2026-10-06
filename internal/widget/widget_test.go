package widget

import (
	"strings"
	"testing"
	"time"

	"github.com/zhide915/footlight/internal/color"
	"github.com/zhide915/footlight/internal/config"
	"github.com/zhide915/footlight/internal/git"
	"github.com/zhide915/footlight/internal/session"
)

func f64(v float64) *float64 { return &v }
func str(v string) *string   { return &v }

func withThinking(s *session.Session) *session.Session {
	cp := *s
	cp.Thinking = &session.Thinking{Enabled: true}
	return &cp
}

func mustBuild(t *testing.T, typ string, opts map[string]any) Widget {
	t.Helper()
	w, err := Build(config.WidgetSpec{Type: typ, Options: opts})
	if err != nil {
		t.Fatalf("Build(%q): %v", typ, err)
	}
	return w
}

func offCtx(s *session.Session, g *git.GitInfo) Ctx {
	return Ctx{Session: s, Git: g, Color: color.Off, Thresh: color.DefaultThreshold()}
}

func TestBuildUnknownType(t *testing.T) {
	if _, err := Build(config.WidgetSpec{Type: "nope"}); err == nil {
		t.Error("unknown widget type should error (drives ⚠ cfg)")
	}
}

func TestBuildBadColorOption(t *testing.T) {
	if _, err := Build(config.WidgetSpec{Type: "model", Options: map[string]any{"fg": "notacolor"}}); err == nil {
		t.Error("invalid fg color should error")
	}
}

func TestWidgetRender(t *testing.T) {
	full := &session.Session{
		Model:         session.Model{DisplayName: "Opus 4.8"},
		Workspace:     session.Workspace{CurrentDir: `C:\a\b\footlight`},
		ContextWindow: &session.ContextWindow{UsedPercentage: f64(21)},
		Cost:          &session.Cost{TotalCostUSD: 13.65, TotalDurationMS: 5917175, TotalLinesAdded: 1210, TotalLinesRemoved: 106},
		Effort:        &session.Effort{Level: "high"},
		SessionName:   str("X"),
		RateLimits:    &session.RateLimits{FiveHour: session.RateWindow{UsedPercentage: 45}, SevenDay: session.RateWindow{UsedPercentage: 17}},
	}
	empty := &session.Session{}

	cases := []struct {
		name string
		typ  string
		opts map[string]any
		s    *session.Session
		g    *git.GitInfo
		want string
		ok   bool
	}{
		{"model", "model", nil, full, nil, "Opus 4.8", true},
		{"model/absent", "model", nil, empty, nil, "", false},
		{"cost", "cost", nil, full, nil, "$13.65", true},
		{"duration", "duration", nil, full, nil, "1h38m", true},
		{"effort", "effort", nil, full, nil, "high", true},
		{"effort+thinking", "effort", nil, withThinking(full), nil, "high + thinking", true},
		{"session", "session", nil, full, nil, "X", true},
		{"dir/basename", "dir", nil, full, nil, "footlight", true},
		{"dir/full", "dir", map[string]any{"full": true}, full, nil, `C:\a\b\footlight`, true},
		{"context", "context", nil, full, nil, "21% ctx", true},
		{"context/bar", "context", map[string]any{"bar": true}, full, nil, "██░░░░░░░░ 21%", true},
		{"context/absent", "context", nil, empty, nil, "", false},
		{"lines", "lines", nil, full, nil, "+1210 -106", true},
		{"lines/absent", "lines", nil, empty, nil, "", false},
		{"git", "git", nil, full, &git.GitInfo{Branch: "main", Ahead: 1, Untracked: 2}, "main ↑1 ?2", true},
		{"git/absent", "git", nil, full, nil, "", false},
		{"limit_5h", "limit_5h", nil, full, nil, "5h 45%", true},
		{"limit_7d", "limit_7d", nil, full, nil, "7d 17%", true},
		{"limit/absent", "limit_5h", nil, empty, nil, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := mustBuild(t, c.typ, c.opts)
			got, ok := w.Render(offCtx(c.s, c.g))
			if got != c.want || ok != c.ok {
				t.Errorf("= (%q, %v), want (%q, %v)", got, ok, c.want, c.ok)
			}
		})
	}
}

func TestDefaultColors(t *testing.T) {
	s := &session.Session{
		Model:       session.Model{DisplayName: "M"},
		Workspace:   session.Workspace{CurrentDir: "/a/b"},
		Cost:        &session.Cost{},
		Effort:      &session.Effort{Level: "high"},
		SessionName: str("S"),
	}
	on := Ctx{Session: s, Color: color.On, Thresh: color.DefaultThreshold()}

	defaults := map[string]string{
		"model": "38;2;224;161;136", "dir": "38;2;232;179;57", "git": "38;2;91;192;190",
		"cost": "38;2;95;197;155", "duration": "38;2;141;180;226", "effort": "38;2;198;120;221",
		"session": "38;2;209;159;180",
	}
	g := &git.GitInfo{Branch: "main"}
	for typ, seq := range defaults {
		w := mustBuild(t, typ, nil)
		got, ok := w.Render(Ctx{Session: s, Git: g, Color: color.On, Thresh: on.Thresh})
		if !ok || !strings.Contains(got, seq) {
			t.Errorf("%s default color: got %q, want contains %q", typ, got, seq)
		}
	}

	w := mustBuild(t, "model", map[string]any{"fg": "red"})
	if got, _ := w.Render(on); !strings.Contains(got, "38;5;1") || strings.Contains(got, "38;2;224;161;136") {
		t.Errorf("fg override: got %q, want red (38;5;1) not the model default", got)
	}
}

func TestLimitResetTime(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	resets := now.Add(2*time.Hour + 14*time.Minute).Unix()
	s := &session.Session{RateLimits: &session.RateLimits{
		FiveHour: session.RateWindow{UsedPercentage: 45, ResetsAt: resets},
	}}
	ctx := Ctx{Session: s, Color: color.Off, Thresh: color.DefaultThreshold(), Now: now}

	if got, _ := mustBuild(t, "limit_5h", nil).Render(ctx); got != "5h 45% (2h14m)" {
		t.Errorf("= %q, want %q", got, "5h 45% (2h14m)")
	}

	past := &session.Session{RateLimits: &session.RateLimits{
		FiveHour: session.RateWindow{UsedPercentage: 45, ResetsAt: now.Add(-time.Hour).Unix()},
	}}
	if got, _ := mustBuild(t, "limit_5h", nil).Render(Ctx{Session: past, Color: color.Off, Thresh: color.DefaultThreshold(), Now: now}); got != "5h 45%" {
		t.Errorf("past reset = %q, want %q", got, "5h 45%")
	}
}

func TestNeedsGit(t *testing.T) {
	with, _ := Resolve(config.Config{Lines: []config.Line{{Widgets: []config.WidgetSpec{{Type: "model"}, {Type: "git"}}}}})
	if !NeedsGit(with) {
		t.Error("NeedsGit = false, want true (git widget present)")
	}
	without, _ := Resolve(config.Config{Lines: []config.Line{{Widgets: []config.WidgetSpec{{Type: "model"}}}}})
	if NeedsGit(without) {
		t.Error("NeedsGit = true, want false (no git widget)")
	}
}
