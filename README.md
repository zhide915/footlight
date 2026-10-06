# footlight

A configurable status line for [Claude Code](https://docs.claude.com/en/docs/claude-code),
written in Go. Claude Code pipes a JSON description of the current session into
the program on stdin; footlight prints one formatted, colored status string back.

```
footlight · main ?3 · +1210 -106 · Opus 4.8 (1M context) · high · 21% ctx · 1h38m · $13.65 · 5h 45% (2h14m) · 7d 17% (3d4h)
```

## Install

Build from source (Go 1.26+):

```sh
go install github.com/zhide915/footlight@latest
```

or clone and `go build .`. Tagged releases (`v*`) publish static binaries for
linux/darwin/windows on amd64 and arm64.

## Setup

```sh
footlight init
```

`init` does two things, both idempotent and non-destructive:

- Writes a default `~/.zhide915/footlight.toml` if one doesn't already exist (it never
  overwrites an existing config).
- Wires `statusLine` into Claude Code's `settings.json` (under
  `$CLAUDE_CONFIG_DIR`, else `~/.claude`). If `footlight` on your `PATH` is
  the binary you ran, the command is `footlight`. Otherwise, it's the binary's
  absolute path. `init` prints the command it writes. It changes only the
  `statusLine` entry, so your key order and formatting stay as they are. If a
  `statusLine` pointing elsewhere is already set, `init` refuses unless given
  `--force`. `--force` also rewrites an existing footlight command, which
  repairs one that points at an old install. It replaces only `command` and
  `type` and keeps other fields, such as `padding`, when both are present.
  Before changing an existing file, it writes a `.bak`. It warns if a
  project-level `.claude/settings.json` (or `settings.local.json`) defines its
  own `statusLine`, since that overrides the global one.

## Configuration

Config lives at `~/.zhide915/footlight.toml`. Each `[[line]]` is one row of the status
line; `widgets` is a list where each entry is either a bare widget name or a
table `{ type = "name", ... }` carrying that widget's options. An optional
per-line `sep` overrides the default separator (`" · "`).

```toml
[[line]]
widgets = ["dir", "git", "lines", "model", "effort", { type = "context", bar = true }, "duration", "cost", "limit_5h", "limit_7d"]

[threshold]
warn_pct = 50    # usage > 50% -> warn color
danger_pct = 70  # usage > 70% -> danger color
```

A missing config falls back to the defaults shown above. A malformed config also
falls back to defaults, and the status line gains a trailing `⚠ cfg` marker.

### Widgets

| Widget | Shows | Options | Color |
| --- | --- | --- | --- |
| `model` | model display name | `fg`, `bg` | default `#E0A188`, overridable |
| `dir` | current directory; basename unless `full` | `full`, `fg`, `bg` | default `#E8B339`, overridable |
| `git` | branch, then `↑ahead ↓behind +staged ~unstaged ?untracked` | `fg`, `bg` | default `#5BC0BE`, overridable |
| `lines` | lines added/removed this session (`+N -N`) | — | fixed green `#98C379` / red `#E06C75` |
| `cost` | session cost in USD | `fg`, `bg` | default `#5FC59B`, overridable |
| `duration` | session wall-clock time | `fg`, `bg` | default `#8DB4E2`, overridable |
| `effort` | effort level, plus `+ thinking` when enabled | `fg`, `bg` | default `#C678DD`, overridable |
| `session` | session name | `fg`, `bg` | default `#D19FB4`, overridable |
| `context` | context-window usage; `bar = true` draws a block-gauge | `bar`, `width` (default 10) | by threshold |
| `limit_5h` | 5-hour rate-limit usage + time until reset | — | by threshold |
| `limit_7d` | 7-day rate-limit usage + time until reset | — | by threshold |

A widget renders nothing (and is dropped from the line) when its data is absent
from the session payload. The `context` and `limit_*` widgets take their color
from the usage threshold rather than `fg`/`bg`; `lines` is always green/red.

### Colors

`fg`/`bg` and the threshold colors accept three forms:

- a name: `red`, `green`, `bright_blue`, `gray`, … (the 16 standard colors)
- a 256-palette index: `0`–`255`
- a truecolor hex: `#ff8800` or the short `#f80`

Color is on by default. Set `color = false` at the top of the config, or set the
`NO_COLOR` environment variable, to emit plain text.

The `[threshold]` table tunes the by-usage coloring: `warn_pct` and `danger_pct`
percentages, and `ok_color` / `warn_color` / `danger_color`.

### Full example

<details>
<summary>A complete <code>~/.zhide915/footlight.toml</code>.</summary>

Unset keys take the default shown in the comment; unknown keys are ignored.

```toml
# ~/.zhide915/footlight.toml

# Master color switch (default true). The NO_COLOR env var also forces it off.
color = true

# Coloring for the usage widgets (context, limit_5h, limit_7d), by percent used.
[threshold]
warn_pct = 50            # over this percent -> warn_color   (default 50)
danger_pct = 70          # over this percent -> danger_color (default 70)
ok_color = "#98C379"     # at or below warn_pct              (default #98C379, green)
warn_color = "#E5C07B"   # default #E5C07B (yellow)
danger_color = "#E06C75" # default #E06C75 (red)

# Each [[line]] is one row, rendered top to bottom. A widget is a bare name or a
# table { type = "name", <options> }. Repeat [[line]] for more rows.
[[line]]
widgets = [
  { type = "dir", full = true },                  # full path instead of the basename
  "git",
  "lines",
  { type = "model", fg = "cyan", bg = "#222" },   # fg/bg override the widget's default color
  "effort",
]
sep = " · "  # separator between widgets on this row (default " · ")

[[line]]
widgets = [
  { type = "context", bar = true, width = 12 },   # draw a gauge; width in cells (default 10)
  "duration",
  "cost",
  "limit_5h",
  "limit_7d",
]
```

`fg`/`bg` are accepted on every widget but ignored by `context` and `limit_*`
(colored by the usage threshold) and `lines` (always green/red).

</details>

## Commands

| Command | Behavior |
| --- | --- |
| `footlight` | Read session JSON on stdin, print the status line. This is what Claude Code invokes. |
| `footlight init [--force]` | Write the default config and wire `settings.json` (see Setup). |
| `footlight dump [file]` | Render like the default command, and also save raw stdin to `file` — useful for capturing a real payload to test against. |
