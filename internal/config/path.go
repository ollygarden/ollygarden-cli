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
