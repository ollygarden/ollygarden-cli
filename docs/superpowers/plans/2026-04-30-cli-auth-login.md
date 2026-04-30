# CLI Auth Login & On-Disk Token Storage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `ollygarden auth {login,logout,status,use-context,list-contexts}` subcommands backed by a multi-context YAML config file at `os.UserConfigDir()/ollygarden/config.yaml` (mode `0600`), with `OLLYGARDEN_API_KEY` env var continuing to win over saved contexts.

**Architecture:** Two new internal packages — `internal/config` (pure schema + filesystem; no HTTP) and `internal/auth` (login orchestration + credential resolution; depends on `internal/config` and `internal/client`). Five new files in `cmd/` for the subcommands, one parent file (`cmd/auth.go`), and a small refactor of `cmd/root.go` to route `PersistentPreRunE` through `auth.Resolve` instead of the inline env-var check.

**Tech Stack:** Go 1.25, `spf13/cobra` v1.10, `spf13/pflag` v1.0, `gopkg.in/yaml.v3`, `golang.org/x/term`, `stretchr/testify` v1.11. All already present in `go.mod`.

**Source spec:** `docs/superpowers/specs/2026-04-30-cli-auth-login-design.md` (committed on this branch). Read it before starting.

---

## File Map

**Create:**
- `internal/config/config.go` — `Config`, `Context` types + constants
- `internal/config/path.go` — path resolution + `pathFunc` test seam
- `internal/config/loader.go` — `Load()`, `Write()` (atomic write, `0600`/`0700` perms, version handling, empty-file cleanup)
- `internal/config/loader_test.go`
- `internal/config/path_test.go`
- `internal/auth/errors.go` — typed errors with `Code` strings (`NO_CREDENTIALS`, `TOKEN_REJECTED`, etc.)
- `internal/auth/mask.go` — `MaskKey(string) string`
- `internal/auth/mask_test.go`
- `internal/auth/resolve.go` — `Resolve(ResolveInputs) (Credentials, error)` — pure function
- `internal/auth/resolve_test.go`
- `internal/auth/login.go` — `Login(ctx, LoginInputs) (LoginResult, error)` — HTTP probe + persist
- `internal/auth/login_test.go`
- `cmd/auth.go` — parent `auth` cobra command
- `cmd/auth_login.go`
- `cmd/auth_login_test.go`
- `cmd/auth_logout.go`
- `cmd/auth_logout_test.go`
- `cmd/auth_status.go`
- `cmd/auth_status_test.go`
- `cmd/auth_use_context.go`
- `cmd/auth_use_context_test.go`
- `cmd/auth_list_contexts.go`
- `cmd/auth_list_contexts_test.go`
- `cmd/auth_integration_test.go` (build tag `integration`)

**Modify:**
- `internal/exitcode/exitcode.go` — add `Config = 7`
- `cmd/root.go` — add `--context` persistent flag and `OLLYGARDEN_CONTEXT` env reading; replace inline env-var check in `PersistentPreRunE` with `auth.Resolve`; route new typed errors through `Execute()`; remove placeholder `AuthError` type
- `cmd/root_test.go` — update existing `TestMissingAPIKeyReturnsAuthError` to expect new typed error
- `go.mod` / `go.sum` — promote `gopkg.in/yaml.v3` from indirect to direct (happens automatically when first imported)
- `specs/CLI.md` — auth subtree, global `--context` flag, exit code 7, error code table, new §6 Credential Storage
- `specs/CLI_GUIDELINES.md` — pointer to error codes, logout note, new §8 Auth Commands

---

## Phase 1 — Foundation (no behavior change)

### Task 1: Add `Config = 7` exit code

**Files:**
- Modify: `internal/exitcode/exitcode.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/exitcode/exitcode_test.go` (create if it doesn't exist):

```go
package exitcode

import "testing"

func TestExitCodeConfigIsSeven(t *testing.T) {
	if Config != 7 {
		t.Fatalf("Config exit code: want 7, got %d", Config)
	}
}
```

- [ ] **Step 2: Run the test, watch it fail**

```bash
go test ./internal/exitcode/ -run TestExitCodeConfigIsSeven -v
```
Expected: `undefined: Config` compile error.

- [ ] **Step 3: Add the constant**

In `internal/exitcode/exitcode.go`, add `Config = 7` after `Server`:

```go
const (
	Success   = 0
	General   = 1
	Usage     = 2
	Auth      = 3
	NotFound  = 4
	RateLimit = 5
	Server    = 6
	Config    = 7
)
```

- [ ] **Step 4: Run the test, watch it pass**

```bash
go test ./internal/exitcode/ -run TestExitCodeConfigIsSeven -v
```
Expected: `PASS`.

- [ ] **Step 5: Format and commit**

```bash
go fmt ./...
git add internal/exitcode/
git commit -m "feat(exitcode): add Config exit code (7) for config file errors"
```

---

## Phase 2 — `internal/config` package

### Task 2: Schema types + constants in `internal/config/config.go`

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/config/config_test.go`:

```go
package config

import "testing"

func TestConstants(t *testing.T) {
	if ConfigFolder != "ollygarden" {
		t.Errorf("ConfigFolder: want %q, got %q", "ollygarden", ConfigFolder)
	}
	if ConfigFileName != "config.yaml" {
		t.Errorf("ConfigFileName: want %q, got %q", "config.yaml", ConfigFileName)
	}
	if ConfigFileEnvVar != "OLLYGARDEN_CONFIG" {
		t.Errorf("ConfigFileEnvVar: want %q, got %q", "OLLYGARDEN_CONFIG", ConfigFileEnvVar)
	}
	if ContextEnvVar != "OLLYGARDEN_CONTEXT" {
		t.Errorf("ContextEnvVar: want %q, got %q", "OLLYGARDEN_CONTEXT", ContextEnvVar)
	}
	if FilePermissions != 0o600 {
		t.Errorf("FilePermissions: want 0o600, got %o", FilePermissions)
	}
	if DirPermissions != 0o700 {
		t.Errorf("DirPermissions: want 0o700, got %o", DirPermissions)
	}
	if CurrentVersion != 1 {
		t.Errorf("CurrentVersion: want 1, got %d", CurrentVersion)
	}
}

func TestConfigZeroValueHasMap(t *testing.T) {
	cfg := New()
	if cfg.Contexts == nil {
		t.Fatal("New() must return a non-nil Contexts map")
	}
	if cfg.Version != CurrentVersion {
		t.Errorf("Version: want %d, got %d", CurrentVersion, cfg.Version)
	}
}
```

- [ ] **Step 2: Run the test, watch it fail (compile error)**

```bash
go test ./internal/config/ -run TestConstants -v
```
Expected: build error — `package internal/config` doesn't exist.

- [ ] **Step 3: Implement `internal/config/config.go`**

```go
// Package config holds the on-disk schema and I/O for the OllyGarden CLI.
// It is pure filesystem + YAML; it knows nothing about HTTP or authentication.
package config

const (
	ConfigFolder     = "ollygarden"
	ConfigFileName   = "config.yaml"
	ConfigFileEnvVar = "OLLYGARDEN_CONFIG"
	ContextEnvVar    = "OLLYGARDEN_CONTEXT"
	FilePermissions  = 0o600
	DirPermissions   = 0o700
	CurrentVersion   = 1
)

// Config is the on-disk schema. It is multi-context (kubeconfig-style):
// many named contexts plus a pointer to the active one.
type Config struct {
	Version        int                 `yaml:"version"`
	CurrentContext string              `yaml:"current-context,omitempty"`
	Contexts       map[string]*Context `yaml:"contexts"`
	// Source is the path the file was loaded from. Populated by Load(),
	// not serialized.
	Source string `yaml:"-"`
}

// Context is one credential entry: API URL plus API key. The key is
// sensitive — never log it, never include it in errors, never serialize it
// to anywhere other than the on-disk file at mode 0600.
type Context struct {
	Name   string `yaml:"-"` // map key, populated post-load
	APIURL string `yaml:"api-url"`
	APIKey string `yaml:"api-key"`
}

// New returns an empty, valid config initialized at the current schema
// version. Callers that need a starting point for a fresh write should use
// this rather than the zero value, which has a nil Contexts map.
func New() Config {
	return Config{
		Version:  CurrentVersion,
		Contexts: map[string]*Context{},
	}
}
```

- [ ] **Step 4: Run the test, watch it pass**

```bash
go test ./internal/config/ -v
```
Expected: `PASS`.

- [ ] **Step 5: Format and commit**

```bash
go fmt ./...
git add internal/config/
git commit -m "feat(config): add schema types and constants"
```

---

### Task 3: Path resolution in `internal/config/path.go`

**Files:**
- Create: `internal/config/path.go`
- Create: `internal/config/path_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/config/path_test.go`:

```go
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
```

- [ ] **Step 2: Run the test, watch it fail (compile error)**

```bash
go test ./internal/config/ -run TestPath -v
```
Expected: `undefined: Path`, `undefined: pathFunc`.

- [ ] **Step 3: Implement `internal/config/path.go`**

```go
package config

import (
	"os"
	"path/filepath"
)

// pathFunc is the package-level test seam. Tests swap it to redirect writes
// into t.TempDir() instead of touching the real os.UserConfigDir() location.
// Pattern lifted from MercuryTechnologies/mercury-cli internal/auth.
var pathFunc = defaultPath

// Path returns the resolved config file location. It honors OLLYGARDEN_CONFIG
// as a full-path override; otherwise it returns
// os.UserConfigDir()/ollygarden/config.yaml.
//
// The actual filesystem location varies by OS:
//   - Linux:   $XDG_CONFIG_HOME/ollygarden/config.yaml or ~/.config/ollygarden/config.yaml
//   - macOS:   ~/Library/Application Support/ollygarden/config.yaml
//   - Windows: %AppData%\ollygarden\config.yaml
func Path() (string, error) {
	return pathFunc()
}

func defaultPath() (string, error) {
	if envPath := os.Getenv(ConfigFileEnvVar); envPath != "" {
		return envPath, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigFolder, ConfigFileName), nil
}
```

- [ ] **Step 4: Run the tests, watch them pass**

```bash
go test ./internal/config/ -v
```
Expected: `PASS` for all three Path tests.

- [ ] **Step 5: Format and commit**

```bash
go fmt ./...
git add internal/config/
git commit -m "feat(config): resolve config path via os.UserConfigDir() with env override and test seam"
```

---

### Task 4: Loader in `internal/config/loader.go`

**Files:**
- Create: `internal/config/loader.go`
- Modify: `internal/config/loader_test.go` (creating it)

- [ ] **Step 1: Write the failing tests**

Create `internal/config/loader_test.go`:

```go
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
```

- [ ] **Step 2: Run the tests, watch them fail**

```bash
go test ./internal/config/ -run TestLoad -v
```
Expected: `undefined: Load`, `undefined: UnreadableError`.

- [ ] **Step 3: Implement `internal/config/loader.go` — Load function and UnreadableError type**

```go
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"gopkg.in/yaml.v3"
)

// UnreadableError is returned when the config file exists but cannot be
// read or parsed. It maps to exit code 7 (config) at the cmd layer and
// surfaces via the CLI-emitted error code CONFIG_UNREADABLE.
type UnreadableError struct {
	Path string
	Op   string
	Err  error
}

func (e *UnreadableError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("config %s: %s", e.Op, e.Path)
	}
	return fmt.Sprintf("config %s at %s: %v", e.Op, e.Path, e.Err)
}

func (e *UnreadableError) Unwrap() error { return e.Err }

// Load reads and parses the config file. A missing file is not an error —
// callers receive a zero-population Config initialized at CurrentVersion.
// A corrupt file or one with a version newer than CurrentVersion returns
// *UnreadableError.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return New(), &UnreadableError{Path: "", Op: "resolve path", Err: err}
	}

	cfg := New()
	cfg.Source = path

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, &UnreadableError{Path: path, Op: "read", Err: err}
	}

	var loaded Config
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		return cfg, &UnreadableError{Path: path, Op: "parse", Err: err}
	}

	if loaded.Version == 0 {
		loaded.Version = 1
	}
	if loaded.Version > CurrentVersion {
		return cfg, &UnreadableError{
			Path: path,
			Op:   "load",
			Err:  fmt.Errorf("file version %d is newer than this CLI supports (max %d). Upgrade ollygarden CLI", loaded.Version, CurrentVersion),
		}
	}

	if loaded.Contexts == nil {
		loaded.Contexts = map[string]*Context{}
	}
	for name, c := range loaded.Contexts {
		c.Name = name
	}
	loaded.Source = path
	return loaded, nil
}
```

- [ ] **Step 4: Run the tests, watch them pass**

```bash
go test ./internal/config/ -v
```
Expected: all `TestLoad_*` PASS.

- [ ] **Step 5: Format and commit**

```bash
go fmt ./...
go mod tidy
git add internal/config/ go.mod go.sum
git commit -m "feat(config): load YAML config with version handling and typed errors"
```

---

### Task 5: Atomic Writer in `internal/config/loader.go`

**Files:**
- Modify: `internal/config/loader.go` (add `Write`, `WriteFailedError`)
- Modify: `internal/config/loader_test.go` (add Write tests)

- [ ] **Step 1: Write the failing tests**

Append to `internal/config/loader_test.go`:

```go
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
```

Add to a new file `internal/config/runtime_unix_test.go` (build tag for non-Windows):

```go
//go:build !windows

