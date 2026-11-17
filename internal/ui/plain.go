package ui

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Plain writes line-oriented progress to w.
type Plain struct {
	mu        sync.Mutex
	w         io.Writer
	quiet     bool
	json      bool
	lastPhase string
	lastPct   int
	lastTime  time.Time
	err       error
}

// NewPlain returns a line reporter. json suppresses all progress; quiet keeps
// only warnings and errors.
func NewPlain(w io.Writer, quiet, json bool) *Plain {
	return &Plain{w: w, quiet: quiet, json: json, lastPct: -1}
}

// Event implements Reporter.
func (p *Plain) Event(e Event) {
	if p.json {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	switch e.Kind {
	case EvPhase:
		if p.quiet {
			return
		}
		p.lastPhase = e.Phase
		fmt.Fprintf(p.w, "→ %s\n", e.Phase)
	case EvProgress:
		if p.quiet || e.Total <= 0 {
			return
		}
		pct := e.Current * 100 / e.Total
		now := time.Now()
		if pct == p.lastPct {
			return
		}
		if pct-p.lastPct < 5 && now.Sub(p.lastTime) < 800*time.Millisecond && pct < 100 {
			return
		}
		p.lastPct, p.lastTime = pct, now
		fmt.Fprintf(p.w, "  %-10s %3d%%  %d/%d\n", e.Phase, pct, e.Current, e.Total)
	case EvCurrent:
		if p.quiet || e.Detail == "" {
			return
		}
		fmt.Fprintf(p.w, "  %s: %s\n", e.Phase, e.Detail)
	case EvLog:
		if p.quiet {
			return
		}
		fmt.Fprintf(p.w, "  %s\n", e.Detail)
	case EvError:
		if e.Err != nil {
			p.err = e.Err
			fmt.Fprintf(p.w, "error: %v\n", e.Err)
		}
	case EvDone:
		if p.quiet {
			return
		}
		fmt.Fprintf(p.w, "done in %s\n", e.Duration.Round(time.Millisecond))
		for _, w := range e.Warnings {
			fmt.Fprintf(p.w, "warning: %s\n", w)
		}
	}
}
