package git

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"
)

type GitInfo struct {
	Branch    string
	Ahead     int
	Behind    int
	Staged    int
	Unstaged  int
	Untracked int
}

// Status runs `git status` in dir and parses the result. It returns a nil
// GitInfo and nil error when dir is not a repo or git is unavailable: for a
// status line, the absence of git is normal, not an error to report.
func Status(ctx context.Context, dir string) (*GitInfo, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "status", "-b", "--porcelain=v2")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, nil
	}
	return Parse(out.Bytes()), nil
}

// Parse reads `git status --porcelain=v2 -b` output. Header lines start with
// "# branch.head <name>" and "# branch.ab +<ahead> -<behind>". Changed entries
// start with "1 "/"2 " and carry a two-char XY field where column X (index 2)
// is the staged state and column Y (index 3) is the unstaged state, with "."
// meaning unchanged. "? " lines are untracked files.
func Parse(b []byte) *GitInfo {
	g := &GitInfo{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			head := strings.TrimPrefix(line, "# branch.head ")
			if head == "(detached)" {
				g.Branch = "detached"
			} else {
				g.Branch = head
			}
		case strings.HasPrefix(line, "# branch.ab "):
			if ab := strings.Fields(strings.TrimPrefix(line, "# branch.ab ")); len(ab) == 2 {
				g.Ahead = absAtoi(ab[0])
				g.Behind = absAtoi(ab[1])
			}
		case strings.HasPrefix(line, "1 "), strings.HasPrefix(line, "2 "):
			if len(line) >= 4 {
				if line[2] != '.' {
					g.Staged++
				}
				if line[3] != '.' {
					g.Unstaged++
				}
			}
		case strings.HasPrefix(line, "? "):
			g.Untracked++
		}
	}
	return g
}

func absAtoi(s string) int {
	n, _ := strconv.Atoi(strings.TrimLeft(s, "+-"))
	return n
}
