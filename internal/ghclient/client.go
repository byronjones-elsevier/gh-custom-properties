package ghclient

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.github.com"
	apiVersion     = "2022-11-28"
)

// Client talks to the GitHub REST API for custom repository properties.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// ResolveToken finds a GitHub token to authenticate with, in precedence order:
// an explicit override (e.g. from a CLI flag), the GITHUB_TOKEN environment
// variable, then the token cached by the `gh` CLI. It returns an error naming
// all of the places it looked if none of them yield a token.
func ResolveToken(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		return tok, nil
	}
	if tok, err := ghCLIToken(); err == nil && tok != "" {
		return tok, nil
	}
	return "", fmt.Errorf("no GitHub token found: set GITHUB_TOKEN, run `gh auth login`, or pass --token")
}

func ghCLIToken() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "gh", "auth", "token").Output()
	if err != nil {
		return "", fmt.Errorf("gh auth token: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// New creates a Client authenticated with token.
func New(token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    defaultBaseURL,
		token:      token,
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string, body []byte) (*http.Request, error) {
	var reader *strings.Reader
	if body != nil {
		reader = strings.NewReader(string(body))
	} else {
		reader = strings.NewReader("")
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// apiError describes a non-2xx response from the GitHub API.
type apiError struct {
	StatusCode int
	Method     string
	Path       string
	Body       string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("%s %s: unexpected status %d: %s", e.Method, e.Path, e.StatusCode, e.Body)
}
