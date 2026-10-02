package import_source_view

import (
	"bytes"
	"io"
	"reflect"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

var testTargets = []string{"/home/user/.claude", "/home/user/.cursor", "/home/user/.config/opencode"}

func TestImportSourceView_AllStartChecked(t *testing.T) {
	m := New(testTargets)
	selected := m.toggle.GetSelected()
	if len(selected) != len(testTargets) {
		t.Errorf("expected all %d sources checked, got %d", len(testTargets), len(selected))
	}
}

func TestImportSourceView_ToggleUnchecksSource(t *testing.T) {
	m := New(testTargets)
	// space on first item should uncheck it
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	selected := m2.(Model).toggle.GetSelected()
	if len(selected) != len(testTargets)-1 {
		t.Errorf("expected %d selected after toggle, got %d", len(testTargets)-1, len(selected))
	}
}

func TestImportSourceView_SelectAll(t *testing.T) {
	m := New(testTargets)
	// deselect all first, then re-select
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("A")})
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	selected := m3.(Model).toggle.GetSelected()
	if len(selected) != len(testTargets) {
		t.Errorf("expected %d selected after 'a', got %d", len(testTargets), len(selected))
	}
}

func TestImportSourceView_DeselectAll(t *testing.T) {
	m := New(testTargets)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("A")})
	selected := m2.(Model).toggle.GetSelected()
	if len(selected) != 0 {
		t.Errorf("expected 0 selected after 'A', got %d", len(selected))
	}
}

func TestImportSourceView_EnterEmitsConfirmedMsgWithCheckedSources(t *testing.T) {
	m := New(testTargets)
	// Uncheck the second source before confirming
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})           // move cursor to second
	m3, _ := m2.(Model).Update(tea.KeyMsg{Type: tea.KeySpace}) // uncheck it

	_, cmd := m3.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command on enter")
	}
	msg := cmd()
	confirmed, ok := msg.(ConfirmedMsg)
	if !ok {
		t.Fatalf("expected ConfirmedMsg, got %T", msg)
	}
	expected := []string{testTargets[0], testTargets[2]}
	if !reflect.DeepEqual(confirmed.Sources, expected) {
		t.Errorf("Sources: got %v, want %v", confirmed.Sources, expected)
	}
}

func TestImportSourceView_CtrlSEmitsConfirmedMsg(t *testing.T) {
	m := New(testTargets)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd == nil {
		t.Fatal("expected command on ctrl+s")
	}
	if _, ok := cmd().(ConfirmedMsg); !ok {
		t.Errorf("expected ConfirmedMsg on ctrl+s")
	}
}

func TestImportSourceView_QEmitsBackMsg(t *testing.T) {
	m := New(testTargets)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("expected command on 'q'")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Errorf("expected BackMsg on 'q', got %T", cmd())
	}
}

func TestImportSourceView_EscEmitsBackMsg(t *testing.T) {
	m := New(testTargets)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected command on esc")
	}
	if _, ok := cmd().(BackMsg); !ok {
		t.Errorf("expected BackMsg on esc, got %T", cmd())
	}
}

func TestImportSourceView_CtrlCQuits(t *testing.T) {
	m := New(testTargets)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected command on ctrl+c")
	}
	if cmd() != tea.Quit() {
		t.Errorf("expected tea.Quit on ctrl+c")
	}
}

func TestImportSourceView_Teatest(t *testing.T) {
	m := New(testTargets)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	if err := tm.Quit(); err != nil {
		t.Fatal(err)
	}
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	out, err := io.ReadAll(tm.Output())
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(out, []byte(".claude")) {
		t.Errorf("output should contain '.claude'")
	}
	if !bytes.Contains(out, []byte(".cursor")) {
		t.Errorf("output should contain '.cursor'")
	}
	if !bytes.Contains(out, []byte("Import")) {
		t.Errorf("output should contain 'Import'")
	}
}
