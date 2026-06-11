package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultYAMLFileUsesExplicitPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.yaml")
	if err := os.WriteFile(path, []byte("http:\n  addr: \":8080\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("PP_CONFIG_FILE", path)
	loadedPath, err := LoadDefaultYAMLFile()
	if err != nil {
		t.Fatalf("LoadDefaultYAMLFile() error = %v", err)
	}
	if loadedPath != path {
		t.Fatalf("loadedPath = %q, want %q", loadedPath, path)
	}
}

func TestLoadDefaultYAMLFileReturnsErrorWhenMissing(t *testing.T) {
	t.Setenv("PP_CONFIG_FILE", filepath.Join(t.TempDir(), "missing.yaml"))
	_, err := LoadDefaultYAMLFile()
	if err == nil {
		t.Fatal("LoadDefaultYAMLFile() error = nil, want missing file error")
	}
}