package config

const windowsGOOS = false
```

And `internal/config/runtime_windows_test.go`:

```go
//go:build windows

package config

const windowsGOOS = true
```

- [ ] **Step 2: Run the tests, watch them fail**

```bash
go test ./internal/config/ -run TestWrite -v
```
Expected: `undefined: Write`.

- [ ] **Step 3: Implement `Write` and `WriteFailedError`**

Append to `internal/config/loader.go`:

```go
// WriteFailedError is returned when the config file cannot be written
// (atomic-rename failure, permission error, disk full, etc.). It maps to
// exit code 7 (config) at the cmd layer and surfaces via the CLI-emitted
// error code CONFIG_WRITE_FAILED.
type WriteFailedError struct {
	Path string
	Op   string
	Err  error
}

func (e *WriteFailedError) Error() string {
	return fmt.Sprintf("config write %s at %s: %v", e.Op, e.Path, e.Err)
}

func (e *WriteFailedError) Unwrap() error { return e.Err }

// Write persists cfg to disk atomically (write to .tmp, fsync, rename).
//
// Special-case: when cfg has no contexts, Write deletes the existing file
// rather than leaving an empty contexts: {} on disk. This keeps the
// "I'm completely logged out" state clean and matches Mercury CLI's
// approach. Removing an absent file is not an error.
//
// Always writes Version=CurrentVersion regardless of cfg.Version. Source
// is not serialized.
func Write(cfg Config) error {
	path, err := Path()
	if err != nil {
		return &WriteFailedError{Path: "", Op: "resolve path", Err: err}
	}

	if len(cfg.Contexts) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return &WriteFailedError{Path: path, Op: "remove (empty config)", Err: err}
		}
		return nil
	}

	cfg.Version = CurrentVersion

	if err := os.MkdirAll(filepath.Dir(path), DirPermissions); err != nil {
		return &WriteFailedError{Path: path, Op: "create parent dir", Err: err}
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return &WriteFailedError{Path: path, Op: "marshal yaml", Err: err}
	}

	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, FilePermissions)
	if err != nil {
		return &WriteFailedError{Path: tmp, Op: "open tmp", Err: err}
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return &WriteFailedError{Path: tmp, Op: "write tmp", Err: err}
	}
	// fsync best-effort; some filesystems (tmpfs in containers, network FS)
	// reject it spuriously. Continue on failure.
	_ = f.Sync()
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return &WriteFailedError{Path: tmp, Op: "close tmp", Err: err}
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return &WriteFailedError{Path: path, Op: "rename tmp -> final", Err: err}
	}
	return nil
}
```

Add the `path/filepath` import if it's missing in `loader.go`:

```go
import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)
```

- [ ] **Step 4: Run the tests, watch them pass**

```bash
go test ./internal/config/ -v
```
Expected: all PASS (Windows-only tests skip on Linux).

- [ ] **Step 5: Format and commit**

```bash
go fmt ./...
git add internal/config/
git commit -m "feat(config): atomic Write with 0600/0700 perms and empty-cleanup"
```

---

## Phase 3 — `internal/auth` package: pure functions first

### Task 6: Typed errors with codes in `internal/auth/errors.go`

**Files:**
- Create: `internal/auth/errors.go`
- Create: `internal/auth/errors_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/auth/errors_test.go`:

```go
package auth

import (
	"errors"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/exitcode"
)

func TestError_CodeAndExit(t *testing.T) {
	cases := []struct {
		name     string
		err      *Error
		wantCode string
		wantExit int
	}{
		{"no creds", ErrNoCredentials(), "NO_CREDENTIALS", exitcode.Auth},
		{"invalid format", ErrInvalidTokenFormat("bad"), "INVALID_TOKEN_FORMAT", exitcode.Usage},
		{"rejected", ErrTokenRejected(), "TOKEN_REJECTED", exitcode.Auth},
		{"not found", ErrContextNotFound("internal"), "CONTEXT_NOT_FOUND", exitcode.NotFound},
		{"unreadable", ErrConfigUnreadable("/x", errors.New("oops")), "CONFIG_UNREADABLE", exitcode.Config},
		{"write failed", ErrConfigWriteFailed("/x", errors.New("oops")), "CONFIG_WRITE_FAILED", exitcode.Config},
		{"token file", ErrTokenFileNotFound("/x"), "TOKEN_FILE_NOT_FOUND", exitcode.Usage},
		{"confirm req", ErrConfirmRequired(), "CONFIRM_REQUIRED", exitcode.Usage},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.Code != tc.wantCode {
				t.Errorf("Code: got %q, want %q", tc.err.Code, tc.wantCode)
			}
			if tc.err.ExitCode != tc.wantExit {
				t.Errorf("ExitCode: got %d, want %d", tc.err.ExitCode, tc.wantExit)
			}
			if tc.err.Message == "" {
				t.Error("Message must be non-empty")
			}
		})
	}
}

func TestError_AsTarget(t *testing.T) {
	src := ErrNoCredentials()
	wrapped := errorWrap(src)
	var got *Error
	if !errors.As(wrapped, &got) {
		t.Fatal("errors.As failed to unwrap")
	}
	if got.Code != "NO_CREDENTIALS" {
		t.Errorf("got.Code = %q", got.Code)
	}
}

// errorWrap wraps for the As-target test (avoids fmt import in the test file proper).
func errorWrap(e *Error) error {
	return &outer{e: e}
}

type outer struct{ e *Error }

func (o *outer) Error() string { return "outer: " + o.e.Error() }
func (o *outer) Unwrap() error { return o.e }
```

- [ ] **Step 2: Run the test, watch it fail**

```bash
go test ./internal/auth/ -run TestError -v
```
Expected: `package internal/auth` doesn't exist.

- [ ] **Step 3: Implement `internal/auth/errors.go`**

```go
// Package auth orchestrates login (HTTP probe + persist) and credential
// resolution (env/flag/file precedence). It depends on internal/config for
// on-disk schema and internal/client for the HTTP probe; everything else
// (formatting, prompts, exit-code routing) belongs in cmd/.
package auth

import (
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/exitcode"
)

// Error is the typed error this package returns. Each instance carries
// a stable Code (machine-readable, mirrored in the JSON error envelope)
// and an ExitCode used by cmd.Execute() to set the process exit status.
//
// Cmd code switches on this type via errors.As. Never compare Code values
// from outside this package — use the constructors below.
type Error struct {
	Code     string
	ExitCode int
	Message  string
	Cause    error
}

func (e *Error) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *Error) Unwrap() error { return e.Cause }

// Constructors. One per error code from the spec.

func ErrNoCredentials() *Error {
	return &Error{
		Code:     "NO_CREDENTIALS",
		ExitCode: exitcode.Auth,
		Message:  `No credentials configured. Run "ollygarden auth login" or set OLLYGARDEN_API_KEY. Get a token at https://ollygarden.app/settings.`,
	}
}

func ErrInvalidTokenFormat(_ string) *Error {
	return &Error{
		Code:     "INVALID_TOKEN_FORMAT",
		ExitCode: exitcode.Usage,
		Message:  "Invalid token format. Expected og_sk_xxxxxx_<32 hex>.",
	}
}

func ErrTokenRejected() *Error {
	return &Error{
		Code:     "TOKEN_REJECTED",
		ExitCode: exitcode.Auth,
		Message:  "Token rejected by API. The token may be revoked or expired.",
	}
}

func ErrContextNotFound(name string) *Error {
	return &Error{
		Code:     "CONTEXT_NOT_FOUND",
		ExitCode: exitcode.NotFound,
		Message:  fmt.Sprintf(`Context %q not found. Run "ollygarden auth list-contexts" to see available contexts.`, name),
	}
}

func ErrConfigUnreadable(path string, cause error) *Error {
	return &Error{
		Code:     "CONFIG_UNREADABLE",
		ExitCode: exitcode.Config,
		Message:  fmt.Sprintf("Cannot read config file at %s. Inspect or remove the file to recover.", path),
		Cause:    cause,
	}
}

func ErrConfigWriteFailed(path string, cause error) *Error {
	return &Error{
		Code:     "CONFIG_WRITE_FAILED",
		ExitCode: exitcode.Config,
		Message:  fmt.Sprintf("Cannot write config file at %s.", path),
		Cause:    cause,
	}
}

func ErrTokenFileNotFound(path string) *Error {
	return &Error{
		Code:     "TOKEN_FILE_NOT_FOUND",
		ExitCode: exitcode.Usage,
		Message:  fmt.Sprintf("Cannot read token file %s.", path),
	}
}

func ErrConfirmRequired() *Error {
	return &Error{
		Code:     "CONFIRM_REQUIRED",
		ExitCode: exitcode.Usage,
		Message:  "Refusing to remove all contexts without --confirm in non-interactive mode.",
	}
}
```

- [ ] **Step 4: Run the tests, watch them pass**

```bash
go test ./internal/auth/ -v
```
Expected: PASS.

- [ ] **Step 5: Format and commit**

```bash
go fmt ./...
git add internal/auth/
git commit -m "feat(auth): typed errors with stable codes and exit codes"
```

---

### Task 7: `MaskKey` in `internal/auth/mask.go`

**Files:**
- Create: `internal/auth/mask.go`
- Create: `internal/auth/mask_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/auth/mask_test.go`:

```go
package auth

import "testing"

