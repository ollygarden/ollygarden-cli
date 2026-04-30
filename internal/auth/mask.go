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
