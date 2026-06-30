<p align="center">
  <h1 align="center">devr</h1>
  <p align="center">
    <strong>A TUI log viewer for your running Go service.</strong>
  </p>
  <p align="center">
    <a href="https://github.com/pabiadzinski/devr/actions/workflows/ci.yml"><img src="https://github.com/pabiadzinski/devr/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
    <a href="https://codecov.io/gh/pabiadzinski/devr"><img src="https://codecov.io/gh/pabiadzinski/devr/branch/main/graph/badge.svg" alt="Coverage"></a>
    <a href="https://goreportcard.com/report/github.com/pabiadzinski/devr"><img src="https://goreportcard.com/badge/github.com/pabiadzinski/devr" alt="Go Report Card"></a>
  </p>
  <p align="center">
    <code>devr app run</code> builds and runs your app, then pipes its logs into a<br>
    full-screen TUI you can filter, search, and pretty-print – instead of<br>
    squinting at a scrolling wall of stdout. Auto-reloads on save.
  </p>
</p>

<p align="center">
  <img src="demo/demo.gif" alt="devr demo" width="800">
</p>

<br>

## Why devr?

`go run .` dumps your app's logs as an unsearchable wall of text. Need to see
only errors? Find that one request? Read a JSON line without it word-wrapping
into mush? You can't – you scroll, squint, and re-run.

**devr** runs your Go app and opens its logs in a real viewer:

- **Filter by level** – one keystroke to show only errors, warnings, info, or debug
- **Search & highlight** – jump between matches, or hide everything that doesn't match
- **JSON pretty-print** – pop any log line open into a formatted panel
- **Vim navigation** – `j/k`, `g/G`, half/full-page scroll, top/middle/bottom
- **Detach without killing** – press `q` to leave the TUI; your app keeps running
- **Live reload** – `devr app watch` rebuilds and restarts on every `.go` save, logs keep flowing
- **Zero config** – auto-detects `cmd/*/main.go`, parses JSON / `key=value` / plain logs out of the box

## devr vs `go run` / air

|                              | `go run` | air | **devr** |
| ---------------------------- | :------: | :-: | :------: |
| Rebuild & restart on save    |    ❌    | ✅  |    ✅    |
| Logs                         | raw stdout | raw stdout | **searchable TUI** |
| Filter by log level          |    ❌    | ❌  |    ✅    |
| Search with highlight        |    ❌    | ❌  |    ✅    |
| JSON pretty-print            |    ❌    | ❌  |    ✅    |
| Vim navigation               |    ❌    | ❌  |    ✅    |
| Detach, app keeps running    |    ❌    | ❌  |    ✅    |

air is great at reloading. devr reloads **and** gives you somewhere to actually read the logs.

## Install

```bash
go install github.com/pabiadzinski/devr/cmd/devr@latest
```

## Get started

```bash
cd your-go-project
devr app run
```

That's it. Your app is building, running, and you're in the log viewer.

## Log Viewer

The heart of devr: a full-screen TUI that tails your running app in real time.

| Key | Action |
|-----|--------|
| `j/k`, `arrows` | Navigate |
| `Ctrl+D/U` | Half page scroll |
| `Ctrl+F/B` | Full page scroll |
| `H/M/L` | Top / middle / bottom of screen |
| `g` / `G` | Jump to top / bottom |
| `1` `2` `3` `4` | Filter: error, warn, info, debug |
| `0` | Clear filter |
| `/` | Filter lines (search + hide non-matching) |
| `s` | Search with highlight (no filtering) |
| `n` / `N` | Next / prev search match |
| `Tab` | JSON pretty-print panel |
| `w` | Toggle line wrap (on by default) |
| `y` | Copy line to clipboard |
| `Alt+Enter` | Insert marker line |
| `Enter` | Insert blank line |
| `q` | Detach (process keeps running) |
| `Ctrl+C` x2 | Stop process & exit |

devr reads JSON logs out of the box (`{"level":"info","msg":"ready"}`), and also
plain text (`INFO server started`) or `key=value` lines. Configure the level field
and values in `.devr.yaml` if your format differs – see [Configuration](#configuration).

## Run & Watch

```bash
devr app run              # build, start, open log viewer
devr app run --race=false # disable race detector (enabled by default)
devr app run --no-env     # skip loading .env file
devr app run --env-file .env.local
devr app watch            # same, but auto-restart on .go changes
devr app watch --debounce 1s
devr app stop             # send SIGTERM to the background process
devr app attach           # reattach to a running process
devr app logs             # view logs from last run
devr app ps               # list all managed processes
```

CLI flags override `.devr.yaml` values. Race detector is enabled by default.

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
  race: true              # enabled by default, use --race=false to disable
  flags: ["-trimpath"]    # extra go build flags

run:
  env_file: ".env"        # or .env.local, etc. (defaults to .env)
  no_env: false           # skip loading env file

watch:
  extensions: [".go"]
  exclude: ["vendor", "node_modules"]
  debounce: 500ms

logs:
  format: auto            # auto, json, or text
  level_field: level      # JSON field or key=value field for level
  level_values:
    error: ["error", "err", "fatal"]
    warn: ["warn", "warning"]
    info: ["info"]
    debug: ["debug", "trace"]
  highlight_fields: ["msg"]

notify: true  # macOS desktop notifications on build/crash failures
```

By default, `devr` uses `logs.format: auto`: it first tries JSON logs, then falls back to plain text / `key=value` parsing. This works out of the box for logs like `{"level":"info","msg":"ready"}`, `INFO server started`, or `lvl=warning msg="slow query"` if you set `level_field: lvl`.

Without a config file, devr uses sensible defaults – it just works.

## License

MIT
