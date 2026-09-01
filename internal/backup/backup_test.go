package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
)

func TestWrite_SingleRepo(t *testing.T) {
	dir := t.TempDir()
	repos := []RepoSnapshot{
		{Owner: "octocat", Repo: "hello-world", PropertiesBefore: []ghclient.PropertyValue{{Name: "team", Value: "platform"}}},
	}

	path, err := Write(dir, ModeSingle, repos)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	wantPattern := regexp.MustCompile(`^octocat-hello-world-\d{8}-\d{6}\.json$`)
	if base := filepath.Base(path); !wantPattern.MatchString(base) {
		t.Errorf("filename %q does not match expected pattern", base)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read backup file: %v", err)
	}
	var got Snapshot
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal backup file: %v", err)
	}
	if got.Mode != ModeSingle || len(got.Repos) != 1 || got.Repos[0].Owner != "octocat" {
		t.Errorf("unexpected snapshot contents: %+v", got)
	}
}

func TestWrite_Batch(t *testing.T) {
	dir := t.TempDir()
	repos := []RepoSnapshot{
		{Owner: "octocat", Repo: "a"},
		{Owner: "octocat", Repo: "b"},
	}

	path, err := Write(dir, ModeBatch, repos)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	wantPattern := regexp.MustCompile(`^batch-\d{8}-\d{6}\.json$`)
	if base := filepath.Base(path); !wantPattern.MatchString(base) {
		t.Errorf("filename %q does not match expected pattern", base)
	}
}

func TestWrite_NoRepos(t *testing.T) {
	if _, err := Write(t.TempDir(), ModeSingle, nil); err == nil {
		t.Fatal("expected an error for empty repo list")
	}
}

func TestWrite_AvoidsCollision(t *testing.T) {
	dir := t.TempDir()
	repos := []RepoSnapshot{{Owner: "octocat", Repo: "hello-world"}}

	first, err := Write(dir, ModeSingle, repos)
	if err != nil {
		t.Fatalf("Write (first): %v", err)
	}
	second, err := Write(dir, ModeSingle, repos)
	if err != nil {
		t.Fatalf("Write (second): %v", err)
	}
	if first == second {
		t.Errorf("expected distinct paths, got the same path twice: %s", first)
	}
	for _, p := range []string{first, second} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist: %v", p, err)
		}
	}
}

func TestDefaultDir(t *testing.T) {
	dir, err := DefaultDir()
	if err != nil {
		t.Fatalf("DefaultDir: %v", err)
	}
	if filepath.Base(dir) != "backups" || filepath.Base(filepath.Dir(dir)) != ".gh-custom-properties" {
		t.Errorf("DefaultDir = %q, want to end in .gh-custom-properties/backups", dir)
	}
}
