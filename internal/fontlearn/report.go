// Package fontlearn parses arbitrary TrueType/OpenType files into a readable
// design report. It does not modify or embed the source font.
package fontlearn

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode/utf16"

	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// TableInfo is one sfnt table directory entry.
type TableInfo struct {
	Tag      string `json:"tag"`
	Offset   uint32 `json:"offset"`
	Length   uint32 `json:"length"`
	Checksum uint32 `json:"checksum"`
}

// NameRecord is one decoded name-table string.
type NameRecord struct {
	Platform uint16 `json:"platform"`
	Encoding uint16 `json:"encoding"`
	Language uint16 `json:"language"`
	ID       uint16 `json:"id"`
	Value    string `json:"value"`
}

// LayoutFeature is one GSUB/GPOS feature, including its lookups.
type LayoutFeature struct {
	Table      string   `json:"table"`
	Tag        string   `json:"tag"`
	Lookups    []uint16 `json:"lookups"`
	LookupType []uint16 `json:"lookupTypes"`
}

// Axis is one fvar variation axis.
type Axis struct {
	Tag     string  `json:"tag"`
	Min     float64 `json:"min"`
	Default float64 `json:"default"`
	Max     float64 `json:"max"`
	Flags   uint16  `json:"flags"`
	NameID  uint16  `json:"nameId"`
}

// KernPair is one legacy kern pair.
type KernPair struct {
	Left  uint16 `json:"left"`
	Right uint16 `json:"right"`
	Value int16  `json:"value"`
}

// GlyphStats summarizes a sample of outlines.
type GlyphStats struct {
	Sampled       int `json:"sampled"`
	MaxContours   int `json:"maxContours"`
	MaxPoints     int `json:"maxPoints"`
	Composite     int `json:"composite"`
	EmptyOutlines int `json:"emptyOutlines"`
}

// OutlineAnalysis holds inferred style signals from outline sampling.
type OutlineAnalysis struct {
	Samples      int     `json:"samples"`
	Contrast     float64 `json:"contrast"`     // 0 uniform .. 0.9+ Didone/hairline
	Serif        string  `json:"serif"`        // none|slab|bracketed|hairline
	Terminal     string  `json:"terminal"`     // flat|round|angled|flare
	Skeleton     string  `json:"skeleton"`     // geometric|humanist|serif
	MeanStem     float64 `json:"meanStem"`     // average stem width in UPM
	StemCV       float64 `json:"stemCV"`       // coefficient of variation of stems
	EndOvershoot float64 `json:"endOvershoot"` // (end-width / mid-width) - 1, >=0 means flare/serif
	Done         bool    `json:"done"`
}

// CmapSubtable describes one cmap record.
type CmapSubtable struct {
	Platform   uint16 `json:"platform"`
	Encoding   uint16 `json:"encoding"`
	Format     uint16 `json:"format"`
	GlyphCount int    `json:"glyphCount"`
}

// Report is the complete learning report for one font.
type Report struct {
	File          string          `json:"file"`
	Family        string          `json:"family"`
	Style         string          `json:"style"`
	Version       string          `json:"version"`
	License       string          `json:"license"`
	UnitsPerEm    int             `json:"unitsPerEm"`
	Glyphs        int             `json:"glyphs"`
	WeightClass   int             `json:"weightClass"`
	WidthClass    int             `json:"widthClass"`
	ItalicAngle   float64         `json:"italicAngle"`
	Ascender      int             `json:"ascender"`
	Descender     int             `json:"descender"`
	LineGap       int             `json:"lineGap"`
	XHeight       int             `json:"xHeight"`
	CapHeight     int             `json:"capHeight"`
	BBox          [4]int16        `json:"bbox"`
	Tables        []TableInfo     `json:"tables"`
	Names         []NameRecord    `json:"names"`
	Features      []LayoutFeature `json:"features"`
	Scripts       []string        `json:"scripts"`
	Kerning       int             `json:"kerningPairs"`
	SampleKern    []KernPair      `json:"sampleKerning,omitempty"`
	Cmap          []CmapSubtable  `json:"cmap"`
	Axes          []Axis          `json:"axes,omitempty"`
	HasColor      bool            `json:"hasColor"`
	HasVariable   bool            `json:"hasVariable"`
	GlyphStats    GlyphStats      `json:"glyphStats"`
	Outline       OutlineAnalysis `json:"outline,omitempty"`
	Coverage      map[string]int  `json:"coverage"`
	CoverageTotal int             `json:"coverageTotal"`
	Warnings      []string        `json:"warnings,omitempty"`
	Suggested     Suggested       `json:"suggested"`
}

