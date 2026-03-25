<p align="center">
  <h1 align="center">devr</h1>
  <p align="center">
    <strong>A zero-config dev runner for Go projects</strong>
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
devr app run --env .env   # same, but load .env file
devr app watch            # same, but auto-restart on .go changes
devr app watch -e .env    # watch with env file
devr app stop             # send SIGTERM to the background process
devr app attach           # reattach to a running process
devr app logs             # view logs from last run
devr app ps               # list all managed processes
```

### Test

```bash
devr test run             # run tests with compact output
devr test run -f dots     # minimal dot output
devr test run -f verbose  # full verbose
devr test run -r TestFoo  # run specific tests
devr test bench           # run benchmarks
devr test cover           # coverage report, opens in browser
```

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
| `g` / `G` | Jump to top / bottom |
| `/` | Search & filter |
| `1` `2` `3` `4` | Filter: error, warn, info, debug |
| `0` | Show all |
| `Tab` | JSON pretty-print panel |
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
  env_file: ".env"       # or .env.local, etc. (empty by default)

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
