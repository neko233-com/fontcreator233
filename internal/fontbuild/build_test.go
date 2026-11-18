package fontbuild

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"io"
	"testing"
	"time"

	"github.com/neko233-com/fontcreator233/internal/geom"
	"golang.org/x/image/font/sfnt"
)

func testConfig() Config {
	return Config{
		Family:      "Test Grotesk",
		Style:       "Regular",
		Version:     "1.0.0",
		Designer:    "Tests",
		Description: "A test font.",
		License:     "OFL-1.1",
		LicenseURL:  "https://scripts.sil.org/OFL",
		Copyright:   "Copyright 2026 Tests",
		SampleText:  "Aa 中",
		Ascender:    800,
		Descender:   -200,
		LineGap:     0,
		CapHeight:   720,
		XHeight:     520,
		WeightClass: 400,
		WidthClass:  5,
		Created:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Glyphs: []Glyph{
			{Rune: 0, Advance: 600, Path: geom.RectPath(50, 0, 550, 700)},
			{Rune: 'A', Advance: 600, Path: geom.RectPath(50, 0, 550, 700)},
			{Rune: ' ', Advance: 300},
			{Rune: '中', Advance: 1000, Path: geom.RectPath(100, 0, 900, 700)},
		},
	}
}

func TestBuildContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := BuildContext(ctx, testConfig()); err == nil {
		t.Fatal("expected context cancellation")
	}
}

func TestBuildTTF(t *testing.T) {
	cfg := testConfig()
	data, err := Build(cfg)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(data)%4 != 0 {
		t.Fatalf("font length %d not 4-byte aligned", len(data))
	}
	f, err := sfnt.Parse(data)
	if err != nil {
		t.Fatalf("sfnt parse: %v", err)
	}
	if f.UnitsPerEm() != 1000 {
		t.Fatalf("upm = %d", f.UnitsPerEm())
	}
	if f.NumGlyphs() != 4 {
		t.Fatalf("glyphs = %d", f.NumGlyphs())
	}
	var buf sfnt.Buffer
	gid, err := f.GlyphIndex(&buf, 'A')
	if err != nil || gid != 1 {
		t.Fatalf("A gid = %d err=%v", gid, err)
	}
	adv, err := f.GlyphAdvance(&buf, gid, 1000, 0)
	if err != nil || adv != 600 {
		t.Fatalf("A advance = %v err=%v", adv, err)
	}
	segs, err := f.LoadGlyph(&buf, gid, 1000, nil)
	if err != nil || len(segs) == 0 {
		t.Fatalf("load A: segs=%d err=%v", len(segs), err)
	}
	bounds := segs.Bounds()
	if w := int(bounds.Max.X - bounds.Min.X); w != 500 {
		t.Fatalf("A width = %d, want 500 (coordinate sign encoding)", w)
	}
	metrics, err := f.Metrics(&buf, 1000, 0)
	if err != nil {
		t.Fatal(err)
	}
	if metrics.Ascent != 800 || metrics.Descent != 200 {
		t.Fatalf("metrics = %#v", metrics)
	}
	name, _ := f.Name(&buf, sfnt.NameIDFamily)
	if name != cfg.Family {
		t.Fatalf("family = %q", name)
	}
	sub, _ := f.Name(&buf, sfnt.NameIDSubfamily)
	if sub != cfg.Style {
		t.Fatalf("subfamily = %q", sub)
	}
	ps, _ := f.Name(&buf, sfnt.NameIDPostScript)
	if ps != "TestGrotesk-Regular" {
		t.Fatalf("postscript = %q", ps)
	}

	for _, tag := range []string{"head", "hhea", "maxp", "hmtx", "cmap", "loca", "glyf", "name", "OS/2", "post"} {
		if len(tableBytes(t, data, tag)) == 0 {
			t.Fatalf("missing table %s", tag)
		}
	}

	var sum uint32
	for i := 0; i+4 <= len(data); i += 4 {
		sum += binary.BigEndian.Uint32(data[i:])
	}
	if sum != 0xB1B0AFBA {
		t.Fatalf("font checksum = %#x, want 0xB1B0AFBA", sum)
	}
}

