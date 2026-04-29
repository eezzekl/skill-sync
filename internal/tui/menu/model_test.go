package menu

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMenuSelection(t *testing.T) {
	m := New()

	// Initial selection is "Init"
	// Let's press "enter" and see if we get the right message
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e', 'n', 't', 'e', 'r'}})
	if cmd == nil {
		t.Fatal("Expected command to be returned on enter")
	}

	msg := cmd()
	selMsg, ok := msg.(MenuSelectionMsg)
	if !ok {
		t.Fatalf("Expected MenuSelectionMsg, got %T", msg)
	}

	if selMsg.Selection != "Init" {
		t.Errorf("Expected selection 'Init', got %q", selMsg.Selection)
	}

	// Move down and press enter
	m2, _ = m2.Update(tea.KeyMsg{Type: tea.KeyDown})
	m3, cmd2 := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd2 == nil {
		t.Fatal("Expected command to be returned on enter after moving down")
	}
	
	msg2 := cmd2()
	selMsg2, ok := msg2.(MenuSelectionMsg)
	if !ok {
		t.Fatalf("Expected MenuSelectionMsg, got %T", msg2)
	}

	if selMsg2.Selection != "Sync" {
		t.Errorf("Expected selection 'Sync', got %q", selMsg2.Selection)
	}

	// Test Quit
	_, cmdQuit := m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmdQuit != nil && cmdQuit() != tea.Quit() {
		t.Errorf("Expected tea.Quit on 'q'")
	}
}
