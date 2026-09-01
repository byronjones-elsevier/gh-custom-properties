package tui

import (
	"fmt"
	"strings"
)

// optionPicker is a small cursor-driven list used for single/multi-select
// value editing and for picking an org-schema property name to add.
type optionPicker struct {
	options    []string
	cursor     int // index into visibleIndices(), not options directly
	multi      bool
	checked    map[int]bool // keyed by index into options, unaffected by filtering
	maxVisible int          // 0 = show every option; otherwise scroll to keep cursor in view

	filter    string // "?" starts composing this; narrows options to a case-insensitive substring match
	filtering bool   // true while actively typing the filter (enter locks it in, esc clears it)
}

func newOptionPicker(options []string, multi bool) *optionPicker {
	return &optionPicker{options: options, multi: multi, checked: map[int]bool{}}
}

// visibleIndices returns the indices into options currently shown, in
// order: every index when there's no filter, otherwise only those matching
// it (via the shared filterIndices helper).
func (p *optionPicker) visibleIndices() []int {
	if p.filter == "" {
		out := make([]int, len(p.options))
		for i := range out {
			out[i] = i
		}
		return out
	}
	return filterIndices(p.options, p.filter)
}

func (p *optionPicker) up() {
	if p.cursor > 0 {
		p.cursor--
	}
}

func (p *optionPicker) down() {
	if p.cursor < len(p.visibleIndices())-1 {
		p.cursor++
	}
}

func (p *optionPicker) pageUp() {
	p.cursor -= pageSize(p.maxVisible)
	if p.cursor < 0 {
		p.cursor = 0
	}
}

func (p *optionPicker) pageDown() {
	p.cursor += pageSize(p.maxVisible)
	if last := len(p.visibleIndices()) - 1; p.cursor > last {
		p.cursor = last
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

// toggle flips the checked state of the option currently under the cursor
// (which is an index into the filtered view — translated to the
// corresponding index into options, since checked state must survive the
// filter being cleared or changed).
func (p *optionPicker) toggle() {
	indices := p.visibleIndices()
	if p.cursor < 0 || p.cursor >= len(indices) {
		return
	}
	idx := indices[p.cursor]
	p.checked[idx] = !p.checked[idx]
}

// startFilter begins composing a filter query; subsequent runes/backspace
// go to it instead of navigating until stopFilterTyping or clearFilter.
func (p *optionPicker) startFilter() { p.filtering = true }

// stopFilterTyping locks in the current filter text (if any) and returns to
// normal navigation — pressed via enter while actively typing. The filter
// itself, and the narrowed view it produces, remain in effect.
func (p *optionPicker) stopFilterTyping() { p.filtering = false }

// clearFilter drops the filter entirely, restoring the full option list.
func (p *optionPicker) clearFilter() {
	p.filter = ""
	p.filtering = false
	p.cursor = 0
}

func (p *optionPicker) appendFilterRune(r rune) {
	p.filter += string(r)
	p.cursor = 0
}

func (p *optionPicker) backspaceFilter() {
	if p.filter == "" {
		return
	}
	runes := []rune(p.filter)
	p.filter = string(runes[:len(runes)-1])
	p.cursor = 0
}

// preselectSingle moves the cursor to value, if present among the options.
// Only meaningful before any filter is applied (construction time), when
// visibleIndices() is the identity mapping.
func (p *optionPicker) preselectSingle(value string) {
	for i, o := range p.options {
		if o == value {
			p.cursor = i
			return
		}
	}
}

// preselectMulti checks every option present in values.
func (p *optionPicker) preselectMulti(values []string) {
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}
	for i, o := range p.options {
		if set[o] {
			p.checked[i] = true
		}
	}
}

func (p *optionPicker) selectedValue() string {
	indices := p.visibleIndices()
	if p.cursor < 0 || p.cursor >= len(indices) {
		return ""
	}
	return p.options[indices[p.cursor]]
}

func (p *optionPicker) selectedValues() []string {
	out := make([]string, 0, len(p.checked))
	for i, o := range p.options {
		if p.checked[i] {
			out = append(out, o)
		}
	}
	return out
}

func (p *optionPicker) View() string {
	indices := p.visibleIndices()
	start, end := visibleWindow(len(indices), p.cursor, p.maxVisible)

	var b strings.Builder
	if p.filtering || p.filter != "" {
		b.WriteString(dimStyle.Render("Filter: "+p.filter) + "\n")
	}
	if len(indices) == 0 {
		b.WriteString(dimStyle.Render("  (no matches)") + "\n")
	}
	if start > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ↑ %d more above", start)) + "\n")
	}
	for i := start; i < end; i++ {
		idx := indices[i]
		o := p.options[idx]
		prefix := "  "
		label := o
		if p.multi {
			if p.checked[idx] {
				label = "[x] " + o
			} else {
				label = "[ ] " + o
			}
		}
		if i == p.cursor {
			prefix = cursorStyle.Render("> ")
			label = selectedStyle.Render(label)
		}
		b.WriteString(prefix + label + "\n")
	}
	if end < len(indices) {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ↓ %d more below", len(indices)-end)) + "\n")
	}
	return b.String()
}
