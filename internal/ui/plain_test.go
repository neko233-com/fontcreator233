package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestPlainReporter(t *testing.T) {
	var b bytes.Buffer
	p := NewPlain(&b, false, false)
	p.Event(Event{Kind: EvPhase, Phase: "derive"})
	p.Event(Event{Kind: EvProgress, Phase: "derive", Current: 50, Total: 100})
	p.Event(Event{Kind: EvDone, Duration: 2 * time.Second, Warnings: []string{"missing 16"}})
	out := b.String()
	for _, want := range []string{"derive", "50%", "done in 2s", "warning: missing 16"} {
		if !strings.Contains(out, want) {
			t.Fatalf("plain output missing %q:\n%s", want, out)
		}
	}
}

func TestPlainJSONSuppresses(t *testing.T) {
	var b bytes.Buffer
	p := NewPlain(&b, false, true)
	p.Event(Event{Kind: EvPhase, Phase: "derive"})
	p.Event(Event{Kind: EvProgress, Phase: "derive", Current: 1, Total: 2})
	if b.Len() != 0 {
		t.Fatalf("json plain wrote %q", b.String())
	}
}

func TestPlainQuiet(t *testing.T) {
	var b bytes.Buffer
	p := NewPlain(&b, true, false)
	p.Event(Event{Kind: EvPhase, Phase: "derive"})
	p.Event(Event{Kind: EvProgress, Phase: "derive", Current: 1, Total: 2})
	if b.Len() != 0 {
		t.Fatalf("quiet wrote %q", b.String())
	}
}
