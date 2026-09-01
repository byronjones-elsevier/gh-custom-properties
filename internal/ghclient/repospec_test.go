package ghclient

import "testing"

func TestParseRepoSpec(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{name: "shorthand", input: "octocat/hello-world", wantOwner: "octocat", wantRepo: "hello-world"},
		{name: "https url", input: "https://github.com/octocat/hello-world", wantOwner: "octocat", wantRepo: "hello-world"},
		{name: "https url with .git suffix", input: "https://github.com/octocat/hello-world.git", wantOwner: "octocat", wantRepo: "hello-world"},
		{name: "https url trailing slash", input: "https://github.com/octocat/hello-world/", wantOwner: "octocat", wantRepo: "hello-world"},
		{name: "https url with extra path", input: "https://github.com/octocat/hello-world/settings/access", wantOwner: "octocat", wantRepo: "hello-world"},
		{name: "http url", input: "http://github.com/octocat/hello-world", wantOwner: "octocat", wantRepo: "hello-world"},
		{name: "ssh url", input: "git@github.com:octocat/hello-world.git", wantOwner: "octocat", wantRepo: "hello-world"},
		{name: "repo name with dot", input: "octocat/my.repo", wantOwner: "octocat", wantRepo: "my.repo"},
		{name: "whitespace padded", input: "  octocat/hello-world  ", wantOwner: "octocat", wantRepo: "hello-world"},
		{name: "empty", input: "", wantErr: true},
		{name: "missing repo", input: "octocat", wantErr: true},
		{name: "not github", input: "https://gitlab.com/octocat/hello-world", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ParseRepoSpec(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseRepoSpec(%q) = %q, %q, nil; want error", tt.input, owner, repo)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRepoSpec(%q) returned unexpected error: %v", tt.input, err)
			}
			if owner != tt.wantOwner || repo != tt.wantRepo {
				t.Errorf("ParseRepoSpec(%q) = %q, %q; want %q, %q", tt.input, owner, repo, tt.wantOwner, tt.wantRepo)
			}
		})
	}
}
