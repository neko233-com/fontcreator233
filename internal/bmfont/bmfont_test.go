package bmfont

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/neko233-com/fontcreator233/internal/geom"
)

func testGlyphs() []Glyph {
	return []Glyph{
		{Rune: ' ', Name: "space", Advance: 250, Path: nil},
		{Rune: 'A', Name: "A", Advance: 600, Path: geom.RectPath(50, 0, 550, 700)},
		{Rune: 'B', Name: "B", Advance: 600, Path: geom.Circle(300, 350, 250)},
	}
}

func TestBuildBinaryFormat(t *testing.T) {
	dir := t.TempDir()
	paths, err := Build(Options{
		OutDir: dir, BaseName: "Test", Size: 32, Padding: 2, AtlasSize: 128,
		Ascender: 800, Descender: -200, Weight: 500,
	}, testGlyphs(), []KernPair{{Left: "A", Right: "B", Value: -80}})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("paths = %v", paths)
	}
	data, err := os.ReadFile(filepath.Join(dir, "Test.fnt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data[:4], []byte{'B', 'M', 'F', 3}) {
		t.Fatalf("bad magic: %q", data[:4])
	}
	blocks := map[byte][]byte{}
	for off := 4; off < len(data); {
		kind := data[off]
		size := int(binary.LittleEndian.Uint32(data[off+1:]))
		blocks[kind] = data[off+5 : off+5+size]
		off += 5 + size
	}
	for _, kind := range []byte{1, 2, 3, 4, 5} {
		if _, ok := blocks[kind]; !ok {
			t.Fatalf("missing block %d", kind)
		}
	}
	if binary.LittleEndian.Uint16(blocks[1][0:]) != 32 {
		t.Fatal("bad fontSize")
	}
	if binary.LittleEndian.Uint16(blocks[2][0:]) < 1 {
		t.Fatal("bad lineHeight")
	}
	if got := binary.LittleEndian.Uint16(blocks[2][4:]); got != 128 {
		t.Fatalf("atlas width = %d", got)
	}
	if len(blocks[4]) != 3*20 {
		t.Fatalf("char block = %d bytes", len(blocks[4]))
	}
	ids := []uint32{
		binary.LittleEndian.Uint32(blocks[4][0:]),
		binary.LittleEndian.Uint32(blocks[4][20:]),
		binary.LittleEndian.Uint32(blocks[4][40:]),
	}
	if ids[0] != ' ' || ids[1] != 'A' || ids[2] != 'B' {
		t.Fatalf("char ids = %v", ids)
	}
	if len(blocks[5]) != 10 {
		t.Fatalf("kerning block = %d bytes", len(blocks[5]))
	}
	f, err := os.Open(filepath.Join(dir, "Test_0.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 128 || img.Bounds().Dy() != 128 {
		t.Fatalf("atlas size = %v", img.Bounds())
	}
}

func TestBuildDeterministic(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	opts := Options{OutDir: a, BaseName: "Test", Size: 24, AtlasSize: 128, Ascender: 800, Descender: -200}
	if _, err := Build(opts, testGlyphs(), nil); err != nil {
		t.Fatal(err)
	}
	opts.OutDir = b
	if _, err := Build(opts, testGlyphs(), nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Test.fnt", "Test_0.png"} {
		x, err := os.ReadFile(filepath.Join(a, name))
		if err != nil {
			t.Fatal(err)
		}
		y, err := os.ReadFile(filepath.Join(b, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(x, y) {
			t.Fatalf("%s differs between builds", name)
		}
	}
}

func TestMultiPage(t *testing.T) {
	// Force many glyphs into a tiny atlas so several pages are produced.
	var glyphs []Glyph
	for r := rune(0x20); r <= 0x7E; r++ {
		glyphs = append(glyphs, Glyph{Rune: r, Name: string(r), Advance: 600, Path: geom.RectPath(0, 0, 500, 700)})
	}
	paths, err := Build(Options{
		OutDir: t.TempDir(), BaseName: "Pages", Size: 32, AtlasSize: 64,
		Ascender: 800, Descender: -200,
	}, glyphs, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) < 3 {
		t.Fatalf("expected multiple pages, got %v", paths)
	}
}
