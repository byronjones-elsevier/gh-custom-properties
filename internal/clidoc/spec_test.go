package clidoc

import (
	"strings"
	"testing"
)

func TestRenderText_ContainsFlagsAndSections(t *testing.T) {
	out := GHCustomProperties.RenderText()
	for _, want := range []string{"-f, --file", "-h, -H", "AUTHENTICATION:", "BACKUPS:"} {
		if !strings.Contains(out, want) {
			t.Errorf("RenderText() missing %q\n---\n%s", want, out)
		}
	}
}

func TestRenderMan_WellFormed(t *testing.T) {
	out := GHCustomProperties.RenderMan()
	for _, want := range []string{".TH GH-CUSTOM-PROPERTIES 1", ".SH NAME", ".SH OPTIONS", ".SH AUTHENTICATION"} {
		if !strings.Contains(out, want) {
			t.Errorf("RenderMan() missing %q\n---\n%s", want, out)
		}
	}
}

func TestRenderHTML_Valid(t *testing.T) {
	out, err := GHCustomProperties.RenderHTML()
	if err != nil {
		t.Fatalf("RenderHTML: %v", err)
	}
	for _, want := range []string{"<title>gh-custom-properties", "<h2>Flags</h2>", "&lt;path&gt;"} {
		if !strings.Contains(out, want) {
			t.Errorf("RenderHTML() missing %q", want)
		}
	}
}

func TestFlagLabel_ShortNamesFirst(t *testing.T) {
	f := Flag{Names: []string{"help", "h", "H", "HELP", "?"}}
	got := f.FlagLabel()
	want := "-h, -H, -?, --help, --HELP"
	if got != want {
		t.Errorf("FlagLabel() = %q, want %q", got, want)
	}
}
