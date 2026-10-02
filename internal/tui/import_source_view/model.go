package import_source_view

import (
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eezzekl/skill-sync/internal/tui/shared_toggle"
	"github.com/eezzekl/skill-sync/internal/tui/styles"
)

// ConfirmedMsg carries the subset of source directories the user kept checked.
type ConfirmedMsg struct{ Sources []string }

// BackMsg signals the root model to return to the main menu.
type BackMsg struct{}

// Model wraps shared_toggle for agent-source multi-selection.
type Model struct {
	toggle  shared_toggle.Model
	sources []string // parallel to toggle items; index → full target path
}

// New builds the model from a list of resolved target directories.
// All sources start checked.
func New(targets []string) Model {
	items := make([]shared_toggle.Item, len(targets))
	for i, t := range targets {
		items[i] = shared_toggle.Item{
			Label:    filepath.Base(t) + "  (" + t + ")",
			Value:    t,
			Selected: true,
		}
	}
	return Model{
		toggle:  shared_toggle.New(items),
		sources: targets,
	}
}

func (m Model) Init() tea.Cmd { return m.toggle.Init() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "ctrl+s":
			var checked []string
			for _, item := range m.toggle.GetSelected() {
				checked = append(checked, item.Value)
			}
			return m, func() tea.Msg { return ConfirmedMsg{Sources: checked} }
		case "q", "esc":
			return m, func() tea.Msg { return BackMsg{} }
		case "ctrl+c":
			return m, tea.Quit
		case "a":
			m.toggle = m.toggle.SelectAll()
			return m, nil
		case "A":
			m.toggle = m.toggle.DeselectAll()
			return m, nil
		}
	}
	var cmd tea.Cmd
	var updated tea.Model
	updated, cmd = m.toggle.Update(msg)
	m.toggle = updated.(shared_toggle.Model)
	return m, cmd
}

func (m Model) View() string {
	var s strings.Builder
	s.WriteString(styles.TitleStyle.Render("Import — Select Sources"))
	s.WriteString("\n\nChoose which agent directories to search:\n")
	s.WriteString(m.toggle.View())
	s.WriteString(styles.HelpStyle.Render(
		"\n[↑/↓] navigate • [space] toggle • [a] all • [A] none • [enter/ctrl+s] next • [q/esc] back",
	))
	return s.String()
}
