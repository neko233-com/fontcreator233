// Package fontbuild writes deterministic TrueType and WOFF fonts without an
// external compiler.
package fontbuild

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/neko233-com/fontcreator233/internal/geom"
	"github.com/neko233-com/fontcreator233/internal/parallel"
)

// Glyph is one output glyph. Glyphs[0] should be .notdef; Build prepends one
// when it is missing.
type Glyph struct {
	Rune    rune
	Name    string
	Advance int
	Path    geom.Path
	NoCmap  bool
}

// KernPair is one GPOS/kern pair adjustment.
type KernPair struct {
	Left  string
	Right string
	Value int16
}

// Ligature maps a component sequence to one glyph.
type Ligature struct {
	Feature    string
	Components []string
	Glyph      string
}

// Substitution is a one-for-one GSUB replacement for a feature tag.
type Substitution struct {
	Feature string
	From    string
	To      string
}

// ContextualRule invokes a ligature only when the exact input sequence is
// present. It is emitted as a real GSUB chaining-context lookup (type 6).
// Backtrack and lookahead glyph sequences optionally constrain the context.
type ContextualRule struct {
	Feature   string
	Backtrack []string
	Input     []string
	Lookahead []string
	Glyph     string
}

// Config is the complete font metadata and glyph set.
type Config struct {
	Family      string
	Style       string
	Version     string
	Designer    string
	Description string
	License     string
	LicenseURL  string
	Copyright   string
	SampleText  string

	Ascender  int
	Descender int
	LineGap   int
	CapHeight int
	XHeight   int

	WeightClass   int
	WidthClass    int
	ItalicAngle   float64
	FixedPitch    bool
	Created       time.Time
	Jobs          int
	Variable      bool
	Cursive       bool
	Accent        string
	Color         *ColorGlyph
	ColorSet      *ColorSet
	Glyphs        []Glyph
	Kerning       []KernPair
	Ligatures     []Ligature
	Substitutions []Substitution
	Contextual    []ContextualRule
}

type glyphBox struct {
	empty        bool
	xMin, yMin   int16
	xMax, yMax   int16
	advance, lsb int
}

type table struct {
	tag  string
	data []byte
}

// Build compiles a TrueType font.
func Build(cfg Config) ([]byte, error) {
	return BuildContext(context.Background(), cfg)
}

