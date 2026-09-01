package tui

import (
	"fmt"
	"strings"
)

// optionPicker is a small cursor-driven list used for single/multi-select
// value editing and for picking an org-schema property name to add.
type optionPicker struct {
	options    []string
	cursor     int
	multi      bool
	checked    map[int]bool
	maxVisible int // 0 = show every option; otherwise scroll to keep cursor in view
}

func newOptionPicker(options []string, multi bool) *optionPicker {
	return &optionPicker{options: options, multi: multi, checked: map[int]bool{}}
}

func (p *optionPicker) up() {
	if p.cursor > 0 {
		p.cursor--
	}
}

func (p *optionPicker) down() {
	if p.cursor < len(p.options)-1 {
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
	if last := len(p.options) - 1; p.cursor > last {
		p.cursor = last
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
}

func (p *optionPicker) toggle() {
	p.checked[p.cursor] = !p.checked[p.cursor]
}

// preselectSingle moves the cursor to value, if present among the options.
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
	if p.cursor < 0 || p.cursor >= len(p.options) {
		return ""
	}
	return p.options[p.cursor]
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
	start, end := visibleWindow(len(p.options), p.cursor, p.maxVisible)

	var b strings.Builder
	if start > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ↑ %d more above", start)) + "\n")
	}
	for i := start; i < end; i++ {
		o := p.options[i]
		prefix := "  "
		label := o
		if p.multi {
			if p.checked[i] {
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
	if end < len(p.options) {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ↓ %d more below", len(p.options)-end)) + "\n")
	}
	return b.String()
}