// Suggested is a first-pass recipe suggestion; visual judgment stays with the
// agent.
type Suggested struct {
	Skeleton string  `json:"skeleton"`
	Weight   int     `json:"weight"`
	Width    float64 `json:"width"`
	Slant    float64 `json:"slant"`
	Serif    string  `json:"serif"`
	Note     string  `json:"note"`
}

// Analyze reads a TTF/OTF byte stream.
func Analyze(filename string, data []byte) (Report, error) {
	tables, err := parseTables(data)
	if err != nil {
		return Report{}, err
	}
	byTag := map[string][]byte{}
	for _, t := range tables {
		if int(t.Offset)+int(t.Length) <= len(data) {
			byTag[t.Tag] = data[t.Offset : t.Offset+t.Length]
		}
	}
	f, err := sfnt.Parse(data)
	if err != nil {
		return Report{}, fmt.Errorf("parse sfnt: %w", err)
	}
	r := Report{
		File:       filename,
		UnitsPerEm: int(f.UnitsPerEm()),
		Glyphs:     f.NumGlyphs(),
		Tables:     tables,
		Coverage:   map[string]int{},
		Cmap:       parseCmap(byTag["cmap"]),
	}
	var buf sfnt.Buffer
	r.Family, _ = f.Name(&buf, sfnt.NameIDFamily)
	r.Style, _ = f.Name(&buf, sfnt.NameIDSubfamily)
	r.Version, _ = f.Name(&buf, sfnt.NameIDVersion)
	r.License, _ = f.Name(&buf, sfnt.NameIDLicense)
	if pt := f.PostTable(); pt != nil {
		r.ItalicAngle = pt.ItalicAngle
	}
	parseNames(byTag["name"], &r)
	parseOS2(byTag["OS/2"], &r)
	parseHead(byTag["head"], &r)
	parseHhea(byTag["hhea"], &r)
	parseMaxp(byTag["maxp"], &r)
	parsePost(byTag["post"], &r)
	gsubFeatures, gsubScripts := parseLayout(byTag["GSUB"], "GSUB")
	r.Features = append(r.Features, gsubFeatures...)
	r.Scripts = append(r.Scripts, gsubScripts...)
	gposFeatures, gposScripts := parseLayout(byTag["GPOS"], "GPOS")
	r.Features = append(r.Features, gposFeatures...)
	r.Scripts = append(r.Scripts, gposScripts...)
	r.Features = mergeFeatures(r.Features)
	r.Scripts = dedupeStrings(r.Scripts)
	sort.Strings(r.Scripts)
	r.Kerning, r.SampleKern = parseKern(byTag["kern"])
	r.Axes = parseFvar(byTag["fvar"])
	r.HasVariable = len(r.Axes) > 0 || byTag["gvar"] != nil || byTag["HVAR"] != nil || byTag["MVAR"] != nil
	for _, tag := range []string{"COLR", "CPAL", "SVG ", "sbix", "CBDT", "CBLC"} {
		if byTag[tag] != nil {
			r.HasColor = true
		}
	}
	if r.WeightClass == 0 {
		r.WeightClass = 400
	}
	if r.WidthClass == 0 {
		r.WidthClass = 5
	}
	scanGlyphs(f, &buf, &r)
	r.Suggested = suggest(&r)
	// Outline analysis needs the same sfnt.Font handle; reuse buf state.
	// Skip when UPM or xHeight/ascender are missing from the input font.
	if r.UnitsPerEm > 0 && r.XHeight > 0 && r.Ascender > 0 {
		r.Outline = analyzeOutline(f, &buf, r.UnitsPerEm, r.XHeight, r.CapHeight, r.Ascender)
		r.Suggested = suggest(&r) // re-evaluate with outline hints
	}
	if r.Family == "" {
		r.Warnings = append(r.Warnings, "name table has no family name")
	}
	return r, nil
}

