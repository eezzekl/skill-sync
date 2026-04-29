package config_view

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ezzek/skill-sync/internal/tui/shared_toggle"
	"github.com/ezzek/skill-sync/internal/tui/styles"
)

type UpdateConfigMsg struct {
	Targets []string
}

// BackMsg signals the root model to return to the main menu.
type BackMsg struct{}

type Model struct {
	toggle shared_toggle.Model
}

func New(allTargets []string, activeTargets []string) Model {
	activeMap := make(map[string]bool)
	for _, t := range activeTargets {
		activeMap[t] = true
	}

	var items []shared_toggle.Item
	for _, t := range allTargets {
		items = append(items, shared_toggle.Item{
			Label:    t,
			Value:    t,
			Selected: activeMap[t],
		})
	}

	return Model{
		toggle: shared_toggle.New(items),
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
			return m, func() tea.Msg { return UpdateConfigMsg{Targets: targets} }
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
	s.WriteString(styles.TitleStyle.Render("Configure Sync Targets"))
	s.WriteString("\n\nSelect the target directories to sync skills to:\n")
	s.WriteString(m.toggle.View())
	s.WriteString(styles.HelpStyle.Render("\n[space/enter] toggle • [s] save • [q/esc] back to menu"))
	return s.String()
}