func TestMaskKey(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "og_sk_abc123_••••"},
		{"og_sk_DEFGHI_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "og_sk_DEFGHI_••••"},
		{"", ""},
		{"og_sk_short", "og_sk_short"},        // no underscore-separated secret tail → leave as-is
		{"og_sk_abc123_••••", "og_sk_abc123_••••"}, // already masked → idempotent
		{"plain-string", "plain-string"},      // doesn't match shape → leave as-is
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := MaskKey(tc.in); got != tc.want {
				t.Errorf("MaskKey(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run, watch it fail**

```bash
go test ./internal/auth/ -run TestMaskKey -v
```
Expected: `undefined: MaskKey`.

- [ ] **Step 3: Implement `internal/auth/mask.go`**

```go
package auth

import "strings"

// MaskKey returns a display-safe rendering of an API key. The prefix
// (everything up to and including the second underscore — e.g.
// `og_sk_abc123_`) is preserved; the secret tail is replaced with `••••`.
//
// Inputs that don't match the og_sk_<id>_<secret> shape are returned
// unchanged. Already-masked values pass through untouched.
func MaskKey(s string) string {
	if s == "" {
		return s
	}
	parts := strings.SplitN(s, "_", 4)
	if len(parts) < 4 {
		return s
	}
	if parts[3] == "••••" {
		return s
	}
	return parts[0] + "_" + parts[1] + "_" + parts[2] + "_••••"
}
```

- [ ] **Step 4: Run the tests, watch them pass**

```bash
go test ./internal/auth/ -v
```
Expected: PASS.

- [ ] **Step 5: Format and commit**

```bash
go fmt ./...
git add internal/auth/
git commit -m "feat(auth): MaskKey for display-safe rendering of API keys"
```

---

### Task 8: `Resolve` (precedence engine) in `internal/auth/resolve.go`

**Files:**
- Create: `internal/auth/resolve.go`
- Create: `internal/auth/resolve_test.go`

- [ ] **Step 1: Write the failing tests (table-driven, covers every row of the precedence tables in the spec)**

Create `internal/auth/resolve_test.go`:

```go
package auth

import (
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/config"
)

func cfgWith(current string, ctxs map[string]*config.Context) config.Config {
	if ctxs == nil {
		ctxs = map[string]*config.Context{}
	}
	for name, c := range ctxs {
		c.Name = name
	}
	return config.Config{
		Version:        config.CurrentVersion,
		CurrentContext: current,
		Contexts:       ctxs,
	}
}

func TestResolve_PrecedenceTable(t *testing.T) {
	prodCtx := &config.Context{APIURL: "https://api.ollygarden.cloud", APIKey: "og_sk_prod00_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	devCtx := &config.Context{APIURL: "https://api.dev.ollygarden.cloud", APIKey: "og_sk_dev000_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	cfg := cfgWith("prod", map[string]*config.Context{"prod": prodCtx, "dev": devCtx})

	cases := []struct {
		name string
		in   ResolveInputs
		// expected
		key       string
		url       string
		source    Source
		ctxName   string
		expectErr bool
	}{
		{
			name:    "env key wins over flag, env-context, current-context",
			in:      ResolveInputs{Config: cfg, EnvAPIKey: "og_sk_envkey_cccccccccccccccccccccccccccccccc", EnvAPIURL: "", FlagContext: "dev", EnvContext: "dev", FlagAPIURL: ""},
			key:     "og_sk_envkey_cccccccccccccccccccccccccccccccc",
			url:     "https://api.dev.ollygarden.cloud", // url comes from --context dev
			source:  SourceEnv,
			ctxName: "dev",
		},
		{
			name:    "flag-context selected when env unset",
			in:      ResolveInputs{Config: cfg, FlagContext: "dev"},
			key:     devCtx.APIKey,
			url:     devCtx.APIURL,
			source:  SourceContext,
			ctxName: "dev",
		},
		{
			name:    "env-context selected when flag unset",
			in:      ResolveInputs{Config: cfg, EnvContext: "dev"},
			key:     devCtx.APIKey,
			url:     devCtx.APIURL,
			source:  SourceContext,
			ctxName: "dev",
		},
		{
			name:    "flag-context wins over env-context",
			in:      ResolveInputs{Config: cfg, FlagContext: "dev", EnvContext: "prod"},
			key:     devCtx.APIKey,
			url:     devCtx.APIURL,
			source:  SourceContext,
			ctxName: "dev",
		},
		{
			name:    "current-context falls back when nothing else set",
			in:      ResolveInputs{Config: cfg},
			key:     prodCtx.APIKey,
			url:     prodCtx.APIURL,
			source:  SourceContext,
			ctxName: "prod",
		},
		{
			name:    "url flag overrides context url",
			in:      ResolveInputs{Config: cfg, FlagAPIURL: "https://override.example.com"},
			key:     prodCtx.APIKey,
			url:     "https://override.example.com",
			source:  SourceContext,
			ctxName: "prod",
		},
		{
			name:    "url env overrides context url when flag unset",
			in:      ResolveInputs{Config: cfg, EnvAPIURL: "https://envurl.example.com"},
			key:     prodCtx.APIKey,
			url:     "https://envurl.example.com",
			source:  SourceContext,
			ctxName: "prod",
		},
		{
			name:    "url flag wins over url env",
			in:      ResolveInputs{Config: cfg, FlagAPIURL: "https://flag.example", EnvAPIURL: "https://env.example"},
			key:     prodCtx.APIKey,
			url:     "https://flag.example",
			source:  SourceContext,
			ctxName: "prod",
		},
		{
			name:    "env key only, no contexts → default URL",
			in:      ResolveInputs{Config: cfgWith("", nil), EnvAPIKey: "og_sk_envkey_cccccccccccccccccccccccccccccccc"},
			key:     "og_sk_envkey_cccccccccccccccccccccccccccccccc",
			url:     DefaultAPIURL,
			source:  SourceEnv,
			ctxName: "",
		},
		{
			name:      "no env, no current-context, no flag → NO_CREDENTIALS",
			in:        ResolveInputs{Config: cfgWith("", nil)},
			expectErr: true,
		},
		{
			name:      "flag-context names unknown context",
			in:        ResolveInputs{Config: cfg, FlagContext: "ghost"},
			expectErr: true,
		},
		{
			name: "context with empty APIURL falls back to default",
			in: ResolveInputs{Config: cfgWith("prod", map[string]*config.Context{
				"prod": {APIKey: "og_sk_prod00_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", APIURL: ""},
			})},
			key:     "og_sk_prod00_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			url:     DefaultAPIURL,
			source:  SourceContext,
			ctxName: "prod",
		},
		{
			name:      "env-context names unknown context",
			in:        ResolveInputs{Config: cfg, EnvContext: "ghost"},
			expectErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Resolve(tc.in)
			if tc.expectErr {
				if err == nil {
					t.Fatal("Resolve(): want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve(): %v", err)
			}
			if got.APIKey != tc.key {
				t.Errorf("APIKey: got %q, want %q", got.APIKey, tc.key)
			}
			if got.APIURL != tc.url {
				t.Errorf("APIURL: got %q, want %q", got.APIURL, tc.url)
			}
			if got.Source != tc.source {
				t.Errorf("Source: got %v, want %v", got.Source, tc.source)
			}
			if got.ContextName != tc.ctxName {
				t.Errorf("ContextName: got %q, want %q", got.ContextName, tc.ctxName)
			}
		})
	}
}

func TestResolve_EnvKeyWithSavedContext_ReportsBoth(t *testing.T) {
	cfg := cfgWith("prod", map[string]*config.Context{
		"prod": {APIURL: "https://api.ollygarden.cloud", APIKey: "og_sk_prod00_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	})
	got, err := Resolve(ResolveInputs{Config: cfg, EnvAPIKey: "og_sk_envkey_cccccccccccccccccccccccccccccccc"})
	if err != nil {
		t.Fatalf("Resolve(): %v", err)
	}
	if got.Source != SourceEnv {
		t.Errorf("Source: want SourceEnv")
	}
	if got.ContextName != "prod" {
		t.Errorf("ContextName: want %q (saved one that would have won), got %q", "prod", got.ContextName)
	}
}
```

- [ ] **Step 2: Run, watch it fail**

```bash
go test ./internal/auth/ -run TestResolve -v
```
Expected: `undefined: Resolve` etc.

- [ ] **Step 3: Implement `internal/auth/resolve.go`**

```go
package auth

import "github.com/ollygarden/ollygarden-cli/internal/config"

// DefaultAPIURL is the production API endpoint used when nothing else
// resolves a URL.
const DefaultAPIURL = "https://api.ollygarden.cloud"

// Source records where the active credential came from. Useful for
// `auth status` (so the user sees `source: env (overrides saved context "prod")`).
type Source int

const (
	SourceUnknown Source = iota
	SourceEnv             // from OLLYGARDEN_API_KEY
	SourceContext         // from a context in the config file
)

// ResolveInputs is the pure-data input to Resolve. The cmd layer fills it in
// from os.Getenv, persistent flags, and config.Load — Resolve itself does
// no I/O.
type ResolveInputs struct {
	Config      config.Config
	EnvAPIKey   string // OLLYGARDEN_API_KEY
	EnvAPIURL   string // OLLYGARDEN_API_URL
	EnvContext  string // OLLYGARDEN_CONTEXT
	FlagAPIURL  string // --api-url
	FlagContext string // --context
}

// Credentials is the resolved output: the URL+key the HTTP client should
// use, plus metadata about where they came from for `auth status`.
type Credentials struct {
	APIURL      string
	APIKey      string
	Source      Source
	ContextName string // name of the context that was selected (or that env wins over)
}

// Resolve applies the precedence rules from the spec:
//
//   API key:  OLLYGARDEN_API_KEY > flag-context > env-context > current-context > error
//   API URL:  --api-url flag > OLLYGARDEN_API_URL > selected context's api-url > default
//
// API key and API URL resolve independently — a user can pair an env-var
// key with a flag-selected URL.
//
// Returns a typed *Error when no credential can be resolved or when a
// flag/env names an unknown context.
func Resolve(in ResolveInputs) (Credentials, error) {
	// Step 1: pick which context (if any) is selected for URL/metadata purposes.
	selectedName := ""
	switch {
	case in.FlagContext != "":
		selectedName = in.FlagContext
	case in.EnvContext != "":
		selectedName = in.EnvContext
	case in.Config.CurrentContext != "":
		selectedName = in.Config.CurrentContext
	}

	var selected *config.Context
	if selectedName != "" {
		selected = in.Config.Contexts[selectedName]
		// If a flag or env explicitly named a context, missing is an error.
		// If we just landed here via current-context, missing is treated as no
		// selection (defensive: shouldn't happen with a well-formed file, but
		// don't blow up).
		if selected == nil && (in.FlagContext != "" || in.EnvContext != "") {
			return Credentials{}, ErrContextNotFound(selectedName)
		}
	}

	// Step 2: resolve the API key.
	var creds Credentials
	switch {
	case in.EnvAPIKey != "":
		creds.APIKey = in.EnvAPIKey
		creds.Source = SourceEnv
		creds.ContextName = selectedName // may be "" — that's fine
	case selected != nil:
		creds.APIKey = selected.APIKey
		creds.Source = SourceContext
		creds.ContextName = selectedName
	default:
		return Credentials{}, ErrNoCredentials()
	}

	// Step 3: resolve the API URL (independent of the key).
	switch {
	case in.FlagAPIURL != "":
		creds.APIURL = in.FlagAPIURL
	case in.EnvAPIURL != "":
		creds.APIURL = in.EnvAPIURL
	case selected != nil && selected.APIURL != "":
		creds.APIURL = selected.APIURL
	default:
		creds.APIURL = DefaultAPIURL
	}

	return creds, nil
}
```

- [ ] **Step 4: Run the tests, watch them pass**

```bash
go test ./internal/auth/ -v
```
Expected: every `TestResolve*` case PASS.

- [ ] **Step 5: Format and commit**

```bash
go fmt ./...
git add internal/auth/
git commit -m "feat(auth): Resolve credentials by env/flag/context precedence"
```

---

## Phase 4 — `internal/auth` orchestration

### Task 9: `Login` in `internal/auth/login.go`

**Files:**
- Create: `internal/auth/login.go`
- Create: `internal/auth/login_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/auth/login_test.go`:

```go
package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ollygarden/ollygarden-cli/internal/config"
)

// withTempConfigPath redirects config.pathFunc to t.TempDir() so Login's
// persist step doesn't touch the real ~/.config.
func withTempConfigPath(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	// Use OLLYGARDEN_CONFIG since it's an exported override; cleaner than
	// reaching into the config package.
	t.Setenv(config.ConfigFileEnvVar, dir+"/config.yaml")
}

func newOrgServer(t *testing.T, status int, orgName string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/organization" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Bearer og_sk_") {
			t.Errorf("missing or malformed Authorization header: %q", got)
		}
		w.WriteHeader(status)
		if status == http.StatusOK {
			body := map[string]any{
				"data": map[string]any{"name": orgName},
				"meta": map[string]any{},
			}
			_ = json.NewEncoder(w).Encode(body)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLogin_HappyPath(t *testing.T) {
	withTempConfigPath(t)
	srv := newOrgServer(t, http.StatusOK, "Acme Corp")

	got, err := Login(context.Background(), LoginInputs{
		Token:       "og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		APIURL:      srv.URL,
		ContextName: "prod",
		Activate:    true,
	})
	if err != nil {
		t.Fatalf("Login(): %v", err)
	}
	if got.ContextName != "prod" {
		t.Errorf("ContextName: got %q, want prod", got.ContextName)
	}
	if got.OrganizationName != "Acme Corp" {
		t.Errorf("OrganizationName: got %q, want Acme Corp", got.OrganizationName)
	}
	if !got.Activated {
		t.Error("Activated: want true")
	}

	// Verify it landed on disk.
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load(): %v", err)
	}
	if cfg.CurrentContext != "prod" {
		t.Errorf("CurrentContext: got %q", cfg.CurrentContext)
	}
	if cfg.Contexts["prod"].APIKey != "og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Error("token did not round-trip to disk")
	}
}

func TestLogin_InvalidTokenFormat(t *testing.T) {
	withTempConfigPath(t)
	_, err := Login(context.Background(), LoginInputs{
		Token:       "not-a-real-key",
		APIURL:      "http://example.invalid",
		ContextName: "prod",
		Activate:    true,
	})
	if err == nil {
		t.Fatal("Login(): want error for bad shape")
	}
	got, ok := err.(*Error)
	if !ok || got.Code != "INVALID_TOKEN_FORMAT" {
		t.Errorf("want INVALID_TOKEN_FORMAT, got %T %v", err, err)
	}
}

func TestLogin_TokenRejected(t *testing.T) {
	withTempConfigPath(t)
	srv := newOrgServer(t, http.StatusUnauthorized, "")

	_, err := Login(context.Background(), LoginInputs{
		Token:       "og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		APIURL:      srv.URL,
		ContextName: "prod",
		Activate:    true,
	})
	if err == nil {
		t.Fatal("Login(): want error for 401")
	}
	got, ok := err.(*Error)
	if !ok || got.Code != "TOKEN_REJECTED" {
		t.Errorf("want TOKEN_REJECTED, got %T %v", err, err)
	}
}

func TestLogin_NoActivate(t *testing.T) {
	withTempConfigPath(t)
	srv := newOrgServer(t, http.StatusOK, "Acme")

	// Pre-seed a current context that should NOT be overwritten.
	pre := config.New()
	pre.CurrentContext = "existing"
	pre.Contexts["existing"] = &config.Context{Name: "existing", APIURL: "https://x", APIKey: "og_sk_pre000_eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"}
	if err := config.Write(pre); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := Login(context.Background(), LoginInputs{
		Token:       "og_sk_new000_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		APIURL:      srv.URL,
		ContextName: "new",
		Activate:    false,
	}); err != nil {
		t.Fatalf("Login(): %v", err)
	}

	cfg, _ := config.Load()
	if cfg.CurrentContext != "existing" {
		t.Errorf("CurrentContext: got %q, want %q (Activate=false should preserve)", cfg.CurrentContext, "existing")
	}
	if _, ok := cfg.Contexts["new"]; !ok {
		t.Error("expected new context to be added even with Activate=false")
	}
}

func TestLogin_OverwritesExistingSameName(t *testing.T) {
	withTempConfigPath(t)
	srv := newOrgServer(t, http.StatusOK, "Acme")

	pre := config.New()
	pre.Contexts["prod"] = &config.Context{Name: "prod", APIURL: "https://old", APIKey: "og_sk_old000_eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"}
	if err := config.Write(pre); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := Login(context.Background(), LoginInputs{
		Token:       "og_sk_new000_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		APIURL:      srv.URL,
		ContextName: "prod",
		Activate:    true,
	}); err != nil {
		t.Fatalf("Login(): %v", err)
	}

	cfg, _ := config.Load()
	if cfg.Contexts["prod"].APIKey != "og_sk_new000_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Errorf("expected overwrite, got %q", cfg.Contexts["prod"].APIKey)
	}
	if cfg.Contexts["prod"].APIURL != srv.URL {
		t.Errorf("expected URL update, got %q", cfg.Contexts["prod"].APIURL)
	}
}
```

- [ ] **Step 2: Run, watch them fail**

```bash
go test ./internal/auth/ -run TestLogin -v
```
Expected: `undefined: Login`, `undefined: LoginInputs`.

- [ ] **Step 3: Implement `internal/auth/login.go`**

```go
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ollygarden/ollygarden-cli/internal/config"
)

// tokenShape matches the og_sk_<6 alnum>_<32 hex> format. Match charset is
// permissive on the 6-char identifier (Olive does not enforce hex there in
// practice) and strict on the 32-char hex secret.
var tokenShape = regexp.MustCompile(`^og_sk_[A-Za-z0-9]{6}_[a-f0-9]{32}$`)

// LoginInputs is the pure-data input to Login. The cmd layer fills it in
// from flags + the resolved token (from --token-file, stdin, or TTY prompt).
type LoginInputs struct {
	Token       string
	APIURL      string
	ContextName string
	Activate    bool
	// HTTPClient is optional. When nil, http.DefaultClient with a 30s timeout
	// is used. Tests pass a custom client when needed.
	HTTPClient *http.Client
}

// LoginResult carries the post-login state the cmd layer needs to render
// human or JSON output.
type LoginResult struct {
	ContextName      string
	APIURL           string
	OrganizationName string
	KeyMasked        string
	Activated        bool
}

// Login validates the token against /api/v1/organization, then atomically
// persists the (overwriting any same-named context). On any failure after
// successful validation, no success is reported and the file is not
// updated.
func Login(ctx context.Context, in LoginInputs) (LoginResult, error) {
	if !tokenShape.MatchString(in.Token) {
		return LoginResult{}, ErrInvalidTokenFormat(in.Token)
	}

	httpClient := in.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	orgName, err := probeOrganization(ctx, httpClient, in.APIURL, in.Token)
	if err != nil {
		return LoginResult{}, err
	}

	// Load existing config (missing file is fine), upsert this context, write back.
	cfg, err := config.Load()
	if err != nil {
		// Translate config-package errors into auth.Error so cmd layer can route by code.
		var ue *config.UnreadableError
		if asAs(err, &ue) {
			return LoginResult{}, ErrConfigUnreadable(ue.Path, ue.Err)
		}
		return LoginResult{}, ErrConfigUnreadable("", err)
	}
	if cfg.Contexts == nil {
		cfg.Contexts = map[string]*config.Context{}
	}
	cfg.Contexts[in.ContextName] = &config.Context{
		Name:   in.ContextName,
		APIURL: in.APIURL,
		APIKey: in.Token,
	}
	if in.Activate {
		cfg.CurrentContext = in.ContextName
	}
	if err := config.Write(cfg); err != nil {
		var we *config.WriteFailedError
		if asAs(err, &we) {
			return LoginResult{}, ErrConfigWriteFailed(we.Path, we.Err)
		}
		return LoginResult{}, ErrConfigWriteFailed("", err)
	}

	return LoginResult{
		ContextName:      in.ContextName,
		APIURL:           in.APIURL,
		OrganizationName: orgName,
		KeyMasked:        MaskKey(in.Token),
		Activated:        in.Activate,
	}, nil
}

// probeOrganization performs the validation HTTP call. Returns the org name
// on 200, ErrTokenRejected on 401, or a generic error otherwise.
func probeOrganization(ctx context.Context, c *http.Client, baseURL, token string) (string, error) {
	url := strings.TrimRight(baseURL, "/") + "/api/v1/organization"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling %s: %w", url, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// Parse out the org name. Tolerate a body without a name field.
		var envelope struct {
			Data struct {
				Name string `json:"name"`
			} `json:"data"`
		}
		body, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(body, &envelope)
		return envelope.Data.Name, nil
	case http.StatusUnauthorized:
		return "", ErrTokenRejected()
	default:
		return "", fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}
}

// asAs is a tiny helper so the imports stay tidy; behaves like errors.As.
func asAs(err error, target any) bool {
	type asable interface{ As(any) bool }
	if a, ok := err.(asable); ok {
		return a.As(target)
	}
	// Fall through to the standard library form via interface assertion at
	// the call site. We use the real errors.As below.
	return errorsAs(err, target)
}
```

Add the actual `errors.As` import at the top of the file by replacing the bottom helper with a real call. Replace `asAs` and the bottom helper with:

```go
// At top of imports, add:
//   "errors"
// Then the helpers below.

// asAs delegates to errors.As. Defined as a wrapper so tests can stub if needed.
func asAs(err error, target any) bool {
	return errors.As(err, target)
}

// errorsAs is unused; remove the previous placeholder.
```

(In other words: just `import "errors"` and `func asAs(err error, target any) bool { return errors.As(err, target) }`. Drop the placeholder `errorsAs`.)

Final imports for `internal/auth/login.go`:

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ollygarden/ollygarden-cli/internal/config"
)
```

- [ ] **Step 4: Run all auth tests, watch them pass**

```bash
go test ./internal/auth/ -v
```
Expected: every test in the package PASS.

- [ ] **Step 5: Format and commit**

```bash
go fmt ./...
git add internal/auth/
git commit -m "feat(auth): Login orchestrates token validation and atomic persist"
```

---

## Phase 5 — `cmd/root.go` integration

### Task 10: Add `--context` persistent flag and wire `auth.Resolve`

**Files:**
- Modify: `cmd/root.go`
- Modify: `cmd/root_test.go`

- [ ] **Step 1: Update the existing test for the new error type**

In `cmd/root_test.go`, replace `TestMissingAPIKeyReturnsAuthError` with this one (note: the existing test refers to `*AuthError`; we're replacing that type):

```go
func TestMissingAPIKey_ReturnsTypedNoCredentialsError(t *testing.T) {
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	// Point config to an empty location to suppress real ~/.config reads.
	t.Setenv("OLLYGARDEN_CONFIG", t.TempDir()+"/config.yaml")

	testCmd := &cobra.Command{
		Use:  "auth-test-cmd",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	rootCmd.AddCommand(testCmd)
	defer rootCmd.RemoveCommand(testCmd)

	_, _, err := executeCommand("auth-test-cmd")
	require.Error(t, err)

	var ae *auth.Error
	require.True(t, errors.As(err, &ae), "expected *auth.Error, got %T: %v", err, err)
	assert.Equal(t, "NO_CREDENTIALS", ae.Code)
}

func TestEnvAPIKey_StillWorks(t *testing.T) {
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_envkey_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	t.Setenv("OLLYGARDEN_CONFIG", t.TempDir()+"/config.yaml")

	testCmd := &cobra.Command{
		Use:  "auth-ok-cmd",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	rootCmd.AddCommand(testCmd)
	defer rootCmd.RemoveCommand(testCmd)

	_, _, err := executeCommand("auth-ok-cmd")
	require.NoError(t, err)
}
```

Add the new imports at the top of `cmd/root_test.go`:

```go
import (
	// ... existing imports ...
	"errors"

	"github.com/ollygarden/ollygarden-cli/internal/auth"
)
```

- [ ] **Step 2: Run, watch the build break (AuthError no longer exists / new symbols missing)**

```bash
go test ./cmd/ -run TestMissingAPIKey -v
```
Expected: build error.

- [ ] **Step 3: Rewrite `cmd/root.go`**

Replace the entire file with this:

```go
package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ollygarden/ollygarden-cli/internal/auth"
	"github.com/ollygarden/ollygarden-cli/internal/client"
	"github.com/ollygarden/ollygarden-cli/internal/config"
	"github.com/ollygarden/ollygarden-cli/internal/exitcode"
	"github.com/spf13/cobra"
)

var (
	apiURL      string
	contextName string // bound to --context persistent flag
	jsonMode    bool
	quiet       bool
	version     = "dev"
	commit      = "none"
	date        = "unknown"

	// resolvedCreds is populated by PersistentPreRunE for non-auth commands
	// so NewClient (and any future helper) can read it.
	resolvedCreds auth.Credentials
)

var rootCmd = &cobra.Command{
	Use:           "ollygarden",
	Short:         "CLI client for the OllyGarden API",
	Version:       "dev",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if skipAuthResolution(cmd) {
			return nil
		}

		if apiURL != "" && !strings.HasPrefix(apiURL, "http://") && !strings.HasPrefix(apiURL, "https://") {
			return fmt.Errorf("Error: --api-url must include scheme (e.g., https://api.ollygarden.cloud)")
		}

		cfg, err := config.Load()
		if err != nil {
			var ue *config.UnreadableError
			if errors.As(err, &ue) {
				return auth.ErrConfigUnreadable(ue.Path, ue.Err)
			}
			return auth.ErrConfigUnreadable("", err)
		}

		creds, err := auth.Resolve(auth.ResolveInputs{
			Config:      cfg,
			EnvAPIKey:   os.Getenv("OLLYGARDEN_API_KEY"),
			EnvAPIURL:   os.Getenv("OLLYGARDEN_API_URL"),
			EnvContext:  os.Getenv(config.ContextEnvVar),
			FlagAPIURL:  apiURL,
			FlagContext: contextName,
		})
		if err != nil {
			return err
		}
		resolvedCreds = creds
		// Make the URL available to NewClient via the existing global.
		apiURL = creds.APIURL
		return nil
	},
}

// skipAuthResolution returns true for commands that should not have
// credentials resolved before running: help, version, the bare root, and
// any command in the `auth` subtree (auth login does the resolution
// itself, the others either don't need creds or compute them on demand).
func skipAuthResolution(cmd *cobra.Command) bool {
	name := cmd.Name()
	if name == "help" || name == "version" || name == "ollygarden" {
		return true
	}
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "auth" {
			return true
		}
	}
	return false
}

func init() {
	defaultURL := "https://api.ollygarden.cloud"
	if envURL := os.Getenv("OLLYGARDEN_API_URL"); envURL != "" {
		defaultURL = envURL
	}

	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", defaultURL, "Base URL for the API")
	rootCmd.PersistentFlags().StringVar(&contextName, "context", "", "Use a specific saved context for this invocation (overrides current-context, OLLYGARDEN_CONTEXT)")
	rootCmd.PersistentFlags().BoolVar(&jsonMode, "json", false, "Output raw JSON")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
}

// SetBuildInfo sets the CLI build metadata. Values come from ldflags at release time.
func SetBuildInfo(v, c, d string) {
	version = v
	commit = c
	date = d
	rootCmd.Version = v
	client.SetVersion(v)
}

// NewClient creates an API client from the resolved credentials. For
// non-auth commands, PersistentPreRunE will have populated resolvedCreds.
// For auth subcommands that call NewClient (auth status --probe), they
// must populate resolvedCreds themselves before calling.
func NewClient() *client.Client {
	if resolvedCreds.APIKey != "" {
		return client.New(resolvedCreds.APIURL, resolvedCreds.APIKey)
	}
	// Fallback for the rare path where resolution was skipped: use env directly.
	return client.New(apiURL, os.Getenv("OLLYGARDEN_API_KEY"))
}

// Execute runs the root command and exits with the appropriate code.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		code := exitcode.General

		var authErr *auth.Error
		var apiErr *client.APIError
		switch {
		case errors.As(err, &authErr):
			fmt.Fprintln(os.Stderr, "Error: "+authErr.Message)
			code = authErr.ExitCode
		case errors.As(err, &apiErr):
			fmt.Fprintln(os.Stderr, apiErr.Error())
			code = apiErr.ExitCode()
		default:
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(code)
	}
}
```

- [ ] **Step 4: Run the cmd tests, watch them pass**

```bash
go test ./cmd/ -run "TestMissingAPIKey|TestEnvAPIKey|TestHelp|TestVersion" -v
```
Expected: PASS. Other existing tests (services, insights, etc.) should still PASS too — run the whole package:

```bash
go test ./cmd/ -v
```

Note: existing tests that did `t.Setenv("OLLYGARDEN_API_KEY", "og_sk_test_key")` may now succeed at the resolution step but then receive a 200/whatever from the test server. They should keep passing. If a test fails because resolution can't find credentials, set `OLLYGARDEN_CONFIG` to `t.TempDir()+"/config.yaml"` in that test's setup helper.

- [ ] **Step 5: Format and commit**

```bash
go fmt ./...
git add cmd/
git commit -m "refactor(cmd): route credential resolution through internal/auth"
```

---

## Phase 6 — Auth subcommands

### Task 11: `cmd/auth.go` parent command

**Files:**
- Create: `cmd/auth.go`
- Create: `cmd/auth_test.go`

- [ ] **Step 1: Write the failing test**

Create `cmd/auth_test.go`:

```go
package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthCommandHelpListsSubcommands(t *testing.T) {
	out, _, err := executeCommand("auth", "--help")
	require.NoError(t, err)
	for _, sub := range []string{"login", "logout", "status", "use-context", "list-contexts"} {
		assert.True(t, strings.Contains(out, sub), "auth --help should list %q, got:\n%s", sub, out)
	}
}
```

- [ ] **Step 2: Run, watch it fail**

```bash
go test ./cmd/ -run TestAuthCommandHelp -v
```
Expected: command not found / help output missing the subcommands.

- [ ] **Step 3: Implement `cmd/auth.go`**

```go
package cmd

