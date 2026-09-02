// Package ghclient talks to the GitHub REST API for custom repository properties.
package ghclient

import (
	"fmt"
	"regexp"
	"strings"
)

// repoSpecPattern matches "owner/repo", an https(s) GitHub URL, or a git@ SSH URL,
// each optionally followed by extra path segments (e.g. /settings) or a trailing ".git".
var repoSpecPattern = regexp.MustCompile(`^(?:https?://(?:www\.)?github\.com/|git@github\.com:)?([A-Za-z0-9][A-Za-z0-9._-]*)/([A-Za-z0-9._-]+?)(?:\.git)?(?:/.*)?/?$`)

// ParseRepoSpec extracts the owner and repo name from a GitHub repo URL
// (https://github.com/owner/repo, https://github.com/owner/repo.git,
// git@github.com:owner/repo.git) or an "owner/repo" shorthand.
func ParseRepoSpec(input string) (owner, repo string, err error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", "", fmt.Errorf("repo spec is empty")
	}

	m := repoSpecPattern.FindStringSubmatch(trimmed)
	if m == nil {
		return "", "", fmt.Errorf("%q is not a recognizable GitHub repo (want owner/repo or a github.com URL)", input)
	}

	owner, repo = m[1], m[2]
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("%q is not a recognizable GitHub repo (want owner/repo or a github.com URL)", input)
	}
	return owner, repo, nil
}
