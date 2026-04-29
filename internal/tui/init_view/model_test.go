package init_view

import (
	"bytes"
	"io"
	"reflect"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestInitView_Update(t *testing.T) {
	discovered := []string{".claude/skills", ".cursor/skills", ".config/opencode/skills"}
	existing := []string{".claude/skills"}

	m := New(discovered, existing)

	// Test pre-checked items
	selected := m.toggle.GetSelected()
	if len(selected) != 1 || selected[0].Value != ".claude/skills" {
		t.Errorf("expected 1 selected item (.claude/skills), got %v", selected)
	}

	// Test navigation and toggling via embedded toggle
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown}) // cursor down to .cursor/skills
	m = newM.(Model)
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace}) // toggle it
	m = newM.(Model)

	selected = m.toggle.GetSelected()
	if len(selected) != 2 {
		t.Errorf("expected 2 selected items, got %d", len(selected))
	}

	// Test 's' to save
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = newM.(Model)
	
	if cmd == nil {
		t.Fatal("expected command on save")
	}

	msg := cmd()
	initMsg, ok := msg.(InitConfigMsg)
	if !ok {
		t.Fatalf("expected InitConfigMsg, got %T", msg)
	}

	expected := []string{".claude/skills", ".cursor/skills"}
	if !reflect.DeepEqual(initMsg.Targets, expected) {
		t.Errorf("expected targets %v, got %v", expected, initMsg.Targets)
	}

	// Test 'q' returns to menu (BackMsg), not tea.Quit
	newM, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	m = newM.(Model)

	if cmd == nil {
		t.Fatal("expected command on 'q'")
	}
	msg = cmd()
	if _, ok := msg.(BackMsg); !ok {
		t.Errorf("expected BackMsg on 'q', got %T", msg)
	}

	// ctrl+c should still quit the app
	_, cmdQuit := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmdQuit == nil {
		t.Fatal("expected command on ctrl+c")
	}
	if cmdQuit() != tea.Quit() {
		t.Errorf("expected tea.Quit on ctrl+c")
	}
}

func TestInitView_Teatest(t *testing.T) {
	discovered := []string{"target1", "target2"}
	existing := []string{"target1"}
	m := New(discovered, existing)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// just verify it renders properly
	if err := tm.Quit(); err != nil {
		t.Fatal(err)
	}
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	out, err := io.ReadAll(tm.Output())
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(out, []byte("target1")) {
		t.Errorf("output should contain 'target1'")
	}
	if !bytes.Contains(out, []byte("target2")) {
		t.Errorf("output should contain 'target2'")
	}
	if !bytes.Contains(out, []byte("Initialize Configuration")) {
		t.Errorf("output should contain 'Initialize Configuration'")
	}
}
