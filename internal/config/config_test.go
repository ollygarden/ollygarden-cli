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
