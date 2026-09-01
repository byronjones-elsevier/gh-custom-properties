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
