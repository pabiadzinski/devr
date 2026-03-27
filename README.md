<p align="center">
  <h1 align="center">devr</h1>
  <p align="center">
    <strong>A zero-config dev runner for Go projects</strong>
  </p>
  <p align="center">
    <a href="https://github.com/pabiadzinski/devr/actions/workflows/ci.yml"><img src="https://github.com/pabiadzinski/devr/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
    <a href="https://codecov.io/gh/pabiadzinski/devr"><img src="https://codecov.io/gh/pabiadzinski/devr/branch/main/graph/badge.svg" alt="Coverage"></a>
    <a href="https://goreportcard.com/report/github.com/pabiadzinski/devr"><img src="https://goreportcard.com/badge/github.com/pabiadzinski/devr" alt="Go Report Card"></a>
  </p>
  <p align="center">
    Build, run, watch, test – all from one command.<br>
    With a built-in TUI log viewer that has vim keybindings, log level filtering, and JSON pretty-printing.
  </p>
</p>

<p align="center">
  <img src="demo/demo.gif" alt="devr demo" width="800">
</p>

<br>

## Why devr?

Most Go dev workflows look like this: `go run .`, manually restart on every change, no log filtering or search. **devr** replaces all of that with a single command.

- **One command to run** – `devr app run` builds, starts, and opens a log viewer
- **Auto-restart on changes** – `devr app watch` rebuilds on every `.go` file save
- **TUI log viewer** – search, filter by level, JSON preview, vim navigation
- **Pretty test output** – dots, compact, or verbose – with failure summaries
- **Zero config** – works out of the box, optional `.devr.yaml` for customization

## Install

```bash
go install github.com/pabiadzinski/devr/cmd/devr@latest
```

## Get started

```bash
devr init myapp && cd myapp
devr app run
```

That's it. Your app is running and you're in the log viewer.

## Usage

### Run & Watch

```bash
devr app run              # build, start, open log viewer
devr app run --race       # enable race detector
devr app run --no-env     # skip loading .env file
devr app run --env-file .env.local
devr app watch            # same, but auto-restart on .go changes
devr app watch --debounce 1s
devr app stop             # send SIGTERM to the background process
devr app attach           # reattach to a running process
devr app logs             # view logs from last run
devr app ps               # list all managed processes
```

CLI flags override `.devr.yaml` config. For example, `--race` adds `-race` even if not in `build.flags`.

### Test

```bash
devr test run             # run tests with compact output
devr test run --race      # enable race detector
devr test run -f dots     # minimal dot output
devr test run -f verbose  # full verbose
devr test run -r TestFoo  # run specific tests
devr test bench           # run benchmarks
devr test cover           # coverage report, opens in browser
devr test cover --profile custom.out
```

All build commands use `build.flags` from `.devr.yaml` (e.g. `-race` by default). CLI `--race` flag overrides this.

### Scaffold

```bash
devr init myapp           # creates go.mod, main.go, .gitignore
```

## Log Viewer

The built-in log viewer gives you a full-screen TUI with real-time tailing:

| Key | Action |
|-----|--------|
| `j/k`, `arrows` | Navigate |
| `Ctrl+D/U` | Half page scroll |
| `Ctrl+F/B` | Full page scroll |
| `H/M/L` | Top / middle / bottom of screen |
| `g` / `G` | Jump to top / bottom |
| `/` | Filter lines (search + hide non-matching) |
| `s` | Search with highlight (no filtering) |
| `n` / `N` | Next / prev search match |
| `1` `2` `3` `4` | Filter: error, warn, info, debug |
| `0` | Clear filter |
| `w` | Toggle line wrap (on by default) |
| `Alt+Enter` | Insert marker line |
| `Tab` | JSON pretty-print panel |
| `Enter` | Insert blank line |
| `y` | Copy line to clipboard |
| `q` | Detach (process keeps running) |
| `Ctrl+C` x2 | Stop process & exit |

## Shell Completions

```bash
command devr completion fish | source   # fish (add to ~/.config/fish/config.fish)
eval "$(devr completion bash)"  # bash (add to ~/.bashrc)
eval "$(devr completion zsh)"   # zsh  (add to ~/.zshrc)
```

## Configuration

Drop a `.devr.yaml` in your project root to customize behavior. All fields are optional:

```yaml
build:
  cmd_pattern: "cmd/*/main.go"
  flags: ["-race"]

run:
  env_file: ".env"       # or .env.local, etc. (defaults to .env)

watch:
  extensions: [".go"]
  exclude: ["vendor", "node_modules"]
  debounce: 500ms

logs:
  format: auto          # auto, json, or text
  level_field: level    # JSON field or key=value field for level
  level_values:
    error: ["error", "err", "fatal"]
    warn: ["warn", "warning"]
    info: ["info"]
    debug: ["debug", "trace"]
  highlight_fields: ["msg"]

test:
  cover_profile: "coverage.out"

notify: true  # macOS desktop notifications on build/crash failures
```

By default, `devr` uses `logs.format: auto`: it first tries JSON logs, then falls back to plain text / `key=value` parsing. This works out of the box for logs like `{"level":"info","msg":"ready"}`, `INFO server started`, or `lvl=warning msg="slow query"` if you set `level_field: lvl`.

Without a config file, devr uses sensible defaults – it just works.

## License

MIT
