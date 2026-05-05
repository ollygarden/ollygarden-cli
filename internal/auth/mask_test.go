package auth

import "testing"

func TestMaskKey(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"og_sk_abc123_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "og_sk_abc123_••••"},
		{"og_sk_DEFGHI_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "og_sk_DEFGHI_••••"},
		{"", ""},
		{"og_sk_short", "og_sk_short"}, // no underscore-separated secret tail → leave as-is
		{"og_sk_abc123_••••", "og_sk_abc123_••••"},                                             // already masked → idempotent
		{"plain-string", "plain-string"},                                                       // doesn't match shape → leave as-is
		{"foo_bar_baz_qux", "foo_bar_baz_qux"},                                                 // wrong prefix → unchanged
		{"og_sk__aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "og_sk__aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, // empty id → unchanged
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := MaskKey(tc.in); got != tc.want {
				t.Errorf("MaskKey(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
