package config_view

import (
	"bytes"
	"io"
	"reflect"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestConfigView_Update(t *testing.T) {
	allTargets := []string{"target1", "target2", "target3"}
	activeTargets := []string{"target2"}

	m := New(allTargets, activeTargets)

	selected := m.toggle.GetSelected()
	if len(selected) != 1 || selected[0].Value != "target2" {
		t.Errorf("expected 1 selected item (target2), got %v", selected)
	}

	// toggle target1
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = newM.(Model)

	// Test 's' to save
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = newM.(Model)
	
	if cmd == nil {
		t.Fatal("expected command on save")
	}

	msg := cmd()
	updateMsg, ok := msg.(UpdateConfigMsg)
	if !ok {
		t.Fatalf("expected UpdateConfigMsg, got %T", msg)
	}

	expected := []string{"target1", "target2"}
	if !reflect.DeepEqual(updateMsg.Targets, expected) {
		t.Errorf("expected targets %v, got %v", expected, updateMsg.Targets)
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

func TestConfigView_Teatest(t *testing.T) {
	all := []string{"t1", "t2"}
	active := []string{"t1"}
	m := New(all, active)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	if err := tm.Quit(); err != nil {
		t.Fatal(err)
	}
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	out, err := io.ReadAll(tm.Output())
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(out, []byte("Configure Sync Targets")) {
		t.Errorf("output should contain title")
	}
	if !bytes.Contains(out, []byte("t1")) {
		t.Errorf("output should contain t1")
	}
}
