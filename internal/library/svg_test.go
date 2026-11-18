package library

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/neko233-com/fontcreator233/internal/geom"
	"github.com/neko233-com/fontcreator233/internal/svgout"
)

func TestCodepointFromFilename(t *testing.T) {
	tests := map[string]rune{
		"U+4E2D.svg":  0x4E2D,
		"u+0041.svg":  'A',
		"uni4E2D.svg": 0x4E2D,
		"A.svg":       'A',
	}
	for name, want := range tests {
		got, ok := CodepointFromFilename(name)
		if !ok || got != want {
			t.Fatalf("%s = %U ok=%v, want %U", name, got, ok, want)
		}
	}
	if _, ok := CodepointFromFilename("glyph.svg"); ok {
		t.Fatal("glyph.svg should not map to one codepoint")
	}
}

func TestSVGRejectsTransform(t *testing.T) {
	data := []byte(`<svg xmlns="http://www.w3.org/2000/svg" transform="scale(2)"><path d="M 0 0 L 100 0 L 100 100 Z"/></svg>`)
	if _, err := ParseSVG(data, "scaled.svg"); err == nil {
		t.Fatal("expected transform rejection")
	}
}

func TestSVGRoundTrip(t *testing.T) {
	var b bytes.Buffer
	p := geom.Circle(300, 300, 200)
	if err := svgout.WriteGlyph(&b, "U+25CF", 600, p); err != nil {
		t.Fatal(err)
	}
	ov, err := ParseSVG(b.Bytes(), "circle.svg")
	if err != nil {
		t.Fatal(err)
	}
	if ov.Advance != 600 || len(ov.Path) != 1 {
		t.Fatalf("override = %#v", ov)
	}
	if ov.Path.Bounds().Width() < 390 || ov.Path.Bounds().Width() > 410 {
		t.Fatalf("bounds = %#v", ov.Path.Bounds())
	}
}

func TestLoadDirAndErrors(t *testing.T) {
	dir := t.TempDir()
	data := []byte(`<svg xmlns="http://www.w3.org/2000/svg" data-advance="500"><path d="M 0 0 L 100 0 L 100 100 Z"/></svg>`)
	if err := os.WriteFile(filepath.Join(dir, "U+0041.svg"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadDir(dir)
	if err != nil || len(m) != 1 || m['A'].Advance != 500 {
		t.Fatalf("m = %#v err=%v", m, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.svg"), []byte(`<svg><path d="M 0 0 A 1 1 0 0 1 2 2"/></svg>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDir(dir); err == nil {
		t.Fatal("expected bad filename/arc error")
	}
}
