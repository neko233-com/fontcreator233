package fontbuild

import (
	"fmt"
	"math/bits"
	"sort"
)

type layoutFeatureSpec struct {
	Tag        string
	LookupType uint16
	Subtable   []byte
}

// layoutTable assembles a minimal OpenType GSUB/GPOS table with a DFLT/latn
// script list, one feature, and one lookup.
func layoutTable(featureTag string, lookupType uint16, subtable []byte) []byte {
	return layoutTableMulti([]layoutFeatureSpec{{Tag: featureTag, LookupType: lookupType, Subtable: subtable}})
}

func layoutTableMulti(features []layoutFeatureSpec) []byte {
	if len(features) == 0 {
		return nil
	}
	scriptList := buildScriptList([]string{"DFLT", "latn"}, len(features))
	featureList := buildFeatureList(features)
	lookupList := buildLookupList(features)
	headerLen := 10
	scriptOff := headerLen
	featureOff := scriptOff + len(scriptList)
	lookupOff := featureOff + len(featureList)
	out := make([]byte, headerLen+len(scriptList)+len(featureList)+len(lookupList))
	putU32(out, 0, 0x00010000)
	putU16(out, 4, uint16(scriptOff))
	putU16(out, 6, uint16(featureOff))
	putU16(out, 8, uint16(lookupOff))
	copy(out[scriptOff:], scriptList)
	copy(out[featureOff:], featureList)
	copy(out[lookupOff:], lookupList)
	return out
}

func buildScriptList(tags []string, featureCount int) []byte {
	scriptTable := make([]byte, 10+2*featureCount)
	putU16(scriptTable, 0, 4)      // DefaultLangSys offset
	putU16(scriptTable, 2, 0)      // LangSysCount
	putU16(scriptTable, 4, 0)      // LookupOrder
	putU16(scriptTable, 6, 0xFFFF) // ReqFeatureIndex
	putU16(scriptTable, 8, uint16(featureCount))
	for i := 0; i < featureCount; i++ {
		putU16(scriptTable, 10+2*i, uint16(i))
	}
	header := 2 + 6*len(tags)
	out := make([]byte, header+len(scriptTable))
	putU16(out, 0, uint16(len(tags)))
	for i, tag := range tags {
		rec := 2 + i*6
		copy(out[rec:rec+4], tag)
		putU16(out, rec+4, uint16(header))
	}
	copy(out[header:], scriptTable)
	return out
}

func buildFeatureList(features []layoutFeatureSpec) []byte {
	header := 2 + 6*len(features)
	total := header + 6*len(features)
	out := make([]byte, total)
	putU16(out, 0, uint16(len(features)))
	off := header
	for i, f := range features {
		rec := 2 + i*6
		copy(out[rec:rec+4], f.Tag)
		putU16(out, rec+4, uint16(off))
		feature := out[off : off+6]
		putU16(feature, 0, 0) // FeatureParams
		putU16(feature, 2, 1)
		putU16(feature, 4, uint16(i)) // lookup index
		off += 6
	}
	return out
}

func buildLookupList(features []layoutFeatureSpec) []byte {
	header := 2 + 2*len(features)
	offsets := make([]int, len(features))
	off := header
	total := header
	for i, f := range features {
		offsets[i] = off
		off += 8 + len(f.Subtable)
		total = off
	}
	out := make([]byte, total)
	putU16(out, 0, uint16(len(features)))
	for i, f := range features {
		putU16(out, 2+2*i, uint16(offsets[i]))
		lookup := out[offsets[i] : offsets[i]+8+len(f.Subtable)]
		putU16(lookup, 0, f.LookupType)
		putU16(lookup, 2, 0)
		putU16(lookup, 4, 1)
		putU16(lookup, 6, 8)
		copy(lookup[8:], f.Subtable)
	}
	return out
}

func glyphIDs(glyphs []Glyph) map[string]uint16 {
	ids := make(map[string]uint16, len(glyphs))
	for i, g := range glyphs {
		ids[g.Name] = uint16(i)
	}
	return ids
}

