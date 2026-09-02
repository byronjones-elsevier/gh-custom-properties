package tui

import (
	"strings"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/knownprops"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// editorOutcome reports whether a valueEditor.Update call finished the
// editor (with a result ready) or left it in progress / cancelled.
type editorOutcome int

const (
	outcomeNone editorOutcome = iota
	outcomeDone
	outcomeCancelled
)

type editorStep int

const (
	stepName editorStep = iota
	stepValue
)

// valueEditor is a type-aware editor for one property's value. When adding a
// new property it first walks the user through picking a name (from the org
// schema, or freeform text if no schema is available), then edits the value
// using a widget appropriate to the property's type.
type valueEditor struct {
	isAdd      bool
	deleteMode bool // true => stop after the name step, Result().Value is always nil
	step       editorStep

	nameCandidates []ghclient.PropertyDefinition
	namePicker     *optionPicker
	nameInput      textinput.Model
	freeformName   bool

	name string
	def  *ghclient.PropertyDefinition // nil => freeform string value

	stringInput   textinput.Model
	picker        *optionPicker
	validationErr error // set when the string input fails a knownprops format check

	maxVisible int // 0 = unbounded; otherwise applied to namePicker/picker so long option lists scroll
}

// newAddEditor starts an editor for adding a new property. candidates is the
// set of schema-defined property names not already set on the repo; it is
// ignored (and freeform name entry used instead) when freeform is true.
// maxVisible bounds how many options a name/value picker shows at once
// before scrolling (0 = unbounded), sized to the terminal by the caller.
func newAddEditor(candidates []ghclient.PropertyDefinition, freeform bool, maxVisible int) *valueEditor {
	e := &valueEditor{isAdd: true, step: stepName, freeformName: freeform, nameCandidates: candidates, maxVisible: maxVisible}
	if freeform {
		e.nameInput = newTextInput(nil)
	} else {
		names := make([]string, len(candidates))
		for i, c := range candidates {
			names[i] = c.Name
		}
		e.namePicker = newOptionPicker(names, false)
		e.namePicker.maxVisible = maxVisible
	}
	return e
}

// newDeleteEditor starts an editor that only asks for a property name (via
// the same schema-picker/freeform-text choice as newAddEditor) and completes
// as soon as a name is chosen; Result().Value is always nil.
func newDeleteEditor(candidates []ghclient.PropertyDefinition, freeform bool, maxVisible int) *valueEditor {
	e := newAddEditor(candidates, freeform, maxVisible)
	e.deleteMode = true
	return e
}

// newEditEditor starts an editor for an existing property's value. def is
// nil when the property isn't described by the org schema, in which case
// the value is edited as freeform text.
func newEditEditor(name string, def *ghclient.PropertyDefinition, current any, maxVisible int) *valueEditor {
	e := &valueEditor{isAdd: false, step: stepValue, name: name, def: def, maxVisible: maxVisible}
	e.initValueStep(current)
	return e
}

func (e *valueEditor) initValueStep(current any) {
	if e.def == nil {
		e.stringInput = newTextInput(current)
		return
	}
	switch e.def.Type {
	case ghclient.PropertyTypeSingleSelect:
		e.picker = newOptionPicker(e.def.AllowedValues, false)
		e.picker.maxVisible = e.maxVisible
		if s, ok := current.(string); ok {
			e.picker.preselectSingle(s)
		}
	case ghclient.PropertyTypeMultiSelect:
		e.picker = newOptionPicker(e.def.AllowedValues, true)
		e.picker.maxVisible = e.maxVisible
		if ss, ok := current.([]string); ok {
			e.picker.preselectMulti(ss)
		}
	case ghclient.PropertyTypeTrueFalse:
		e.picker = newOptionPicker([]string{"true", "false"}, false)
		e.picker.maxVisible = e.maxVisible
		if s, ok := current.(string); ok {
			e.picker.preselectSingle(s)
		}
	default:
		e.stringInput = newTextInput(current)
	}
}

func newTextInput(current any) textinput.Model {
	ti := textinput.New()
	if s, ok := current.(string); ok {
		ti.SetValue(s)
	}
	ti.Focus()
	ti.CursorEnd()
	return ti
}

func (e *valueEditor) Init() tea.Cmd {
	return textinput.Blink
}

// setMaxVisible updates how many options a name/value picker shows at once,
// e.g. in response to a terminal resize while the editor is open.
func (e *valueEditor) setMaxVisible(n int) {
	e.maxVisible = n
	if e.namePicker != nil {
		e.namePicker.maxVisible = n
	}
	if e.picker != nil {
		e.picker.maxVisible = n
	}
}

// Update advances the editor by one message. The returned outcome tells the
// caller whether to keep showing the editor (outcomeNone), read Result()
// because the user confirmed (outcomeDone), or discard the editor because
// the user backed out (outcomeCancelled).
func (e *valueEditor) Update(msg tea.Msg) (tea.Cmd, editorOutcome) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return e.forwardToActiveInput(msg), outcomeNone
	}

	if keyMsg.Type == tea.KeyEsc {
		// A live filter absorbs the first Esc (clearing it); only once
		// there's no filter left active does Esc cancel the editor.
		if p := e.activePicker(); p != nil && (p.filtering || p.filter != "") {
			p.clearFilter()
			return nil, outcomeNone
		}
		return nil, outcomeCancelled
	}

	if e.step == stepName {
		return e.updateNameStep(keyMsg)
	}
	return e.updateValueStep(keyMsg)
}

