package shared_toggle

import (
	"bytes"
	"io"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestSharedToggle_Update(t *testing.T) {
	items := []Item{
		{Label: "Item 1", Value: "val1"},
		{Label: "Item 2", Value: "val2"},
		{Label: "Item 3", Value: "val3"},
	}

	tests := []struct {
		name         string
		startCursor  int
		startItems   []Item
		keys         []string
		expectCursor int
		expectChecks []bool
	}{
		{
			name:         "move down",
			startCursor:  0,
			startItems:   items,
			keys:         []string{"down", "j"},
			expectCursor: 2,
			expectChecks: []bool{false, false, false},
		},
		{
			name:         "move up",
			startCursor:  2,
			startItems:   items,
			keys:         []string{"up", "k"},
			expectCursor: 0,
			expectChecks: []bool{false, false, false},
		},
		{
			name:         "stay within bounds bottom",
			startCursor:  2,
			startItems:   items,
			keys:         []string{"down", "j"},
			expectCursor: 2,
			expectChecks: []bool{false, false, false},
		},
		{
			name:         "stay within bounds top",
			startCursor:  0,
			startItems:   items,
			keys:         []string{"up", "k"},
			expectCursor: 0,
			expectChecks: []bool{false, false, false},
		},
		{
			name:         "toggle with space",
			startCursor:  0,
			startItems:   items,
			keys:         []string{" ", "down", " "},
			expectCursor: 1,
			expectChecks: []bool{true, true, false},
		},
		{
			name:         "toggle with enter",
			startCursor:  1,
			startItems:   items,
			keys:         []string{"enter"},
			expectCursor: 1,
			expectChecks: []bool{false, true, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Deep copy items for safety
			testItems := make([]Item, len(tt.startItems))
			copy(testItems, tt.startItems)

			m := New(testItems)
			m.Cursor = tt.startCursor

			var finalModel tea.Model = m
			for _, k := range tt.keys {
				var msg tea.KeyMsg
				switch k {
				case "up":
					msg = tea.KeyMsg{Type: tea.KeyUp}
				case "down":
					msg = tea.KeyMsg{Type: tea.KeyDown}
				case "enter":
					msg = tea.KeyMsg{Type: tea.KeyEnter}
				case "ctrl+c":
					msg = tea.KeyMsg{Type: tea.KeyCtrlC}
				default:
					msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
				}
				finalModel, _ = finalModel.Update(msg)
			}

			model := finalModel.(Model)

			if model.Cursor != tt.expectCursor {
				t.Errorf("expected cursor %d, got %d", tt.expectCursor, model.Cursor)
			}

			for i, expectCheck := range tt.expectChecks {
				if model.Items[i].Selected != expectCheck {
					t.Errorf("expected item %d selected to be %v, got %v", i, expectCheck, model.Items[i].Selected)
				}
			}
		})
	}
}

func TestSharedToggle_SelectDeselectAll(t *testing.T) {
	items := []Item{
		{Label: "A", Value: "a", Selected: false},
		{Label: "B", Value: "b", Selected: true},
		{Label: "C", Value: "c", Selected: false},
	}

	t.Run("SelectAll sets all to true", func(t *testing.T) {
		m := New(items)
		m2 := m.SelectAll()
		for i, item := range m2.Items {
			if !item.Selected {
				t.Errorf("item %d should be selected", i)
			}
		}
		// original unchanged
		if m.Items[0].Selected {
			t.Error("original item 0 should not be selected")
		}
	})

	t.Run("DeselectAll sets all to false", func(t *testing.T) {
		m := New(items)
		m2 := m.DeselectAll()
		for i, item := range m2.Items {
			if item.Selected {
				t.Errorf("item %d should not be selected", i)
			}
		}
		// original unchanged
		if !m.Items[1].Selected {
			t.Error("original item 1 should remain selected")
		}
	})
}

func TestSharedToggle_Teatest(t *testing.T) {
	items := []Item{
		{Label: "Test1", Value: "1"},
		{Label: "Test2", Value: "2"},
	}
	m := New(items)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	tm.Send(tea.KeyMsg{Type: tea.KeySpace}) // toggle Test1
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})  // move to Test2
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // toggle Test2

	// Quit to finish
	if err := tm.Quit(); err != nil {
		t.Fatal(err)
	}
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	finalModel := tm.FinalModel(t).(Model)

	if finalModel.Cursor != 1 {
		t.Errorf("expected cursor 1, got %d", finalModel.Cursor)
	}

	selected := finalModel.GetSelected()
	if len(selected) != 2 {
		t.Errorf("expected 2 selected items, got %d", len(selected))
	}

	out, err := io.ReadAll(tm.Output())
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(out, []byte("Test1")) {
		t.Errorf("output should contain 'Test1'")
	}
}
