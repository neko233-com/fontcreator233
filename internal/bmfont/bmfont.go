// Package bmfont writes AngelCode BMFont (.fnt v3) files and PNG atlases from
// the generated vector glyphs. It is fully offline and deterministic.
package bmfont

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/neko233-com/fontcreator233/internal/geom"
)

// Glyph is one cmap glyph with its source outline.
type Glyph struct {
	Rune    rune
	Name    string
	Advance int
	Path    geom.Path
}

// KernPair is a kerning adjustment in font units between two glyph names.
type KernPair struct {
	Left, Right string
	Value       int16
}

// Options controls atlas generation.
type Options struct {
	OutDir    string
	BaseName  string
	Size      int    // pixel size of the em square
	Padding   int    // padding around each glyph, in pixels
	AtlasSize int    // width and height of each atlas page
	Charset   string // "latin" (default), "latin-ext", "all", or an explicit string
	Ascender  int
	Descender int
	LineGap   int
	Weight    int
	Italic    bool
}

type raster struct {
	id         uint32
	w, h       int
	xoff, yoff int
	advance    int
	alpha      []byte
}

type placement struct {
	r     raster
	page  int
	pageX int
	pageY int
}

type kernRecord struct {
	first, second uint32
	amount        int16
}

// Build writes <BaseName>.fnt and one or more <BaseName>_<n>.png pages. It
// returns the written file paths.
func Build(o Options, glyphs []Glyph, kern []KernPair) ([]string, error) {
	if o.OutDir == "" {
		return nil, fmt.Errorf("bmfont: empty output directory")
	}
	if o.Size <= 0 {
		o.Size = 32
	}
	if o.Padding < 0 {
		o.Padding = 0
	}
	if o.AtlasSize <= 0 {
		o.AtlasSize = 512
	}
	if o.BaseName == "" {
		o.BaseName = "Font"
	}
	if o.Ascender <= 0 {
		o.Ascender = 800
	}
	if o.Descender >= 0 {
		o.Descender = -200
	}
	selected := selectGlyphs(o.Charset, glyphs)
	if len(selected) == 0 {
		return nil, fmt.Errorf("bmfont: empty charset")
	}
	scale := float64(o.Size) / 1000.0

	nameToRune := make(map[string]rune, len(glyphs))
	selectedRunes := make(map[rune]bool, len(selected))
	for _, g := range glyphs {
		nameToRune[g.Name] = g.Rune
	}
	for _, g := range selected {
		selectedRunes[g.Rune] = true
	}

	items := make([]placement, 0, len(selected))
	for _, g := range selected {
		r := rasterize(g, o, scale)
		r.id = uint32(g.Rune)
		items = append(items, placement{r: r})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].r.h != items[j].r.h {
			return items[i].r.h > items[j].r.h
		}
		return items[i].r.advance < items[j].r.advance
	})

	pageImages := []*image.NRGBA{newAtlas(o.AtlasSize)}
	pageX, pageY, rowH := 0, 0, 0
	for i := range items {
		r := &items[i].r
		if r.w == 0 || r.h == 0 {
			items[i].page = -1 // empty glyph; no atlas rectangle
			continue
		}
		if pageX+r.w > o.AtlasSize {
			pageX = 0
			pageY += rowH + 1
			rowH = 0
		}
		if pageY+r.h > o.AtlasSize {
			pageImages = append(pageImages, newAtlas(o.AtlasSize))
			pageX, pageY, rowH = 0, 0, 0
		}
		items[i].page = len(pageImages) - 1
		items[i].pageX, items[i].pageY = pageX, pageY
		copyInto(pageImages[len(pageImages)-1], r, pageX, pageY)
		pageX += r.w + 1
		if r.h > rowH {
			rowH = r.h
		}
	}

	pageNames := make([]string, len(pageImages))
	paths := make([]string, 0, len(pageImages)+1)
	for i, img := range pageImages {
		pageNames[i] = fmt.Sprintf("%s_%d.png", o.BaseName, i)
		p := filepath.Join(o.OutDir, pageNames[i])
		if err := writePNG(p, img); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}

	chars := make([]charRecord, 0, len(items))
	for _, it := range items {
		chars = append(chars, charRecord{
			r:    it.r,
			page: byte(maxInt(it.page, 0)),
			x:    it.pageX,
			y:    it.pageY,
		})
	}
	sort.Slice(chars, func(i, j int) bool { return chars[i].r.id < chars[j].r.id })

	var kerns []kernRecord
	for _, kp := range kern {
		a, okA := nameToRune[kp.Left]
		b, okB := nameToRune[kp.Right]
		if !okA || !okB || !selectedRunes[a] || !selectedRunes[b] {
			continue
		}
		kerns = append(kerns, kernRecord{
			first: uint32(a), second: uint32(b),
			amount: int16(round(float64(kp.Value) * scale)),
		})
	}
	sort.Slice(kerns, func(i, j int) bool {
		if kerns[i].first != kerns[j].first {
			return kerns[i].first < kerns[j].first
		}
		return kerns[i].second < kerns[j].second
	})

	fntPath := filepath.Join(o.OutDir, o.BaseName+".fnt")
	if err := os.WriteFile(fntPath, writeFNT(o, pageNames, o.AtlasSize, chars, kerns), 0o644); err != nil {
		return nil, err
	}
	return append(paths, fntPath), nil
}

