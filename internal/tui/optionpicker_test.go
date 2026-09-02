package tui

import "testing"

func TestOptionPicker_PageUpDown(t *testing.T) {
	options := make([]string, 40)
	for i := range options {
		options[i] = string(rune('a' + i%26))
	}
	p := newOptionPicker(options, false)
	p.maxVisible = 10

	p.pageDown()
	if p.cursor != 10 {
		t.Fatalf("cursor after one pageDown = %d, want 10", p.cursor)
	}
	p.pageDown()
	p.pageDown()
	p.pageDown()
	p.pageDown()
	if p.cursor != len(options)-1 {
		t.Errorf("cursor after paging past the end = %d, want clamped to %d", p.cursor, len(options)-1)
	}

	p.pageUp()
	if p.cursor != len(options)-1-10 {
		t.Errorf("cursor after one pageUp = %d, want %d", p.cursor, len(options)-1-10)
	}

	for i := 0; i < 10; i++ {
		p.pageUp()
	}
	if p.cursor != 0 {
		t.Errorf("cursor after paging past the start = %d, want clamped to 0", p.cursor)
	}
}

func TestOptionPicker_Filter(t *testing.T) {
	p := newOptionPicker([]string{"Alpha", "Beta", "Gamma", "Delta"}, false)

	p.filter = "ta"
	got := p.visibleIndices()
	if len(got) != 2 || got[0] != 1 || got[1] != 3 { // "Beta", "Delta"
		t.Fatalf("visibleIndices() = %v, want [1 3] (Beta, Delta)", got)
	}
	if p.selectedValue() != "Beta" {
		t.Errorf("selectedValue() = %q, want %q (cursor stays 0 into the filtered view)", p.selectedValue(), "Beta")
	}

	p.down()
	if p.selectedValue() != "Delta" {
		t.Errorf("after down(): selectedValue() = %q, want %q", p.selectedValue(), "Delta")
	}
	if p.cursor >= len(p.visibleIndices()) {
		t.Fatal("down() should not move past the filtered view's end")
	}
	p.down() // already at the last filtered item; should not move further
	if p.selectedValue() != "Delta" {
		t.Errorf("down() past the end = %q, want to stay on %q", p.selectedValue(), "Delta")
	}

	p.clearFilter()
	if p.filter != "" || p.filtering {
		t.Error("clearFilter() should reset both filter and filtering")
	}
	if len(p.visibleIndices()) != 4 {
		t.Errorf("visibleIndices() after clear = %d, want all 4", len(p.visibleIndices()))
	}
}

func TestOptionPicker_ToggleSurvivesFilterChange(t *testing.T) {
	p := newOptionPicker([]string{"Alpha", "Beta", "Gamma"}, true)
	p.filter = "beta"
	p.toggle() // toggles whatever's at cursor 0 in the filtered view: "Beta"

	p.clearFilter()
	if !p.checked[1] { // raw index of "Beta"
		t.Error("toggling while filtered should check the underlying option, surviving a filter clear")
	}
	values := p.selectedValues()
	if len(values) != 1 || values[0] != "Beta" {
		t.Errorf("selectedValues() = %v, want [Beta]", values)
	}
}

func TestOptionPicker_PageSizeFallsBackWhenUnbounded(t *testing.T) {
	options := make([]string, 40)
	for i := range options {
		options[i] = "x"
	}
	p := newOptionPicker(options, false) // maxVisible left at 0 (unbounded)

	p.pageDown()
	if p.cursor != defaultPageSize {
		t.Errorf("cursor = %d, want defaultPageSize (%d)", p.cursor, defaultPageSize)
	}
}