// BuildContext is Build with cooperative cancellation between glyphs and
// tables.
func BuildContext(ctx context.Context, cfg Config) ([]byte, error) {
	cfg = normalizeConfig(cfg)
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	glyphs := cfg.Glyphs
	type encResult struct {
		data []byte
		box  glyphBox
	}
	results, err := parallel.Map(ctx, glyphs, cfg.Jobs, func(ctx context.Context, i int, g Glyph) (encResult, error) {
		if err := ctx.Err(); err != nil {
			return encResult{}, err
		}
		data, box, err := encodeGlyph(g)
		if err != nil {
			return encResult{}, fmt.Errorf("glyph %d (%U): %w", i, g.Rune, err)
		}
		return encResult{data: data, box: box}, nil
	})
	if err != nil {
		return nil, err
	}
	encoded := make([][]byte, len(glyphs))
	boxes := make([]glyphBox, len(glyphs))
	var maxPoints, maxContours, maxAdvance int
	var fontBox glyphBox
	fontBox.empty = true
	for i, g := range glyphs {
		box := results[i].box
		if g.Advance < 0 || g.Advance > 65535 {
			return nil, fmt.Errorf("glyph %U advance %d out of range", g.Rune, g.Advance)
		}
		box.advance = g.Advance
		if !box.empty {
			box.lsb = int(box.xMin)
		}
		encoded[i] = results[i].data
		boxes[i] = box
		if box.advance > maxAdvance {
			maxAdvance = box.advance
		}
		points, contours := countPoints(g.Path)
		if points > maxPoints {
			maxPoints = points
		}
		if contours > maxContours {
			maxContours = contours
		}
		if !box.empty {
			if fontBox.empty {
				fontBox = box
				fontBox.empty = false
			} else {
				if box.xMin < fontBox.xMin {
					fontBox.xMin = box.xMin
				}
				if box.yMin < fontBox.yMin {
					fontBox.yMin = box.yMin
				}
				if box.xMax > fontBox.xMax {
					fontBox.xMax = box.xMax
				}
				if box.yMax > fontBox.yMax {
					fontBox.yMax = box.yMax
				}
			}
		}
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	glyf, loca := buildGlyf(encoded)
	cmapData, err := buildCmap(glyphs)
	if err != nil {
		return nil, err
	}
	tables := []table{
		{"OS/2", buildOS2(cfg, glyphs, fontBox)},
		{"cmap", cmapData},
		{"glyf", glyf},
		{"head", buildHead(cfg, fontBox)},
		{"hhea", buildHhea(cfg, boxes, maxAdvance)},
		{"hmtx", buildHmtx(boxes)},
		{"loca", loca},
		{"maxp", buildMaxp(len(glyphs), maxPoints, maxContours)},
		{"name", buildName(cfg)},
		{"post", buildPost(cfg)},
	}
	hasMarks := hasMarkGlyphs(glyphs)
	if hasMarks {
		tables = append(tables, table{"GDEF", buildGDEF(glyphs)})
	}
	if len(cfg.Kerning) > 0 {
		if kern, err := buildKernTable(glyphs, cfg.Kerning); err != nil {
			return nil, err
		} else if len(kern) > 0 {
			tables = append(tables, table{"kern", kern})
		}
	}
	if len(cfg.Kerning) > 0 || hasMarks || cfg.Cursive {
		if gpos, err := buildGPOSTable(glyphs, cfg.Kerning, cfg.CapHeight, cfg.XHeight, cfg.Cursive); err != nil {
			return nil, err
		} else if len(gpos) > 0 {
			tables = append(tables, table{"GPOS", gpos})
		}
	}
	if len(cfg.Ligatures) > 0 || len(cfg.Substitutions) > 0 || len(cfg.Contextual) > 0 {
		if gsub, err := buildGSUBTable(glyphs, cfg.Ligatures, cfg.Substitutions, cfg.Contextual); err != nil {
			return nil, err
		} else if len(gsub) > 0 {
			tables = append(tables, table{"GSUB", gsub})
		}
	}
	if cfg.ColorSet != nil && len(cfg.ColorSet.Glyphs) > 0 {
		if colr := buildCOLRMulti(glyphs, cfg.ColorSet.Glyphs); len(colr) > 0 {
			tables = append(tables, table{"COLR", colr})
			tables = append(tables, table{"CPAL", buildCPALMulti(cfg.ColorSet.Palettes)})
		}
	} else if cfg.Color != nil {
		if colr := buildCOLR(glyphs, cfg.Color); len(colr) > 0 {
			tables = append(tables, table{"COLR", colr})
			tables = append(tables, table{"CPAL", buildCPAL(cfg.Accent)})
		}
	}
	if cfg.Variable {
		gvar, err := buildGvar(glyphs)
		if err != nil {
			return nil, err
		}
		if len(gvar) > 0 {
			tables = append(tables, table{"fvar", buildFvar(cfg)})
			tables = append(tables, table{"gvar", gvar})
			if hvar := buildHVAR(glyphs); len(hvar) > 0 {
				tables = append(tables, table{"HVAR", hvar})
			}
			if mvar := buildMVAR(cfg); len(mvar) > 0 {
				tables = append(tables, table{"MVAR", mvar})
			}
		}
	}
	sort.Slice(tables, func(i, j int) bool { return tables[i].tag < tables[j].tag })
	out, headOffset, err := assemble(tables)
	if err != nil {
		return nil, err
	}
	if headOffset < 0 {
		return nil, fmt.Errorf("head table missing")
	}
	var sum uint32
	for i := 0; i+4 <= len(out); i += 4 {
		sum += binary.BigEndian.Uint32(out[i:])
	}
	adjustment := uint32(0xB1B0AFBA) - sum
	binary.BigEndian.PutUint32(out[headOffset+8:], adjustment)
	return out, nil
}

func normalizeConfig(cfg Config) Config {
	if len(cfg.Glyphs) == 0 || cfg.Glyphs[0].Rune != 0 {
		notdef := Glyph{Rune: 0, Advance: 600, Path: notdefPath()}
		cfg.Glyphs = append([]Glyph{notdef}, cfg.Glyphs...)
	}
	if len(cfg.Glyphs[0].Path) == 0 {
		cfg.Glyphs[0].Path = notdefPath()
	}
	if cfg.Version == "" {
		cfg.Version = "1.0.0"
	}
	if cfg.License == "" {
		cfg.License = "OFL-1.1"
	}
	if cfg.LicenseURL == "" && cfg.License == "OFL-1.1" {
		cfg.LicenseURL = "https://scripts.sil.org/OFL"
	}
	if cfg.Copyright == "" {
		cfg.Copyright = "Copyright 2026 fontcreator233"
	}
	if cfg.Created.IsZero() {
		cfg.Created = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if cfg.WeightClass == 0 {
		cfg.WeightClass = 400
	}
	if cfg.WidthClass == 0 {
		cfg.WidthClass = 5
	}
	for i := range cfg.Glyphs {
		if cfg.Glyphs[i].Name == "" {
			cfg.Glyphs[i].Name = fmt.Sprintf("gid%05d", i)
		}
	}
	return cfg
}

func validateConfig(cfg Config) error {
	if strings.TrimSpace(cfg.Family) == "" {
		return fmt.Errorf("empty family")
	}
	if len(cfg.Glyphs) > 65535 {
		return fmt.Errorf("too many glyphs: %d", len(cfg.Glyphs))
	}
	if cfg.Ascender < -32768 || cfg.Ascender > 32767 || cfg.Descender < -32768 || cfg.Descender > 32767 {
		return fmt.Errorf("vertical metrics out of int16 range")
	}
	if cfg.WeightClass < 1 || cfg.WeightClass > 1000 {
		return fmt.Errorf("weight class %d out of range", cfg.WeightClass)
	}
	if cfg.WidthClass < 1 || cfg.WidthClass > 9 {
		return fmt.Errorf("width class %d out of range", cfg.WidthClass)
	}
	seen := map[rune]bool{}
	names := map[string]bool{}
	for i, g := range cfg.Glyphs {
		if g.Name == "" {
			g.Name = fmt.Sprintf("gid%05d", i)
		}
		if names[g.Name] {
			return fmt.Errorf("duplicate glyph name %q", g.Name)
		}
		names[g.Name] = true
		if !g.NoCmap && g.Rune != 0 && seen[g.Rune] {
			return fmt.Errorf("duplicate rune %U", g.Rune)
		}
		if !g.NoCmap {
			seen[g.Rune] = true
		}
	}
	return nil
}

func notdefPath() geom.Path {
	outer := geom.RectPath(50, 0, 550, 700)
	inner := geom.RectPath(120, 80, 480, 620).Reverse(0)
	return append(outer, inner...)
}

func countPoints(p geom.Path) (int, int) {
	points, contours := 0, 0
	for _, c := range p {
		if len(c) == 0 {
			continue
		}
		contours++
		points += len(c)
	}
	return points, contours
}

func buildGlyf(encoded [][]byte) (glyf, loca []byte) {
	offsets := make([]uint32, len(encoded)+1)
	var buf []byte
	for i, data := range encoded {
		offsets[i] = uint32(len(buf))
		buf = append(buf, data...)
		for len(buf)%4 != 0 {
			buf = append(buf, 0)
		}
	}
	offsets[len(encoded)] = uint32(len(buf))
	loca = make([]byte, 4*len(offsets))
	for i, off := range offsets {
		binary.BigEndian.PutUint32(loca[i*4:], off)
	}
	return buf, loca
}

func assemble(tables []table) ([]byte, int, error) {
	n := len(tables)
	headerLen := 12 + 16*n
	offset := headerLen
	offsets := make([]int, n)
	total := headerLen
	for i, t := range tables {
		offsets[i] = offset
		offset += len(t.data)
		for offset%4 != 0 {
			offset++
		}
		total = offset
	}
	out := make([]byte, total)
	binary.BigEndian.PutUint32(out[0:], 0x00010000)
	binary.BigEndian.PutUint16(out[4:], uint16(n))
	searchRange, entrySelector := 16, 0
	for (1 << (entrySelector + 1)) <= n {
		entrySelector++
		searchRange *= 2
	}
	binary.BigEndian.PutUint16(out[6:], uint16(searchRange))
	binary.BigEndian.PutUint16(out[8:], uint16(entrySelector))
	binary.BigEndian.PutUint16(out[10:], uint16(16*n-searchRange))
	headOffset := -1
	for i, t := range tables {
		rec := out[12+i*16:]
		copy(rec[:4], t.tag)
		binary.BigEndian.PutUint32(rec[4:], checksum(t.data))
		binary.BigEndian.PutUint32(rec[8:], uint32(offsets[i]))
		binary.BigEndian.PutUint32(rec[12:], uint32(len(t.data)))
		copy(out[offsets[i]:], t.data)
		if t.tag == "head" {
			headOffset = offsets[i]
		}
	}
	return out, headOffset, nil
}

func checksum(data []byte) uint32 {
	var sum uint32
	for i := 0; i < len(data); i += 4 {
		var word [4]byte
		copy(word[:], data[i:minInt(i+4, len(data))])
		sum += binary.BigEndian.Uint32(word[:])
	}
	return sum
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func appendU16(b []byte, v uint16) []byte {
	return append(b, byte(v>>8), byte(v))
}

func appendI16(b []byte, v int16) []byte {
	return appendU16(b, uint16(v))
}

func appendU32(b []byte, v uint32) []byte {
	return append(b, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

func putU16(b []byte, off int, v uint16) { binary.BigEndian.PutUint16(b[off:], v) }
func putI16(b []byte, off int, v int16)  { binary.BigEndian.PutUint16(b[off:], uint16(v)) }
func putU32(b []byte, off int, v uint32) { binary.BigEndian.PutUint32(b[off:], v) }

func roundInt16(v float64) (int16, error) {
	r := math.Round(v)
	if r < -32768 || r > 32767 {
		return 0, fmt.Errorf("coordinate %v out of int16 range", v)
	}
	return int16(r), nil
}

func utf16be(s string) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, 2*len(units))
	for i, u := range units {
		binary.BigEndian.PutUint16(out[i*2:], u)
	}
	return out
}
