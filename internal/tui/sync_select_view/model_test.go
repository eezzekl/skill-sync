package sync_select_view

import (
	"reflect"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ezzek/skill-sync/internal/models"
)

func makeSkills(ids ...string) []models.SkillSyncInfo {
	skills := make([]models.SkillSyncInfo, len(ids))
	for i, id := range ids {
		skills[i] = models.SkillSyncInfo{
			ID:           id,
			SourceAgent:  "Claude Code",
			TargetAgents: []string{"Cursor"},
		}
	}
	return skills
}

func TestSyncSelectView_Update(t *testing.T) {
	skills := makeSkills("skill-a", "skill-b", "skill-c")
	m := New(skills)

	if m.totalSkills != 3 {
		t.Errorf("expected 3 skills, got %d", m.totalSkills)
	}

	t.Run("all skills pre-selected", func(t *testing.T) {
		selected := m.toggle.GetSelected()
		if len(selected) != 3 {
			t.Errorf("expected all 3 skills pre-selected, got %d", len(selected))
		}
	})

	t.Run("toggle deselects a skill", func(t *testing.T) {
		m2, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
		root := m2.(Model)
		if len(root.toggle.GetSelected()) != 2 {
			t.Errorf("expected 2 selected after toggle, got %d", len(root.toggle.GetSelected()))
		}
	})

	t.Run("a selects all", func(t *testing.T) {
		m2, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace}) // deselect first
		m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
		if len(m3.(Model).toggle.GetSelected()) != 3 {
			t.Errorf("expected 3 selected after 'a', got %d", len(m3.(Model).toggle.GetSelected()))
		}
	})

	t.Run("A deselects all", func(t *testing.T) {
		m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("A")})
		if len(m2.(Model).toggle.GetSelected()) != 0 {
			t.Errorf("expected 0 selected after 'A', got %d", len(m2.(Model).toggle.GetSelected()))
		}
	})

	t.Run("s emits SyncSelectedMsg with selected skills", func(t *testing.T) {
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
		if cmd == nil {
			t.Fatal("expected command on 's'")
		}
		msg := cmd()
		syncMsg, ok := msg.(SyncSelectedMsg)
		if !ok {
			t.Fatalf("expected SyncSelectedMsg, got %T", msg)
		}
		expected := []string{"skill-a", "skill-b", "skill-c"}
		if !reflect.DeepEqual(syncMsg.SkillIDs, expected) {
			t.Errorf("expected %v, got %v", expected, syncMsg.SkillIDs)
		}
	})

	t.Run("q emits BackMsg", func(t *testing.T) {
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		if cmd == nil {
			t.Fatal("expected command on 'q'")
		}
		if _, ok := cmd().(BackMsg); !ok {
			t.Errorf("expected BackMsg on 'q'")
		}
	})

	t.Run("esc emits BackMsg", func(t *testing.T) {
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		if cmd == nil {
			t.Fatal("expected command on esc")
		}
		if _, ok := cmd().(BackMsg); !ok {
			t.Errorf("expected BackMsg on esc")
		}
	})

	t.Run("ctrl+c quits app", func(t *testing.T) {
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd == nil {
			t.Fatal("expected command on ctrl+c")
		}
		if cmd() != tea.Quit() {
			t.Errorf("expected tea.Quit on ctrl+c")
		}
	})

	t.Run("conflict skills not counted in toggle", func(t *testing.T) {
		mixed := append(makeSkills("skill-a"), models.SkillSyncInfo{
			ID:         "conflict-skill",
			IsConflict: true,
		})
		mc := New(mixed)
		if mc.totalSkills != 1 {
			t.Errorf("expected 1 actionable skill, got %d", mc.totalSkills)
		}
		if mc.conflictCount != 1 {
			t.Errorf("expected 1 conflict, got %d", mc.conflictCount)
		}
	})

	t.Run("empty skill list handles gracefully", func(t *testing.T) {
		empty := New([]models.SkillSyncInfo{})
		if empty.totalSkills != 0 {
			t.Errorf("expected 0 skills")
		}
		view := empty.View()
		if view == "" {
			t.Error("expected non-empty view even with no skills")
		}
	})
}
