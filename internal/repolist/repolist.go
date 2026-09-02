// Package repolist parses the --file argument for batch mode: a plain text
// file listing one GitHub repo per line.
package repolist

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
)

// Entry is one successfully parsed repo from the list.
type Entry struct {
	Owner string
	Repo  string
	Line  int // 1-based source line number, for error reporting
}

// Skipped is one line that could not be parsed as a repo.
type Skipped struct {
	Line int
	Text string
	Err  error
}

// Result is the outcome of parsing a repo-list file: the repos that parsed
// successfully, and any lines that were skipped with the reason why.
type Result struct {
	Entries []Entry
	Skipped []Skipped
}

// LoadFile reads and parses the repo list at path.
func LoadFile(path string) (Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return Result{}, fmt.Errorf("open repo list %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return Parse(f)
}

// Parse reads one repo (owner/repo or a GitHub URL) per line from r. Blank
// lines and lines starting with '#' are ignored. Lines that fail to parse
// are collected in Result.Skipped rather than aborting the whole file.
func Parse(r io.Reader) (Result, error) {
	var res Result
	scanner := bufio.NewScanner(r)
	for lineNo := 1; scanner.Scan(); lineNo++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		owner, repo, err := ghclient.ParseRepoSpec(text)
		if err != nil {
			res.Skipped = append(res.Skipped, Skipped{Line: lineNo, Text: text, Err: err})
			continue
		}
		res.Entries = append(res.Entries, Entry{Owner: owner, Repo: repo, Line: lineNo})
	}
	if err := scanner.Err(); err != nil {
		return Result{}, fmt.Errorf("read repo list: %w", err)
	}
	return res, nil
}
