package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/backup"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	menubar "github.com/jejacks0n/bubbletea-menubar"
)

// singleListHelpKeys adapts the property list's key bindings to
// bubbles/help's KeyMap interface for renderFooterPanel.
type singleListHelpKeys struct {
	keys listKeyMap
}

func (k singleListHelpKeys) ShortHelp() []key.Binding {
	return []key.Binding{
		k.keys.Up, k.keys.Add, k.keys.Edit, k.keys.Delete, k.keys.Save,
		keyBinding("?", "filter"), keyBinding("F5", "refresh"), keyBinding("F1", "help"),
		k.keys.Quit,
	}
}

func (k singleListHelpKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.keys.Up, k.keys.Down},
		{k.keys.Add, k.keys.Edit, k.keys.Delete, k.keys.Save},
		{keyBinding("?", "filter"), keyBinding("F5", "refresh"), keyBinding("F1", "help")},
		{k.keys.Quit},
	}
}

type singleScreen int

const (
	screenInput singleScreen = iota
	screenLoading
	screenList
	screenEdit
	screenConfirmDelete
	screenApplying
	screenResult
)

// loadedMsg carries the result of fetching a repo's properties and its
// org's schema (schema/schemaErr set independently, since a missing schema
// is a soft fallback, not a fatal error for the whole load).
type loadedMsg struct {
	owner, repo string
	properties  []ghclient.PropertyValue
	schema      []ghclient.PropertyDefinition
	schemaErr   error
	err         error
}

// appliedMsg carries the result of writing a backup and applying an edit.
type appliedMsg struct {
	backupPath string
	err        error
}

// singleRepoModel drives the single-repo add/edit/delete flow.
type singleRepoModel struct {
	api       ghclient.PropertiesAPI
	backupDir string

	screen singleScreen
	keys   listKeyMap
	spin   spinner.Model

	repoInput textinput.Model
	owner     string
	repo      string

	properties   []ghclient.PropertyValue // working set, mutated by add/edit/delete
	loaded       []ghclient.PropertyValue // as fetched, used as the backup's "before" state
	schema       []ghclient.PropertyDefinition
	schemaByName map[string]ghclient.PropertyDefinition

	cursor          int
	filter          string // "?" starts composing this; narrows the property list to a case-insensitive substring match on name
	filtering       bool
	pendingDeleteAt int // -1 when not confirming a delete
	editor          *valueEditor
	pendingFlash    *pendingFlash // set by a hub-screen command key; see keys.go
	help            help.Model    // renders the boxed footer panel; see renderFooterPanel

	width, height int // last known terminal size, from tea.WindowSizeMsg

	backupPath string
	err        error
}

// newSingleRepoModel builds the single-repo model. If ownerRepo is empty the
// user is prompted for a repo on screenInput; otherwise loading starts
// immediately.
func newSingleRepoModel(api ghclient.PropertiesAPI, backupDir, owner, repo string) *singleRepoModel {
	m := &singleRepoModel{
		api:             api,
		backupDir:       backupDir,
		keys:            defaultListKeyMap(),
		spin:            newSpinner(),
		pendingDeleteAt: -1,
		help:            newHelpModel(),
		owner:           owner,
		repo:            repo,
	}
	if owner == "" || repo == "" {
		m.screen = screenInput
		ti := textinput.New()
		ti.Placeholder = "owner/repo or https://github.com/owner/repo"
		ti.Focus()
		m.repoInput = ti
	} else {
		m.screen = screenLoading
	}
	return m
}

func newSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return s
}

func (m *singleRepoModel) Init() tea.Cmd {
	if m.screen == screenLoading {
		return tea.Batch(m.spin.Tick, m.loadCmd())
	}
	return textinput.Blink
}

func (m *singleRepoModel) loadCmd() tea.Cmd {
	owner, repo := m.owner, m.repo
	api := m.api
	return func() tea.Msg {
		ctx := context.Background()
		props, err := api.GetRepoProperties(ctx, owner, repo)
		if err != nil {
			return loadedMsg{owner: owner, repo: repo, err: err}
		}
		schema, schemaErr := api.GetOrgSchema(ctx, owner)
		return loadedMsg{owner: owner, repo: repo, properties: props, schema: schema, schemaErr: schemaErr}
	}
}

func (m *singleRepoModel) applyCmd() tea.Cmd {
	owner, repo := m.owner, m.repo
	dir := m.backupDir
	before := append([]ghclient.PropertyValue{}, m.loaded...)
	after := append([]ghclient.PropertyValue{}, m.properties...)
	api := m.api
	return func() tea.Msg {
		path, err := backup.Write(dir, backup.ModeSingle, []backup.RepoSnapshot{
			{Owner: owner, Repo: repo, PropertiesBefore: before},
		})
		if err != nil {
			return appliedMsg{err: fmt.Errorf("write backup: %w", err)}
		}
		if err := api.SetRepoProperties(context.Background(), owner, repo, after); err != nil {
			return appliedMsg{backupPath: path, err: err}
		}
		return appliedMsg{backupPath: path}
	}
}

