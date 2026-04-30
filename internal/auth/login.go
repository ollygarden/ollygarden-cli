package auth

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
		if errors.As(err, &ue) {
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
		if errors.As(err, &we) {
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
