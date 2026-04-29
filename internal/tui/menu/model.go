package menu

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ezzek/skill-sync/internal/tui/styles"
)

type MenuSelectionMsg struct {
	Selection string
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type Model struct {
	list list.Model
}

func New() Model {
	items := []list.Item{
		item{title: "Init", desc: "Initialize configuration and discover skills"},
		item{title: "Sync", desc: "Synchronize skills across directories"},
		item{title: "Verify", desc: "Check for drift without syncing"},
		item{title: "Config", desc: "Configure sync targets"},
		item{title: "Quit", desc: "Exit the application"},
	}

	m := list.New(items, list.NewDefaultDelegate(), 0, 0)
	m.Title = "Skill Sync Menu"
	m.SetShowHelp(false)
	m.SetShowStatusBar(false)
	m.Styles.Title = styles.TitleStyle

	return Model{list: m}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := styles.BaseStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				if i.title == "Quit" {
					return m, tea.Quit
				}
				return m, func() tea.Msg { return MenuSelectionMsg{Selection: i.title} }
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return styles.BaseStyle.Render(m.list.View())
}
