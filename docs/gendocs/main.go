// Command gendocs regenerates docs/gh-custom-properties.1 (man page) and
// docs/gh-custom-properties.html (HTML help) from the CLI spec in
// internal/clidoc, so the two never drift from what -h/--help prints.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/clidoc"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	spec := clidoc.GHCustomProperties
	outDir := "docs"

	manPath := filepath.Join(outDir, spec.Name+".1")
	if err := os.WriteFile(manPath, []byte(spec.RenderMan()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", manPath, err)
	}

	html, err := spec.RenderHTML()
	if err != nil {
		return fmt.Errorf("render HTML: %w", err)
	}
	htmlPath := filepath.Join(outDir, spec.Name+".html")
	if err := os.WriteFile(htmlPath, []byte(html), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", htmlPath, err)
	}

	fmt.Println("wrote", manPath, "and", htmlPath)
	return nil
}
