// Package library imports and exports external SVG glyph libraries.
package library

import (
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/neko233-com/fontcreator233/internal/geom"
)

// Override is one external glyph.
type Override struct {
	Path    geom.Path
	Advance int
	Source  string
}

var (
	uPlusRE = regexp.MustCompile(`(?i)^U\+([0-9a-f]{2,6})\.svg$`)
	uniRE   = regexp.MustCompile(`(?i)^uni([0-9a-f]{4,6})\.svg$`)
)

// CodepointFromFilename extracts a code point from U+XXXX.svg, uniXXXX.svg, or
// a single-character filename.
func CodepointFromFilename(name string) (rune, bool) {
	base := filepath.Base(name)
	if m := uPlusRE.FindStringSubmatch(base); m != nil {
		n, err := strconv.ParseUint(m[1], 16, 32)
		if err == nil && n <= 0x10FFFF {
			return rune(n), true
		}
	}
	if m := uniRE.FindStringSubmatch(base); m != nil {
		n, err := strconv.ParseUint(m[1], 16, 32)
		if err == nil && n <= 0x10FFFF {
			return rune(n), true
		}
	}
	if strings.HasSuffix(strings.ToLower(base), ".svg") {
		base = base[:len(base)-4]
	}
	if utf8.RuneCountInString(base) == 1 {
		r, _ := utf8.DecodeRuneInString(base)
		return r, true
	}
	return 0, false
}

type svgXML struct {
	DataAdvance string `xml:"data-advance,attr"`
	Width       string `xml:"width,attr"`
	Paths       []struct {
		D string `xml:"d,attr"`
	} `xml:"path"`
}

// ParseSVG parses a fontcreator233 glyph SVG. Coordinates are font units with
// baseline at 0 and Y up (matching viewBox="0 -250 1000 1250").
func ParseSVG(data []byte, source string) (Override, error) {
	var doc svgXML
	if err := xml.Unmarshal(data, &doc); err != nil {
		return Override{}, fmt.Errorf("%s: parse SVG: %w", source, err)
	}
	if len(doc.Paths) == 0 {
		return Override{}, fmt.Errorf("%s: SVG has no <path>", source)
	}
	var path geom.Path
	for _, p := range doc.Paths {
		cmds, err := geom.ParseCommands(p.D)
		if err != nil {
			return Override{}, fmt.Errorf("%s: %w", source, err)
		}
		path = append(path, commandsToPath(cmds)...)
	}
	if len(path) == 0 {
		return Override{}, fmt.Errorf("%s: SVG path is empty", source)
	}
	advance := 0
	if doc.DataAdvance != "" {
		n, err := strconv.Atoi(strings.TrimSpace(doc.DataAdvance))
		if err != nil || n <= 0 {
			return Override{}, fmt.Errorf("%s: invalid data-advance %q", source, doc.DataAdvance)
		}
		advance = n
	} else if doc.Width != "" {
		w, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(doc.Width), "px"), 64)
		if err == nil && w > 0 {
			advance = int(math.Round(w))
		}
	}
	if advance <= 0 {
		b := path.Bounds()
		advance = int(math.Round(b.MaxX - b.MinX + 100))
	}
	return Override{Path: path, Advance: advance, Source: source}, nil
}

func commandsToPath(cmds []geom.Cmd) geom.Path {
	var path geom.Path
	var cur geom.Contour
	flush := func() {
		if len(cur) > 0 {
			path = append(path, cur)
			cur = nil
		}
	}
	var start, current geom.Point
	for _, c := range cmds {
		switch c.Kind {
		case geom.CmdMove:
			flush()
			current = c.P[0]
			start = current
			cur = append(cur, geom.Node{P: current, On: true})
		case geom.CmdLine:
			current = c.P[0]
			cur = append(cur, geom.Node{P: current, On: true})
		case geom.CmdQuad:
			cur = append(cur, geom.Node{P: c.P[0], On: false}, geom.Node{P: c.P[1], On: true})
			current = c.P[1]
		case geom.CmdCubic:
			flattenCubic(current, c.P[0], c.P[1], c.P[2], &cur)
			current = c.P[2]
		case geom.CmdClose:
			if len(cur) > 1 && cur[0].P == cur[len(cur)-1].P {
				cur = cur[:len(cur)-1]
			}
			flush()
			current = start
		}
	}
	flush()
	return path
}

func flattenCubic(p0, c1, c2, p1 geom.Point, out *geom.Contour) {
	const steps = 24
	for i := 1; i <= steps; i++ {
		t := float64(i) / steps
		u := 1 - t
		p := geom.Point{
			X: u*u*u*p0.X + 3*u*u*t*c1.X + 3*u*t*t*c2.X + t*t*t*p1.X,
			Y: u*u*u*p0.Y + 3*u*u*t*c1.Y + 3*u*t*t*c2.Y + t*t*t*p1.Y,
		}
		*out = append(*out, geom.Node{P: p, On: true})
	}
}

// LoadDir loads every SVG override in dir.
func LoadDir(dir string) (map[rune]Override, error) {
	out := map[rune]Override{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".svg") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		r, ok := CodepointFromFilename(name)
		if !ok {
			return nil, fmt.Errorf("%s: filename must be U+XXXX.svg, uniXXXX.svg, or one character", name)
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		ov, err := ParseSVG(data, name)
		if err != nil {
			return nil, err
		}
		if _, exists := out[r]; exists {
			return nil, fmt.Errorf("%s: duplicate override for U+%04X", name, r)
		}
		out[r] = ov
	}
	return out, nil
}
