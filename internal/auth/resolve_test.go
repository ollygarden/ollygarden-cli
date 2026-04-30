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
