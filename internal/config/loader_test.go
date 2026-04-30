package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// withTempPath redirects pathFunc to a file inside t.TempDir() and returns the path.
func withTempPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	original := pathFunc
	pathFunc = func() (string, error) { return path, nil }
	t.Cleanup(func() { pathFunc = original })
	return path
}

func TestLoad_MissingFile_ReturnsEmptyConfig(t *testing.T) {
	withTempPath(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if cfg.Contexts == nil {
		t.Fatal("Load(): Contexts must be non-nil even when file is missing")
	}
	if len(cfg.Contexts) != 0 {
		t.Errorf("Load(): want empty Contexts, got %d", len(cfg.Contexts))
	}
	if cfg.Version != CurrentVersion {
		t.Errorf("Load(): Version want %d, got %d", CurrentVersion, cfg.Version)
	}
}

func TestLoad_RoundTrip(t *testing.T) {
	path := withTempPath(t)
	yaml := []byte("version: 1\ncurrent-context: prod\ncontexts:\n  prod:\n    api-url: https://api.ollygarden.cloud\n    api-key: og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n")
	if err := os.WriteFile(path, yaml, 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if cfg.CurrentContext != "prod" {
		t.Errorf("CurrentContext: got %q, want %q", cfg.CurrentContext, "prod")
	}
	prod, ok := cfg.Contexts["prod"]
	if !ok {
		t.Fatal("expected context 'prod'")
	}
	if prod.Name != "prod" {
		t.Errorf("Name: got %q, want %q", prod.Name, "prod")
	}
	if prod.APIURL != "https://api.ollygarden.cloud" {
		t.Errorf("APIURL: got %q", prod.APIURL)
	}
	if prod.APIKey != "og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Errorf("APIKey: got %q", prod.APIKey)
	}
	if cfg.Source != path {
		t.Errorf("Source: got %q, want %q", cfg.Source, path)
	}
}

func TestLoad_MissingVersion_TreatedAsOne(t *testing.T) {
	path := withTempPath(t)
	yaml := []byte("contexts:\n  prod:\n    api-url: https://api.ollygarden.cloud\n    api-key: og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n")
	if err := os.WriteFile(path, yaml, 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("Version: got %d, want 1", cfg.Version)
	}
}

func TestLoad_NewerVersion_ReturnsTypedError(t *testing.T) {
	path := withTempPath(t)
	if err := os.WriteFile(path, []byte("version: 99\ncontexts: {}\n"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_, err := Load()
	if err == nil {
		t.Fatal("Load(): expected error for newer version")
	}
	var ue *UnreadableError
	if !errors.As(err, &ue) {
		t.Fatalf("expected *UnreadableError, got %T", err)
	}
}

func TestLoad_CorruptYAML_ReturnsTypedError(t *testing.T) {
	path := withTempPath(t)
	if err := os.WriteFile(path, []byte("version: [this is not valid"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	_, err := Load()
	if err == nil {
		t.Fatal("Load(): expected error for corrupt YAML")
	}
	var ue *UnreadableError
	if !errors.As(err, &ue) {
		t.Fatalf("expected *UnreadableError, got %T", err)
	}
}

func TestLoad_UnknownFields_Ignored(t *testing.T) {
	path := withTempPath(t)
	yaml := []byte("version: 1\nfuture-field: hello\ncontexts:\n  prod:\n    api-url: https://example.com\n    api-key: og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n    extra: ignored\n")
	if err := os.WriteFile(path, yaml, 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := Load(); err != nil {
		t.Fatalf("Load(): unknown fields should be ignored: %v", err)
	}
}