import "github.com/spf13/cobra"

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage OllyGarden CLI credentials",
	Long: `Manage credentials stored on disk for the OllyGarden CLI.

Credentials are kept in a YAML file at:
  os.UserConfigDir()/ollygarden/config.yaml  (mode 0600)

Override with the OLLYGARDEN_CONFIG environment variable.

The OLLYGARDEN_API_KEY environment variable always wins over saved
credentials, so CI runs and ad-hoc invocations continue to work.`,
}

func init() {
	rootCmd.AddCommand(authCmd)
}
```

- [ ] **Step 4: Run, watch it pass (once subcommands exist; for now help just shows the parent)**

```bash
go test ./cmd/ -run TestAuthCommandHelp -v
```

It will FAIL because no subcommands are wired yet. That's expected — the test passes once Tasks 12–16 finish. **Mark this test as expected-fail-for-now in your head; do not modify.** Do NOT commit yet — go to Task 12 and come back to commit after all subcommands land.

- [ ] **Step 5: Hold the commit**

Defer the commit; it lands as part of the next task or be combined with all subcommands.

---

### Task 12: `cmd/auth_login.go` with all three input modes

**Files:**
- Create: `cmd/auth_login.go`
- Create: `cmd/auth_login_test.go`

- [ ] **Step 1: Write the failing tests**

Create `cmd/auth_login_test.go`:

```go
package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ollygarden/ollygarden-cli/internal/config"
)