func parseTables(data []byte) ([]TableInfo, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("font is too small")
	}
	n := int(binary.BigEndian.Uint16(data[4:]))
	if n < 0 || 12+n*16 > len(data) {
		return nil, fmt.Errorf("invalid sfnt table directory")
	}
	out := make([]TableInfo, 0, n)
	for i := 0; i < n; i++ {
		base := 12 + i*16
		t := TableInfo{
			Tag:      string(data[base : base+4]),
			Checksum: binary.BigEndian.Uint32(data[base+4:]),
			Offset:   binary.BigEndian.Uint32(data[base+8:]),
			Length:   binary.BigEndian.Uint32(data[base+12:]),
		}
		if int(t.Offset)+int(t.Length) > len(data) {
			return nil, fmt.Errorf("table %s exceeds file size", t.Tag)
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Tag < out[j].Tag })
	return out, nil
}

func parseNames(data []byte, r *Report) {
	if len(data) < 6 {
		return
	}
	count := int(binary.BigEndian.Uint16(data[2:]))
	stringOffset := int(binary.BigEndian.Uint16(data[4:]))
	if 6+12*count > len(data) {
		return
	}
	for i := 0; i < count; i++ {
		base := 6 + i*12
		platform := binary.BigEndian.Uint16(data[base:])
		encoding := binary.BigEndian.Uint16(data[base+2:])
		language := binary.BigEndian.Uint16(data[base+4:])
		id := binary.BigEndian.Uint16(data[base+6:])
		length := int(binary.BigEndian.Uint16(data[base+8:]))
		off := int(binary.BigEndian.Uint16(data[base+10:]))
		start := stringOffset + off
		if start < 0 || length < 0 || start+length > len(data) {
			continue
		}
		value := decodeName(platform, encoding, data[start:start+length])
		if value == "" {
			continue
		}
		r.Names = append(r.Names, NameRecord{Platform: platform, Encoding: encoding, Language: language, ID: id, Value: value})
	}
	sort.Slice(r.Names, func(i, j int) bool {
		if r.Names[i].ID != r.Names[j].ID {
			return r.Names[i].ID < r.Names[j].ID
		}
		if r.Names[i].Platform != r.Names[j].Platform {
			return r.Names[i].Platform < r.Names[j].Platform
		}
		return r.Names[i].Language < r.Names[j].Language
	})
}

func decodeName(platform, encoding uint16, data []byte) string {
	if platform == 0 || platform == 3 {
		if len(data)%2 != 0 {
			data = data[:len(data)-1]
		}
		u := make([]uint16, len(data)/2)
		for i := range u {
			u[i] = binary.BigEndian.Uint16(data[i*2:])
		}
		return strings.TrimSpace(string(utf16.Decode(u)))
	}
	var b strings.Builder
	for _, c := range data {
		if c >= 32 && c < 127 || c >= 160 {
			b.WriteRune(rune(c))
		}
	}
	return strings.TrimSpace(b.String())
}

func parseOS2(data []byte, r *Report) {
	if len(data) < 6 {
		return
	}
	r.WeightClass = int(binary.BigEndian.Uint16(data[4:]))
	r.WidthClass = int(binary.BigEndian.Uint16(data[6:]))
	if len(data) >= 74 {
		r.Ascender = int(int16(binary.BigEndian.Uint16(data[68:])))
		r.Descender = int(int16(binary.BigEndian.Uint16(data[70:])))
		r.LineGap = int(int16(binary.BigEndian.Uint16(data[72:])))
	}
	if len(data) >= 90 {
		r.XHeight = int(int16(binary.BigEndian.Uint16(data[86:])))
		r.CapHeight = int(int16(binary.BigEndian.Uint16(data[88:])))
	}
}

