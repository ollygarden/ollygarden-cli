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
