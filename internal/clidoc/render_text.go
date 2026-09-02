package clidoc

import (
	"fmt"
	"sort"
	"strings"
)

// FlagLabel renders a flag's aliases as "-f, --file" style, shortest-first,
// single-dash for one-character names and double-dash otherwise.
func (f Flag) FlagLabel() string {
	names := append([]string{}, f.Names...)
	sort.SliceStable(names, func(i, j int) bool { return len(names[i]) < len(names[j]) })

	parts := make([]string, len(names))
	for i, n := range names {
		dash := "--"
		if len(n) == 1 {
			dash = "-"
		}
		parts[i] = dash + n
	}
	label := strings.Join(parts, ", ")
	if f.ValuePlaceholder != "" {
		label += " <" + f.ValuePlaceholder + ">"
	}
	return label
}

// RenderText renders the spec as the plain-text help shown by -h/--help.
func (s Spec) RenderText() string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s - %s\n\n", s.Name, s.Summary)

	b.WriteString("USAGE:\n")
	for _, u := range s.Usage {
		fmt.Fprintf(&b, "  %s\n", u)
	}
	if s.Note != "" {
		fmt.Fprintf(&b, "\n%s\n", s.Note)
	}

	b.WriteString("\nFLAGS:\n")
	labels := make([]string, len(s.Flags))
	for i, f := range s.Flags {
		labels[i] = f.FlagLabel()
	}
	width := 0
	for _, l := range labels {
		if len(l) > width {
			width = len(l)
		}
	}
	for i, f := range s.Flags {
		fmt.Fprintf(&b, "  %-*s  %s\n", width, labels[i], f.Description)
	}

	for _, sec := range s.Section {
		fmt.Fprintf(&b, "\n%s:\n", sec.Title)
		for _, line := range strings.Split(sec.Body, "\n") {
			if line == "" {
				b.WriteString("\n")
			} else {
				fmt.Fprintf(&b, "  %s\n", line)
			}
		}
	}

	return b.String()
}