func parseHead(data []byte, r *Report) {
	if len(data) < 54 {
		return
	}
	r.UnitsPerEm = int(binary.BigEndian.Uint16(data[18:]))
	r.BBox = [4]int16{
		int16(binary.BigEndian.Uint16(data[36:])),
		int16(binary.BigEndian.Uint16(data[38:])),
		int16(binary.BigEndian.Uint16(data[40:])),
		int16(binary.BigEndian.Uint16(data[42:])),
	}
}

func parseHhea(data []byte, r *Report) {
	if len(data) < 36 {
		return
	}
	if r.Ascender == 0 {
		r.Ascender = int(int16(binary.BigEndian.Uint16(data[4:])))
	}
	if r.Descender == 0 {
		r.Descender = int(int16(binary.BigEndian.Uint16(data[6:])))
	}
	if r.LineGap == 0 {
		r.LineGap = int(int16(binary.BigEndian.Uint16(data[8:])))
	}
}

func parseMaxp(data []byte, r *Report) {
	if len(data) < 6 {
		return
	}
	r.Glyphs = int(binary.BigEndian.Uint16(data[4:]))
}

func parsePost(data []byte, r *Report) {
	if len(data) < 8 {
		return
	}
	angle := int32(binary.BigEndian.Uint32(data[4:]))
	r.ItalicAngle = float64(angle) / 65536
}

func parseLayout(data []byte, table string) ([]LayoutFeature, []string) {
	if len(data) < 10 {
		return nil, nil
	}
	scriptOff := int(binary.BigEndian.Uint16(data[4:]))
	featureOff := int(binary.BigEndian.Uint16(data[6:]))
	lookupOff := int(binary.BigEndian.Uint16(data[8:]))
	var scripts []string
	if scriptOff >= 0 && scriptOff+2 <= len(data) {
		n := int(binary.BigEndian.Uint16(data[scriptOff:]))
		for i := 0; i < n; i++ {
			rec := scriptOff + 2 + i*6
			if rec+6 > len(data) {
				break
			}
			scripts = append(scripts, string(data[rec:rec+4]))
		}
	}
	var lookupTypes []uint16
	if lookupOff >= 0 && lookupOff+2 <= len(data) {
		n := int(binary.BigEndian.Uint16(data[lookupOff:]))
		for i := 0; i < n; i++ {
			rec := lookupOff + 2 + i*2
			if rec+2 > len(data) {
				break
			}
			off := lookupOff + int(binary.BigEndian.Uint16(data[rec:]))
			if off+2 <= len(data) {
				lookupTypes = append(lookupTypes, binary.BigEndian.Uint16(data[off:]))
			} else {
				lookupTypes = append(lookupTypes, 0)
			}
		}
	}
	if featureOff < 0 || featureOff+2 > len(data) {
		return nil, scripts
	}
	n := int(binary.BigEndian.Uint16(data[featureOff:]))
	var out []LayoutFeature
	for i := 0; i < n; i++ {
		rec := featureOff + 2 + i*6
		if rec+6 > len(data) {
			break
		}
		tag := string(data[rec : rec+4])
		off := featureOff + int(binary.BigEndian.Uint16(data[rec+4:]))
		if off+4 > len(data) {
			continue
		}
		lookupCount := int(binary.BigEndian.Uint16(data[off+2:]))
		f := LayoutFeature{Table: table, Tag: tag}
		for j := 0; j < lookupCount; j++ {
			pos := off + 4 + j*2
			if pos+2 > len(data) {
				break
			}
			idx := binary.BigEndian.Uint16(data[pos:])
			f.Lookups = append(f.Lookups, idx)
			if int(idx) < len(lookupTypes) {
				f.LookupType = append(f.LookupType, lookupTypes[idx])
			}
		}
		out = append(out, f)
	}
	return out, scripts
}

