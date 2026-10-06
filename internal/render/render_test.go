package render

import (
	"strings"
	"testing"

	"github.com/zhide915/footlight/internal/color"
	"github.com/zhide915/footlight/internal/config"
	"github.com/zhide915/footlight/internal/git"
	"github.com/zhide915/footlight/internal/session"
	"github.com/zhide915/footlight/internal/widget"
)

func fixture() *session.Session {
	pct := 21.0
	name := "Discuss next steps for project"
	return &session.Session{
		Model:         session.Model{DisplayName: "Opus 4.8 (1M context)"},
		Workspace:     session.Workspace{CurrentDir: `C:\tzd\code\zhide915\footlight`},
		ContextWindow: &session.ContextWindow{UsedPercentage: &pct},
		Cost:          &session.Cost{TotalCostUSD: 13.6526, TotalDurationMS: 5917175, TotalLinesAdded: 1210, TotalLinesRemoved: 106},
		Effort:        &session.Effort{Level: "high"},
		SessionName:   &name,
		RateLimits:    &session.RateLimits{FiveHour: session.RateWindow{UsedPercentage: 45}, SevenDay: session.RateWindow{UsedPercentage: 17}},
	}
}

func ctx(s *session.Session, g *git.GitInfo, mode color.Mode) widget.Ctx {
	return widget.Ctx{Session: s, Git: g, Color: mode, Thresh: color.DefaultThreshold()}
}

func TestRenderDefaultLayout(t *testing.T) {
	lines, errs := widget.Resolve(config.Default())
	if len(errs) != 0 {
		t.Fatalf("resolve errors: %v", errs)
	}
	want := "footlight · +1210 -106 · Opus 4.8 (1M context) · high · " +
		"██░░░░░░░░ 21% · 1h38m · $13.65 · 5h 45% · 7d 17%"
	if got := Render(lines, ctx(fixture(), nil, color.Off), false); got != want {
		t.Errorf("Render =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderWithGit(t *testing.T) {
	lines, _ := widget.Resolve(config.Default())
	got := Render(lines, ctx(fixture(), &git.GitInfo{Branch: "main", Untracked: 3}, color.Off), false)
	if !strings.Contains(got, "footlight · main ?3") {
		t.Errorf("git segment missing from line 1:\n%q", got)
	}
}

func TestRenderCfgMarker(t *testing.T) {
	lines, _ := widget.Resolve(config.Default())
	got := Render(lines, ctx(fixture(), nil, color.Off), true)
	if !strings.HasSuffix(got, "⚠ cfg") {
		t.Errorf("expected ⚠ cfg marker, got:\n%q", got)
	}
}

func TestRenderColorOnEmitsAnsi(t *testing.T) {
	lines, _ := widget.Resolve(config.Default())
	got := Render(lines, ctx(fixture(), nil, color.On), false)
	if !strings.Contains(got, "\x1b[") {
		t.Errorf("expected ANSI escapes when color On, got:\n%q", got)
	}
}