func buildLigatureSubtable(glyphs []Glyph, ligatures []Ligature) ([]byte, error) {
	ids := glyphIDs(glyphs)
	groups := map[uint16][][]uint16{}
	for _, lig := range ligatures {
		if len(lig.Components) < 2 || lig.Glyph == "" {
			continue
		}
		components := make([]uint16, 0, len(lig.Components))
		ok := true
		for _, name := range lig.Components {
			gid, exists := ids[name]
			if !exists {
				ok = false
				break
			}
			components = append(components, gid)
		}
		ligGID, exists := ids[lig.Glyph]
		if !ok || !exists {
			continue
		}
		groups[components[0]] = append(groups[components[0]], append(components[1:], ligGID))
	}
	if len(groups) == 0 {
		return nil, nil
	}
	first := make([]uint16, 0, len(groups))
	for gid := range groups {
		first = append(first, gid)
	}
	sort.Slice(first, func(i, j int) bool { return first[i] < first[j] })
	sets := make([][]byte, len(first))
	for i, gid := range first {
		ligs := groups[gid]
		sort.Slice(ligs, func(a, b int) bool {
			if len(ligs[a]) != len(ligs[b]) {
				return len(ligs[a]) > len(ligs[b])
			}
			return ligs[a][len(ligs[a])-1] < ligs[b][len(ligs[b])-1]
		})
		set := make([]byte, 2+2*len(ligs))
		putU16(set, 0, uint16(len(ligs)))
		off := 2 + 2*len(ligs)
		for j, lig := range ligs {
			body := make([]byte, 4+2*(len(lig)-1))
			putU16(body, 0, lig[len(lig)-1])
			putU16(body, 2, uint16(len(lig)))
			for k := 0; k < len(lig)-1; k++ {
				putU16(body, 4+2*k, lig[k])
			}
			putU16(set, 2+2*j, uint16(off))
			set = append(set, body...)
			off += len(body)
		}
		sets[i] = set
	}
	return ligatureSubtable(first, sets), nil
}

func ligatureSubtable(first []uint16, sets [][]byte) []byte {
	header := 6 + 2*len(first)
	coverage := buildCoverage(first)
	total := header + len(coverage)
	for _, set := range sets {
		total += len(set)
	}
	out := make([]byte, total)
	putU16(out, 0, 1)
	putU16(out, 2, uint16(header))
	putU16(out, 4, uint16(len(first)))
	off := header + len(coverage)
	for i, set := range sets {
		putU16(out, 6+2*i, uint16(off))
		copy(out[off:], set)
		off += len(set)
	}
	copy(out[header:], coverage)
	return out
}

func buildCoverage(gids []uint16) []byte {
	b := make([]byte, 4+2*len(gids))
	putU16(b, 0, 1)
	putU16(b, 2, uint16(len(gids)))
	for i, gid := range gids {
		putU16(b, 4+2*i, gid)
	}
	return b
}

func buildGPOS(glyphs []Glyph, pairs []KernPair) ([]byte, error) {
	ids := glyphIDs(glyphs)
	byLeft := map[uint16]map[uint16]int16{}
	for _, p := range pairs {
		left, okL := ids[p.Left]
		right, okR := ids[p.Right]
		if !okL || !okR || left == right {
			continue
		}
		if byLeft[left] == nil {
			byLeft[left] = map[uint16]int16{}
		}
		byLeft[left][right] = p.Value
	}
	if len(byLeft) == 0 {
		return nil, nil
	}
	lefts := make([]uint16, 0, len(byLeft))
	for gid := range byLeft {
		lefts = append(lefts, gid)
	}
	sort.Slice(lefts, func(i, j int) bool { return lefts[i] < lefts[j] })
	coverage := buildCoverage(lefts)
	sets := make([][]byte, len(lefts))
	totalSets := 0
	for i, left := range lefts {
		rights := make([]uint16, 0, len(byLeft[left]))
		for right := range byLeft[left] {
			rights = append(rights, right)
		}
		sort.Slice(rights, func(a, b int) bool { return rights[a] < rights[b] })
		set := make([]byte, 2+4*len(rights))
		putU16(set, 0, uint16(len(rights)))
		for j, right := range rights {
			putU16(set, 2+4*j, right)
			putU16(set, 4+4*j, uint16(byLeft[left][right]))
		}
		sets[i] = set
		totalSets += len(set)
	}
	header := 10 + 2*len(lefts)
	out := make([]byte, header+len(coverage)+totalSets)
	putU16(out, 0, 1) // PairPos format 1
	putU16(out, 2, uint16(header))
	putU16(out, 4, 0x0004) // ValueFormat1: XAdvance
	putU16(out, 6, 0)      // ValueFormat2
	putU16(out, 8, uint16(len(lefts)))
	off := header + len(coverage)
	for i, set := range sets {
		putU16(out, 10+2*i, uint16(off))
		copy(out[off:], set)
		off += len(set)
	}
	copy(out[header:], coverage)
	return out, nil
}

