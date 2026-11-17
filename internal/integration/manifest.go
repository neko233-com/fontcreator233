// Package integration builds the cross-platform asset manifest and charsets.
package integration

import (
	"encoding/json"
	"os"
	"sort"

	"github.com/neko233-com/fontcreator233/internal/unicodes"
)

// Range is an inclusive code point range.
type Range [2]int

// Font is one font file in the manifest.
type Font struct {
	Path    string   `json:"path"`
	WOFF    string   `json:"woff,omitempty"`
	SHA256  string   `json:"sha256"`
	Primary bool     `json:"primary"`
	Scripts []string `json:"scripts"`
	Ranges  []Range  `json:"ranges"`
	Charset string   `json:"charset,omitempty"`
}

// TMP is the Unity TextMeshPro configuration.
type TMP struct {
	Population        string   `json:"population"`
	SamplingPointSize int      `json:"samplingPointSize"`
	Padding           int      `json:"padding"`
	AtlasWidth        int      `json:"atlasWidth"`
	AtlasHeight       int      `json:"atlasHeight"`
	MultiAtlas        bool     `json:"multiAtlas"`
	RenderMode        string   `json:"renderMode"`
	StaticCharsets    []string `json:"staticCharsets"`
}

// Composite sub-font for Unreal.
type Composite struct {
	Path   string  `json:"path"`
	Ranges []Range `json:"ranges"`
}

// Unreal holds the composite font fallback description.
type Unreal struct {
	Default  string      `json:"default"`
	SubFonts []Composite `json:"subFonts"`
}

// Web points at the generated CSS.
type Web struct {
	CSS         string `json:"css"`
	FontDisplay string `json:"fontDisplay"`
}

// Manifest is fontcreator233/integration/v1.
type Manifest struct {
	Schema          string   `json:"schema"`
	Generator       string   `json:"generator"`
	Family          string   `json:"family"`
	Style           string   `json:"style"`
	RequestedStyle  string   `json:"requestedStyle,omitempty"`
	License         string   `json:"license"`
	LicenseFile     string   `json:"licenseFile"`
	Fonts           []Font   `json:"fonts"`
	Fallbacks       []string `json:"fallbacks"`
	Charsets        []string `json:"charsets"`
	UnityTMP        TMP      `json:"unityTMP"`
	UnrealComposite Unreal   `json:"unrealComposite"`
	Web             Web      `json:"web"`
}

// Ranges compresses sorted code points into inclusive ranges.
func Ranges(runes []rune) []Range {
	if len(runes) == 0 {
		return nil
	}
	cp := append([]rune(nil), runes...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	var out []Range
	start, prev := cp[0], cp[0]
	for _, r := range cp[1:] {
		if r == prev {
			continue
		}
		if r == prev+1 {
			prev = r
			continue
		}
		out = append(out, Range{int(start), int(prev)})
		start, prev = r, r
	}
	out = append(out, Range{int(start), int(prev)})
	return out
}

// Scripts returns sorted unique script names.
func Scripts(runes []rune) []string {
	set := map[string]bool{}
	for _, r := range runes {
		set[unicodes.ScriptOf(r)] = true
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// Charsets returns UTF-8 text per script, sorted and deduplicated.
func Charsets(runes []rune) map[string]string {
	seen := map[string]map[rune]bool{}
	for _, r := range runes {
		s := unicodes.ScriptOf(r)
		if seen[s] == nil {
			seen[s] = map[rune]bool{}
		}
		seen[s][r] = true
	}
	out := map[string]string{}
	for s, set := range seen {
		list := make([]rune, 0, len(set))
		for r := range set {
			list = append(list, r)
		}
		sort.Slice(list, func(i, j int) bool { return list[i] < list[j] })
		out[s] = string(list)
	}
	return out
}

// WriteJSON writes the manifest.
func WriteJSON(path string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// ReadJSON reads a manifest.
func ReadJSON(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