func (m *singleRepoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.editor != nil {
			m.editor.setMaxVisible(availableRows(m.height))
		}
		return m, nil
	case loadedMsg:
		return m.handleLoaded(msg)
	case appliedMsg:
		m.screen = screenResult
		m.backupPath = msg.backupPath
		m.err = msg.err
		return m, nil
	case flashElapsedMsg:
		if m.pendingFlash == nil {
			return m, nil
		}
		action := m.pendingFlash.action
		m.pendingFlash = nil
		return action()
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *singleRepoModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenInput:
		if msg.Type == tea.KeyEnter {
			owner, repo, err := ghclient.ParseRepoSpec(m.repoInput.Value())
			if err != nil {
				m.err = err
				return m, nil
			}
			m.err = nil
			m.owner, m.repo = owner, repo
			m.screen = screenLoading
			return m, tea.Batch(m.spin.Tick, m.loadCmd())
		}
		var cmd tea.Cmd
		m.repoInput, cmd = m.repoInput.Update(msg)
		return m, cmd

	case screenList:
		return m.handleListKey(msg)

	case screenEdit:
		cmd, outcome := m.editor.Update(msg)
		switch outcome {
		case outcomeDone:
			result := m.editor.Result()
			m.upsertProperty(result)
			m.editor = nil
			m.screen = screenList
			return m, nil
		case outcomeCancelled:
			m.editor = nil
			m.screen = screenList
			return m, nil
		default:
			return m, cmd
		}

	case screenConfirmDelete:
		switch msg.String() {
		case "y":
			m.properties = append(m.properties[:m.pendingDeleteAt], m.properties[m.pendingDeleteAt+1:]...)
			if m.cursor >= len(m.properties) && m.cursor > 0 {
				m.cursor--
			}
			m.pendingDeleteAt = -1
			m.screen = screenList
		case "n", "esc":
			m.pendingDeleteAt = -1
			m.screen = screenList
		}
		return m, nil

	case screenResult:
		// A deliberate dismissal after work is already saved (or a failure
		// already reported) — not an accidental interrupt, so this quits
		// immediately without the confirm overlay.
		return m, tea.Quit
	}
	return m, nil
}

// filteredPropertyIndices returns the indices into m.properties matching
// m.filter (all of them when the filter is empty).
func (m *singleRepoModel) filteredPropertyIndices() []int {
	names := make([]string, len(m.properties))
	for i, p := range m.properties {
		names[i] = p.Name
	}
	return filterIndices(names, m.filter)
}

func (m *singleRepoModel) clearFilter() {
	m.filter = ""
	m.filtering = false
	m.cursor = 0
}

