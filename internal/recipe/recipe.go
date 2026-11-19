// Package recipe defines the fontcreator233 recipe contract and validation.
package recipe

// Recipe is the complete input to a font build.
type Recipe struct {
	APIVersion string   `yaml:"apiVersion" json:"apiVersion"`
	Meta       Meta     `yaml:"meta" json:"meta"`
	Metrics    Metrics  `yaml:"metrics" json:"metrics"`
	Style      Style    `yaml:"style" json:"style"`
	Coverage   Coverage `yaml:"coverage" json:"coverage"`
	Glyphs     Glyphs   `yaml:"glyphs" json:"glyphs"`
	OpenType   OpenType `yaml:"openType" json:"openType"`
	Output     Output   `yaml:"output" json:"output"`

	// SourcePath is the recipe file path when loaded from disk. It is not part
	// of the serialized contract; relative glyph paths resolve against it.
	SourcePath string `yaml:"-" json:"-"`
}

// Meta describes the font and its generated specimen.
type Meta struct {
	Family      string `yaml:"family" json:"family"`
	Style       string `yaml:"style" json:"style"`
	Version     string `yaml:"version" json:"version"`
	Designer    string `yaml:"designer" json:"designer"`
	Description string `yaml:"description" json:"description"`
	License     string `yaml:"license" json:"license"`
	Accent      string `yaml:"accent" json:"accent"`
	SampleText  string `yaml:"sampleText" json:"sampleText"`
}

// Metrics uses font units for vertical metrics and ratios of the em square for
// xHeight/capHeight.
type Metrics struct {
	Ascender  int     `yaml:"ascender" json:"ascender"`
	Descender int     `yaml:"descender" json:"descender"`
	XHeight   float64 `yaml:"xHeight" json:"xHeight"`
	CapHeight float64 `yaml:"capHeight" json:"capHeight"`
	LineGap   int     `yaml:"lineGap" json:"lineGap"`
}

// Style controls the parametric glyph engine.
type Style struct {
	Skeleton       string  `yaml:"skeleton" json:"skeleton"`
	Weight         int     `yaml:"weight" json:"weight"`
	Width          float64 `yaml:"width" json:"width"`
	Slant          float64 `yaml:"slant" json:"slant"`
	Contrast       float64 `yaml:"contrast" json:"contrast"`
	Terminal       string  `yaml:"terminal" json:"terminal"`
	Serif          string  `yaml:"serif" json:"serif"`
	SerifSize      float64 `yaml:"serifSize" json:"serifSize"`
	Overshoot      float64 `yaml:"overshoot" json:"overshoot"`
	Tracking       float64 `yaml:"tracking" json:"tracking"`
	Roughness      float64 `yaml:"roughness" json:"roughness"`
	JitterRotation float64 `yaml:"jitterRotation" json:"jitterRotation"`
	Seed           int64   `yaml:"seed" json:"seed"`
}

// Coverage selects generated and derived scripts.
type Coverage struct {
	Generated []string     `yaml:"generated" json:"generated"`
	Base      BaseCoverage `yaml:"base" json:"base"`
}

// BaseCoverage selects scripts supplied by the embedded OFL base fonts.
type BaseCoverage struct {
	CJK     string `yaml:"cjk" json:"cjk"`
	Kana    bool   `yaml:"kana" json:"kana"`
	Hangul  bool   `yaml:"hangul" json:"hangul"`
	Symbols bool   `yaml:"symbols" json:"symbols"`
}

// Glyphs points at optional SVG overrides.
type Glyphs struct {
	Overrides string `yaml:"overrides" json:"overrides"`
}

// OpenType holds optional, agent-authored layout rules.
type OpenType struct {
	Contextual []ContextualRule `yaml:"contextual" json:"contextual"`
}

// ContextualRule maps an input character sequence to an existing glyph when
// the sequence appears in text. Feature defaults to calt.
type ContextualRule struct {
	Feature string `yaml:"feature" json:"feature"`
	Input   string `yaml:"input" json:"input"`
	Target  string `yaml:"target" json:"target"`
}

// Output selects formats, platform targets, and specimen behavior.
type Output struct {
	Formats  []string `yaml:"formats" json:"formats"`
	Targets  []string `yaml:"targets" json:"targets"`
	Specimen bool     `yaml:"specimen" json:"specimen"`
	Open     bool     `yaml:"open" json:"open"`
	Variable bool     `yaml:"variable" json:"variable"`
	Cursive  bool     `yaml:"cursive" json:"cursive"`
	SplitCJK int      `yaml:"splitCJK" json:"splitCJK"`
	TMP      TMP      `yaml:"tmp" json:"tmp"`
}

// TMP is the default Unity TextMeshPro font asset configuration.
type TMP struct {
	Population        string `yaml:"population" json:"population"`
	SamplingPointSize int    `yaml:"samplingPointSize" json:"samplingPointSize"`
	Padding           int    `yaml:"padding" json:"padding"`
	AtlasWidth        int    `yaml:"atlasWidth" json:"atlasWidth"`
	AtlasHeight       int    `yaml:"atlasHeight" json:"atlasHeight"`
	MultiAtlas        bool   `yaml:"multiAtlas" json:"multiAtlas"`
	RenderMode        string `yaml:"renderMode" json:"renderMode"`
}

// ValidationError is a field-specific recipe error.
type ValidationError struct {
	Field   string
	Message string
	Value   any
}

func (e *ValidationError) Error() string { return e.Field + ": " + e.Message }
