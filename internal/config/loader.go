package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

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
