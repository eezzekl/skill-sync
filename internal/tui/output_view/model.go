package output_view

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ezzek/skill-sync/internal/tui/styles"
)

const maxVisible = 20

type DoneMsg struct{}

type Model struct {
	lines  []string
	scroll int
}

func New(output string) Model {
	return Model{
		lines: strings.Split(strings.TrimRight(output, "\n"), "\n"),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc", "q", "ctrl+c":
			return m, func() tea.Msg { return DoneMsg{} }
		case "up", "k":
			if m.scroll > 0 {
				m.scroll--
			}
		case "down", "j":
			limit := len(m.lines) - maxVisible
			if limit < 0 {
				limit = 0
			}
			if m.scroll < limit {
				m.scroll++
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	var s strings.Builder

	n := len(m.lines)
	end := m.scroll + maxVisible
	if end > n {
		end = n
	}

	if m.scroll > 0 {
		s.WriteString(styles.SubtleStyle.Render(fmt.Sprintf("  ↑ %d more above", m.scroll)) + "\n")
	}
	for i := m.scroll; i < end; i++ {
		s.WriteString(m.lines[i] + "\n")
	}
	if end < n {
		s.WriteString(styles.SubtleStyle.Render(fmt.Sprintf("  ↓ %d more below", n-end)) + "\n")
	}

	s.WriteString(styles.HelpStyle.Render("\n[↑/↓] scroll • [enter/esc/q] back"))
	return s.String()
}