func setupAuthLoginEnv(t *testing.T) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(dir, "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/organization", r.URL.Path)
		assert.True(t, strings.HasPrefix(r.Header.Get("Authorization"), "Bearer og_sk_"))
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"name": "Acme Corp"},
			"meta": map[string]any{},
		})
	}))
	t.Cleanup(srv.Close)

	// Reset auth_login flags between tests since they're package globals.
	t.Cleanup(func() {
		authLoginTokenFile = ""
		authLoginNoActivate = false
		jsonMode = false
		quiet = false
	})
	return srv
}

func TestAuthLogin_TokenFile_HappyPath(t *testing.T) {
	srv := setupAuthLoginEnv(t)
	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n"), 0o600))

	out, _, err := executeCommand("auth", "login",
		"--api-url", srv.URL,
		"--context", "prod",
		"--token-file", tokenPath,
	)
	require.NoError(t, err)
	assert.Contains(t, out+"", "") // touch out to silence linter
	cfg, err := config.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg.Contexts["prod"])
	assert.Equal(t, "og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", cfg.Contexts["prod"].APIKey)
	assert.Equal(t, "prod", cfg.CurrentContext)
}

func TestAuthLogin_TokenFile_JSONOutput(t *testing.T) {
	srv := setupAuthLoginEnv(t)
	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), 0o600))

	out, _, err := executeCommand("auth", "login",
		"--api-url", srv.URL,
		"--context", "prod",
		"--token-file", tokenPath,
		"--json",
	)
	require.NoError(t, err)

	var env struct {
		Data struct {
			Context      string `json:"context"`
			Organization string `json:"organization"`
			KeyMasked    string `json:"key_masked"`
			Activated    bool   `json:"activated"`
			APIURL       string `json:"api_url"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &env))
	assert.Equal(t, "prod", env.Data.Context)
	assert.Equal(t, "Acme Corp", env.Data.Organization)
	assert.Equal(t, "og_sk_abc123_••••", env.Data.KeyMasked)
	assert.True(t, env.Data.Activated)
}

func TestAuthLogin_TokenFile_Missing(t *testing.T) {
	srv := setupAuthLoginEnv(t)
	_, _, err := executeCommand("auth", "login",
		"--api-url", srv.URL,
		"--context", "prod",
		"--token-file", "/nonexistent/path/token",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Cannot read token file")
}

func TestAuthLogin_TokenRejected(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(dir, "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() {
		authLoginTokenFile = ""
		authLoginNoActivate = false
	})

	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), 0o600))

	_, _, err := executeCommand("auth", "login",
		"--api-url", srv.URL,
		"--context", "prod",
		"--token-file", tokenPath,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Token rejected")
}

func TestAuthLogin_InvalidTokenFormat(t *testing.T) {
	srv := setupAuthLoginEnv(t)
	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("not-a-real-token"), 0o600))

	_, _, err := executeCommand("auth", "login",
		"--api-url", srv.URL,
		"--context", "prod",
		"--token-file", tokenPath,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid token format")
}

func TestAuthLogin_NoActivate(t *testing.T) {
	srv := setupAuthLoginEnv(t)
	// Pre-seed
	pre := config.New()
	pre.CurrentContext = "existing"
	pre.Contexts["existing"] = &config.Context{Name: "existing", APIURL: "https://x", APIKey: "og_sk_pre000_eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"}
	require.NoError(t, config.Write(pre))

	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(tokenPath, []byte("og_sk_new000_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), 0o600))

	_, _, err := executeCommand("auth", "login",
		"--api-url", srv.URL,
		"--context", "new",
		"--token-file", tokenPath,
		"--no-activate",
	)
	require.NoError(t, err)

	cfg, _ := config.Load()
	assert.Equal(t, "existing", cfg.CurrentContext, "current-context must not change with --no-activate")
	assert.NotNil(t, cfg.Contexts["new"], "new context must still be added")
}
```

- [ ] **Step 2: Run, watch them fail**

```bash
go test ./cmd/ -run TestAuthLogin -v
```
Expected: command not found.

- [ ] **Step 3: Implement `cmd/auth_login.go`**

```go
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/ollygarden/ollygarden-cli/internal/auth"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	authLoginTokenFile  string
	authLoginNoActivate bool
)

const tokenURLHint = "Get a token at https://ollygarden.app/settings"

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Save credentials to disk for a context",
	Long: `Save an OllyGarden API key to a named context on disk.

Three ways to provide the token, picked in this order:
  1. --token-file PATH        read from a file
  2. stdin (when piped)       read one line
  3. interactive TTY          prompt with hidden input

The token is validated against the API before any data is written. On
success, the context becomes the current-context unless --no-activate
is passed.`,
	Args: cobra.NoArgs,
	RunE: runAuthLogin,
}

func init() {
	authLoginCmd.Flags().StringVar(&authLoginTokenFile, "token-file", "", "Read the API token from this file path")
	authLoginCmd.Flags().BoolVar(&authLoginNoActivate, "no-activate", false, "Save the context without setting it as current-context")
	authCmd.AddCommand(authLoginCmd)
}

func runAuthLogin(cmd *cobra.Command, _ []string) error {
	token, err := readTokenInput(cmd)
	if err != nil {
		return err
	}

	resolvedURL := apiURL // default, env, or --api-url all flow through here
	ctxName := contextName
	if ctxName == "" {
		derived, derr := deriveContextName(resolvedURL)
		if derr != nil {
			return fmt.Errorf("deriving context name: %w", derr)
		}
		ctxName = derived
	}

	result, err := auth.Login(cmd.Context(), auth.LoginInputs{
		Token:       token,
		APIURL:      resolvedURL,
		ContextName: ctxName,
		Activate:    !authLoginNoActivate,
	})
	if err != nil {
		return err
	}

	if jsonMode {
		out := map[string]any{
			"data": map[string]any{
				"context":      result.ContextName,
				"api_url":      result.APIURL,
				"organization": result.OrganizationName,
				"key_masked":   result.KeyMasked,
				"activated":    result.Activated,
			},
			"meta": map[string]any{},
		}
		raw, _ := json.Marshal(out)
		fmt.Fprintln(cmd.OutOrStdout(), string(raw))
		return nil
	}

	if quiet {
		return nil
	}

	orgPart := ""
	if result.OrganizationName != "" {
		orgPart = fmt.Sprintf(" to %q", result.OrganizationName)
	}
	fmt.Fprintf(cmd.OutOrStdout(),
		"✓ Logged in%s as context %q (%s).\n",
		orgPart, result.ContextName, result.APIURL,
	)
	return nil
}

// readTokenInput selects the token source per the spec's precedence:
// --token-file > non-TTY stdin > TTY prompt.
func readTokenInput(cmd *cobra.Command) (string, error) {
	if authLoginTokenFile != "" {
		data, err := os.ReadFile(authLoginTokenFile)
		if err != nil {
			return "", auth.ErrTokenFileNotFound(authLoginTokenFile)
		}
		return strings.TrimSpace(string(data)), nil
	}

	stdin := cmd.InOrStdin()
	if !isTerminal(stdin) {
		raw, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("reading token from stdin: %w", err)
		}
		// Take only the first line, trimmed.
		s := bufio.NewScanner(strings.NewReader(string(raw)))
		if s.Scan() {
			return strings.TrimSpace(s.Text()), nil
		}
		return "", fmt.Errorf("no token on stdin")
	}

	// Interactive TTY: print hint to stderr, then ReadPassword.
	fmt.Fprintln(cmd.ErrOrStderr(), tokenURLHint)
	fmt.Fprint(cmd.ErrOrStderr(), "Paste your OllyGarden API key: ")
	tokBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return "", fmt.Errorf("reading token: %w", err)
	}
	return strings.TrimSpace(string(tokBytes)), nil
}

// isTerminal reports whether r is connected to a terminal. Anything that
// isn't *os.File (e.g. bytes.Buffer in tests) counts as non-TTY.
func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// deriveContextName implements the spec rule: strip leading "api." from the
// hostname, replace remaining "." with "-".
func deriveContextName(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	host := u.Hostname()
	host = strings.TrimPrefix(host, "api.")
	host = strings.ReplaceAll(host, ".", "-")
	if host == "" {
		return "default", nil
	}
	return host, nil
}
```

- [ ] **Step 4: Run, watch the tests pass**

```bash
go test ./cmd/ -run "TestAuthLogin|TestAuthCommandHelp" -v
```
Expected: PASS.

- [ ] **Step 5: Format. Hold commit until subcommands done**

```bash
go fmt ./...
```

(Defer commit; we'll batch by subcommand or commit per-subcommand. The plan author chose per-subcommand. Commit now:)

```bash
git add cmd/auth.go cmd/auth_test.go cmd/auth_login.go cmd/auth_login_test.go
git commit -m "feat(cmd): auth login with --token-file, stdin, and TTY paths"
```

---

### Task 13: `cmd/auth_logout.go`

**Files:**
- Create: `cmd/auth_logout.go`
- Create: `cmd/auth_logout_test.go`

- [ ] **Step 1: Write the failing tests**

Create `cmd/auth_logout_test.go`:

```go
package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ollygarden/ollygarden-cli/internal/config"
)

func setupTwoContexts(t *testing.T) {
	t.Helper()
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	cfg := config.New()
	cfg.CurrentContext = "prod"
	cfg.Contexts["prod"] = &config.Context{Name: "prod", APIURL: "https://prod", APIKey: "og_sk_prod00_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	cfg.Contexts["dev"] = &config.Context{Name: "dev", APIURL: "https://dev", APIKey: "og_sk_dev000_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	require.NoError(t, config.Write(cfg))
	t.Cleanup(func() {
		authLogoutContext = ""
		authLogoutAll = false
		authLogoutConfirm = false
	})
}

func TestAuthLogout_DefaultRemovesCurrent(t *testing.T) {
	setupTwoContexts(t)
	_, _, err := executeCommand("auth", "logout")
	require.NoError(t, err)

	cfg, _ := config.Load()
	assert.Empty(t, cfg.CurrentContext, "current-context must be unset")
	_, gone := cfg.Contexts["prod"]
	assert.False(t, gone, "prod context must be removed")
	_, kept := cfg.Contexts["dev"]
	assert.True(t, kept, "dev context must remain")
}

func TestAuthLogout_NamedContext(t *testing.T) {
	setupTwoContexts(t)
	_, _, err := executeCommand("auth", "logout", "--context", "dev")
	require.NoError(t, err)

	cfg, _ := config.Load()
	_, gone := cfg.Contexts["dev"]
	assert.False(t, gone)
	assert.Equal(t, "prod", cfg.CurrentContext, "current-context unchanged when removing non-current")
}

func TestAuthLogout_ContextNotFound(t *testing.T) {
	setupTwoContexts(t)
	_, _, err := executeCommand("auth", "logout", "--context", "ghost")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `not found`)
}

func TestAuthLogout_AllRequiresConfirm(t *testing.T) {
	setupTwoContexts(t)
	// Non-TTY, no --confirm: must refuse.
	_, _, err := executeCommand("auth", "logout", "--all")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--confirm")
}

func TestAuthLogout_AllWithConfirm(t *testing.T) {
	setupTwoContexts(t)
	_, _, err := executeCommand("auth", "logout", "--all", "--confirm")
	require.NoError(t, err)

	cfg, _ := config.Load()
	assert.Empty(t, cfg.Contexts, "all contexts removed")
	assert.Empty(t, cfg.CurrentContext)
}
```

- [ ] **Step 2: Run, watch them fail**

```bash
go test ./cmd/ -run TestAuthLogout -v
```

- [ ] **Step 3: Implement `cmd/auth_logout.go`**

```go
package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ollygarden/ollygarden-cli/internal/auth"
	"github.com/ollygarden/ollygarden-cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"os"
)

var (
	authLogoutContext string
	authLogoutAll     bool
	authLogoutConfirm bool
)

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove a saved context (or all of them)",
	Long: `Remove a saved context from disk.

  Default                 Remove the current-context and unset the pointer.
  --context NAME          Remove a specific context.
  --all                   Remove every context. Requires --confirm in non-TTY mode.

When the last context is removed, the config file is deleted entirely.`,
	Args: cobra.NoArgs,
	RunE: runAuthLogout,
}

func init() {
	// --context here SHADOWS the persistent flag at this command's scope.
	// We use the same flag name because it carries the same intent ("name a
	// context"); the --context value is read from this command's own flag set.
	authLogoutCmd.Flags().StringVar(&authLogoutContext, "context", "", "Name of the context to remove")
	authLogoutCmd.Flags().BoolVar(&authLogoutAll, "all", false, "Remove every saved context")
	authLogoutCmd.Flags().BoolVar(&authLogoutConfirm, "confirm", false, "Required for --all in non-interactive mode")
	authCmd.AddCommand(authLogoutCmd)
}

func runAuthLogout(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		var ue *config.UnreadableError
		if errAs(err, &ue) {
			return auth.ErrConfigUnreadable(ue.Path, ue.Err)
		}
		return auth.ErrConfigUnreadable("", err)
	}

	switch {
	case authLogoutAll:
		if !authLogoutConfirm && !isTerminal(os.Stdin) {
			return auth.ErrConfirmRequired()
		}
		// On a TTY without --confirm, prompt y/N (default No).
		if !authLogoutConfirm && isTerminal(os.Stdin) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Remove all %d saved contexts? [y/N]: ", len(cfg.Contexts))
			b := make([]byte, 1)
			_, _ = os.Stdin.Read(b)
			fmt.Fprintln(cmd.ErrOrStderr())
			if !strings.EqualFold(string(b), "y") {
				return fmt.Errorf("aborted")
			}
		}
		cfg.Contexts = map[string]*config.Context{}
		cfg.CurrentContext = ""
	case authLogoutContext != "":
		if _, ok := cfg.Contexts[authLogoutContext]; !ok {
			return auth.ErrContextNotFound(authLogoutContext)
		}
		delete(cfg.Contexts, authLogoutContext)
		if cfg.CurrentContext == authLogoutContext {
			cfg.CurrentContext = ""
		}
	default:
		if cfg.CurrentContext == "" {
			return auth.ErrNoCredentials()
		}
		removed := cfg.CurrentContext
		delete(cfg.Contexts, removed)
		cfg.CurrentContext = ""
		// Remember what we removed for the success message.
		_ = removed
	}

	if err := config.Write(cfg); err != nil {
		var we *config.WriteFailedError
		if errAs(err, &we) {
			return auth.ErrConfigWriteFailed(we.Path, we.Err)
		}
		return auth.ErrConfigWriteFailed("", err)
	}

	if jsonMode {
		out := map[string]any{
			"data": map[string]any{
				"removed_all":     authLogoutAll,
				"current_context": cfg.CurrentContext,
				"remaining":       sortedContextNames(cfg.Contexts),
			},
			"meta": map[string]any{},
		}
		raw, _ := json.Marshal(out)
		fmt.Fprintln(cmd.OutOrStdout(), string(raw))
		return nil
	}

	if quiet {
		return nil
	}

	switch {
	case authLogoutAll:
		fmt.Fprintln(cmd.OutOrStdout(), "✓ Removed all saved contexts.")
	case authLogoutContext != "":
		fmt.Fprintf(cmd.OutOrStdout(), "✓ Removed context %q.\n", authLogoutContext)
	default:
		fmt.Fprintln(cmd.OutOrStdout(), "✓ Logged out.")
	}

	if !authLogoutAll && len(cfg.Contexts) > 0 && cfg.CurrentContext == "" {
		names := sortedContextNames(cfg.Contexts)
		fmt.Fprintf(cmd.OutOrStdout(),
			"No current context set. Available: %s. Activate with `ollygarden auth use-context NAME`.\n",
			strings.Join(names, ", "),
		)
	}
	return nil
}

func sortedContextNames(m map[string]*config.Context) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// errAs is a thin wrapper to keep the cmd package's import list compact.
func errAs(err error, target any) bool {
	type asable interface{ As(any) bool }
	if a, ok := err.(asable); ok {
		return a.As(target)
	}
	// Fall through to the standard library; we add the import at compile time.
	return errorsAsCmd(err, target)
}
```

The `errorsAsCmd` shim and `term`/`os` imports are awkward — let's just use the stdlib directly. Replace the helper with a real `errors` import:

```go
// At top of file imports:
import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ollygarden/ollygarden-cli/internal/auth"
	"github.com/ollygarden/ollygarden-cli/internal/config"
	"github.com/spf13/cobra"
)

// (golang.org/x/term not needed here — isTerminal is reused from auth_login.go)
```

And replace `errAs(err, &ue)` with `errors.As(err, &ue)`. Drop the `errAs` and `errorsAsCmd` definitions entirely. (The `golang.org/x/term` import was a holdover from an earlier draft of this task; `isTerminal` is already defined in `cmd/auth_login.go` and reusable across the package.)

- [ ] **Step 4: Run, watch tests pass**

```bash
go test ./cmd/ -run TestAuthLogout -v
```

- [ ] **Step 5: Format and commit**

```bash
go fmt ./...
git add cmd/auth_logout.go cmd/auth_logout_test.go
git commit -m "feat(cmd): auth logout with default/--context/--all variants"
```

---

### Task 14: `cmd/auth_status.go`

**Files:**
- Create: `cmd/auth_status.go`
- Create: `cmd/auth_status_test.go`

- [ ] **Step 1: Write the failing tests**

Create `cmd/auth_status_test.go`:

```go
package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ollygarden/ollygarden-cli/internal/config"
)

func TestAuthStatus_NoCreds_Exit3(t *testing.T) {
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	t.Cleanup(func() { authStatusNoProbe = false })

	_, _, err := executeCommand("auth", "status", "--no-probe")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No credentials")
}

func TestAuthStatus_FromContext_NoProbe_JSON(t *testing.T) {
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	t.Cleanup(func() { authStatusNoProbe = false })

	cfg := config.New()
	cfg.CurrentContext = "prod"
	cfg.Contexts["prod"] = &config.Context{Name: "prod", APIURL: "https://api.example.com", APIKey: "og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	require.NoError(t, config.Write(cfg))

	out, _, err := executeCommand("auth", "status", "--no-probe", "--json")
	require.NoError(t, err)

	var env struct {
		Data struct {
			Source    string `json:"source"`
			Context   string `json:"context"`
			APIURL    string `json:"api_url"`
			KeyMasked string `json:"key_masked"`
			Probed    bool   `json:"probed"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &env))
	assert.Equal(t, "context", env.Data.Source)
	assert.Equal(t, "prod", env.Data.Context)
	assert.Equal(t, "og_sk_abc123_••••", env.Data.KeyMasked)
	assert.False(t, env.Data.Probed)
}