func parseKern(data []byte) (int, []KernPair) {
	if len(data) < 4 {
		return 0, nil
	}
	tables := int(binary.BigEndian.Uint16(data[2:]))
	off := 4
	total := 0
	var sample []KernPair
	for i := 0; i < tables && off+6 <= len(data); i++ {
		version := binary.BigEndian.Uint16(data[off:])
		length := int(binary.BigEndian.Uint16(data[off+2:]))
		coverage := binary.BigEndian.Uint16(data[off+4:])
		if version != 0 || length < 6 || off+length > len(data) {
			break
		}
		format := coverage >> 8
		if format == 0 && length >= 14 {
			n := int(binary.BigEndian.Uint16(data[off+6:]))
			if n > 0 && off+14+6*n <= len(data) {
				total += n
				for j := 0; j < n && len(sample) < 20; j++ {
					base := off + 14 + j*6
					sample = append(sample, KernPair{
						Left:  binary.BigEndian.Uint16(data[base:]),
						Right: binary.BigEndian.Uint16(data[base+2:]),
						Value: int16(binary.BigEndian.Uint16(data[base+4:])),
					})
				}
			}
		}
		off += length
	}
	return total, sample
}

func parseFvar(data []byte) []Axis {
	if len(data) < 16 {
		return nil
	}
	axisOffset := int(binary.BigEndian.Uint16(data[4:]))
	axisCount := int(binary.BigEndian.Uint16(data[8:]))
	axisSize := int(binary.BigEndian.Uint16(data[10:]))
	if axisSize < 20 || axisOffset < 0 || axisOffset+axisCount*axisSize > len(data) {
		return nil
	}
	var axes []Axis
	for i := 0; i < axisCount; i++ {
		base := axisOffset + i*axisSize
		axes = append(axes, Axis{
			Tag:     string(data[base : base+4]),
			Min:     fixed16(data[base+4:]),
			Default: fixed16(data[base+8:]),
			Max:     fixed16(data[base+12:]),
			Flags:   binary.BigEndian.Uint16(data[base+16:]),
			NameID:  binary.BigEndian.Uint16(data[base+18:]),
		})
	}
	return axes
}

func fixed16(data []byte) float64 {
	return float64(int32(binary.BigEndian.Uint32(data))) / 65536
}

func parseCmap(data []byte) []CmapSubtable {
	if len(data) < 4 {
		return nil
	}
	n := int(binary.BigEndian.Uint16(data[2:]))
	var out []CmapSubtable
	for i := 0; i < n; i++ {
		rec := 4 + i*8
		if rec+8 > len(data) {
			break
		}
		platform := binary.BigEndian.Uint16(data[rec:])
		encoding := binary.BigEndian.Uint16(data[rec+2:])
		off := int(binary.BigEndian.Uint32(data[rec+4:]))
		if off+2 > len(data) {
			continue
		}
		format := binary.BigEndian.Uint16(data[off:])
		sub := CmapSubtable{Platform: platform, Encoding: encoding, Format: format}
		switch format {
		case 4:
			if off+8 <= len(data) {
				sub.GlyphCount = int(binary.BigEndian.Uint16(data[off+6:])) / 2
			}
		case 6:
			if off+10 <= len(data) {
				sub.GlyphCount = int(binary.BigEndian.Uint16(data[off+8:]))
			}
		case 12:
			if off+16 <= len(data) {
				sub.GlyphCount = int(binary.BigEndian.Uint32(data[off+12:]))
			}
		}
		out = append(out, sub)
	}
	return out
}

func scanGlyphs(f *sfnt.Font, buf *sfnt.Buffer, r *Report) {
	var sampled int
	for runeValue := rune(0); runeValue <= 0x323AF; runeValue++ {
		if runeValue >= 0xD800 && runeValue <= 0xDFFF {
			continue
		}
		gid, err := f.GlyphIndex(buf, runeValue)
		if err != nil {
			continue
		}
		if gid == 0 {
			continue
		}
		r.Coverage[scriptOf(runeValue)]++
		r.CoverageTotal++
		if sampled >= 3000 {
			continue
		}
		segs, err := f.LoadGlyph(buf, gid, fixed.Int26_6(f.UnitsPerEm()), nil)
		if err != nil {
			continue
		}
		sampled++
		if len(segs) == 0 {
			r.GlyphStats.EmptyOutlines++
		}
		points := 0
		for _, seg := range segs {
			if seg.Op != 0 {
				points++
			}
		}
		if len(segs) > r.GlyphStats.MaxContours {
			r.GlyphStats.MaxContours = len(segs)
		}
		if points > r.GlyphStats.MaxPoints {
			r.GlyphStats.MaxPoints = points
		}
	}
	r.GlyphStats.Sampled = sampled
}

