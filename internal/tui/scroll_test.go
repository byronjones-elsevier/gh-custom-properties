package tui

import (
	"strings"
	"testing"
)

func TestVisibleWindow(t *testing.T) {
	tests := []struct {
		name       string
		n, cursor  int
		maxVisible int
		wantStart  int
		wantEnd    int
	}{
		{name: "unbounded shows everything", n: 40, cursor: 20, maxVisible: 0, wantStart: 0, wantEnd: 40},
		{name: "fits within max shows everything", n: 5, cursor: 2, maxVisible: 10, wantStart: 0, wantEnd: 5},
		{name: "cursor near start", n: 40, cursor: 0, maxVisible: 10, wantStart: 0, wantEnd: 10},
		{name: "cursor near end", n: 40, cursor: 39, maxVisible: 10, wantStart: 30, wantEnd: 40},
		{name: "cursor in middle stays centered", n: 40, cursor: 20, maxVisible: 10, wantStart: 15, wantEnd: 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := visibleWindow(tt.n, tt.cursor, tt.maxVisible)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("visibleWindow(%d, %d, %d) = %d, %d; want %d, %d",
					tt.n, tt.cursor, tt.maxVisible, start, end, tt.wantStart, tt.wantEnd)
			}
			if tt.maxVisible > 0 && (tt.cursor < start || tt.cursor >= end) {
				t.Errorf("visibleWindow(%d, %d, %d) = [%d, %d) does not contain cursor", tt.n, tt.cursor, tt.maxVisible, start, end)
			}
		})
	}
}

func TestPinFooter(t *testing.T) {
	t.Run("no footer returns content unchanged", func(t *testing.T) {
		if got := pinFooter("a\nb", "", 10); got != "a\nb" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("unknown height appends footer without padding", func(t *testing.T) {
		got := pinFooter("a\nb", "footer", 0)
		want := "a\nb\nfooter"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("pads footer to the last line at a known height", func(t *testing.T) {
		content := "a\nb"  // 2 lines
		footer := "footer" // 1 line
		got := pinFooter(content, footer, 10)
		lines := strings.Split(got, "\n")
		if len(lines) != 10 {
			t.Fatalf("got %d lines, want 10:\n%q", len(lines), got)
		}
		if lines[len(lines)-1] != "footer" {
			t.Errorf("last line = %q, want %q", lines[len(lines)-1], "footer")
		}
	})

	t.Run("content taller than height still appends footer immediately", func(t *testing.T) {
		content := strings.Repeat("x\n", 20) + "x" // 21 lines
		got := pinFooter(content, "footer", 5)
		lines := strings.Split(got, "\n")
		if lines[len(lines)-1] != "footer" {
			t.Errorf("last line = %q, want %q", lines[len(lines)-1], "footer")
		}
	})
}

func TestAvailableRows(t *testing.T) {
	if got := availableRows(0); got != 0 {
		t.Errorf("availableRows(0) = %d, want 0 (unbounded)", got)
	}
	if got := availableRows(-5); got != 0 {
		t.Errorf("availableRows(-5) = %d, want 0 (unbounded)", got)
	}
	if got := availableRows(24); got <= 0 {
		t.Errorf("availableRows(24) = %d, want a positive row count", got)
	}
	if got := availableRows(5); got < 3 {
		t.Errorf("availableRows(5) = %d, want at least the floor of 3", got)
	}
}