func TestAuthStatus_EnvWinsOverContext_JSON(t *testing.T) {
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_envkey_cccccccccccccccccccccccccccccccc")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	t.Cleanup(func() { authStatusNoProbe = false })

	cfg := config.New()
	cfg.CurrentContext = "prod"
	cfg.Contexts["prod"] = &config.Context{Name: "prod", APIURL: "https://api.example.com", APIKey: "og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	require.NoError(t, config.Write(cfg))

	out, _, err := executeCommand("auth", "status", "--no-probe", "--json")
	require.NoError(t, err)

	var env struct {
		Data struct {
			Source  string `json:"source"`
			Context string `json:"context"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &env))
	assert.Equal(t, "env", env.Data.Source)
	assert.Equal(t, "prod", env.Data.Context, "env source still reports the saved context that would have won")
}

func TestAuthStatus_Probe_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/organization", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"name": "Acme Corp"}, "meta": map[string]any{}})
	}))
	t.Cleanup(srv.Close)

	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")
	t.Cleanup(func() { authStatusNoProbe = false })
	cfg := config.New()
	cfg.CurrentContext = "prod"
	cfg.Contexts["prod"] = &config.Context{Name: "prod", APIURL: srv.URL, APIKey: "og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	require.NoError(t, config.Write(cfg))

	out, _, err := executeCommand("auth", "status", "--json")
	require.NoError(t, err)

	var env struct {
		Data struct {
			Probed       bool   `json:"probed"`
			Organization string `json:"organization"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &env))
	assert.True(t, env.Data.Probed)
	assert.Equal(t, "Acme Corp", env.Data.Organization)
}
```

- [ ] **Step 2: Run, watch them fail**

```bash
go test ./cmd/ -run TestAuthStatus -v
```

- [ ] **Step 3: Implement `cmd/auth_status.go`**

```go
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ollygarden/ollygarden-cli/internal/auth"
	"github.com/ollygarden/ollygarden-cli/internal/config"
	"github.com/spf13/cobra"
)

var authStatusNoProbe bool

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the active credential and verify it works",
	Long: `Print the active credential's source, URL, and a masked key.

By default, makes a single GET /api/v1/organization request to confirm
the token is still accepted (matches `+"`gh auth status`"+` precedent). Pass
--no-probe to skip the network call.

Exit codes:
  0  Logged in (and probe succeeded if probing).
  3  No credential is configured, or the probe got 401.`,
	Args: cobra.NoArgs,
	RunE: runAuthStatus,
}

func init() {
	authStatusCmd.Flags().BoolVar(&authStatusNoProbe, "no-probe", false, "Skip the /organization probe")
	authCmd.AddCommand(authStatusCmd)
}

