package exitcode

import "testing"

func TestExitCodeConfigIsSeven(t *testing.T) {
	if Config != 7 {
		t.Fatalf("Config exit code: want 7, got %d", Config)
	}
}
