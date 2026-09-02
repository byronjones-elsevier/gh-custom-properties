// Command gh-custom-properties is a terminal UI for viewing, adding,
// editing, and deleting GitHub custom properties on a repo, or across a
// batch of repos listed in a file.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/clidoc"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/config"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/repolist"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/termkeys"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

//go:generate go run ./docs/gendocs

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

	tui.ApplyPalette(tui.Palette{
		Title:    cfg.Colors.Title,
		Header:   cfg.Colors.Header,
		Cursor:   cfg.Colors.Cursor,
		Selected: cfg.Colors.Selected,
		Dim:      cfg.Colors.Dim,
		Error:    cfg.Colors.Error,
		Success:  cfg.Colors.Success,
		Warn:     cfg.Colors.Warn,
		Help:     cfg.Colors.Help,
	})

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

	// Must happen before tea.Program.Run() starts reading stdin itself —
	// the probe briefly takes over raw-mode stdin reading on its own.
	if altEnabled, err := termkeys.EnableIfAvailable(); err == nil && altEnabled {
		defer termkeys.Disable()
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

	// Usage strings are omitted here (left "") since fs.Usage is overridden
	// with printUsage below; clidoc.GHCustomProperties is the single source
	// of truth for the descriptions shown to the user.
	fs.StringVar(&opts.filePath, "file", "", "")
	fs.StringVar(&opts.filePath, "f", "", "")
	fs.StringVar(&opts.token, "token", "", "")
	fs.StringVar(&opts.token, "t", "", "")
	fs.StringVar(&opts.backupDir, "backup-dir", "", "")
	fs.BoolVar(&help, "help", false, "")
	fs.BoolVar(&help, "h", false, "")
	fs.BoolVar(&help, "H", false, "")
	fs.BoolVar(&help, "HELP", false, "")
	fs.BoolVar(&help, "?", false, "")
	fs.BoolVar(&showVersion, "version", false, "")
	fs.BoolVar(&showVersion, "v", false, "")

	if err := fs.Parse(args); err != nil {
		return cliOptions{}, 2, true
	}
	if help {
		printUsage(os.Stdout)
		return cliOptions{}, 0, true
	}
	if showVersion {
		fmt.Println(clidoc.GHCustomProperties.Name, clidoc.GHCustomProperties.Version)
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
	_, _ = fmt.Fprint(w, clidoc.GHCustomProperties.RenderText())
}
