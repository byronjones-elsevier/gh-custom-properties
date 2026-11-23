package main

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/browser"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/clidoc"
)

func readDoc(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}

func TestSpec_RequirementIDsAreUniqueAndReferencedFilesExist(t *testing.T) {
	spec := readDoc(t, "ENGINEERING.md")

	// Only the requirements table defines IDs; later tables refer to them.
	start := strings.Index(spec, "## 2. Requirements")
	end := strings.Index(spec, "## 3. External interfaces")
	if start < 0 || end < start {
		t.Fatal("ENGINEERING.md is missing section 2 or 3")
	}
	reqs := spec[start:end]

	idRow := regexp.MustCompile(`(?m)^\| (REQ-[A-Z]+-\d+) \|`)
	seen := map[string]bool{}
	for _, m := range idRow.FindAllStringSubmatch(reqs, -1) {
		if seen[m[1]] {
			t.Errorf("duplicate requirement ID %s", m[1])
		}
		seen[m[1]] = true
	}
	if len(seen) == 0 {
		t.Fatal("no requirement IDs found in ENGINEERING.md")
	}

	for _, path := range regexp.MustCompile("`((?:internal|agents)/[A-Za-z0-9_./-]+\\.(?:go|md))`").FindAllStringSubmatch(spec, -1) {
		if _, err := os.Stat(path[1]); err != nil {
			t.Errorf("ENGINEERING.md references %s, which does not exist", path[1])
		}
	}
}

func TestSpec_DocumentsEveryCLIFlagAndTheBrowserURL(t *testing.T) {
	spec := readDoc(t, "ENGINEERING.md")
	for _, f := range clidoc.GHCustomProperties.Flags {
		for _, n := range f.Names {
			dashes := "--"
			if len(n) == 1 {
				dashes = "-"
			}
			if n == "H" || n == "HELP" || n == "?" || n == "h" {
				continue // help aliases are covered by REQ-CLI-1's own list
			}
			if !strings.Contains(spec, "`"+dashes+n) {
				t.Errorf("ENGINEERING.md does not document flag %s%s", dashes, n)
			}
		}
	}
	want := browser.SettingsURL("<owner>", "<repo>")
	want = strings.NewReplacer("%3Cowner%3E", "<owner>", "%3Crepo%3E", "<repo>").Replace(want)
	if !strings.Contains(spec, want) {
		t.Errorf("ENGINEERING.md does not quote the settings URL %q", want)
	}
}

// Git-derived facts (commit ids, PR numbers, merge status) go stale at merge
// time, so CHANGELOG.md and HANDOFF.md must not carry them; every entry must
// carry an ISO 8601 UTC timestamp instead.
func TestLivingDocs_HoldNoGitFactsAndEntriesAreTimestamped(t *testing.T) {
	banned := []*regexp.Regexp{
		regexp.MustCompile(`\b[0-9a-f]{7,40}\b`),
		regexp.MustCompile(`#\d+`),
		regexp.MustCompile(`(?i)\bPR\s*\d+`),
		regexp.MustCompile(`(?i)\b(?:awaiting review|pending merge|merged at|merge commit)\b`),
	}
	stamp := regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`)

	for _, name := range []string{"CHANGELOG.md", "HANDOFF.md"} {
		doc := readDoc(t, name)
		if !stamp.MatchString(doc) {
			t.Errorf("%s has no ISO 8601 UTC timestamp", name)
		}
		for _, re := range banned {
			if m := re.FindString(doc); m != "" {
				t.Errorf("%s holds a git-derived fact matching %v: %q", name, re, m)
			}
		}
	}

	for _, entry := range strings.Split(readDoc(t, "CHANGELOG.md"), "\n### ")[1:] {
		if !stamp.MatchString(entry) {
			t.Errorf("CHANGELOG entry has no timestamp:\n### %s", strings.SplitN(entry, "\n", 2)[0])
		}
	}
}
