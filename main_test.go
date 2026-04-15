package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCachePath_Deterministic(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "hello.go")
	os.WriteFile(src, []byte("package main\nfunc main(){}\n"), 0644)

	info, err := os.Stat(src)
	if err != nil {
		t.Fatal(err)
	}

	a := cachePath("/tmp/cache", src, info)
	b := cachePath("/tmp/cache", src, info)
	if a != b {
		t.Fatalf("cachePath not deterministic: %q != %q", a, b)
	}
}

func TestCachePath_ChangesOnDifferentPath(t *testing.T) {
	dir := t.TempDir()

	f1 := filepath.Join(dir, "a.go")
	f2 := filepath.Join(dir, "b.go")
	content := []byte("package main\nfunc main(){}\n")
	os.WriteFile(f1, content, 0644)
	os.WriteFile(f2, content, 0644)

	info1, _ := os.Stat(f1)
	info2, _ := os.Stat(f2)

	p1 := cachePath("/tmp/cache", f1, info1)
	p2 := cachePath("/tmp/cache", f2, info2)
	if p1 == p2 {
		t.Fatal("different source paths should produce different cache paths")
	}
}

func TestCacheDir_Default(t *testing.T) {
	t.Setenv("GORUN_CACHE", "")
	dir := cacheDir()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".cache", "gorun")
	if dir != want {
		t.Fatalf("cacheDir() = %q, want %q", dir, want)
	}
}

func TestCacheDir_EnvOverride(t *testing.T) {
	t.Setenv("GORUN_CACHE", "/custom/cache")
	dir := cacheDir()
	if dir != "/custom/cache" {
		t.Fatalf("cacheDir() = %q, want /custom/cache", dir)
	}
}

func TestHumanSize(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "    0B"},
		{512, "  512B"},
		{1024, "  1.0K"},
		{1536, "  1.5K"},
		{1 << 20, "  1.0M"},
		{3 << 20, "  3.0M"},
	}
	for _, tt := range tests {
		got := humanSize(tt.in)
		if got != tt.want {
			t.Errorf("humanSize(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
