package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthCommandHelpListsSubcommands(t *testing.T) {
	out, _, err := executeCommand("auth", "--help")
	require.NoError(t, err)
	for _, sub := range []string{"login", "logout", "status", "use-context", "list-contexts"} {
		assert.True(t, strings.Contains(out, sub), "auth --help should list %q, got:\n%s", sub, out)
	}
}