// activePicker returns whichever optionPicker is currently shown (namePicker
// on the name step, picker on the value step), or nil when the active step
// uses a text input instead.
func (e *valueEditor) activePicker() *optionPicker {
	if e.step == stepName {
		return e.namePicker
	}
	return e.picker
}

// updatePicker handles a keypress for whichever optionPicker is active.
// While composing a filter ("?" then typing), rune/space/backspace keys go
// to the filter text instead of navigating or toggling; enter locks the
// filter in (returning to normal navigation/toggle/confirm) rather than
// confirming the outer selection. Returns outcomeDone once the user
// confirms a selection (enter or tab while not composing a filter).
func updatePicker(p *optionPicker, keyMsg tea.KeyMsg) editorOutcome {
	if p.filtering {
		switch s := keyMsg.String(); {
		case s == "enter":
			p.stopFilterTyping()
		case keyMsg.Type == tea.KeyBackspace:
			p.backspaceFilter()
		case s == "up":
			p.up()
		case s == "down":
			p.down()
		case isPageUpKey(s):
			p.pageUp()
		case isPageDownKey(s):
			p.pageDown()
		case keyMsg.Type == tea.KeySpace:
			p.appendFilterRune(' ')
		case keyMsg.Type == tea.KeyRunes:
			for _, r := range keyMsg.Runes {
				p.appendFilterRune(r)
			}
		}
		return outcomeNone
	}

	switch s := keyMsg.String(); {
	case s == "up" || s == "k":
		p.up()
	case s == "down" || s == "j":
		p.down()
	case isPageUpKey(s):
		p.pageUp()
	case isPageDownKey(s):
		p.pageDown()
	case s == "?":
		p.startFilter()
	case s == " ":
		if p.multi {
			p.toggle()
		}
	case s == "enter" || s == "tab":
		return outcomeDone
	}
	return outcomeNone
}

func (e *valueEditor) forwardToActiveInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch {
	case e.step == stepName && e.freeformName:
		e.nameInput, cmd = e.nameInput.Update(msg)
	case e.step == stepValue && e.picker == nil:
		e.stringInput, cmd = e.stringInput.Update(msg)
	}
	return cmd
}

func (e *valueEditor) updateNameStep(keyMsg tea.KeyMsg) (tea.Cmd, editorOutcome) {
	if e.freeformName {
		if keyMsg.Type == tea.KeyEnter || keyMsg.String() == "tab" {
			name := strings.TrimSpace(e.nameInput.Value())
			if name == "" {
				return nil, outcomeNone
			}
			e.name = name
			if e.deleteMode {
				return nil, outcomeDone
			}
			e.step = stepValue
			e.initValueStep(nil)
			return textinput.Blink, outcomeNone
		}
		var cmd tea.Cmd
		e.nameInput, cmd = e.nameInput.Update(keyMsg)
		return cmd, outcomeNone
	}

	if updatePicker(e.namePicker, keyMsg) != outcomeDone {
		return nil, outcomeNone
	}
	indices := e.namePicker.visibleIndices()
	if e.namePicker.cursor < 0 || e.namePicker.cursor >= len(indices) {
		return nil, outcomeNone
	}
	chosen := e.nameCandidates[indices[e.namePicker.cursor]]
	e.name = chosen.Name
	e.def = &chosen
	if e.deleteMode {
		return nil, outcomeDone
	}
	e.step = stepValue
	e.initValueStep(chosen.DefaultValue)
	return textinput.Blink, outcomeNone
}

