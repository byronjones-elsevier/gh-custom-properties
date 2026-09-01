package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/backup"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

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
	schemaErr    error

	cursor          int
	pendingDeleteAt int // -1 when not confirming a delete
	editor          *valueEditor

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

func (m *singleRepoModel) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, quitRequestedCmd
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.properties)-1 {
			m.cursor++
		}
	case key.Matches(msg, m.keys.Add):
		candidates := m.unsetSchemaProperties()
		if m.schema != nil && len(candidates) == 0 {
			m.err = fmt.Errorf("every org-defined property already has a value on this repo")
			return m, nil
		}
		m.err = nil
		m.editor = newAddEditor(candidates, m.schema == nil, availableRows(m.height))
		m.screen = screenEdit
		return m, m.editor.Init()
	case key.Matches(msg, m.keys.Edit):
		if len(m.properties) == 0 {
			return m, nil
		}
		p := m.properties[m.cursor]
		var def *ghclient.PropertyDefinition
		if d, ok := m.schemaByName[p.Name]; ok {
			def = &d
		}
		m.err = nil
		m.editor = newEditEditor(p.Name, def, p.Value, availableRows(m.height))
		m.screen = screenEdit
		return m, m.editor.Init()
	case key.Matches(msg, m.keys.Delete):
		if len(m.properties) == 0 {
			return m, nil
		}
		m.pendingDeleteAt = m.cursor
		m.screen = screenConfirmDelete
	case key.Matches(msg, m.keys.Save):
		if len(m.properties) == 0 {
			return m, nil
		}
		m.screen = screenApplying
		return m, m.applyCmd()
	}
	return m, nil
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

// header returns the persistent "owner/repo" banner shown at the top of
// every screen once a repo has been chosen (i.e. every screen but the
// initial repo-input prompt, where there's nothing to show yet).
func (m *singleRepoModel) header() string {
	if m.owner == "" || m.repo == "" {
		return ""
	}
	return renderHeader(m.owner + "/" + m.repo)
}

func (m *singleRepoModel) View() string {
	return boxStyle.Render(m.header() + m.viewBody())
}

func (m *singleRepoModel) viewBody() string {
	switch m.screen {
	case screenInput:
		return m.viewInput()
	case screenLoading:
		return fmt.Sprintf("\n  %s Loading properties...\n", m.spin.View())
	case screenList:
		return m.viewList()
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
	b.WriteString("\n" + helpLine(keyBinding("enter", "load"), keyBinding("q", "quit")))
	return b.String()
}

func (m *singleRepoModel) viewList() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Custom properties") + "\n\n")

	if m.schema == nil {
		b.WriteString(warnStyle.Render("org schema unavailable — editing values as freeform text") + "\n\n")
	}

	if len(m.properties) == 0 {
		b.WriteString(dimStyle.Render("(no custom properties set)") + "\n")
	} else {
		start, end := visibleWindow(len(m.properties), m.cursor, availableRows(m.height))
		if start > 0 {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  ↑ %d more above", start)) + "\n")
		}
		for i := start; i < end; i++ {
			p := m.properties[i]
			line := fmt.Sprintf("%-30s %s", p.Name, formatValue(p.Value))
			if i == m.cursor {
				b.WriteString(cursorStyle.Render("> ") + selectedStyle.Render(line) + "\n")
			} else {
				b.WriteString("  " + line + "\n")
			}
		}
		if end < len(m.properties) {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  ↓ %d more below", len(m.properties)-end)) + "\n")
		}
	}

	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render(m.err.Error()) + "\n")
	}

	b.WriteString("\n" + helpLine(
		m.keys.Up, m.keys.Down, m.keys.Add, m.keys.Edit, m.keys.Delete, m.keys.Save, m.keys.Quit,
	))
	return b.String()
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
