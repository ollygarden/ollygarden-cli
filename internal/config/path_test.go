package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPath_EnvVarOverride(t *testing.T) {
	t.Setenv(ConfigFileEnvVar, "/explicit/path/foo.yaml")
	got, err := Path()
	if err != nil {
		t.Fatalf("Path(): %v", err)
	}
	if got != "/explicit/path/foo.yaml" {
		t.Errorf("Path(): got %q, want %q", got, "/explicit/path/foo.yaml")
	}
}

func TestPath_DefaultUsesUserConfigDir(t *testing.T) {
	t.Setenv(ConfigFileEnvVar, "")

	dir, err := os.UserConfigDir()
	if err != nil {
		t.Skipf("os.UserConfigDir unavailable on this host: %v", err)
	}
	want := filepath.Join(dir, ConfigFolder, ConfigFileName)

	got, err := Path()
	if err != nil {
		t.Fatalf("Path(): %v", err)
	}
	if got != want {
		t.Errorf("Path(): got %q, want %q", got, want)
	}
}

func TestPath_TestSeam(t *testing.T) {
	original := pathFunc
	t.Cleanup(func() { pathFunc = original })

	pathFunc = func() (string, error) {
		return "/swapped/by/test.yaml", nil
	}
	got, err := Path()
	if err != nil {
		t.Fatalf("Path(): %v", err)
	}
	if got != "/swapped/by/test.yaml" {
		t.Errorf("Path(): got %q, want %q", got, "/swapped/by/test.yaml")
	}
}
