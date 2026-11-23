// Package browser builds github.com URLs for a repository and opens them in
// the user's default web browser.
package browser

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
)

// SettingsURL returns the web URL of a repository's Custom properties
// settings page.
func SettingsURL(owner, repo string) string {
	return fmt.Sprintf("https://github.com/%s/%s/settings/custom-properties",
		url.PathEscape(owner), url.PathEscape(repo))
}

// Open launches the platform's default browser on target without waiting for
// it to exit. It is a variable so tests can substitute a fake.
var Open = func(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	// Reap the launcher in the background so it doesn't linger as a zombie.
	go func() { _ = cmd.Wait() }()
	return nil
}