func (e *valueEditor) updateValueStep(keyMsg tea.KeyMsg) (tea.Cmd, editorOutcome) {
	if keyMsg.String() == "shift+tab" {
		e.backToNameStep()
		return textinput.Blink, outcomeNone
	}

	if e.picker != nil {
		return nil, updatePicker(e.picker, keyMsg)
	}

	if keyMsg.Type == tea.KeyEnter || keyMsg.String() == "tab" {
		if kind, ok := knownprops.Validators[e.name]; ok {
			if err := knownprops.Validate(kind, e.stringInput.Value()); err != nil {
				e.validationErr = err
				return nil, outcomeNone
			}
		}
		e.validationErr = nil
		return nil, outcomeDone
	}
	e.validationErr = nil
	var cmd tea.Cmd
	e.stringInput, cmd = e.stringInput.Update(keyMsg)
	return cmd, outcomeNone
}

// backToNameStep returns from the value step to the name step, only
// meaningful when adding a new property (there's a name step to go back
// to — editing an existing property's value has none, so this is a no-op).
// The name picker/input keeps whatever the user had already entered; only
// the value-step picker is cleared, since it's rebuilt fresh once the name
// is (re)confirmed — which also means the user can pick a different
// property to add, not just retype the same one's value.
func (e *valueEditor) backToNameStep() {
	if !e.isAdd {
		return
	}
	e.step = stepName
	e.picker = nil
	e.validationErr = nil
}

// Result returns the property name/value pair the user configured. Only
// meaningful after Update has returned outcomeDone.
func (e *valueEditor) Result() ghclient.PropertyValue {
	if e.deleteMode {
		return ghclient.PropertyValue{Name: e.name, Value: nil}
	}
	if e.picker != nil {
		if e.picker.multi {
			return ghclient.PropertyValue{Name: e.name, Value: e.picker.selectedValues()}
		}
		return ghclient.PropertyValue{Name: e.name, Value: e.picker.selectedValue()}
	}
	return ghclient.PropertyValue{Name: e.name, Value: e.stringInput.Value()}
}

func (e *valueEditor) View() string {
	var b strings.Builder

	if e.step == stepName {
		title := "Add property"
		if e.deleteMode {
			title = "Delete property"
		}
		b.WriteString(titleStyle.Render(title) + "\n\n")
		if e.freeformName {
			b.WriteString("Name: " + e.nameInput.View() + "\n")
		} else if e.deleteMode {
			b.WriteString("Choose a property to remove from this repo:\n\n")
			b.WriteString(e.namePicker.View())
		} else {
			b.WriteString("Choose a property defined by the org (values not yet set on this repo):\n\n")
			b.WriteString(e.namePicker.View())
		}
		nextLabel := "next"
		if e.deleteMode {
			nextLabel = "confirm"
		}
		hints := []key.Binding{key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", nextLabel))}
		if !e.freeformName {
			hints = append(hints, keyBinding("?", "filter"))
		}
		hints = append(hints, key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")))
		b.WriteString("\n" + helpLine(hints...))
		return b.String()
	}

	title := "Edit " + e.name
	if e.isAdd {
		title = "Set value for " + e.name
	}
	b.WriteString(titleStyle.Render(title) + "\n\n")

	switch {
	case e.picker != nil && e.picker.multi:
		b.WriteString(e.picker.View())
		b.WriteString("\n" + helpLine(
			key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "toggle")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
			keyBinding("?", "filter"),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		))
	case e.picker != nil:
		b.WriteString(e.picker.View())
		b.WriteString("\n" + helpLine(
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
			keyBinding("?", "filter"),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		))
	default:
		b.WriteString("Value: " + e.stringInput.View() + "\n")
		if e.validationErr != nil {
			b.WriteString(errorStyle.Render(e.validationErr.Error()) + "\n")
		}
		b.WriteString("\n" + helpLine(
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		))
	}
	return b.String()
}
