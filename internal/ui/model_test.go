package ui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModelProgressAndCancel(t *testing.T) {
	events := make(chan Event, 4)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := newModel("Test Font / Regular", events, cancel)
	_, _ = m.Update(eventMsg(Event{Kind: EvPhase, Phase: "generate"}))
	_, _ = m.Update(eventMsg(Event{Kind: EvProgress, Phase: "generate", Current: 50, Total: 100}))
	view := m.View()
	for _, want := range []string{"fontcreator233", "generate", "50%"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q did not return a command")
	}
	select {
	case <-ctx.Done():
	default:
		t.Fatal("q did not cancel context")
	}
	if !m.quitting {
		t.Fatal("quitting flag not set")
	}
}
