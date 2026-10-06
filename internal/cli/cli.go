package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/zhide915/footlight/internal/color"
	"github.com/zhide915/footlight/internal/config"
	"github.com/zhide915/footlight/internal/git"
	"github.com/zhide915/footlight/internal/render"
	"github.com/zhide915/footlight/internal/session"
	"github.com/zhide915/footlight/internal/widget"
)

// gitTimeout caps the git subprocess so a slow or huge repo can never stall the
// prompt; on timeout the git segment is simply omitted.
const gitTimeout = 100 * time.Millisecond

var commands = map[string]func(args []string) int{
	"dump": runDump,
	"init": runInit,
}

func Main(args []string) int {
	if len(args) > 0 {
		if cmd, ok := commands[args[0]]; ok {
			return cmd(args[1:])
		}
	}
	return runStatusline()
}

func runStatusline() int {
	s, err := session.Parse(os.Stdin)
	if err != nil {
		fmt.Println()
		return 0
	}
	fmt.Println(statusLine(s))
	return 0
}

// runDump renders like the default status line but also tees raw stdin to the
// file named in args[0], for capturing real Claude Code payloads to test with.
func runDump(args []string) int {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Println()
		return 0
	}
	if len(args) > 0 {
		_ = os.WriteFile(args[0], data, 0o644)
	}
	s, err := session.Parse(bytes.NewReader(data))
	if err != nil {
		fmt.Println()
		return 0
	}
	fmt.Println(statusLine(s))
	return 0
}

func statusLine(s *session.Session) string {
	cfg, loadErr := config.Load()
	lines, resolveErrs := widget.Resolve(cfg)
	cfgErr := loadErr != nil || len(resolveErrs) > 0

	// Only shell out to git when a git widget is actually configured.
	var gi *git.GitInfo
	if widget.NeedsGit(lines) {
		gi = gitInfo(s)
	}
	ctx := widget.Ctx{
		Session: s,
		Git:     gi,
		Color:   color.ResolveMode(os.Getenv("NO_COLOR") != "", cfg.Color),
		Thresh:  cfg.Threshold,
		Now:     time.Now(),
	}
	return render.Render(lines, ctx, cfgErr)
}

func gitInfo(s *session.Session) *git.GitInfo {
	dir := s.Workspace.CurrentDir
	if dir == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	info, _ := git.Status(ctx, dir)
	return info
}