func (m *singleRepoModel) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEsc && (m.filtering || m.filter != "") {
		m.clearFilter()
		return m, nil
	}

	if m.filtering {
		indices := m.filteredPropertyIndices()
		switch s := msg.String(); {
		case s == "enter":
			m.filtering = false
		case msg.Type == tea.KeyBackspace:
			if m.filter != "" {
				runes := []rune(m.filter)
				m.filter = string(runes[:len(runes)-1])
				m.cursor = 0
			}
		case s == "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case s == "down":
			if m.cursor < len(indices)-1 {
				m.cursor++
			}
		case isPageUpKey(s):
			m.cursor -= pageSize(availableRows(m.height))
			if m.cursor < 0 {
				m.cursor = 0
			}
		case isPageDownKey(s):
			m.cursor += pageSize(availableRows(m.height))
			if last := len(indices) - 1; m.cursor > last {
				m.cursor = last
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
		case msg.Type == tea.KeySpace:
			m.filter += " "
			m.cursor = 0
		case msg.Type == tea.KeyRunes:
			m.filter += string(msg.Runes)
			m.cursor = 0
		}
		return m, nil
	}

	indices := m.filteredPropertyIndices()
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m.deferAction("q", func() (tea.Model, tea.Cmd) { return m, quitRequestedCmd })
	case msg.Type == tea.KeyF5:
		m.screen = screenLoading
		m.err = nil
		return m, tea.Batch(m.spin.Tick, m.loadCmd())
	case msg.String() == "?":
		m.filtering = true
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(indices)-1 {
			m.cursor++
		}
	case isPageUpKey(msg.String()):
		m.cursor -= pageSize(availableRows(m.height))
		if m.cursor < 0 {
			m.cursor = 0
		}
	case isPageDownKey(msg.String()):
		m.cursor += pageSize(availableRows(m.height))
		if last := len(indices) - 1; m.cursor > last {
			m.cursor = last
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
	case key.Matches(msg, m.keys.Add):
		candidates := m.unsetSchemaProperties()
		if m.schema != nil && len(candidates) == 0 {
			m.err = fmt.Errorf("every org-defined property already has a value on this repo")
			return m, nil
		}
		m.err = nil
		return m.deferAction("a", func() (tea.Model, tea.Cmd) {
			m.editor = newAddEditor(candidates, m.schema == nil, availableRows(m.height))
			m.screen = screenEdit
			return m, m.editor.Init()
		})
	case key.Matches(msg, m.keys.Edit):
		if m.cursor < 0 || m.cursor >= len(indices) {
			return m, nil
		}
		p := m.properties[indices[m.cursor]]
		var def *ghclient.PropertyDefinition
		if d, ok := m.schemaByName[p.Name]; ok {
			def = &d
		}
		m.err = nil
		return m.deferAction("enter/e", func() (tea.Model, tea.Cmd) {
			m.editor = newEditEditor(p.Name, def, p.Value, availableRows(m.height))
			m.screen = screenEdit
			return m, m.editor.Init()
		})
	case key.Matches(msg, m.keys.Delete):
		if m.cursor < 0 || m.cursor >= len(indices) {
			return m, nil
		}
		deleteAt := indices[m.cursor]
		return m.deferAction("d", func() (tea.Model, tea.Cmd) {
			m.pendingDeleteAt = deleteAt
			m.screen = screenConfirmDelete
			return m, nil
		})
	case key.Matches(msg, m.keys.Save):
		if len(m.properties) == 0 {
			return m, nil
		}
		return m.deferAction("s", func() (tea.Model, tea.Cmd) {
			m.screen = screenApplying
			return m, m.applyCmd()
		})
	}
	return m, nil
}

// deferAction defers a hub-screen command's effect until its footer key
// finishes flashing (see keys.go's pendingFlash), instead of running it
// immediately.
func (m *singleRepoModel) deferAction(label string, action func() (tea.Model, tea.Cmd)) (tea.Model, tea.Cmd) {
	m.pendingFlash = &pendingFlash{label: label, action: action}
	return m, flashTick()
}

func (m *singleRepoModel) handleLoaded(msg loadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.screen = screenResult
		return m, nil
	}
	m.properties = append([]ghclient.PropertyValue{}, msg.properties...)
	m.loaded = append([]ghclient.PropertyValue{}, msg.properties...)
	sort.Slice(m.properties, func(i, j int) bool { return m.properties[i].Name < m.properties[j].Name })

	if msg.schemaErr == nil {
		sortSchema(msg.schema)
		m.schema = msg.schema
		m.schemaByName = make(map[string]ghclient.PropertyDefinition, len(msg.schema))
		for _, d := range msg.schema {
			m.schemaByName[d.Name] = d
		}
	} else if msg.schemaErr != ghclient.ErrSchemaUnavailable {
		m.err = fmt.Errorf("fetching org schema: %w", msg.schemaErr)
	}
	m.screen = screenList
	return m, nil
}

// unsetSchemaProperties returns schema-defined properties that don't
// currently have a value on the repo, i.e. valid candidates for "add".
func (m *singleRepoModel) unsetSchemaProperties() []ghclient.PropertyDefinition {
	if m.schema == nil {
		return nil
	}
	set := make(map[string]bool, len(m.properties))
	for _, p := range m.properties {
		set[p.Name] = true
	}
	var out []ghclient.PropertyDefinition
	for _, d := range m.schema {
		if !set[d.Name] {
			out = append(out, d)
		}
	}
	return out
}

func (m *singleRepoModel) upsertProperty(v ghclient.PropertyValue) {
	for i, p := range m.properties {
		if p.Name == v.Name {
			m.properties[i] = v
			return
		}
	}
	m.properties = append(m.properties, v)
	sort.Slice(m.properties, func(i, j int) bool { return m.properties[i].Name < m.properties[j].Name })
}

// headerText returns the plain "owner/repo" identity text shown on the
// right side of the top bar once a repo has been chosen, or "" before that
// (the initial repo-input prompt, where there's nothing to show yet).
func (m *singleRepoModel) headerText() string {
	if m.owner == "" || m.repo == "" {
		return ""
	}
	return m.owner + "/" + m.repo
}

