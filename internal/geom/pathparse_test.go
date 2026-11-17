package geom

import (
	"math"
	"strings"
	"testing"
)

func TestParseCommandsAbsolute(t *testing.T) {
	cmds, err := ParseCommands("M 0 0 L 100 0 L 100 100 Z")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(cmds) != 4 {
		t.Fatalf("cmds = %d, want 4", len(cmds))
	}
	if cmds[0].Kind != CmdMove || cmds[0].P[0] != (Point{0, 0}) {
		t.Fatalf("move = %#v", cmds[0])
	}
	if cmds[1].Kind != CmdLine || cmds[1].P[0] != (Point{100, 0}) {
		t.Fatalf("line = %#v", cmds[1])
	}
	if cmds[3].Kind != CmdClose {
		t.Fatalf("close = %#v", cmds[3])
	}
}

func TestParseCommandsRelativeAndSmooth(t *testing.T) {
	d := "m 10 10 l 20 0 h 10 v 10 q 5 -5 10 0 t 10 0 c 0 10 10 10 10 0 s 10 -10 10 0 z"
	cmds, err := ParseCommands(d)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []struct {
		kind CmdKind
		p    [2]Point // last two arguments only
	}{
		{CmdMove, [2]Point{{10, 10}, {10, 10}}},
		{CmdLine, [2]Point{{30, 10}, {30, 10}}},
		{CmdLine, [2]Point{{40, 10}, {40, 10}}},
		{CmdLine, [2]Point{{40, 20}, {40, 20}}},
		{CmdQuad, [2]Point{{45, 15}, {50, 20}}},
		{CmdQuad, [2]Point{{55, 25}, {60, 20}}},
		{CmdCubic, [2]Point{{70, 30}, {70, 20}}},
		{CmdCubic, [2]Point{{80, 10}, {80, 20}}},
		{CmdClose, [2]Point{{80, 20}, {80, 20}}},
	}
	if len(cmds) != len(want) {
		t.Fatalf("cmds = %d, want %d: %#v", len(cmds), len(want), cmds)
	}
	for i, w := range want {
		if cmds[i].Kind != w.kind {
			t.Fatalf("cmd %d kind = %v, want %v", i, cmds[i].Kind, w.kind)
		}
		var got Point
		switch w.kind {
		case CmdMove, CmdLine:
			got = cmds[i].P[0]
		case CmdQuad:
			got = cmds[i].P[1]
			if cmds[i].P[0] != w.p[0] {
				t.Fatalf("cmd %d control = %v, want %v", i, cmds[i].P[0], w.p[0])
			}
		case CmdCubic:
			got = cmds[i].P[2]
			if cmds[i].P[1] != w.p[0] {
				t.Fatalf("cmd %d control2 = %v, want %v", i, cmds[i].P[1], w.p[0])
			}
		case CmdClose:
			got = Point{80, 20}
		}
		if got != w.p[1] {
			t.Fatalf("cmd %d end = %v, want %v (cmd %#v)", i, got, w.p[1], cmds[i])
		}
	}
}

func TestParseCommandsRejectsBadInput(t *testing.T) {
	tests := []struct {
		d    string
		want string
	}{
		{"M 0", "offset"},
		{"L 10 10", "must start with a move"},
		{"M 0 0 A 10 10 0 0 1 20 20", "arc commands are not supported"},
		{"M 0 0 L NaN 0", "non-finite"},
		{"M 0 0 L Inf 0", "non-finite"},
		{"", "empty"},
	}
	for _, tt := range tests {
		_, err := ParseCommands(tt.d)
		if err == nil {
			t.Fatalf("%q: expected error", tt.d)
		}
		if !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("%q: error = %v, want substring %q", tt.d, err, tt.want)
		}
	}
}

func TestParseCommandsCapsPoints(t *testing.T) {
	var b strings.Builder
	b.WriteString("M 0 0")
	for i := 0; i < 200001; i++ {
		b.WriteString(" L 1 1")
	}
	_, err := ParseCommands(b.String())
	if err == nil || !strings.Contains(err.Error(), "too many") {
		t.Fatalf("error = %v, want too many", err)
	}
}

func TestFlattenCubic(t *testing.T) {
	cmds, err := ParseCommands("M 0 0 C 0 100 100 100 100 0")
	if err != nil {
		t.Fatal(err)
	}
	pts := Flatten(cmds, 0.5)
	if len(pts) < 10 {
		t.Fatalf("points = %d, want >= 10", len(pts))
	}
	if pts[0] != (Point{0, 0}) || pts[len(pts)-1] != (Point{100, 0}) {
		t.Fatalf("endpoints = %v .. %v", pts[0], pts[len(pts)-1])
	}
	maxY := 0.0
	for _, p := range pts {
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	if math.Abs(maxY-75) > 1 {
		t.Fatalf("max y = %v, want ~75", maxY)
	}
}

func TestFlattenClosed(t *testing.T) {
	cmds, _ := ParseCommands("M 0 0 L 10 0 L 10 10 Z")
	pts := Flatten(cmds, 0.1)
	if len(pts) != 4 || pts[3] != (Point{0, 0}) {
		t.Fatalf("pts = %v, want closed square", pts)
	}
}