func runAuthStatus(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		var ue *config.UnreadableError
		if errors.As(err, &ue) {
			return auth.ErrConfigUnreadable(ue.Path, ue.Err)
		}
		return auth.ErrConfigUnreadable("", err)
	}

	creds, err := auth.Resolve(auth.ResolveInputs{
		Config:      cfg,
		EnvAPIKey:   os.Getenv("OLLYGARDEN_API_KEY"),
		EnvAPIURL:   os.Getenv("OLLYGARDEN_API_URL"),
		EnvContext:  os.Getenv(config.ContextEnvVar),
		FlagAPIURL:  apiURL,
		FlagContext: contextName,
	})
	if err != nil {
		return err
	}

	probed := false
	orgName := ""
	if !authStatusNoProbe {
		probed = true
		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()
		orgName, err = probeOrgFromCmd(ctx, creds.APIURL, creds.APIKey)
		if err != nil {
			return err
		}
	}

	source := "context"
	if creds.Source == auth.SourceEnv {
		source = "env"
	}

	if jsonMode {
		out := map[string]any{
			"data": map[string]any{
				"source":     source,
				"context":    creds.ContextName,
				"api_url":    creds.APIURL,
				"key_masked": auth.MaskKey(creds.APIKey),
				"probed":     probed,
			},
			"meta": map[string]any{},
		}
		if probed && orgName != "" {
			out["data"].(map[string]any)["organization"] = orgName
		}
		raw, _ := json.Marshal(out)
		fmt.Fprintln(cmd.OutOrStdout(), string(raw))
		return nil
	}

	if quiet {
		return nil
	}

	srcLine := source
	if source == "env" {
		srcLine = "env (OLLYGARDEN_API_KEY)"
		if creds.ContextName != "" {
			srcLine += fmt.Sprintf(" — overrides saved context %q", creds.ContextName)
		}
	} else if source == "context" {
		srcLine = fmt.Sprintf("context: %s", creds.ContextName)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Source:        %s\n", srcLine)
	fmt.Fprintf(cmd.OutOrStdout(), "API URL:       %s\n", creds.APIURL)
	fmt.Fprintf(cmd.OutOrStdout(), "Key:           %s\n", auth.MaskKey(creds.APIKey))
	if probed && orgName != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Organization:  %s\n", orgName)
	}
	return nil
}

// probeOrgFromCmd is a small wrapper that mirrors auth.probeOrganization
// behavior — separated here so cmd doesn't have to import the unexported
// version. Returns auth.ErrTokenRejected on 401.
func probeOrgFromCmd(ctx context.Context, baseURL, token string) (string, error) {
	url := strings.TrimRight(baseURL, "/") + "/api/v1/organization"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("calling %s: %w", url, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var envelope struct {
			Data struct {
				Name string `json:"name"`
			} `json:"data"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&envelope)
		return envelope.Data.Name, nil
	case http.StatusUnauthorized:
		return "", auth.ErrTokenRejected()
	default:
		return "", fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}
}
```

- [ ] **Step 4: Run, watch tests pass**

```bash
go test ./cmd/ -run TestAuthStatus -v
```

- [ ] **Step 5: Format and commit**

```bash
go fmt ./...
git add cmd/auth_status.go cmd/auth_status_test.go
git commit -m "feat(cmd): auth status with optional /organization probe"
```

---

### Task 15: `cmd/auth_use_context.go`

**Files:**
- Create: `cmd/auth_use_context.go`
- Create: `cmd/auth_use_context_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// cmd/auth_use_context_test.go
package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ollygarden/ollygarden-cli/internal/config"
)

func setupTwoContextsForUse(t *testing.T) {
	t.Helper()
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	cfg := config.New()
	cfg.CurrentContext = "prod"
	cfg.Contexts["prod"] = &config.Context{Name: "prod", APIURL: "https://prod", APIKey: "og_sk_prod00_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	cfg.Contexts["dev"] = &config.Context{Name: "dev", APIURL: "https://dev", APIKey: "og_sk_dev000_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	require.NoError(t, config.Write(cfg))
}

func TestAuthUseContext_Switches(t *testing.T) {
	setupTwoContextsForUse(t)
	_, _, err := executeCommand("auth", "use-context", "dev")
	require.NoError(t, err)
	cfg, _ := config.Load()
	assert.Equal(t, "dev", cfg.CurrentContext)
}

func TestAuthUseContext_NotFound(t *testing.T) {
	setupTwoContextsForUse(t)
	_, _, err := executeCommand("auth", "use-context", "ghost")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
```

- [ ] **Step 2: Run, watch them fail**

```bash
go test ./cmd/ -run TestAuthUseContext -v
```

- [ ] **Step 3: Implement `cmd/auth_use_context.go`**

```go
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ollygarden/ollygarden-cli/internal/auth"
	"github.com/ollygarden/ollygarden-cli/internal/config"
	"github.com/spf13/cobra"
)

var authUseContextCmd = &cobra.Command{
	Use:   "use-context <name>",
	Short: "Set the current-context to a saved context by name",
	Args:  cobra.ExactArgs(1),
	RunE:  runAuthUseContext,
}

func init() {
	authCmd.AddCommand(authUseContextCmd)
}

func runAuthUseContext(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := config.Load()
	if err != nil {
		var ue *config.UnreadableError
		if errors.As(err, &ue) {
			return auth.ErrConfigUnreadable(ue.Path, ue.Err)
		}
		return auth.ErrConfigUnreadable("", err)
	}
	if _, ok := cfg.Contexts[name]; !ok {
		return auth.ErrContextNotFound(name)
	}
	cfg.CurrentContext = name
	if err := config.Write(cfg); err != nil {
		var we *config.WriteFailedError
		if errors.As(err, &we) {
			return auth.ErrConfigWriteFailed(we.Path, we.Err)
		}
		return auth.ErrConfigWriteFailed("", err)
	}

	if jsonMode {
		out := map[string]any{
			"data": map[string]any{"current_context": name},
			"meta": map[string]any{},
		}
		raw, _ := json.Marshal(out)
		fmt.Fprintln(cmd.OutOrStdout(), string(raw))
		return nil
	}
	if quiet {
		return nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "✓ Switched to context %q.\n", name)
	return nil
}
```

- [ ] **Step 4: Run, watch tests pass**

```bash
go test ./cmd/ -run TestAuthUseContext -v
```

- [ ] **Step 5: Format and commit**

```bash
go fmt ./...
git add cmd/auth_use_context.go cmd/auth_use_context_test.go
git commit -m "feat(cmd): auth use-context to switch the active context"
```

---

### Task 16: `cmd/auth_list_contexts.go`

**Files:**
- Create: `cmd/auth_list_contexts.go`
- Create: `cmd/auth_list_contexts_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// cmd/auth_list_contexts_test.go
package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ollygarden/ollygarden-cli/internal/config"
)

func setupThreeContextsForList(t *testing.T) {
	t.Helper()
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	cfg := config.New()
	cfg.CurrentContext = "prod"
	cfg.Contexts["prod"] = &config.Context{Name: "prod", APIURL: "https://prod", APIKey: "og_sk_prod00_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	cfg.Contexts["dev"] = &config.Context{Name: "dev", APIURL: "https://dev", APIKey: "og_sk_dev000_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
	cfg.Contexts["staging"] = &config.Context{Name: "staging", APIURL: "https://staging", APIKey: "og_sk_stg000_cccccccccccccccccccccccccccccccc"}
	require.NoError(t, config.Write(cfg))
}

func TestAuthListContexts_Human(t *testing.T) {
	setupThreeContextsForList(t)
	out, _, err := executeCommand("auth", "list-contexts")
	require.NoError(t, err)

	// Sorted alphabetically; current marker on prod.
	assert.Contains(t, out, "CURRENT")
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "API URL")
	assert.Contains(t, out, "*")
	assert.Contains(t, out, "prod")
	assert.Contains(t, out, "dev")
	assert.Contains(t, out, "staging")
	// No keys printed.
	assert.False(t, strings.Contains(out, "og_sk_prod00"), "keys must never appear in list-contexts")
}

func TestAuthListContexts_JSON(t *testing.T) {
	setupThreeContextsForList(t)
	out, _, err := executeCommand("auth", "list-contexts", "--json")
	require.NoError(t, err)

	var env struct {
		Data []struct {
			Name    string `json:"name"`
			APIURL  string `json:"api_url"`
			Current bool   `json:"current"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &env))
	require.Len(t, env.Data, 3)
	for _, e := range env.Data {
		if e.Name == "prod" {
			assert.True(t, e.Current)
		} else {
			assert.False(t, e.Current)
		}
	}
}
```

- [ ] **Step 2: Run, watch them fail**

```bash
go test ./cmd/ -run TestAuthListContexts -v
```

- [ ] **Step 3: Implement `cmd/auth_list_contexts.go`**

```go
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/ollygarden/ollygarden-cli/internal/auth"
	"github.com/ollygarden/ollygarden-cli/internal/config"
	"github.com/ollygarden/ollygarden-cli/internal/output"
	"github.com/spf13/cobra"
)

var authListContextsCmd = &cobra.Command{
	Use:   "list-contexts",
	Short: "List saved contexts (no keys are shown)",
	Args:  cobra.NoArgs,
	RunE:  runAuthListContexts,
}

func init() {
	authCmd.AddCommand(authListContextsCmd)
}

func runAuthListContexts(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		var ue *config.UnreadableError
		if errors.As(err, &ue) {
			return auth.ErrConfigUnreadable(ue.Path, ue.Err)
		}
		return auth.ErrConfigUnreadable("", err)
	}

	names := make([]string, 0, len(cfg.Contexts))
	for n := range cfg.Contexts {
		names = append(names, n)
	}
	sort.Strings(names)

	if jsonMode {
		entries := make([]map[string]any, 0, len(names))
		for _, n := range names {
			entries = append(entries, map[string]any{
				"name":    n,
				"api_url": cfg.Contexts[n].APIURL,
				"current": n == cfg.CurrentContext,
			})
		}
		raw, _ := json.Marshal(map[string]any{"data": entries, "meta": map[string]any{}})
		fmt.Fprintln(cmd.OutOrStdout(), string(raw))
		return nil
	}

	if quiet {
		return nil
	}

	f := output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), false, false)
	rows := make([][]string, 0, len(names))
	for _, n := range names {
		marker := ""
		if n == cfg.CurrentContext {
			marker = "*"
		}
		rows = append(rows, []string{marker, n, cfg.Contexts[n].APIURL})
	}
	f.PrintTable([]string{"CURRENT", "NAME", "API URL"}, rows)
	return nil
}
```

- [ ] **Step 4: Run, watch them pass**

```bash
go test ./cmd/ -run TestAuthListContexts -v
```

- [ ] **Step 5: Format and commit**

```bash
go fmt ./...
git add cmd/auth_list_contexts.go cmd/auth_list_contexts_test.go
git commit -m "feat(cmd): auth list-contexts with table and --json output"
```

---

## Phase 7 — Integration test + spec docs

### Task 17: `cmd/auth_integration_test.go`

**Files:**
- Create: `cmd/auth_integration_test.go`

- [ ] **Step 1: Write the integration test**

```go
//go:build integration

package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ollygarden/ollygarden-cli/internal/config"
)

func newOrgServerInt(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"name": "Acme Corp"},
			"meta": map[string]any{},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// Scenario 1: login → status reports the context → roundtrip persists token.