type charRecord struct {
	r    raster
	page byte
	x, y int
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func selectGlyphs(charset string, glyphs []Glyph) []Glyph {
	var want map[rune]bool
	switch charset {
	case "", "latin":
		want = rangeSet(0x20, 0x7E)
	case "latin-ext":
		want = rangeSet(0x20, 0x24F)
		for r := 0x2000; r <= 0x206F; r++ {
			want[rune(r)] = true
		}
	case "all":
		var out []Glyph
		for _, g := range glyphs {
			if g.Rune != 0 {
				out = append(out, g)
			}
		}
		sortGlyphs(out)
		return out
	default:
		want = map[rune]bool{}
		for _, r := range charset {
			want[r] = true
		}
	}
	var out []Glyph
	seen := map[rune]bool{}
	for _, g := range glyphs {
		if g.Rune == 0 || !want[g.Rune] || seen[g.Rune] {
			continue
		}
		seen[g.Rune] = true
		out = append(out, g)
	}
	sortGlyphs(out)
	return out
}

func sortGlyphs(g []Glyph) {
	sort.Slice(g, func(i, j int) bool { return g[i].Rune < g[j].Rune })
}

func rangeSet(lo, hi int) map[rune]bool {
	m := make(map[rune]bool, hi-lo+1)
	for r := lo; r <= hi; r++ {
		m[rune(r)] = true
	}
	return m
}

// rasterize converts one vector outline into a tightly cropped anti-aliased
// bitmap using 4x supersampling and an even-odd scanline fill.
func rasterize(g Glyph, o Options, scale float64) raster {
	r := raster{advance: round(float64(g.Advance) * scale)}
	if len(g.Path) == 0 {
		return r
	}
	box := g.Path.Bounds()
	pad := o.Padding
	minPX := int(math.Floor(box.MinX*scale)) - pad
	maxPX := int(math.Ceil(box.MaxX*scale)) + pad
	minPY := int(math.Floor(box.MinY*scale)) - pad
	maxPY := int(math.Ceil(box.MaxY*scale)) + pad
	if maxPX <= minPX || maxPY <= minPY {
		return r
	}
	const ss = 4
	w, h := maxPX-minPX, maxPY-minPY
	cover := make([]uint16, w*ss*h*ss)
	type edge struct{ x1, y1, x2, y2 float64 }
	var edges []edge
	for _, contour := range g.Path {
		if len(contour) < 3 {
			continue
		}
		for i := range contour {
			a, b := contour[i].P, contour[(i+1)%len(contour)].P
			edges = append(edges, edge{
				x1: (a.X*scale - float64(minPX)) * ss,
				y1: (float64(maxPY) - a.Y*scale) * ss,
				x2: (b.X*scale - float64(minPX)) * ss,
				y2: (float64(maxPY) - b.Y*scale) * ss,
			})
		}
	}
	xs := make([]float64, 0, len(edges))
	for sy := 0; sy < h*ss; sy++ {
		sampleY := float64(sy) + 0.5
		xs = xs[:0]
		for _, e := range edges {
			if (e.y1 <= sampleY && e.y2 > sampleY) || (e.y2 <= sampleY && e.y1 > sampleY) {
				xs = append(xs, e.x1+(sampleY-e.y1)*(e.x2-e.x1)/(e.y2-e.y1))
			}
		}
		if len(xs) < 2 {
			continue
		}
		sort.Float64s(xs)
		for i := 0; i+1 < len(xs); i += 2 {
			from, to := int(math.Floor(xs[i])), int(math.Ceil(xs[i+1]))
			if from < 0 {
				from = 0
			}
			if to > w*ss {
				to = w * ss
			}
			for x := from; x < to; x++ {
				cover[sy*w*ss+x]++
			}
		}
	}
	alpha := make([]byte, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sum int
			for dy := 0; dy < ss; dy++ {
				for dx := 0; dx < ss; dx++ {
					sum += int(cover[(y*ss+dy)*w*ss+x*ss+dx])
				}
			}
			alpha[y*w+x] = byte(sum * 255 / (ss * ss))
		}
	}
	cx0, cy0, cx1, cy1 := w, h, 0, 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if alpha[y*w+x] > 0 {
				if x < cx0 {
					cx0 = x
				}
				if y < cy0 {
					cy0 = y
				}
				if x >= cx1 {
					cx1 = x + 1
				}
				if y >= cy1 {
					cy1 = y + 1
				}
			}
		}
	}
	if cx1 <= cx0 || cy1 <= cy0 {
		return r
	}
	cw, ch := cx1-cx0, cy1-cy0
	cropped := make([]byte, cw*ch)
	for y := 0; y < ch; y++ {
		copy(cropped[y*cw:(y+1)*cw], alpha[(cy0+y)*w+cx0:(cy0+y)*w+cx1])
	}
	r.w, r.h = cw, ch
	r.xoff = minPX + cx0
	r.yoff = (minPY + cy0) - maxPY
	r.alpha = cropped
	return r
}