func buildGSUBTable(glyphs []Glyph, ligatures []Ligature, substitutions []Substitution) ([]byte, error) {
	byFeature := map[string][]Ligature{}
	for _, lig := range ligatures {
		feature := lig.Feature
		if feature == "" {
			feature = "liga"
		}
		byFeature[feature] = append(byFeature[feature], lig)
	}
	subsByFeature := map[string][]Substitution{}
	for _, sub := range substitutions {
		feature := sub.Feature
		if feature == "" {
			feature = "liga"
		}
		subsByFeature[feature] = append(subsByFeature[feature], sub)
	}
	tags := make([]string, 0, len(byFeature)+len(subsByFeature))
	seen := map[string]bool{}
	for tag := range byFeature {
		seen[tag] = true
		tags = append(tags, tag)
	}
	for tag := range subsByFeature {
		if !seen[tag] {
			tags = append(tags, tag)
		}
	}
	sort.Strings(tags)
	var features []layoutFeatureSpec
	for _, tag := range tags {
		if ligs := byFeature[tag]; len(ligs) > 0 {
			sub, err := buildLigatureSubtable(glyphs, ligs)
			if err != nil {
				return nil, err
			}
			if len(sub) > 0 {
				features = append(features, layoutFeatureSpec{Tag: tag, LookupType: 4, Subtable: sub})
			}
		}
		if subs := subsByFeature[tag]; len(subs) > 0 {
			sub := buildSingleSubtable(glyphs, subs)
			if len(sub) > 0 {
				features = append(features, layoutFeatureSpec{Tag: tag, LookupType: 1, Subtable: sub})
			}
		}
	}
	if len(features) == 0 {
		return nil, nil
	}
	return layoutTableMulti(features), nil
}

func buildSingleSubtable(glyphs []Glyph, subs []Substitution) []byte {
	ids := glyphIDs(glyphs)
	byFrom := map[uint16]uint16{}
	for _, s := range subs {
		from, okF := ids[s.From]
		to, okT := ids[s.To]
		if okF && okT {
			byFrom[from] = to
		}
	}
	if len(byFrom) == 0 {
		return nil
	}
	from := make([]uint16, 0, len(byFrom))
	for gid := range byFrom {
		from = append(from, gid)
	}
	sort.Slice(from, func(i, j int) bool { return from[i] < from[j] })
	coverage := buildCoverage(from)
	header := 6 + 2*len(from)
	out := make([]byte, header+len(coverage))
	putU16(out, 0, 2) // SingleSubst format 2
	putU16(out, 2, uint16(header))
	putU16(out, 4, uint16(len(from)))
	for i, gid := range from {
		putU16(out, 6+2*i, byFrom[gid])
	}
	copy(out[header:], coverage)
	return out
}

func buildGPOSTable(glyphs []Glyph, pairs []KernPair, capHeight, xHeight int) ([]byte, error) {
	var features []layoutFeatureSpec
	if sub, err := buildGPOS(glyphs, pairs); err != nil {
		return nil, err
	} else if len(sub) > 0 {
		features = append(features, layoutFeatureSpec{Tag: "kern", LookupType: 2, Subtable: sub})
	}
	if marks := buildMarkBaseSubtable(glyphs, capHeight, xHeight); len(marks) > 0 {
		features = append(features, layoutFeatureSpec{Tag: "mark", LookupType: 4, Subtable: marks})
	}
	if stack := buildMarkMarkSubtable(glyphs); len(stack) > 0 {
		features = append(features, layoutFeatureSpec{Tag: "mkmk", LookupType: 6, Subtable: stack})
	}
	if len(features) == 0 {
		return nil, nil
	}
	sort.Slice(features, func(i, j int) bool { return features[i].Tag < features[j].Tag })
	return layoutTableMulti(features), nil
}

func buildKernTable(glyphs []Glyph, pairs []KernPair) ([]byte, error) {
	ids := glyphIDs(glyphs)
	type kp struct {
		left, right uint16
		value       int16
	}
	var list []kp
	for _, p := range pairs {
		left, okL := ids[p.Left]
		right, okR := ids[p.Right]
		if !okL || !okR || left == right || p.Value == 0 {
			continue
		}
		list = append(list, kp{left: left, right: right, value: p.Value})
	}
	if len(list) == 0 {
		return nil, nil
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].left != list[j].left {
			return list[i].left < list[j].left
		}
		return list[i].right < list[j].right
	})
	if len(list) > 10920 {
		return nil, fmt.Errorf("kern: too many pairs (%d)", len(list))
	}
	subLen := 14 + 6*len(list)
	if subLen > 65535 {
		return nil, fmt.Errorf("kern: table too large")
	}
	sub := make([]byte, subLen)
	putU16(sub, 0, 0)
	putU16(sub, 2, uint16(subLen))
	putU16(sub, 4, 0x0001)
	n := len(list)
	putU16(sub, 6, uint16(n))
	entrySelector := bits.Len(uint(n)) - 1
	searchRange := 1 << entrySelector * 6
	putU16(sub, 8, uint16(searchRange))
	putU16(sub, 10, uint16(entrySelector))
	putU16(sub, 12, uint16(n*6-searchRange))
	for i, p := range list {
		base := 14 + 6*i
		putU16(sub, base, p.left)
		putU16(sub, base+2, p.right)
		putU16(sub, base+4, uint16(p.value))
	}
	out := make([]byte, 4+len(sub))
	putU16(out, 0, 0)
	putU16(out, 2, 1)
	copy(out[4:], sub)
	return out, nil
}