func scriptOf(r rune) string {
	switch {
	case r >= 0x0370 && r <= 0x03FF, r >= 0x1F00 && r <= 0x1FFF:
		return "greek"
	case r >= 0x0400 && r <= 0x052F:
		return "cyrillic"
	case r >= 0x1100 && r <= 0x11FF, r >= 0x3130 && r <= 0x318F, r >= 0xA960 && r <= 0xA97F, r >= 0xAC00 && r <= 0xD7AF:
		return "hangul"
	case r >= 0x3040 && r <= 0x30FF, r >= 0x31F0 && r <= 0x31FF:
		return "kana"
	case r >= 0x2E80 && r <= 0x9FFF, r >= 0xF900 && r <= 0xFAFF, r >= 0x20000:
		return "cjk"
	case r >= 0x0020 && r <= 0x024F, r >= 0x1E00 && r <= 0x1EFF:
		return "latin"
	case r >= 0x2000 && r <= 0x27BF:
		return "symbols"
	default:
		return "other"
	}
}

func suggest(r *Report) Suggested {
	s := Suggested{Weight: r.WeightClass, Width: 100, Slant: math.Abs(r.ItalicAngle), Serif: "none", Skeleton: "geometric"}
	switch r.WidthClass {
	case 1, 2:
		s.Width = 82
	case 3:
		s.Width = 92
	case 4:
		s.Width = 96
	case 6, 7:
		s.Width = 108
	case 8, 9:
		s.Width = 118
	}
	if r.ItalicAngle > 0 {
		s.Serif = "hairline"
		s.Skeleton = "serif"
	}
	if r.Outline.Done {
		s.Skeleton = r.Outline.Skeleton
		s.Serif = r.Outline.Serif
		// s.Contrast is not a Suggested field; we add a Note instead.
		// Terminal also lives in outline; we surface it via Note.
		s.Note = fmt.Sprintf("Stem mean=%.0f CV=%.2f → contrast=%.2f, serif=%s, terminal=%s, skeleton=%s.",
			r.Outline.MeanStem, r.Outline.StemCV, r.Outline.Contrast,
			r.Outline.Serif, r.Outline.Terminal, r.Outline.Skeleton)
	} else {
		s.Note = "Inferred from metrics only; inspect the reference for skeleton, contrast, serif, and terminals."
	}
	return s
}

func mergeFeatures(in []LayoutFeature) []LayoutFeature {
	type key struct{ table, tag string }
	index := map[key]int{}
	var out []LayoutFeature
	for _, f := range in {
		k := key{f.Table, f.Tag}
		if i, ok := index[k]; ok {
			out[i].Lookups = appendUniqueU16(out[i].Lookups, f.Lookups...)
			out[i].LookupType = appendUniqueU16(out[i].LookupType, f.LookupType...)
			continue
		}
		index[k] = len(out)
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Table != out[j].Table {
			return out[i].Table < out[j].Table
		}
		return out[i].Tag < out[j].Tag
	})
	return out
}

func appendUniqueU16(dst []uint16, values ...uint16) []uint16 {
	seen := map[uint16]bool{}
	for _, v := range dst {
		seen[v] = true
	}
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			dst = append(dst, v)
		}
	}
	return dst
}