// topBarItems returns the commands shown in the top bar for the current
// screen — only the property list has commands worth showing there; other
// screens (input, loading, an open editor, ...) show an empty command bar
// with just the identity text on the right, rather than commands that
// aren't actually available right now.
func (m *singleRepoModel) topBarItems() []menubar.MenuItem {
	if m.screen != screenList {
		return nil
	}
	return []menubar.MenuItem{
		{Label: "Add", Hotkey: "a"},
		{Label: "Edit", Hotkey: "e"},
		{Label: "Delete", Hotkey: "d"},
		{Label: "Save", Hotkey: "s"},
		menubar.Separator(),
		{Label: "Filter", Hotkey: "?"},
		{Label: "Refresh", Hotkey: "F5"},
		{Label: "Help", Hotkey: "F1"},
		menubar.Separator(),
		{Label: "Quit", Hotkey: "q"},
	}
}

func (m *singleRepoModel) View() string {
	top := renderTopBar(m.topBarItems(), m.headerText(), m.width)
	if m.screen == screenList {
		content, footer := m.viewListParts()
		return boxStyle.Render(pinFooter(top+"\n\n"+content, footer, m.height))
	}
	return boxStyle.Render(top + "\n\n" + m.viewBody())
}

func (m *singleRepoModel) viewBody() string {
	switch m.screen {
	case screenInput:
		return m.viewInput()
	case screenLoading:
		return fmt.Sprintf("\n  %s Loading properties...\n", m.spin.View())
	case screenEdit:
		return m.editor.View()
	case screenConfirmDelete:
		name := m.properties[m.pendingDeleteAt].Name
		return warnStyle.Render(fmt.Sprintf("Delete property %q?", name)) + "\n\n" + helpLine(
			keyBinding("y", "confirm"),
			keyBinding("n/esc", "cancel"),
		)
	case screenApplying:
		return fmt.Sprintf("\n  %s Applying changes...\n", m.spin.View())
	case screenResult:
		return m.viewResult()
	}
	return ""
}

func (m *singleRepoModel) viewInput() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("gh-custom-properties") + "\n\n")
	b.WriteString("Repo: " + m.repoInput.View() + "\n")
	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render(m.err.Error()) + "\n")
	}
	b.WriteString("\n" + helpLine(keyBinding("enter", "load"), keyBinding("ctrl+c", "quit")))
	return b.String()
}

// viewListParts renders the property list screen, split into scrollable
// content and its footer so View can pin the footer to the terminal's last
// row instead of letting it float wherever the content happens to end.
func (m *singleRepoModel) viewListParts() (content, footer string) {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Custom properties") + "\n\n")

	if m.schema == nil {
		b.WriteString(warnStyle.Render("org schema unavailable — editing values as freeform text") + "\n\n")
	}

	if m.filtering || m.filter != "" {
		b.WriteString(dimStyle.Render("Filter: "+m.filter) + "\n")
	}

	indices := m.filteredPropertyIndices()
	if len(m.properties) == 0 {
		b.WriteString(dimStyle.Render("(no custom properties set)") + "\n")
	} else if len(indices) == 0 {
		b.WriteString(dimStyle.Render("(no matches)") + "\n")
	} else {
		start, end := visibleWindow(len(indices), m.cursor, availableRows(m.height))
		if start > 0 {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  ↑ %d more above", start)) + "\n")
		}
		for i := start; i < end; i++ {
			p := m.properties[indices[i]]
			line := truncateToWidth(fmt.Sprintf("%-30s %s", p.Name, formatValue(p.Value)), rowContentWidth(m.width))
			if i == m.cursor {
				b.WriteString(cursorStyle.Render("> ") + selectedStyle.Render(line) + "\n")
			} else {
				b.WriteString("  " + line + "\n")
			}
		}
		if end < len(indices) {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  ↓ %d more below", len(indices)-end)) + "\n")
		}
	}

	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render(m.err.Error()) + "\n")
	}

	footer = renderFooterPanel(m.help, m.width, singleListHelpKeys{keys: m.keys})
	return b.String(), footer
}

func (m *singleRepoModel) viewResult() string {
	var b strings.Builder
	if m.err != nil {
		b.WriteString(errorStyle.Render("Failed: "+m.err.Error()) + "\n")
		if m.backupPath != "" {
			b.WriteString(dimStyle.Render("Backup of prior values was written to "+m.backupPath) + "\n")
		}
	} else {
		b.WriteString(successStyle.Render("Applied changes") + "\n")
		b.WriteString(dimStyle.Render("Backup of prior values: "+m.backupPath) + "\n")
	}
	b.WriteString("\n" + helpLine(keyBinding("any key", "quit")))
	return b.String()
}

func formatValue(v any) string {
	switch t := v.(type) {
	case nil:
		return dimStyle.Render("(unset)")
	case string:
		return t
	case []string:
		return strings.Join(t, ", ")
	default:
		return fmt.Sprintf("%v", t)
	}
}
