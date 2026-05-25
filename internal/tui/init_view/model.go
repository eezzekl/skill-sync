package init_view

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ezzek/skill-sync/internal/tui/shared_toggle"
	"github.com/ezzek/skill-sync/internal/tui/styles"
)

type InitConfigMsg struct {
	Targets []string
}

// BackMsg signals the root model to return to the main menu.
type BackMsg struct{}

type Model struct {
	toggle     shared_toggle.Model
	agentCount int
}

func New(discovered []string, existing []string) Model {
	existingMap := make(map[string]bool)
	for _, t := range existing {
		cleanT := t
		if filepath.Base(cleanT) == "skills" {
			cleanT = filepath.Dir(cleanT)
		}
		existingMap[cleanT] = true
	}

	var items []shared_toggle.Item
	for _, t := range discovered {
		cleanT := t
		if filepath.Base(cleanT) == "skills" {
			cleanT = filepath.Dir(cleanT)
		}
		items = append(items, shared_toggle.Item{
			Label:    t,
			Value:    t,
			Selected: existingMap[cleanT],
		})
	}

	return Model{
		toggle:     shared_toggle.New(items),
		agentCount: len(discovered),
	}
}

func (m Model) Init() tea.Cmd {
	return m.toggle.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			var targets []string
			for _, item := range m.toggle.GetSelected() {
				targets = append(targets, item.Value)
			}
			return m, func() tea.Msg { return InitConfigMsg{Targets: targets} }
		case "q", "esc":
			return m, func() tea.Msg { return BackMsg{} }
		case "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	var tModel tea.Model
	tModel, cmd = m.toggle.Update(msg)
	m.toggle = tModel.(shared_toggle.Model)

	return m, cmd
}

func (m Model) View() string {
	var s strings.Builder
	s.WriteString(styles.TitleStyle.Render("Initialize Configuration"))
	s.WriteString(fmt.Sprintf("\n\n%d agent(s) found — select targets to sync:\n", m.agentCount))
	s.WriteString(m.toggle.View())
	s.WriteString(styles.HelpStyle.Render("\n[space/enter] toggle • [s] save • [q/esc] back to menu"))
	return s.String()
}
