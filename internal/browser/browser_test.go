package browser

import "testing"

func TestSettingsURL(t *testing.T) {
	tests := []struct {
		name        string
		owner, repo string
		want        string
	}{
		{"plain", "acme", "widgets", "https://github.com/acme/widgets/settings/custom-properties"},
		{"escapes path characters", "ac me", "wid/gets", "https://github.com/ac%20me/wid%2Fgets/settings/custom-properties"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SettingsURL(tt.owner, tt.repo); got != tt.want {
				t.Errorf("SettingsURL(%q, %q) = %q, want %q", tt.owner, tt.repo, got, tt.want)
			}
		})
	}
}