func newAtlas(size int) *image.NRGBA {
	return image.NewNRGBA(image.Rect(0, 0, size, size))
}

func copyInto(img *image.NRGBA, r *raster, dx, dy int) {
	for y := 0; y < r.h; y++ {
		for x := 0; x < r.w; x++ {
			img.SetNRGBA(dx+x, dy+y, color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: r.alpha[y*r.w+x]})
		}
	}
}

func writePNG(path string, img *image.NRGBA) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func writeFNT(o Options, pageNames []string, atlas int, chars []charRecord, kerns []kernRecord) []byte {
	var buf []byte
	buf = append(buf, 'B', 'M', 'F', 3)
	// Block 1: info.
	info := make([]byte, 14)
	var bits byte = 1 // unicode
	if o.Italic {
		bits |= 2
	}
	if o.Weight >= 600 {
		bits |= 4
	}
	binary.LittleEndian.PutUint16(info[0:], uint16(o.Size))
	info[2] = bits
	binary.LittleEndian.PutUint16(info[5:], 100) // stretchH
	info[7] = 4                                  // aa (supersampling level)
	for i := 8; i <= 11; i++ {
		info[i] = byte(o.Padding)
	}
	buf = appendBlock(buf, 1, info)
	// Block 2: common.
	common := make([]byte, 14)
	lineHeight := round(float64(o.Ascender-o.Descender+o.LineGap) * float64(o.Size) / 1000.0)
	base := round(float64(o.Ascender) * float64(o.Size) / 1000.0)
	binary.LittleEndian.PutUint16(common[0:], uint16(lineHeight))
	binary.LittleEndian.PutUint16(common[2:], uint16(base))
	binary.LittleEndian.PutUint16(common[4:], uint16(atlas))
	binary.LittleEndian.PutUint16(common[6:], uint16(atlas))
	binary.LittleEndian.PutUint16(common[8:], uint16(len(pageNames)))
	buf = appendBlock(buf, 2, common)
	// Block 3: pages.
	var pages []byte
	for _, name := range pageNames {
		pages = append(pages, []byte(name)...)
		pages = append(pages, 0)
	}
	buf = appendBlock(buf, 3, pages)
	// Block 4: chars.
	charData := make([]byte, 0, len(chars)*20)
	for _, c := range chars {
		rec := make([]byte, 20)
		binary.LittleEndian.PutUint32(rec[0:], c.r.id)
		binary.LittleEndian.PutUint16(rec[4:], uint16(c.x))
		binary.LittleEndian.PutUint16(rec[6:], uint16(c.y))
		binary.LittleEndian.PutUint16(rec[8:], uint16(c.r.w))
		binary.LittleEndian.PutUint16(rec[10:], uint16(c.r.h))
		putI16LE(rec[12:], int16(c.r.xoff))
		putI16LE(rec[14:], int16(c.r.yoff))
		putI16LE(rec[16:], int16(c.r.advance))
		rec[18] = c.page
		charData = append(charData, rec...)
	}
	buf = appendBlock(buf, 4, charData)
	// Block 5: kerning.
	if len(kerns) > 0 {
		kernData := make([]byte, 0, len(kerns)*10)
		for _, k := range kerns {
			rec := make([]byte, 10)
			binary.LittleEndian.PutUint32(rec[0:], k.first)
			binary.LittleEndian.PutUint32(rec[4:], k.second)
			putI16LE(rec[8:], k.amount)
			kernData = append(kernData, rec...)
		}
		buf = appendBlock(buf, 5, kernData)
	}
	return buf
}

func appendBlock(dst []byte, kind byte, data []byte) []byte {
	head := make([]byte, 5)
	head[0] = kind
	binary.LittleEndian.PutUint32(head[1:], uint32(len(data)))
	return append(append(dst, head...), data...)
}

func putI16LE(b []byte, v int16) {
	binary.LittleEndian.PutUint16(b, uint16(v))
}

func round(v float64) int {
	if v >= 0 {
		return int(v + 0.5)
	}
	return int(v - 0.5)
}
