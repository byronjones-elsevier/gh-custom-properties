package clidoc

import (
	"fmt"
	"strings"
	"time"
)

// manDateFormat parses Spec.Date ("2006-01-02") into the man page's date format.
func manDate(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("2 January 2006")
}

// manEscape escapes troff-significant characters in free text.
func manEscape(s string) string {
	return strings.ReplaceAll(s, "-", `\-`)
}

// RenderMan renders the spec as a troff man page (section 1).
func (s Spec) RenderMan() string {
	var b strings.Builder

	fmt.Fprintf(&b, `.TH %s 1 "%s" "%s %s" "User Commands"
.SH NAME
%s \- %s
.SH SYNOPSIS
`, strings.ToUpper(s.Name), manDate(s.Date), s.Name, s.Version, s.Name, manEscape(s.Summary))

	for _, u := range s.Usage {
		fmt.Fprintf(&b, ".B %s\n", u)
	}

	b.WriteString(".SH DESCRIPTION\n")
	if s.Note != "" {
		fmt.Fprintf(&b, "%s\n", s.Note)
	}

	b.WriteString(".SH OPTIONS\n")
	for _, f := range s.Flags {
		fmt.Fprintf(&b, ".TP\n.B %s\n%s\n", f.FlagLabel(), f.Description)
	}

	for _, sec := range s.Section {
		fmt.Fprintf(&b, ".SH %s\n", strings.ToUpper(sec.Title))
		for _, line := range strings.Split(sec.Body, "\n") {
			if line == "" {
				b.WriteString(".PP\n")
			} else {
				fmt.Fprintf(&b, "%s\n", line)
			}
		}
	}

	return b.String()
}
