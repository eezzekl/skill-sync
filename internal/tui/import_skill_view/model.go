package import_skill_view

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/eezzekl/skill-sync/internal/importer"
	"github.com/eezzekl/skill-sync/internal/tui/styles"
)

// ConfirmedMsg carries the skill the user selected for import.
type ConfirmedMsg struct{ Candidate importer.SkillCandidate }

// BackMsg signals the root model to return to import_source_view.
type BackMsg struct{}

const maxVisible = 15
const descMaxLen = 60

// Model is a single-selection cursor-based list of skill candidates.
type Model struct {
	candidates []importer.SkillCandidate
	cursor     int
	scroll     int
}

// New builds the model from a list of candidates.
// Callers must ensure len(candidates) > 0; use output_view for the empty case.
func New(candidates []importer.SkillCandidate) Model {
	return Model{candidates: candidates}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.scroll {
					m.scroll = m.cursor
				}
			}
		case "down", "j":
			if m.cursor < len(m.candidates)-1 {
				m.cursor++
				if m.cursor >= m.scroll+maxVisible {
					m.scroll = m.cursor - maxVisible + 1
				}
			}
		case "enter":
			c := m.candidates[m.cursor]
			return m, func() tea.Msg { return ConfirmedMsg{Candidate: c} }
		case "esc":
			return m, func() tea.Msg { return BackMsg{} }
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	var s strings.Builder
	s.WriteString(styles.TitleStyle.Render("Import — Select Skill"))
	s.WriteString(fmt.Sprintf("\n\n%d skill(s) found — choose one to import:\n", len(m.candidates)))

	n := len(m.candidates)
	end := m.scroll + maxVisible
	if end > n {
		end = n
	}

	if m.scroll > 0 {
		s.WriteString(styles.SubtleStyle.Render(fmt.Sprintf("  ↑ %d more above", m.scroll)) + "\n")
	}
	for i := m.scroll; i < end; i++ {
		c := m.candidates[i]
		cursor := "  "
		if m.cursor == i {
			cursor = "→ "
		}
		name := c.Metadata.Name
		if name == "" {
			name = c.SkillID
		}
		desc := c.Metadata.Description
		if c.IsConflict {
			desc = "[CONFLICT] " + desc
		}
		if len([]rune(desc)) > descMaxLen {
			desc = string([]rune(desc)[:descMaxLen]) + "…"
		}
		line := fmt.Sprintf("%s%-24s  %-28s  %s", cursor, c.SkillID, name, desc)
		if m.cursor == i {
			s.WriteString(styles.ListItemSelectedStyle.Render(line) + "\n")
		} else {
			s.WriteString(styles.ListItemStyle.Render(line) + "\n")
		}
	}
	if end < n {
		s.WriteString(styles.SubtleStyle.Render(fmt.Sprintf("  ↓ %d more below", n-end)) + "\n")
	}

	s.WriteString(styles.HelpStyle.Render("\n[↑/↓] navigate • [enter] import • [esc] back"))
	return s.String()
}
