package sync_select_view

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ezzek/skill-sync/internal/models"
	"github.com/ezzek/skill-sync/internal/tui/shared_toggle"
	"github.com/ezzek/skill-sync/internal/tui/styles"
)

// SyncSelectedMsg carries the skill IDs the user confirmed for sync.
type SyncSelectedMsg struct {
	SkillIDs []string
}

// BackMsg signals the root model to return to the main menu.
type BackMsg struct{}

type Model struct {
	toggle        shared_toggle.Model
	totalSkills   int
	conflictCount int
}

func New(skills []models.SkillSyncInfo) Model {
	var items []shared_toggle.Item
	conflicts := 0
	for _, s := range skills {
		if s.IsConflict {
			conflicts++
			continue
		}
		label := s.ID + " - " + s.SourceAgent + " → " + strings.Join(s.TargetAgents, ", ")
		items = append(items, shared_toggle.Item{
			Label:    label,
			Value:    s.ID,
			Selected: true,
		})
	}
	return Model{
		toggle:        shared_toggle.New(items),
		totalSkills:   len(items),
		conflictCount: conflicts,
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
			var skillIDs []string
			for _, item := range m.toggle.GetSelected() {
				skillIDs = append(skillIDs, item.Value)
			}
			return m, func() tea.Msg { return SyncSelectedMsg{SkillIDs: skillIDs} }
		case "a":
			m.toggle = m.toggle.SelectAll()
			return m, nil
		case "A":
			m.toggle = m.toggle.DeselectAll()
			return m, nil
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
	s.WriteString(styles.TitleStyle.Render("Sync Skills"))

	if m.totalSkills == 0 {
		s.WriteString("\n\nNo skills require syncing.\n")
		if m.conflictCount > 0 {
			s.WriteString(fmt.Sprintf("\n⚠ %d conflict(s) detected — these will be skipped\n", m.conflictCount))
		}
		s.WriteString(styles.HelpStyle.Render("\n[q/esc] back to menu"))
		return s.String()
	}

	s.WriteString(fmt.Sprintf("\n\n%d skill(s) to sync — select which to include:\n", m.totalSkills))
	s.WriteString(m.toggle.View())
	if m.conflictCount > 0 {
		s.WriteString(fmt.Sprintf("\n⚠ %d conflict(s) detected — these will be skipped\n", m.conflictCount))
	}
	s.WriteString(styles.HelpStyle.Render("\n[space/enter] toggle • [a] all • [A] none • [s] sync • [q/esc] back"))
	return s.String()
}
