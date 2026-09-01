// Command gh-custom-properties is a terminal UI for viewing, adding,
// editing, and deleting GitHub custom properties on a repo, or across a
// batch of repos listed in a file.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/config"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/repolist"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

//go:generate go run ./docs/gendocs

const version = "0.1.0"

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	opts, code, done := parseArgs(args)
	if done {
		return code
	}

	if opts.filePath != "" && opts.repoArg != "" {
		fmt.Fprintln(os.Stderr, "error: --file and a repo argument are mutually exclusive")
		return 1
	}

	cfg, err := config.Load(config.EnvOverrides(), config.Overrides{
		BackupDir: opts.backupDir,
		Token:     opts.token,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	token, err := ghclient.ResolveToken(cfg.Token)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	api := ghclient.New(token)

	var model tea.Model
	if opts.filePath != "" {
		result, err := repolist.LoadFile(opts.filePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
		if len(result.Entries) == 0 {
			fmt.Fprintln(os.Stderr, "error: no valid repos found in", opts.filePath)
			for _, s := range result.Skipped {
				fmt.Fprintf(os.Stderr, "  line %d: %q: %v\n", s.Line, s.Text, s.Err)
			}
			return 1
		}
		model = tui.NewBatch(api, cfg.BackupDir, result.Entries, result.Skipped)
	} else {
		owner, repo := "", ""
		if opts.repoArg != "" {
			owner, repo, err = ghclient.ParseRepoSpec(opts.repoArg)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				return 1
			}
		}
		model = tui.NewSingleRepo(api, cfg.BackupDir, owner, repo)
	}

	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}

type cliOptions struct {
	filePath  string
	token     string
	backupDir string
	repoArg   string
}

// parseArgs parses args and returns the resolved options. done is true when
// main should exit immediately (help/version/usage error was printed);
// code is the process exit code to use in that case.
func parseArgs(args []string) (cliOptions, int, bool) {
	fs := flag.NewFlagSet("gh-custom-properties", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = func() { printUsage(os.Stdout) }

	var opts cliOptions
	var help, showVersion bool

	fs.StringVar(&opts.filePath, "file", "", "batch mode: file listing one repo (owner/repo or URL) per line")
	fs.StringVar(&opts.filePath, "f", "", "shorthand for --file")
	fs.StringVar(&opts.token, "token", "", "GitHub token, overriding GITHUB_TOKEN / `gh auth token`")
	fs.StringVar(&opts.token, "t", "", "shorthand for --token")
	fs.StringVar(&opts.backupDir, "backup-dir", "", "directory to write timestamped backups to (default $HOME/.gh-custom-properties/backups)")
	fs.BoolVar(&help, "help", false, "show help")
	fs.BoolVar(&help, "h", false, "shorthand for --help")
	fs.BoolVar(&help, "H", false, "shorthand for --help")
	fs.BoolVar(&help, "HELP", false, "shorthand for --help")
	fs.BoolVar(&help, "?", false, "shorthand for --help")
	fs.BoolVar(&showVersion, "version", false, "show version")
	fs.BoolVar(&showVersion, "v", false, "shorthand for --version")

	if err := fs.Parse(args); err != nil {
		return cliOptions{}, 2, true
	}
	if help {
		printUsage(os.Stdout)
		return cliOptions{}, 0, true
	}
	if showVersion {
		fmt.Println("gh-custom-properties", version)
		return cliOptions{}, 0, true
	}

	rest := fs.Args()
	if len(rest) > 1 {
		fmt.Fprintln(os.Stderr, "error: expected at most one repo argument, got", len(rest))
		return cliOptions{}, 1, true
	}
	if len(rest) == 1 {
		opts.repoArg = rest[0]
	}
	return opts, 0, false
}

func printUsage(w *os.File) {
	fmt.Fprint(w, `gh-custom-properties - view, add, edit, and delete GitHub custom repo properties

USAGE:
  gh-custom-properties [flags] [owner/repo | repo-url]
  gh-custom-properties --file repos.txt [flags]

If no repo argument or --file is given, the TUI prompts for a repo to load.

FLAGS:
  -f, --file <path>        batch mode: file listing one repo per line
  -t, --token <token>      GitHub token (overrides GITHUB_TOKEN / gh auth token)
      --backup-dir <path>  backup directory (default $HOME/.gh-custom-properties/backups)
  -h, -H, --help, --HELP, -?   show this help
  -v, --version             show version

AUTHENTICATION:
  A GitHub token is required, resolved in this order:
    1. --token
    2. GITHUB_TOKEN environment variable
    3. `+"`gh auth token`"+` (if the gh CLI is installed and logged in)

  The token needs permission to read and write custom properties on the
  target repos, and to read the org's custom-property schema.

REPO LIST FILE FORMAT (--file):
  One repo per line, as "owner/repo" or a github.com URL. Blank lines and
  lines starting with '#' are ignored.

BACKUPS:
  Before any change is applied, a timestamped JSON snapshot of the prior
  property values is written to the backup directory.
`)
}