func dedupeStrings(in []string) []string {
	seen := map[string]bool{}
	out := in[:0]
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// Axes returns the fvar axes without scanning glyph coverage. It is used by
// the specimen command to expose variation sliders for external fonts.
func Axes(data []byte) []Axis {
	tables, err := parseTables(data)
	if err != nil {
		return nil
	}
	for _, t := range tables {
		if t.Tag == "fvar" && int(t.Offset)+int(t.Length) <= len(data) {
			return parseFvar(data[t.Offset : t.Offset+t.Length])
		}
	}
	return nil
}

// Target glyphs for outline style sampling — uppercase for stem width,
// lowercase a/g for skeleton hint.
var latinTargetGlyphs = []rune{
	'H', 'I', 'M', 'N', 'U', 'h', 'n', 'o', 'O', 'l', 't', 'E', 'F',
}

type lineSeg struct{ x1, y1, x2, y2 float64 }

// collectEndpoints gathers every outline endpoint as (x, y). TrueType uses
// quadratic Bézier with Args[0] = endpoint, Args[1] = control; CFF uses cubic
// with Args[2] = endpoint, Args[0]/Args[1] = controls. We emit both Args[0]
// and Args[1] for every segment to be robust across formats, then let the
// per-glyph normalizer dedupe naturally.
func collectEndpoints(f *sfnt.Font, buf *sfnt.Buffer, gid sfnt.GlyphIndex) [][2]float64 {
	outline, err := f.LoadGlyph(buf, gid, fixed.Int26_6(f.UnitsPerEm()), nil)
	if err != nil || len(outline) == 0 {
		return nil
	}
	var pts [][2]float64
	for _, seg := range outline {
		pts = append(pts, [2]float64{float64(seg.Args[0].X) / 64.0, float64(seg.Args[0].Y) / 64.0})
		pts = append(pts, [2]float64{float64(seg.Args[1].X) / 64.0, float64(seg.Args[1].Y) / 64.0})
	}
	return pts
}

func normalizePoints(pts [][2]float64) [][2]float64 {
	if len(pts) < 4 {
		return nil
	}
	xMin, xMax := pts[0][0], pts[0][0]
	yMin, yMax := pts[0][1], pts[0][1]
	for _, p := range pts[1:] {
		if p[0] < xMin {
			xMin = p[0]
		}
		if p[0] > xMax {
			xMax = p[0]
		}
		if p[1] < yMin {
			yMin = p[1]
		}
		if p[1] > yMax {
			yMax = p[1]
		}
	}
	xRange := xMax - xMin
	yRange := yMax - yMin
	if xRange < 0.01 || yRange < 0.01 {
		return nil
	}
	out := make([][2]float64, len(pts))
	for i, p := range pts {
		out[i] = [2]float64{(p[0] - xMin) / xRange, (p[1] - yMin) / yRange}
	}
	return out
}

func stemsAtY(normPts [][2]float64, yCenter, yTolerance float64) []float64 {
	var xs []float64
	for _, p := range normPts {
		if math.Abs(p[1]-yCenter) <= yTolerance {
			xs = append(xs, p[0])
		}
	}
	sort.Float64s(xs)
	if len(xs) < 2 {
		return nil
	}
	var widths []float64
	// Pair adjacent sorted x values (outer contours vs inner contours).
	for i := 0; i+1 < len(xs); i += 2 {
		w := xs[i+1] - xs[i]
		if w > 0.001 {
			widths = append(widths, w)
		}
	}
	return widths
}

func analyzeOutline(f *sfnt.Font, buf *sfnt.Buffer, upm, xHeight, capHeight, ascender int) OutlineAnalysis {
	var allStems, endStems []float64
	var sampled int
	for _, r := range latinTargetGlyphs {
		gid, err := f.GlyphIndex(buf, r)
		if err != nil || gid == 0 {
			continue
		}
		raw := collectEndpoints(f, buf, gid)
		norm := normalizePoints(raw)
		if len(norm) < 6 {
			continue
		}
		sampled++
		// Mid-stem: ~70% of glyph height (body of lowercase/uppercase).
		midStem := stemsAtY(norm, 0.70, 0.08)
		allStems = append(allStems, midStem...)
		// End-stem: near top (where serif/hairline overshoots sit).
		topStem := stemsAtY(norm, 0.92, 0.06)
		endStems = append(endStems, topStem...)
		// Baseline side.
		botStem := stemsAtY(norm, 0.08, 0.06)
		endStems = append(endStems, botStem...)
	}
	oa := OutlineAnalysis{Samples: sampled, Done: sampled > 0}
	if len(allStems) < 3 {
		return oa
	}
	mean, std := stats(allStems)
	oa.MeanStem = mean
	if mean > 0 {
		oa.StemCV = std / mean
		cv := math.Min(oa.StemCV, 2.0)
		oa.Contrast = cv / 2.2
		if oa.Contrast > 0.95 {
			oa.Contrast = 0.95
		}
	}
	if len(endStems) >= 2 {
		eMean, _ := stats(endStems)
		if mean > 0 {
			oa.EndOvershoot = eMean/mean - 1
			if oa.EndOvershoot < 0 {
				oa.EndOvershoot = 0
			}
		}
	}
	switch {
	case oa.EndOvershoot > 0.55:
		oa.Serif = "bracketed"
	case oa.EndOvershoot > 0.30:
		oa.Serif = "slab"
	case oa.EndOvershoot > 0.10:
		oa.Serif = "hairline"
	default:
		oa.Serif = "none"
	}
	switch {
	case oa.Contrast > 0.45 && oa.EndOvershoot < 0.10:
		oa.Terminal = "flat"
	case oa.Contrast > 0.45 && oa.EndOvershoot > 0.10:
		oa.Terminal = "angled"
	case oa.Contrast < 0.20 && oa.EndOvershoot < 0.05:
		oa.Terminal = "round"
	default:
		oa.Terminal = "flare"
	}
	switch {
	case oa.Serif != "none":
		oa.Skeleton = "serif"
	case oa.Contrast > 0.35:
		oa.Skeleton = "humanist"
	default:
		oa.Skeleton = "geometric"
	}
	return oa
}

func stats(xs []float64) (mean, std float64) {
	if len(xs) == 0 {
		return 0, 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	mean = sum / float64(len(xs))
	if len(xs) < 2 {
		return mean, 0
	}
	var varSum float64
	for _, x := range xs {
		d := x - mean
		varSum += d * d
	}
	std = math.Sqrt(varSum / float64(len(xs)-1))
	return mean, std
}

// Markdown renders a human-readable learning report.
func (r Report) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s %s\n\n", r.Family, r.Style)
	fmt.Fprintf(&b, "- File: `%s`\n- Glyphs: %d\n- UPM: %d\n- Weight/width: %d / %d\n- Italic angle: %.1f\n\n", r.File, r.Glyphs, r.UnitsPerEm, r.WeightClass, r.WidthClass, r.ItalicAngle)
	b.WriteString("## Metrics\n\n")
	fmt.Fprintf(&b, "- Ascender: %d\n- Descender: %d\n- Line gap: %d\n- x-height: %d\n- cap-height: %d\n- bbox: %v\n\n", r.Ascender, r.Descender, r.LineGap, r.XHeight, r.CapHeight, r.BBox)
	b.WriteString("## OpenType\n\n")
	if len(r.Features) == 0 {
		b.WriteString("No GSUB/GPOS features detected.\n\n")
	} else {
		for _, f := range r.Features {
			fmt.Fprintf(&b, "- `%s` %s lookups %v types %v\n", f.Table, f.Tag, f.Lookups, f.LookupType)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "- Kern pairs: %d\n- Axes: %d\n- Color tables: %v\n- Variable tables: %v\n\n", r.Kerning, len(r.Axes), r.HasColor, r.HasVariable)
	b.WriteString("## Coverage\n\n")
	keys := make([]string, 0, len(r.Coverage))
	for k := range r.Coverage {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "- %s: %d\n", k, r.Coverage[k])
	}
	fmt.Fprintf(&b, "\nTotal mapped codepoints: %d\n\n", r.CoverageTotal)
	b.WriteString("## Suggested recipe\n\n")
	fmt.Fprintf(&b, "```yaml\nstyle:\n  skeleton: %s\n  weight: %d\n  width: %.0f\n  slant: %.1f\n  serif: %s\n```\n\n", r.Suggested.Skeleton, r.Suggested.Weight, r.Suggested.Width, r.Suggested.Slant, r.Suggested.Serif)
	b.WriteString(r.Suggested.Note + "\n")
	return b.String()
}