func TestItalicBits(t *testing.T) {
	cfg := testConfig()
	cfg.Style = "Italic"
	cfg.ItalicAngle = 8
	data, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	f, err := sfnt.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if f.PostTable() == nil || f.PostTable().ItalicAngle != -8 {
		t.Fatalf("post = %#v", f.PostTable())
	}
	os2 := tableBytes(t, data, "OS/2")
	if fsSelection := binary.BigEndian.Uint16(os2[62:]); fsSelection&1 == 0 {
		t.Fatalf("fsSelection = %#x, want italic bit", fsSelection)
	}
	head := tableBytes(t, data, "head")
	if macStyle := binary.BigEndian.Uint16(head[44:]); macStyle&2 == 0 {
		t.Fatalf("macStyle = %#x, want italic bit", macStyle)
	}
}

func TestRejectsDeltaOverflow(t *testing.T) {
	cfg := testConfig()
	cfg.Glyphs = append(cfg.Glyphs, Glyph{Rune: 'X', Advance: 600, Path: geom.RectPath(-32000, 0, 32000, 100)})
	if _, err := Build(cfg); err == nil {
		t.Fatal("expected delta overflow error")
	}
}

func TestRejectsCmapFormat4Overflow(t *testing.T) {
	cfg := testConfig()
	cfg.Glyphs = cfg.Glyphs[:1]
	for i := 0; i < 9000; i++ {
		cfg.Glyphs = append(cfg.Glyphs, Glyph{Rune: rune(0x1000 + i*2), Advance: 500})
	}
	if _, err := Build(cfg); err == nil {
		t.Fatal("expected cmap format 4 overflow error")
	}
}

func TestBuildPrependsNotdef(t *testing.T) {
	cfg := testConfig()
	cfg.Glyphs = cfg.Glyphs[1:]
	data, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	f, err := sfnt.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if f.NumGlyphs() != 4 {
		t.Fatalf("glyphs = %d, want 4", f.NumGlyphs())
	}
}

func TestBuildDeterministic(t *testing.T) {
	cfg := testConfig()
	a, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("builds differ")
	}
}

func TestBuildWOFF(t *testing.T) {
	cfg := testConfig()
	ttf, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	woff, err := BuildWOFF(ttf)
	if err != nil {
		t.Fatal(err)
	}
	if string(woff[:4]) != "wOFF" {
		t.Fatalf("signature = %q", woff[:4])
	}
	numTables := int(binary.BigEndian.Uint16(woff[12:]))
	if numTables != int(binary.BigEndian.Uint16(ttf[4:])) {
		t.Fatalf("woff tables = %d", numTables)
	}
	for i := 0; i < numTables; i++ {
		rec := woff[44+i*20:]
		tag := string(rec[:4])
		off := binary.BigEndian.Uint32(rec[4:])
		compLen := binary.BigEndian.Uint32(rec[8:])
		origLen := binary.BigEndian.Uint32(rec[12:])
		var got []byte
		if compLen == origLen {
			got = append([]byte(nil), woff[off:off+origLen]...)
		} else {
			raw, err := zlib.NewReader(bytes.NewReader(woff[off : off+compLen]))
			if err != nil {
				t.Fatalf("%s zlib: %v", tag, err)
			}
			got, err = io.ReadAll(raw)
			_ = raw.Close()
			if err != nil {
				t.Fatalf("%s read: %v", tag, err)
			}
		}
		if uint32(len(got)) != origLen {
			t.Fatalf("%s length %d, want %d", tag, len(got), origLen)
		}
		if tag == "glyf" || tag == "head" {
			if !bytes.Equal(got, tableBytes(t, ttf, tag)) {
				t.Fatalf("%s content mismatch", tag)
			}
		}
	}
}

func tableBytes(t *testing.T, data []byte, tag string) []byte {
	t.Helper()
	if len(data) < 12 {
		t.Fatal("short font")
	}
	n := int(binary.BigEndian.Uint16(data[4:]))
	for i := 0; i < n; i++ {
		rec := data[12+i*16:]
		if string(rec[:4]) != tag {
			continue
		}
		off := binary.BigEndian.Uint32(rec[8:])
		length := binary.BigEndian.Uint32(rec[12:])
		return data[off : off+length]
	}
	return nil
}