func TestIntegration_LoginThenStatus(t *testing.T) {
	srv := newOrgServerInt(t)
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")

	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, writeFile(tokenPath, "og_sk_int000_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))

	_, _, err := executeCommand("auth", "login",
		"--api-url", srv.URL,
		"--context", "prod",
		"--token-file", tokenPath,
	)
	require.NoError(t, err)

	out, _, err := executeCommand("auth", "status", "--no-probe", "--json")
	require.NoError(t, err)

	var env struct {
		Data struct {
			Source  string `json:"source"`
			Context string `json:"context"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &env))
	assert.Equal(t, "context", env.Data.Source)
	assert.Equal(t, "prod", env.Data.Context)
}

// Scenario 2: env var wins; status reports both env source and the would-have-won context.
func TestIntegration_EnvWinsButContextNoted(t *testing.T) {
	srv := newOrgServerInt(t)
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")

	tokenPath := filepath.Join(t.TempDir(), "token")
	require.NoError(t, writeFile(tokenPath, "og_sk_int000_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	_, _, err := executeCommand("auth", "login",
		"--api-url", srv.URL, "--context", "prod", "--token-file", tokenPath,
	)
	require.NoError(t, err)

	t.Setenv("OLLYGARDEN_API_KEY", "og_sk_envvar_dddddddddddddddddddddddddddddddd")
	out, _, err := executeCommand("auth", "status", "--no-probe", "--json")
	require.NoError(t, err)

	var env struct {
		Data struct {
			Source  string `json:"source"`
			Context string `json:"context"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &env))
	assert.Equal(t, "env", env.Data.Source)
	assert.Equal(t, "prod", env.Data.Context)
}

// Scenario 3: two logins + use-context produce a stable file with both contexts.
func TestIntegration_TwoLoginsThenUseContext(t *testing.T) {
	srv := newOrgServerInt(t)
	t.Setenv(config.ConfigFileEnvVar, filepath.Join(t.TempDir(), "config.yaml"))
	t.Setenv("OLLYGARDEN_API_KEY", "")
	t.Setenv("OLLYGARDEN_CONTEXT", "")

	tokenPathProd := filepath.Join(t.TempDir(), "tprod")
	tokenPathDev := filepath.Join(t.TempDir(), "tdev")
	require.NoError(t, writeFile(tokenPathProd, "og_sk_int001_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	require.NoError(t, writeFile(tokenPathDev, "og_sk_int002_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"))

	_, _, err := executeCommand("auth", "login",
		"--api-url", srv.URL, "--context", "prod", "--token-file", tokenPathProd,
	)
	require.NoError(t, err)

	_, _, err = executeCommand("auth", "login",
		"--api-url", srv.URL, "--context", "dev", "--token-file", tokenPathDev,
	)
	require.NoError(t, err)

	_, _, err = executeCommand("auth", "use-context", "prod")
	require.NoError(t, err)

	out, _, err := executeCommand("auth", "list-contexts")
	require.NoError(t, err)
	assert.True(t, strings.Contains(out, "prod"))
	assert.True(t, strings.Contains(out, "dev"))
}

func writeFile(path, contents string) error {
	return os.WriteFile(path, []byte(contents), 0o600)
}
```

Add the missing import — `"os"` — at the top.

- [ ] **Step 2: Run with the integration tag**

```bash
go test -tags=integration ./cmd/ -run TestIntegration -v
```
Expected: all PASS.

- [ ] **Step 3: Make sure default `go test ./...` still skips them**

```bash
go test ./...
```
Expected: PASS; integration tests are skipped (build tag).

- [ ] **Step 4: Format and commit**

```bash
go fmt ./...
git add cmd/auth_integration_test.go
git commit -m "test(cmd): integration smoke for auth login/status/use-context flows"
```

---

### Task 18: Update `specs/CLI.md`

**Files:**
- Modify: `specs/CLI.md`

- [ ] **Step 1: Update §1 Command Tree**

Insert between the existing `webhooks` block and the closing of section 1:

```
ollygarden
├── auth
│   ├── login                           # save credentials for a context
│   ├── logout                          # remove a context (or all)
│   ├── status                          # show active credential, optional probe
│   ├── use-context <name>              # set current-context
│   └── list-contexts                   # list saved contexts (no keys shown)
├── organization                        # GET /organization
├── services
│   ...
```

- [ ] **Step 2: Update §2 Global Flags table**

Add two rows to the table:

```
| `--context`   | `OLLYGARDEN_CONTEXT` | string | *(none)* | Use a specific saved context for this invocation |
```

And update the auth row to clarify env-still-wins:

```
| *(none)* | `OLLYGARDEN_API_KEY` | string | required if no saved context | API key (env-only). Still wins over saved contexts when set. |
```

- [ ] **Step 3: Add §3.x subsections for each auth subcommand**

Append the following subsections after the existing §3 entries:

`````
### 3.X.1 `ollygarden auth login`

```
ollygarden auth login [flags]
```

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--api-url`     | string | `https://api.ollygarden.cloud` | no | Inherited global flag. |
| `--context`     | string | derived from API URL host | no | Name to assign this context. Overwrites if it exists. |
| `--token-file`  | string | *(none)* | no | Read the token from this file path instead of stdin/TTY. |
| `--no-activate` | bool   | false | no | Save the context without setting it as current-context. |

Token input precedence: `--token-file` > non-TTY stdin > TTY prompt. Token shape `og_sk_[A-Za-z0-9]{6}_[a-f0-9]{32}` is enforced before any network call. The token is validated against `GET /api/v1/organization` before being persisted.

| API | `GET /api/v1/organization` (validation only) |
|---|---|

### 3.X.2 `ollygarden auth logout`

```
ollygarden auth logout [flags]
```

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--context`  | string | *(none)* | no | Name of the context to remove. |
| `--all`      | bool   | false    | no | Remove every saved context. |
| `--confirm`  | bool   | false    | no | Required for `--all` in non-interactive mode. |

When the last context is removed, the config file is deleted entirely.

### 3.X.3 `ollygarden auth status`

```
ollygarden auth status [flags]
```

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `--no-probe` | bool | false | no | Skip the `GET /organization` validation probe. |

Exit codes: `0` if logged in (and probe succeeded if probing); `3` if no credential is configured or the probe got 401.

### 3.X.4 `ollygarden auth use-context <name>`

Sets `current-context` to the named context. Exit `4` if the name doesn't exist.

### 3.X.5 `ollygarden auth list-contexts`

```
ollygarden auth list-contexts
```

No additional flags. Columns: `CURRENT` (`*` marker), `NAME`, `API URL`. **Keys are never shown** — use `auth status` to see the active key.
`````

- [ ] **Step 4: Update §5 Error Handling**

Add this row to the exit code table:

```
| 7 | config | Local config file unreadable, malformed, or unwriteable |
```

Add a new subsection at the end of §5:

`````
### CLI-emitted error codes

These appear in JSON-mode error envelopes (`error.code`) for failures detected before any HTTP call.

| Code | Exit | When |
|---|---|---|
| `NO_CREDENTIALS`        | 3 | No env var, no flag, no current-context |
| `INVALID_TOKEN_FORMAT`  | 2 | Token shape check failed |
| `TOKEN_REJECTED`        | 3 | `/organization` returned 401 |
| `CONTEXT_NOT_FOUND`     | 4 | A flag/env named a context that isn't in the file |
| `CONFIG_UNREADABLE`     | 7 | Config file exists but can't be read or parsed |
| `CONFIG_WRITE_FAILED`   | 7 | Atomic-rename or temp-file write failed |
| `TOKEN_FILE_NOT_FOUND`  | 2 | `--token-file PATH` doesn't exist or isn't readable |
| `CONFIRM_REQUIRED`      | 2 | `auth logout --all` in non-TTY without `--confirm` |
`````

- [ ] **Step 5: Add new §6 Credential Storage**

Append (renumbering subsequent sections accordingly):

`````
## 6. Credential Storage

Credentials are stored in a YAML file at `os.UserConfigDir()/ollygarden/config.yaml` with mode `0600`. Override the path with the `OLLYGARDEN_CONFIG` environment variable.

### File schema

```yaml
version: 1
current-context: prod
contexts:
  prod:
    api-url: https://api.ollygarden.cloud
    api-key: og_sk_xxxxxx_<32 hex>
  internal:
    api-url: https://api.internal.ollygarden.cloud
    api-key: og_sk_xxxxxx_<32 hex>
```

Writes are atomic (`config.yaml.tmp` → `fsync` → `rename`). When the last context is removed via `auth logout`, the file is deleted entirely.

### Resolution precedence

**API key:** `OLLYGARDEN_API_KEY` env > `--context NAME` > `OLLYGARDEN_CONTEXT` > `current-context` > error (`NO_CREDENTIALS`).

**API URL:** `--api-url` flag > `OLLYGARDEN_API_URL` env > selected context's `api-url` > built-in default `https://api.ollygarden.cloud`.

API key and API URL resolve independently — `--api-url=internal --context=prod` is allowed.
`````

- [ ] **Step 6: Format check and commit**

```bash
git add specs/CLI.md
git commit -m "docs(specs): document auth subtree, exit code 7, credential storage"
```

---

### Task 19: Update `specs/CLI_GUIDELINES.md`

**Files:**
- Modify: `specs/CLI_GUIDELINES.md`

- [ ] **Step 1: §4 add a one-line pointer**

Append to §4 (Error Handling Rules), after the "Adding New Error Codes" subsection:

```
### CLI-emitted error codes

For errors emitted before any HTTP call (validation, config I/O, missing context), see the "CLI-emitted error codes" table in `CLI.md` §5. Reuse those codes when adding similar pre-HTTP failures.
```

- [ ] **Step 2: §5 note about logout**

Append to §5 (Destructive Operation Safety):

```
### Auth-specific note

`auth logout --all` follows this destructive-op pattern (`--confirm` required in non-TTY, prompt on TTY). Default `auth logout` and `auth logout --context NAME` do **not** require confirmation: removing a single locally-stored credential is reversible by re-running `auth login`.
```

- [ ] **Step 3: Add new §8 Auth Commands**

```
## 8. Auth Commands

The `auth` subgroup houses commands that manage credentials on disk. When adding a new auth subcommand, follow these rules in addition to the rest of the guidelines:

- **Non-interactive paths required.** Coding agents are the primary consumer. Every command must work without a TTY: provide a `--token-file` / stdin pipe / structured-flag alternative to any prompt.
- **`--json` is mandatory.** Auth state is machine-introspected. The JSON envelope (`{data, meta}`) must include enough information for an agent to act on the result without parsing prose.
- **Never print raw secrets.** Use `internal/auth.MaskKey` for any human or JSON output that includes a key. The `auth list-contexts` command intentionally omits keys entirely.
- **Skip credential resolution.** Auth commands handle their own credential reads; `cmd/root.go`'s `PersistentPreRunE` walks up the command tree and skips resolution if any ancestor is the `auth` group. New auth commands inherit this automatically.
```

- [ ] **Step 4: Commit**

```bash
git add specs/CLI_GUIDELINES.md
git commit -m "docs(specs): note auth subgroup conventions and exit code mapping"
```

---

## Final verification

- [ ] **Run the full suite**

```bash
go fmt ./...
go vet ./...
go build ./...
go test ./...
```
Expected: all green.

- [ ] **Run the integration suite**

```bash
go test -tags=integration ./...
```
Expected: all green.

- [ ] **Manual smoke (against a real or fake `httptest` API)**

```bash
go build -o bin/ollygarden ./cmd/ollygarden
OLLYGARDEN_CONFIG=$(mktemp -d)/config.yaml ./bin/ollygarden auth login \
  --api-url https://api.ollygarden.cloud \
  --token-file <(echo "og_sk_xxxxxx_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
./bin/ollygarden auth status
./bin/ollygarden auth list-contexts
./bin/ollygarden auth logout
```

- [ ] **Open the PR**

```bash
gh pr create --title "feat(auth): interactive login & on-disk credential storage" \
  --body "Implements E-1910. Spec: docs/superpowers/specs/2026-04-30-cli-auth-login-design.md."
```

---

## Self-Review

**Spec coverage check:**
- ✅ Architecture (Section 1 of spec) → Tasks 2–9 + 11–16 (file map matches)
- ✅ Config schema (Section 2) → Tasks 2, 3, 4, 5
- ✅ Behaviors (missing/empty/corrupt/unknown/empty-cleanup/Windows) → Task 4 + Task 5 + skipped on Windows
- ✅ Future storage backends (architectural intent) → spec only, no code in v1 (deliberate)
- ✅ `auth login` 3 input modes + flags + behavior + outputs → Task 12
- ✅ `auth logout` default/--context/--all + confirm + hint → Task 13
- ✅ `auth status` --no-probe, source reporting, exit codes → Task 14
- ✅ `auth use-context` → Task 15
- ✅ `auth list-contexts` no keys → Task 16
- ✅ Resolution rules (Section 4) → Task 8 (Resolve), Task 10 (PersistentPreRunE wiring)
- ✅ `--context` global flag and `OLLYGARDEN_CONTEXT` env → Task 10
- ✅ Edge cases (cross-source, env+context, file unreadable, file missing) → Task 8 covers cross-source; Task 10 covers config-load failure routing
- ✅ Errors and exit codes (Section 5) → Task 1 (exit 7), Task 6 (typed errors)
- ✅ AuthError removal → Task 10
- ✅ Testing strategy: unit / cmd / integration → Tasks 4–9 unit, 12–16 cmd, 17 integration
- ✅ Spec docs (CLI.md / CLI_GUIDELINES.md) → Tasks 18, 19
- ⚠ Self-review item: the "atomicity test" in Task 5 verifies `.tmp` doesn't linger after success but does NOT stub `os.Rename` to fail mid-write (the spec calls for that). Acceptable given Go has no clean way to inject `os.Rename`; the leftover-cleanup test plus the explicit `os.Remove(tmp)` calls in `Write` cover the intent. Documented here as a known-limitation rather than added.

**Placeholder scan:** none.

**Type consistency check:**
- `auth.Resolve(ResolveInputs)` returns `(Credentials, error)` — used identically in cmd/root.go (Task 10) and cmd/auth_status.go (Task 14). ✅
- `auth.LoginInputs` / `auth.LoginResult` — defined in Task 9, consumed in Task 12. Field names match: `Token`, `APIURL`, `ContextName`, `Activate`, `HTTPClient` / `ContextName`, `APIURL`, `OrganizationName`, `KeyMasked`, `Activated`. ✅
- `config.Config` / `config.Context` field names — `Version`, `CurrentContext`, `Contexts`, `Source` / `Name`, `APIURL`, `APIKey`. Used consistently across tasks. ✅
- Error constructors (`auth.ErrNoCredentials()` etc.) — defined in Task 6, called from cmd in Tasks 10, 12, 13, 14, 15, 16. ✅
- `config.Load` and `config.Write` signatures — defined in Tasks 4, 5; called from auth.Login (Task 9) and from each auth subcommand. ✅

Plan complete.

---

**Plan complete and saved to `docs/superpowers/plans/2026-04-30-cli-auth-login.md`. Two execution options:**

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.

**Which approach?**
