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
	SourceEnv            // from OLLYGARDEN_API_KEY
	SourceContext        // from a context in the config file
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
//	API key:  OLLYGARDEN_API_KEY > flag-context > env-context > current-context > error
//	API URL:  --api-url flag > OLLYGARDEN_API_URL > selected context's api-url > default
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
