package termkeys

import "testing"

// TestEnableIfAvailable_NonTerminal confirms the probe degrades safely (no
// error, not enabled) when stdout isn't a terminal, which is always true
// under `go test` — exercising the real terminal-detection path end to end
// isn't practical in an automated test.
func TestEnableIfAvailable_NonTerminal(t *testing.T) {
	enabled, err := EnableIfAvailable()
	if err != nil {
		t.Fatalf("EnableIfAvailable: %v", err)
	}
	if enabled {
		t.Error("EnableIfAvailable() = true under go test, where stdout is not a terminal")
	}
}
