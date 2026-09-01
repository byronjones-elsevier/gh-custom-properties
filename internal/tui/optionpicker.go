package tui

import "strings"

// optionPicker is a small cursor-driven list used for single/multi-select
// value editing and for picking an org-schema property name to add.
type optionPicker struct {
	options []string
	cursor  int
	multi   bool
	checked map[int]bool
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
	var b strings.Builder
	for i, o := range p.options {
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
	return b.String()
}
