package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ezzek/skill-sync/internal/importer"
	"github.com/ezzek/skill-sync/internal/models"
	"github.com/ezzek/skill-sync/internal/tui/config_view"
	"github.com/ezzek/skill-sync/internal/tui/import_skill_view"
	"github.com/ezzek/skill-sync/internal/tui/import_source_view"
	"github.com/ezzek/skill-sync/internal/tui/init_view"
	"github.com/ezzek/skill-sync/internal/tui/menu"
	"github.com/ezzek/skill-sync/internal/tui/output_view"
	"github.com/ezzek/skill-sync/internal/tui/sync_select_view"
)

func TestRootModelTransitions(t *testing.T) {
	callbacks := Callbacks{
		ScanSkills: func() ([]models.SkillSyncInfo, error) {
			return []models.SkillSyncInfo{
				{ID: "skill-a", SourceAgent: "Claude Code", TargetAgents: []string{"Cursor"}},
				{ID: "skill-b", SourceAgent: "Cursor", TargetAgents: []string{"Claude Code"}},
			}, nil
		},
		RunSync:   func(filter []string) (string, error) { return "sync out", nil },
		RunVerify: func() (string, error) { return "verify out", nil },
	}
	m := NewRootModel(callbacks)

	if m.state != stateMenu {
		t.Errorf("Expected initial state stateMenu, got %v", m.state)
	}

	// 1. Menu -> Init
	m1, _ := m.Update(menu.MenuSelectionMsg{Selection: "Init"})
	root1 := m1.(RootModel)
	if root1.state != stateInit {
		t.Errorf("Expected stateInit, got %v", root1.state)
	}

	// 2. Init -> Output (on InitConfigMsg)
	m2, _ := root1.Update(init_view.InitConfigMsg{Targets: []string{"test-target"}})
	root2 := m2.(RootModel)
	if root2.state != stateOutput {
		t.Errorf("Expected stateOutput, got %v", root2.state)
	}

	// 3. Output -> Menu (on DoneMsg)
	m3, _ := root2.Update(output_view.DoneMsg{})
	root3 := m3.(RootModel)
	if root3.state != stateMenu {
		t.Errorf("Expected stateMenu, got %v", root3.state)
	}

	// 4. Menu -> Config
	m4, _ := root3.Update(menu.MenuSelectionMsg{Selection: "Config"})
	root4 := m4.(RootModel)
	if root4.state != stateConfig {
		t.Errorf("Expected stateConfig, got %v", root4.state)
	}

	// 5. Config -> Output (on UpdateConfigMsg)
	m5, _ := root4.Update(config_view.UpdateConfigMsg{Targets: []string{"test-target"}})
	root5 := m5.(RootModel)
	if root5.state != stateOutput {
		t.Errorf("Expected stateOutput, got %v", root5.state)
	}

	// 6. Menu -> Sync -> SyncSelect (scan step)
	m6, _ := root3.Update(menu.MenuSelectionMsg{Selection: "Sync"})
	root6 := m6.(RootModel)
	if root6.state != stateSyncSelect {
		t.Errorf("Expected stateSyncSelect for Sync, got %v", root6.state)
	}

	// 7. SyncSelect -> Output (on SyncSelectedMsg)
	m7, _ := root6.Update(sync_select_view.SyncSelectedMsg{SkillIDs: []string{"skill-a"}})
	root7 := m7.(RootModel)
	if root7.state != stateOutput {
		t.Errorf("Expected stateOutput after SyncSelectedMsg, got %v", root7.state)
	}

	// 8. Menu -> Verify
	m8, _ := root3.Update(menu.MenuSelectionMsg{Selection: "Verify"})
	root8 := m8.(RootModel)
	if root8.state != stateOutput {
		t.Errorf("Expected stateOutput for Verify, got %v", root8.state)
	}

	// Quit on ctrl+c
	_, cmd := root3.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd() != tea.Quit() {
		t.Errorf("Expected tea.Quit on ctrl+c")
	}
}

func makeImportCallbacks() Callbacks {
	candidate := importer.SkillCandidate{
		SkillID:   "git-expert",
		SourceDir: "/fake/git-expert",
		AgentDir:  "/fake",
	}
	return Callbacks{
		ScanSkills: func() ([]models.SkillSyncInfo, error) { return nil, nil },
		RunSync:    func([]string) (string, error) { return "", nil },
		RunVerify:  func() (string, error) { return "", nil },
		FindSkillsForImport: func(sources []string) ([]importer.SkillCandidate, error) {
			return []importer.SkillCandidate{candidate}, nil
		},
		RunImport: func(c importer.SkillCandidate) (string, error) {
			return "Imported " + c.SkillID, nil
		},
	}
}

func TestRootModel_ImportFlow(t *testing.T) {
	m := NewRootModel(makeImportCallbacks())

	// import_source_view.ConfirmedMsg → stateImportSkill (candidates found)
	t.Run("SourceConfirmed with candidates → stateImportSkill", func(t *testing.T) {
		m2, _ := m.Update(import_source_view.ConfirmedMsg{Sources: []string{"/fake"}})
		if m2.(RootModel).state != stateImportSkill {
			t.Errorf("expected stateImportSkill, got %v", m2.(RootModel).state)
		}
	})

	// import_source_view.ConfirmedMsg with no candidates → stateOutput
	t.Run("SourceConfirmed with no candidates → stateOutput", func(t *testing.T) {
		cbs := makeImportCallbacks()
		cbs.FindSkillsForImport = func([]string) ([]importer.SkillCandidate, error) {
			return nil, nil
		}
		m0 := NewRootModel(cbs)
		m2, _ := m0.Update(import_source_view.ConfirmedMsg{Sources: []string{"/fake"}})
		if m2.(RootModel).state != stateOutput {
			t.Errorf("expected stateOutput, got %v", m2.(RootModel).state)
		}
	})

	// import_source_view.BackMsg → stateMenu
	t.Run("SourceBack → stateMenu", func(t *testing.T) {
		m2, _ := m.Update(import_source_view.BackMsg{})
		if m2.(RootModel).state != stateMenu {
			t.Errorf("expected stateMenu, got %v", m2.(RootModel).state)
		}
	})

	// Transition to stateImportSkill then test skill selection → stateOutput
	t.Run("SkillConfirmed → stateOutput", func(t *testing.T) {
		m2, _ := m.Update(import_source_view.ConfirmedMsg{Sources: []string{"/fake"}})
		candidate := importer.SkillCandidate{SkillID: "git-expert", SourceDir: "/fake/git-expert"}
		m3, _ := m2.(RootModel).Update(import_skill_view.ConfirmedMsg{Candidate: candidate})
		if m3.(RootModel).state != stateOutput {
			t.Errorf("expected stateOutput, got %v", m3.(RootModel).state)
		}
	})

	// import_skill_view.BackMsg → stateImportSource
	t.Run("SkillBack → stateImportSource", func(t *testing.T) {
		m2, _ := m.Update(import_source_view.ConfirmedMsg{Sources: []string{"/fake"}})
		m3, _ := m2.(RootModel).Update(import_skill_view.BackMsg{})
		if m3.(RootModel).state != stateImportSource {
			t.Errorf("expected stateImportSource, got %v", m3.(RootModel).state)
		}
	})
}
