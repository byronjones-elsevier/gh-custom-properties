package repolist

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	input := strings.Join([]string{
		"# a comment",
		"",
		"octocat/hello-world",
		"https://github.com/octocat/other-repo",
		"   ",
		"not a valid repo spec at all !!",
		"another/repo # trailing comment is not stripped, treated as part of the line",
	}, "\n")

	res, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(res.Entries) != 2 {
		t.Fatalf("got %d entries, want 2: %+v", len(res.Entries), res.Entries)
	}
	if res.Entries[0].Owner != "octocat" || res.Entries[0].Repo != "hello-world" || res.Entries[0].Line != 3 {
		t.Errorf("entries[0] = %+v", res.Entries[0])
	}
	if res.Entries[1].Owner != "octocat" || res.Entries[1].Repo != "other-repo" || res.Entries[1].Line != 4 {
		t.Errorf("entries[1] = %+v", res.Entries[1])
	}

	if len(res.Skipped) != 2 {
		t.Fatalf("got %d skipped, want 2: %+v", len(res.Skipped), res.Skipped)
	}
	if res.Skipped[0].Line != 6 {
		t.Errorf("skipped[0].Line = %d, want 6", res.Skipped[0].Line)
	}
}

func TestParse_Empty(t *testing.T) {
	res, err := Parse(strings.NewReader(""))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(res.Entries) != 0 || len(res.Skipped) != 0 {
		t.Errorf("expected empty result, got %+v", res)
	}
}

func TestLoadFile_MissingFile(t *testing.T) {
	if _, err := LoadFile("/nonexistent/path/repos.txt"); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}
