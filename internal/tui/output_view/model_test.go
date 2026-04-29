package output_view

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestOutputView_Update(t *testing.T) {
	m := New("test output")

	keys := []string{"enter", "esc", "q", "ctrl+c"}

	for _, k := range keys {
		t.Run("key "+k, func(t *testing.T) {
			var msg tea.KeyMsg
			switch k {
			case "enter":
				msg = tea.KeyMsg{Type: tea.KeyEnter}
			case "esc":
				msg = tea.KeyMsg{Type: tea.KeyEsc}
			case "ctrl+c":
				msg = tea.KeyMsg{Type: tea.KeyCtrlC}
			default:
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
			}

			_, cmd := m.Update(msg)
			if cmd == nil {
				t.Fatalf("expected command, got nil for key %s", k)
			}
			outMsg := cmd()
			if _, ok := outMsg.(DoneMsg); !ok {
				t.Errorf("expected DoneMsg, got %T", outMsg)
			}
		})
	}

	t.Run("unrelated key", func(t *testing.T) {
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
		if cmd != nil {
			t.Errorf("expected nil cmd, got %v", cmd)
		}
	})

	t.Run("scroll down and up", func(t *testing.T) {
		var linesBuf strings.Builder
		for i := 0; i < 30; i++ {
			fmt.Fprintf(&linesBuf, "line %d\n", i)
		}
		ms := New(linesBuf.String())

		// scroll down 5 times
		var cur tea.Model = ms
		for i := 0; i < 5; i++ {
			cur, _ = cur.Update(tea.KeyMsg{Type: tea.KeyDown})
		}
		if cur.(Model).scroll != 5 {
			t.Errorf("expected scroll=5, got %d", cur.(Model).scroll)
		}

		// scroll up 2 times
		cur, _ = cur.Update(tea.KeyMsg{Type: tea.KeyUp})
		cur, _ = cur.Update(tea.KeyMsg{Type: tea.KeyUp})
		if cur.(Model).scroll != 3 {
			t.Errorf("expected scroll=3, got %d", cur.(Model).scroll)
		}
	})

	t.Run("scroll does not go below zero", func(t *testing.T) {
		ms := New("only one line")
		cur, _ := ms.Update(tea.KeyMsg{Type: tea.KeyUp})
		if cur.(Model).scroll != 0 {
			t.Errorf("expected scroll=0, got %d", cur.(Model).scroll)
		}
	})

	t.Run("scroll does not exceed content length", func(t *testing.T) {
		var linesBuf strings.Builder
		for i := 0; i < 5; i++ {
			fmt.Fprintf(&linesBuf, "line %d\n", i)
		}
		ms := New(linesBuf.String())
		// press down many times on content shorter than window
		var cur tea.Model = ms
		for i := 0; i < 20; i++ {
			cur, _ = cur.Update(tea.KeyMsg{Type: tea.KeyDown})
		}
		if cur.(Model).scroll != 0 {
			t.Errorf("expected scroll=0 for short content, got %d", cur.(Model).scroll)
		}
	})
}

func TestOutputView_Teatest(t *testing.T) {
	m := New("test output string")

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	if err := tm.Quit(); err != nil {
		t.Fatal(err)
	}
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	out, err := io.ReadAll(tm.Output())
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(out, []byte("test output string")) {
		t.Errorf("output should contain 'test output string'")
	}
}
