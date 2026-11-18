package fontbuild

import (
	"fmt"
	"math/bits"
	"sort"
)

// layoutTable assembles a minimal OpenType GSUB/GPOS table with a DFLT/latn
// script list, one feature, and one lookup.
func layoutTable(featureTag string, lookupType uint16, subtable []byte) []byte {
	scriptList := buildScriptList([]string{"DFLT", "latn"})
	featureList := buildFeatureList(featureTag)
	lookupList := buildLookupList(lookupType, subtable)
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

func buildScriptList(tags []string) []byte {
	scriptTable := make([]byte, 10)
	putU16(scriptTable, 0, 4)      // DefaultLangSys offset
	putU16(scriptTable, 2, 0)      // LangSysCount
	putU16(scriptTable, 4, 0)      // LookupOrder
	putU16(scriptTable, 6, 0xFFFF) // ReqFeatureIndex
	putU16(scriptTable, 8, 1)      // FeatureIndexCount
	// The feature index record is the final uint16.
	scriptTable = append(scriptTable, 0, 0)
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

func buildFeatureList(tag string) []byte {
	feature := make([]byte, 6)
	putU16(feature, 0, 0) // FeatureParams
	putU16(feature, 2, 1)
	putU16(feature, 4, 0) // lookup index
	out := make([]byte, 8+len(feature))
	putU16(out, 0, 1)
	copy(out[2:6], tag)
	putU16(out, 6, 8)
	copy(out[8:], feature)
	return out
}

func buildLookupList(lookupType uint16, subtable []byte) []byte {
	lookup := make([]byte, 8+len(subtable))
	putU16(lookup, 0, lookupType)
	putU16(lookup, 2, 0)
	putU16(lookup, 4, 1)
	putU16(lookup, 6, 8)
	copy(lookup[8:], subtable)
	out := make([]byte, 4+len(lookup))
	putU16(out, 0, 1)
	putU16(out, 2, 4)
	copy(out[4:], lookup)
	return out
}

func glyphIDs(glyphs []Glyph) map[string]uint16 {
	ids := make(map[string]uint16, len(glyphs))
	for i, g := range glyphs {
		ids[g.Name] = uint16(i)
	}
	return ids
}

func buildGSUB(glyphs []Glyph, ligatures []Ligature) ([]byte, error) {
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

func buildGSUBTable(glyphs []Glyph, ligatures []Ligature) ([]byte, error) {
	sub, err := buildGSUB(glyphs, ligatures)
	if err != nil || sub == nil {
		return nil, err
	}
	return layoutTable("liga", 4, sub), nil
}

func buildGPOSTable(glyphs []Glyph, pairs []KernPair) ([]byte, error) {
	sub, err := buildGPOS(glyphs, pairs)
	if err != nil || sub == nil {
		return nil, err
	}
	return layoutTable("kern", 2, sub), nil
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
