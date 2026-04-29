package shared_toggle

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ezzek/skill-sync/internal/tui/styles"
)

type Item struct {
	Label    string
	Value    string
	Selected bool
}

type SubmitMsg struct {
	Selected []Item
}

const defaultMaxVisible = 15

type Model struct {
	Items      []Item
	Cursor     int
	maxVisible int
	scroll     int
}

func New(items []Item) Model {
	return Model{
		Items:      items,
		Cursor:     0,
		maxVisible: defaultMaxVisible,
		scroll:     0,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
				if m.Cursor < m.scroll {
					m.scroll = m.Cursor
				}
			}
		case "down", "j":
			if m.Cursor < len(m.Items)-1 {
				m.Cursor++
				if m.Cursor >= m.scroll+m.maxVisible {
					m.scroll = m.Cursor - m.maxVisible + 1
				}
			}
		case " ", "enter":
			if len(m.Items) > 0 {
				copied := make([]Item, len(m.Items))
				copy(copied, m.Items)
				m.Items = copied
				m.Items[m.Cursor].Selected = !m.Items[m.Cursor].Selected
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	var s strings.Builder

	n := len(m.Items)
	end := m.scroll + m.maxVisible
	if end > n {
		end = n
	}

	if m.scroll > 0 {
		s.WriteString(styles.SubtleStyle.Render(fmt.Sprintf("  ↑ %d more above", m.scroll)) + "\n")
	}

	for i := m.scroll; i < end; i++ {
		item := m.Items[i]
		cursor := " "
		if m.Cursor == i {
			cursor = ">"
		}
		checked := " "
		if item.Selected {
			checked = "x"
		}
		line := fmt.Sprintf("%s [%s] %s", cursor, checked, item.Label)
		if m.Cursor == i {
			s.WriteString(styles.ListItemSelectedStyle.Render(line) + "\n")
		} else {
			s.WriteString(styles.ListItemStyle.Render(line) + "\n")
		}
	}

	if end < n {
		s.WriteString(styles.SubtleStyle.Render(fmt.Sprintf("  ↓ %d more below", n-end)) + "\n")
	}

	return s.String()
}

func (m Model) GetSelected() []Item {
	var selected []Item
	for _, item := range m.Items {
		if item.Selected {
			selected = append(selected, item)
		}
	}
	return selected
}

func (m Model) SelectAll() Model {
	copied := make([]Item, len(m.Items))
	copy(copied, m.Items)
	for i := range copied {
		copied[i].Selected = true
	}
	m.Items = copied
	return m
}

func (m Model) DeselectAll() Model {
	copied := make([]Item, len(m.Items))
	copy(copied, m.Items)
	for i := range copied {
		copied[i].Selected = false
	}
	m.Items = copied
	return m
}
