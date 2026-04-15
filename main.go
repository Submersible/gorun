package main

import (
	"crypto/sha1"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const version = "0.1.2"

func usage() {
	fmt.Fprintf(os.Stderr, `gorun %s — compile-on-change runner for single-file Go programs

Usage:
  gorun <script.go> [args...]
  gorun --clean              Remove all cached binaries
  gorun --list               List cached binaries
  gorun --version            Print version

Cache:
  Binaries are cached at $GORUN_CACHE (default: ~/.cache/gorun).
  Cache key is derived from the absolute path, mtime, and size of the
  source file. When you edit the file, the next run recompiles automatically.

Environment:
  GORUN_CACHE    Override cache directory
  GORUN_FLAGS    Extra flags passed to "go build" (e.g. "-ldflags=-s -w")

`, version)
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	switch args[0] {
	case "--help", "-h":
		usage()
		return
	case "--version", "-v":
		fmt.Println(version)
		return
	case "--clean":
		clean()
		return
	case "--list":
		list()
		return
	}

	src := args[0]
	rest := args[1:]

	if !strings.HasSuffix(src, ".go") {
		fmt.Fprintf(os.Stderr, "gorun: %s is not a .go file\n", src)
		os.Exit(1)
	}

	// Resolve absolute path for stable cache key
	abs, err := filepath.Abs(src)
	if err != nil {
		die("resolving path: %v", err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		die("%v", err)
	}

	cacheDir := cacheDir()
	bin := cachePath(cacheDir, abs, info)

	if _, err := os.Stat(bin); err != nil {
		// Compile
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			die("creating cache dir: %v", err)
		}

		buildArgs := []string{"build", "-o", bin}
		if flags := os.Getenv("GORUN_FLAGS"); flags != "" {
			buildArgs = append(buildArgs, strings.Fields(flags)...)
		}
		buildArgs = append(buildArgs, abs)

		cmd := exec.Command("go", buildArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			// Clean up partial binary
			os.Remove(bin)
			die("build failed: %v", err)
		}
	}

	// Exec replaces this process — no child process overhead
	execErr := syscall.Exec(bin, append([]string{bin}, rest...), os.Environ())
	// If exec fails (shouldn't happen), fall back to cmd
	if execErr != nil {
		cmd := exec.Command(bin, rest...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				os.Exit(exit.ExitCode())
			}
			os.Exit(1)
		}
	}
}

// cachePath builds a deterministic binary path from source path + mtime + size.
// Format: <cache>/<short-hash>-<basename>
// The hash covers abs path + mtime + size so any change triggers recompile.
func cachePath(cacheDir, abs string, info os.FileInfo) string {
	mtime := strconv.FormatInt(info.ModTime().UnixNano(), 36)
	size := strconv.FormatInt(info.Size(), 36)
	key := abs + "\x00" + mtime + "\x00" + size

	h := sha1.Sum([]byte(key))
	short := fmt.Sprintf("%x", h[:6])

	base := strings.TrimSuffix(filepath.Base(abs), ".go")
	return filepath.Join(cacheDir, short+"-"+base)
}

func cacheDir() string {
	if d := os.Getenv("GORUN_CACHE"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "gorun")
}

func clean() {
	dir := cacheDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gorun: cache dir %s: %v\n", dir, err)
		return
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			os.Remove(filepath.Join(dir, e.Name()))
			count++
		}
	}
	fmt.Printf("Removed %d cached binaries from %s\n", count, dir)
}

func list() {
	dir := cacheDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gorun: cache dir %s: %v\n", dir, err)
		return
	}
	if len(entries) == 0 {
		fmt.Printf("No cached binaries in %s\n", dir)
		return
	}
	total := int64(0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		total += size
		fmt.Printf("  %s  %s\n", humanSize(size), e.Name())
	}
	fmt.Printf("\n  %s total in %s\n", humanSize(total), dir)
}

func humanSize(b int64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%5.1fM", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%5.1fK", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%5dB", b)
	}
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gorun: "+format+"\n", args...)
	os.Exit(1)
}
