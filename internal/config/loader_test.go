package config

import (
	"errors"
	"io/fs"
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

func TestWrite_RoundTrip(t *testing.T) {
	withTempPath(t)
	cfg := New()
	cfg.CurrentContext = "prod"
	cfg.Contexts["prod"] = &Context{
		Name:   "prod",
		APIURL: "https://api.ollygarden.cloud",
		APIKey: "og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}

	if err := Write(cfg); err != nil {
		t.Fatalf("Write(): %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if loaded.CurrentContext != "prod" {
		t.Errorf("CurrentContext: got %q", loaded.CurrentContext)
	}
	if loaded.Contexts["prod"].APIKey != cfg.Contexts["prod"].APIKey {
		t.Errorf("APIKey roundtrip failed")
	}
}

func TestWrite_FilePermissions0600(t *testing.T) {
	if runtimeIsWindows() {
		t.Skip("POSIX file modes do not apply on Windows")
	}
	path := withTempPath(t)
	cfg := New()
	cfg.Contexts["prod"] = &Context{APIURL: "https://x", APIKey: "og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}

	if err := Write(cfg); err != nil {
		t.Fatalf("Write(): %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != FilePermissions {
		t.Errorf("file perms: got %o, want %o", info.Mode().Perm(), FilePermissions)
	}
}

func TestWrite_DirPermissions0700(t *testing.T) {
	if runtimeIsWindows() {
		t.Skip("POSIX file modes do not apply on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.yaml")
	original := pathFunc
	pathFunc = func() (string, error) { return path, nil }
	t.Cleanup(func() { pathFunc = original })

	cfg := New()
	cfg.Contexts["prod"] = &Context{APIURL: "https://x", APIKey: "og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}

	if err := Write(cfg); err != nil {
		t.Fatalf("Write(): %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, "nested"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != DirPermissions {
		t.Errorf("dir perms: got %o, want %o", info.Mode().Perm(), DirPermissions)
	}
}

func TestWrite_EmptyContexts_DeletesFile(t *testing.T) {
	path := withTempPath(t)
	if err := os.WriteFile(path, []byte("version: 1\ncontexts:\n  prod: {api-url: x, api-key: y}\n"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	cfg := New()
	if err := Write(cfg); err != nil {
		t.Fatalf("Write(): %v", err)
	}

	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected file to be deleted, but it exists (err: %v)", err)
	}
}

func TestWrite_EmptyContexts_NoExistingFile_NoError(t *testing.T) {
	withTempPath(t)
	cfg := New()
	if err := Write(cfg); err != nil {
		t.Fatalf("Write(): want nil for already-absent file, got %v", err)
	}
}

func TestWrite_AtomicLeftoverCleanup(t *testing.T) {
	path := withTempPath(t)
	cfg := New()
	cfg.Contexts["prod"] = &Context{APIURL: "https://x", APIKey: "og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	if err := Write(cfg); err != nil {
		t.Fatalf("Write(): %v", err)
	}
	// Verify the .tmp sibling does not linger after a successful write.
	if _, err := os.Stat(path + ".tmp"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected no .tmp file after successful write")
	}
}

func TestWrite_PersistsCurrentVersion(t *testing.T) {
	path := withTempPath(t)
	cfg := Config{Contexts: map[string]*Context{
		"prod": {APIURL: "https://x", APIKey: "og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}}
	if err := Write(cfg); err != nil {
		t.Fatalf("Write(): %v", err)
	}
	data, _ := os.ReadFile(path)
	if !bytesContainLine(data, "version: 1") {
		t.Errorf("expected file to contain `version: 1` line, got:\n%s", data)
	}
}

func bytesContainLine(b []byte, line string) bool {
	for _, l := range bytesSplitLines(b) {
		if l == line {
			return true
		}
	}
	return false
}

func bytesSplitLines(b []byte) []string {
	var out []string
	start := 0
	for i, c := range b {
		if c == '\n' {
			out = append(out, string(b[start:i]))
			start = i + 1
		}
	}
	if start < len(b) {
		out = append(out, string(b[start:]))
	}
	return out
}

// runtimeIsWindows reports whether the current GOOS is windows. Defined as a
// helper so the skip lines stay readable.
func runtimeIsWindows() bool {
	return windowsGOOS
}
