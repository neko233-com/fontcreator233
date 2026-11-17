package ui

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// BuildFunc is the work the TUI supervises.
type BuildFunc func(context.Context, Reporter) error

type channelReporter struct{ ch chan<- Event }

func (r channelReporter) Event(e Event) {
	if e.Kind == EvDone || e.Kind == EvError {
		r.ch <- e
		return
	}
	select {
	case r.ch <- e:
	default:
	}
}

// RunTUI runs fn under a Bubble Tea progress view. It is only called when
// stderr is a terminal.
func RunTUI(ctx context.Context, title string, fn BuildFunc) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	events := make(chan Event, 512)
	m := newModel(title, events, cancel)
	program := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithInput(os.Stdin))

	errCh := make(chan error, 1)
	go func() {
		err := fn(ctx, channelReporter{ch: events})
		if err != nil {
			select {
			case events <- Event{Kind: EvError, Err: err}:
			default:
			}
		}
		close(events)
		errCh <- err
	}()

	_, runErr := program.Run()
	cancel()
	// Drain any events the stopped program no longer consumes so a build
	// blocked on the final EvDone can return.
	go func() {
		for range events {
		}
	}()
	select {
	case buildErr := <-errCh:
		if runErr != nil {
			return runErr
		}
		return buildErr
	case <-time.After(3 * time.Second):
		return fmt.Errorf("build did not stop after cancellation")
	}
}
