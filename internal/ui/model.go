package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type phaseState struct {
	name    string
	current int
	total   int
	done    bool
}

type model struct {
	title    string
	phases   map[string]*phaseState
	order    []string
	current  string
	done     bool
	err      error
	warnings []string
	summary  string
	start    time.Time
	duration time.Duration
	width    int
	logs     []string
	cancel   context.CancelFunc
	events   <-chan Event
	progress progress.Model
	quitting bool
}

type eventMsg Event
type quitMsg struct{}

func newModel(title string, events <-chan Event, cancel context.CancelFunc) *model {
	return &model{
		title:    title,
		phases:   map[string]*phaseState{},
		events:   events,
		cancel:   cancel,
		start:    time.Now(),
		width:    80,
		progress: progress.New(progress.WithWidth(38), progress.WithDefaultGradient()),
	}
}

func (m *model) Init() tea.Cmd { return waitEvent(m.events) }

func waitEvent(ch <-chan Event) tea.Cmd {
	return func() tea.Msg {
		e, ok := <-ch
		if !ok {
			return quitMsg{}
		}
		return eventMsg(e)
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			if m.cancel != nil {
				m.cancel()
			}
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	case eventMsg:
		return m.handle(Event(msg))
	case quitMsg:
		return m, tea.Quit
	}
	return m, nil
}

func (m *model) handle(e Event) (tea.Model, tea.Cmd) {
	switch e.Kind {
	case EvPhase:
		if m.phases[e.Phase] == nil {
			m.phases[e.Phase] = &phaseState{name: e.Phase}
			m.order = append(m.order, e.Phase)
		}
		m.current = e.Phase
	case EvProgress:
		ph := m.phases[e.Phase]
		if ph == nil {
			ph = &phaseState{name: e.Phase}
			m.phases[e.Phase] = ph
			m.order = append(m.order, e.Phase)
		}
		if e.Current >= ph.current {
			ph.current = e.Current
		}
		if e.Total > 0 {
			ph.total = e.Total
		}
		if ph.total > 0 && ph.current >= ph.total {
			ph.done = true
		}
		m.current = e.Phase
	case EvCurrent:
		m.current = e.Detail
	case EvLog:
		m.logs = append(m.logs, e.Detail)
		if len(m.logs) > 5 {
			m.logs = m.logs[len(m.logs)-5:]
		}
	case EvError:
		m.err = e.Err
		m.done = true
		return m, tea.Tick(600*time.Millisecond, func(time.Time) tea.Msg { return quitMsg{} })
	case EvDone:
		m.done = true
		m.warnings = e.Warnings
		m.summary = e.Summary
		m.duration = e.Duration
		return m, tea.Tick(700*time.Millisecond, func(time.Time) tea.Msg { return quitMsg{} })
	}
	return m, waitEvent(m.events)
}

func (m *model) View() string {
	var b strings.Builder
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Render("fontcreator233")
	b.WriteString(title + "  " + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(m.title) + "\n\n")
	for _, name := range m.order {
		ph := m.phases[name]
		mark := "○"
		if ph.done {
			mark = "●"
		} else if name == m.current {
			mark = "◐"
		}
		counts := ""
		if ph.total > 0 {
			counts = fmt.Sprintf("  %d/%d", ph.current, ph.total)
		}
		b.WriteString(fmt.Sprintf("%s %-12s%s\n", mark, ph.name, counts))
	}
	pct, current, total := m.overall()
	b.WriteString("\n")
	b.WriteString(m.progress.ViewAs(pct))
	b.WriteString(fmt.Sprintf("  %3.0f%%  (%d/%d)\n", pct*100, current, total))
	if m.current != "" && !m.done {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("now: "+m.current) + "\n")
	}
	if !m.done {
		elapsed := time.Since(m.start)
		eta := ""
		if pct > 0.01 {
			eta = fmt.Sprintf("  ETA %s", time.Duration(float64(elapsed)/(pct)-float64(elapsed)).Round(100*time.Millisecond))
		}
		b.WriteString(fmt.Sprintf("elapsed %s%s\n", elapsed.Round(100*time.Millisecond), eta))
	} else {
		b.WriteString(fmt.Sprintf("done in %s\n", m.duration.Round(time.Millisecond)))
		if m.summary != "" {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(m.summary) + "\n")
		}
	}
	if len(m.logs) > 0 {
		b.WriteString("\n")
		for _, l := range m.logs {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(l) + "\n")
		}
	}
	if m.err != nil {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("error: "+m.err.Error()) + "\n")
	}
	for _, w := range m.warnings {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("warning: "+w) + "\n")
	}
	if !m.done {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("q cancel · ctrl+c quit") + "\n")
	}
	return b.String()
}

func (m *model) overall() (pct float64, current, total int) {
	for _, name := range m.order {
		ph := m.phases[name]
		current += ph.current
		total += ph.total
	}
	if total == 0 {
		return 0, 0, 0
	}
	if current > total {
		current = total
	}
	return float64(current) / float64(total), current, total
}
