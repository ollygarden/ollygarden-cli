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
