# gorun

Compile-on-change runner for single-file Go programs. Edit your `.go` file, run it — gorun handles the rest.

```
gorun script.go [args...]
```

First run compiles the binary and caches it. Subsequent runs use the cached binary directly (~2ms startup). When you edit the source file, gorun detects the change and recompiles automatically.

## Install

```bash
go install github.com/Submersible/gorun@latest
```

Or build from source:

```bash
git clone https://github.com/Submersible/gorun.git
cd gorun
go build -o gorun .
cp gorun ~/.local/bin/
```

## How it works

gorun computes a cache key from the source file's **absolute path + modification time + size**. If a cached binary exists for that key, it `exec`s it directly (no child process). If not, it runs `go build` first.

```
~/.cache/gorun/
  a1b2c3-statusline      # cached binary for statusline.go
  d4e5f6-myscript         # cached binary for myscript.go
```

Editing the source file changes its mtime, which changes the cache key, which triggers a recompile on the next run. Old binaries accumulate harmlessly — clean them with `gorun --clean`.

## Commands

```
gorun <file.go> [args...]   Run a Go file (compile if needed)
gorun --clean                Remove all cached binaries
gorun --list                 List cached binaries and sizes
gorun --version              Print version
```

## Environment

| Variable | Default | Description |
|---|---|---|
| `GORUN_CACHE` | `~/.cache/gorun` | Cache directory for compiled binaries |
| `GORUN_FLAGS` | *(empty)* | Extra flags for `go build` (e.g. `-ldflags="-s -w"`) |

## Performance

| Scenario | Time |
|---|---|
| Cold (first compile) | ~350ms |
| Warm (cached binary) | ~2ms |
| Source changed (recompile) | ~350ms |

The cached binary is `exec`'d directly — gorun replaces itself with your program, so there's zero wrapper overhead at runtime.

### Why Go instead of bash?

Tested with [hyperfine](https://github.com/sharkdp/hyperfine) — a real-world statusline script rewritten from bash+jq to Go:

| Runner | Mean | vs bash |
|---|---|---|
| Raw compiled binary | 19.1ms | 5.1x faster |
| gorun (cached) | 19.9ms | 4.9x faster |
| bash + jq | 97.4ms | baseline |

gorun adds <1ms overhead over the raw binary — just a `stat()` + cache key check before `exec`. Writing scripts in Go instead of bash gives you type safety, better error handling, and ~5x faster execution.

## Use cases

- **Status lines** — fast-refreshing scripts that run every second
- **Dev tools** — one-off utilities that don't need a full project
- **Prototyping** — iterate on a single file with compiled-language speed
- **CLI scripts** — replace bash/python scripts with type-safe Go

## Requirements

- Go 1.22+ (uses `go build` under the hood)

## License

MIT
