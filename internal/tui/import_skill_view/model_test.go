package import_skill_view

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/eezzekl/skill-sync/internal/importer"
	"github.com/eezzekl/skill-sync/internal/models"
)

func makeCandidate(skillID, name, desc string, isConflict bool) importer.SkillCandidate {
	return importer.SkillCandidate{
		SkillID:    skillID,
		SourceDir:  "/fake/" + skillID,
		AgentDir:   "/fake",
		IsConflict: isConflict,
		Metadata: models.SkillMetadata{
			Version:     1,
			Name:        name,
			Description: desc,
		},
	}
}

var testCandidates = []importer.SkillCandidate{
	makeCandidate("docker-dev", "Docker Dev", "Docker workflows", false),
	makeCandidate("git-expert", "Git Expert", "Git branching standards", false),
	makeCandidate("k8s-ops", "K8s Ops", "Kubernetes operations", false),
}

func TestImportSkillView_InitialCursorOnFirst(t *testing.T) {
	m := New(testCandidates)
	if m.cursor != 0 {
		t.Errorf("initial cursor: got %d, want 0", m.cursor)
	}
}

func TestImportSkillView_NavigateDown(t *testing.T) {
	m := New(testCandidates)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m2.(Model).cursor != 1 {
		t.Errorf("cursor after down: got %d, want 1", m2.(Model).cursor)
	}
}

func TestImportSkillView_NavigateUp(t *testing.T) {
	m := New(testCandidates)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyUp})
	if m3.(Model).cursor != 0 {
		t.Errorf("cursor after down+up: got %d, want 0", m3.(Model).cursor)
	}
}

func TestImportSkillView_CursorDoesNotGoAboveZero(t *testing.T) {
	m := New(testCandidates)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m2.(Model).cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", m2.(Model).cursor)
	}
}

func TestImportSkillView_CursorDoesNotGoBeyondLast(t *testing.T) {
	m := New(testCandidates)
	var cur tea.Model = m
	for i := 0; i < 10; i++ {
		cur, _ = cur.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if cur.(Model).cursor != len(testCandidates)-1 {
		t.Errorf("cursor should be at last item, got %d", cur.(Model).cursor)
	}
}

func TestImportSkillView_JKNavigation(t *testing.T) {
	m := New(testCandidates)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m2.(Model).cursor != 1 {
		t.Errorf("j key: cursor should be 1, got %d", m2.(Model).cursor)
	}
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if m3.(Model).cursor != 0 {
		t.Errorf("k key: cursor should be 0, got %d", m3.(Model).cursor)
	}
}

func TestImportSkillView_EnterEmitsConfirmedMsgWithSelectedCandidate(t *testing.T) {
	m := New(testCandidates)
	// Move to second candidate
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})

	_, cmd := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command on enter")
	}
	msg := cmd()
	confirmed, ok := msg.(ConfirmedMsg)
	if !ok {
		t.Fatalf("expected ConfirmedMsg, got %T", msg)
	}
	if confirmed.Candidate.SkillID != "git-expert" {
		t.Errorf("expected 'git-expert', got %q", confirmed.Candidate.SkillID)
	}
}

func TestImportSkillView_EscEmitsBackMsg(t *testing.T) {
	m := New(testCandidates)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected command on esc")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Errorf("expected BackMsg on esc, got %T", cmd())
	}
}

func TestImportSkillView_CtrlCQuits(t *testing.T) {
	m := New(testCandidates)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected command on ctrl+c")
	}
	if cmd() != tea.Quit() {
		t.Errorf("expected tea.Quit on ctrl+c")
	}
}

func TestImportSkillView_ViewShowsCursorMarker(t *testing.T) {
	m := New(testCandidates)
	view := m.View()
	if !strings.Contains(view, "→") {
		t.Error("view should contain → cursor marker")
	}
}

func TestImportSkillView_ViewShowsSkillIDs(t *testing.T) {
	m := New(testCandidates)
	view := m.View()
	for _, c := range testCandidates {
		if !strings.Contains(view, c.SkillID) {
			t.Errorf("view should contain skillID %q", c.SkillID)
		}
	}
}

func TestImportSkillView_ViewShowsConflictPrefix(t *testing.T) {
	candidates := []importer.SkillCandidate{
		makeCandidate("conflict-skill", "Conflict", "some desc", true),
		makeCandidate("normal-skill", "Normal", "other desc", false),
	}
	m := New(candidates)
	view := m.View()
	if !strings.Contains(view, "[CONFLICT]") {
		t.Error("view should contain [CONFLICT] for conflicting skill")
	}
	if strings.Count(view, "[CONFLICT]") != 1 {
		t.Error("only one skill should have [CONFLICT] prefix")
	}
}

func TestImportSkillView_ViewFallsBackToSkillIDWhenNameEmpty(t *testing.T) {
	candidates := []importer.SkillCandidate{
		makeCandidate("unnamed-skill", "", "desc", false),
	}
	m := New(candidates)
	view := m.View()
	// SkillID should appear as the name column fallback
	if strings.Count(view, "unnamed-skill") < 2 {
		t.Error("unnamed-skill should appear at least twice (skillID col + name col fallback)")
	}
}

func TestImportSkillView_DescriptionTruncated(t *testing.T) {
	longDesc := strings.Repeat("x", 100)
	candidates := []importer.SkillCandidate{
		makeCandidate("long-skill", "Long", longDesc, false),
	}
	m := New(candidates)
	view := m.View()
	// Full 100-char description must not appear verbatim
	if strings.Contains(view, longDesc) {
		t.Error("description should be truncated in the view")
	}
	// Truncation ellipsis should be present
	if !strings.Contains(view, "…") {
		t.Error("view should contain … for truncated description")
	}
}

func TestImportSkillView_Teatest(t *testing.T) {
	m := New(testCandidates)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 24))

	if err := tm.Quit(); err != nil {
		t.Fatal(err)
	}
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	out, err := io.ReadAll(tm.Output())
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(out, []byte("docker-dev")) {
		t.Errorf("output should contain 'docker-dev'")
	}
	if !bytes.Contains(out, []byte("Import")) {
		t.Errorf("output should contain 'Import'")
	}
}
