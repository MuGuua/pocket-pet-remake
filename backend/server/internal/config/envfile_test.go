package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFileIfPresent(t *testing.T) {
	t.Setenv("PP_EXISTING", "keep-me")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.env")
	content := "# comment\nPP_FOO=bar\nexport PP_BAR=baz\nPP_EXISTING=should-not-override\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded, err := LoadEnvFileIfPresent(path)
	if err != nil {
		t.Fatalf("LoadEnvFileIfPresent() error = %v", err)
	}
	if !loaded {
		t.Fatal("LoadEnvFileIfPresent() loaded = false, want true")
	}
	if got := os.Getenv("PP_FOO"); got != "bar" {
		t.Fatalf("PP_FOO = %q, want %q", got, "bar")
	}
	if got := os.Getenv("PP_BAR"); got != "baz" {
		t.Fatalf("PP_BAR = %q, want %q", got, "baz")
	}
	if got := os.Getenv("PP_EXISTING"); got != "keep-me" {
		t.Fatalf("PP_EXISTING = %q, want %q", got, "keep-me")
	}
}

func TestLoadEnvFileIfPresentMissingFile(t *testing.T) {
	loaded, err := LoadEnvFileIfPresent(filepath.Join(t.TempDir(), "missing.env"))
	if err != nil {
		t.Fatalf("LoadEnvFileIfPresent() error = %v", err)
	}
	if loaded {
		t.Fatal("LoadEnvFileIfPresent() loaded = true, want false")
	}
}
