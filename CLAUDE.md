# CLAUDE.md

Guidance for working in this repository. See `README.md` for user-facing usage.

## What this is

A Claude Code status line. The binary reads one JSON session payload from stdin
and writes one (possibly multi-line) status string to stdout. The only
dependency is `github.com/pelletier/go-toml/v2`.

## Build, test, lint

```sh
go build ./...
go vet ./...
go test ./...
```

CI runs `go vet`, `go test`, and `go build` on every push and PR. Keep the tree
`gofmt`-clean.

## Architecture

Data flows in one direction (`internal/cli/cli.go` orchestrates it):

```
stdin JSON
  → session.Parse        decode into session.Session (pointer fields = optional)
  → config.Load          read ~/.zhide915/footlight.toml, falling back to defaults
  → widget.Resolve       turn the config into renderable widgets
  → git.Status           only if a git widget is configured (100ms timeout)
  → render.Render        join widgets into lines, append ⚠ cfg on config error
stdout
```

Packages under `internal/`:

- `session` — the stdin schema and its decoder. Pointer fields are optional;
  nil means the field was absent.
- `config` — TOML config model and loading. `config.go` is the public model +
  `Load`; `defaults.go` holds the file schema, the file→`Config` conversion, and
  the embedded default config.
- `widget` — `widget.go` is the framework (the `Widget` interface, `Build`,
  `Resolve`, `NeedsGit`, the `baseStyle`/color plumbing); `widgets.go` holds the
  concrete widgets and the `registry` that maps a config type name to a
  constructor.
- `color` — color parsing (named / 256 / hex), ANSI emission, and the usage
  `Threshold`.
- `git` — runs and parses `git status --porcelain=v2 -b`.
- `render` — assembles widget output into the final string.
- `cli` — argument dispatch (`init`, `dump`, default) and the `init` wiring of
  `settings.json`.

## Conventions

- **Fail soft; never block the prompt.** A status line must always print a line
  and exit 0. Parse failures print an empty line; a not-a-repo / missing-git /
  timed-out git call yields no git segment; config errors fall back to defaults
  and surface as a `⚠ cfg` marker rather than an error exit. Preserve this — do
  not add fatal paths to the render flow.

- **Adding a widget:** implement the `Widget` interface (a `Render` method),
  write a `Constructor`, and register it in the `registry` map in `widgets.go`.
  In `widgets.go`, keep each widget as one block ordered `type → constructor →
  Render → widget-specific helpers`, and keep the blocks in the same order as
  the `registry`. Pure formatting helpers go in the section at the bottom.

- **Tests are table-driven**, and case order mirrors the `registry` order so the
  table cross-references the source. Tests assert exact rendered strings,
  including ANSI sequences.
