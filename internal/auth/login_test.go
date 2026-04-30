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
