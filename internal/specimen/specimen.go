// Package specimen writes the offline HTML review page and its local assets.
package specimen

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// GlyphInfo is one code point shown in the virtualized grid.
type GlyphInfo struct {
	Rune   rune
	Script string
}

// Data is everything the specimen needs besides the font file.
type Data struct {
	Family      string
	Style       string
	Description string
	Version     string
	SHA256      string
	RecipeName  string
	Accent      string
	SampleText  string
	Glyphs      []GlyphInfo
	Coverage    map[string]int
}

// OpenFunc opens a path in the desktop browser. Tests inject a recorder.
type OpenFunc func(string) error

// Options controls specimen generation.
type Options struct {
	OutDir        string
	FontPath      string
	FontFile      string
	Data          Data
	AssetsDir     string
	SelfContained bool
	Open          bool
	OpenFunc      OpenFunc
}

type templateData struct {
	Data          Data
	FontRel       string
	GlyphDataRel  string
	AssetRel      string
	FontFaceCSS   template.CSS
	CSS           template.CSS
	JS            template.JS
	VendorJS      template.JS
	GlyphJS       template.JS
	SelfContained bool
}

// Build writes `<font>.glyphs.js` and `<font>.html`, copying shared assets.
func Build(o Options) (string, error) {
	if o.OutDir == "" {
		return "", fmt.Errorf("specimen: empty output directory")
	}
	if o.FontPath == "" {
		return "", fmt.Errorf("specimen: empty font path")
	}
	if err := os.MkdirAll(o.OutDir, 0o755); err != nil {
		return "", err
	}
	d := o.Data
	if d.Accent == "" {
		d.Accent = "#5B8CFF"
	}
	if d.SampleText == "" {
		d.SampleText = "Hamburgefonstiv 字体创造器 0123"
	}
	fontName := o.FontFile
	if fontName == "" {
		fontName = filepath.Base(o.FontPath)
	}
	stem := strings.TrimSuffix(fontName, filepath.Ext(fontName))
	glyphName := stem + ".glyphs.js"
	glyphPath := filepath.Join(o.OutDir, glyphName)

	glyphJS := buildGlyphJS(d)
	if err := os.WriteFile(glyphPath, []byte(glyphJS), 0o644); err != nil {
		return "", err
	}

	td := templateData{Data: d, GlyphDataRel: glyphName, AssetRel: "assets", SelfContained: o.SelfContained, GlyphJS: template.JS(glyphJS)}
	fontRel, err := filepath.Rel(o.OutDir, o.FontPath)
	if err != nil {
		fontRel = fontName
	}
	td.FontRel = filepath.ToSlash(fontRel)
	escapedFont := (&url.URL{Path: td.FontRel}).EscapedPath()

	if !o.SelfContained {
		if _, err := EnsureAssets(o.OutDir, o.AssetsDir); err != nil {
			return "", err
		}
		if o.AssetsDir != "" {
			rel, err := filepath.Rel(o.OutDir, o.AssetsDir)
			if err == nil {
				td.AssetRel = filepath.ToSlash(rel)
			}
		}
		td.FontFaceCSS = template.CSS(fmt.Sprintf(
			`@font-face { font-family: "Specimen"; src: url("./%s") format("truetype"); font-display: swap; }`,
			escapedFont,
		))
	} else {
		css, err := inlineCSS()
		if err != nil {
			return "", err
		}
		js, _ := readEmbedded("webassets/specimen.js")
		vendor, _ := readEmbedded("webassets/vendor/tanstack-virtual.iife.js")
		fontBytes, err := os.ReadFile(o.FontPath)
		if err != nil {
			return "", err
		}
		mime := "font/ttf"
		if strings.HasSuffix(strings.ToLower(o.FontPath), ".woff") {
			mime = "font/woff"
		}
		uri := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(fontBytes)
		td.CSS = template.CSS(css)
		td.JS = template.JS(js)
		td.VendorJS = template.JS(vendor)
		td.FontFaceCSS = template.CSS(fmt.Sprintf(
			`@font-face { font-family: "Specimen"; src: url("%s") format("%s"); font-display: swap; }`,
			uri, map[string]string{"font/ttf": "truetype", "font/woff": "woff"}[mime],
		))
	}

	tmpl, err := template.New("specimen").Parse(string(specimenTemplate))
	if err != nil {
		return "", err
	}
	htmlPath := filepath.Join(o.OutDir, stem+".html")
	f, err := os.Create(htmlPath)
	if err != nil {
		return "", err
	}
	if err := tmpl.Execute(f, td); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	if o.Open && !isCI() {
		open := o.OpenFunc
		if open == nil {
			open = Open
		}
		if err := open(htmlPath); err != nil {
			// Opening is a convenience; the specimen is still valid.
			return htmlPath, nil
		}
	}
	return htmlPath, nil
}

func buildGlyphJS(d Data) string {
	cps := make([]int, 0, len(d.Glyphs))
	for _, g := range d.Glyphs {
		cps = append(cps, int(g.Rune))
	}
	payload := map[string]any{
		"family":   d.Family,
		"style":    d.Style,
		"accent":   d.Accent,
		"cps":      cps,
		"coverage": d.Coverage,
		"samples": map[string]string{
			"latin":    "Hamburgefonstiv 0123",
			"greek":    "Καλημέρα κόσμε",
			"russian":  "Съешь ещё этих мягких булок",
			"chinese":  "天地玄黄，宇宙洪荒。字体创造器",
			"japanese": "いろはにほへと ちりぬるを",
			"korean":   "다람쥐 헌 쳇바퀴에 타고파",
			"symbols":  "€ £ ¥ → ← ↑ ↓ “ ” ‘ ’ — …",
		},
	}
	raw, _ := json.Marshal(payload)
	return "window.FontCreatorData = " + string(raw) + ";\n"
}

func inlineCSS() (string, error) {
	css, err := readEmbedded("webassets/specimen.css")
	if err != nil {
		return "", err
	}
	cssStr := string(css)
	for _, name := range []string{"Geist-Variable.woff2", "GeistMono-Variable.woff2"} {
		data, err := readEmbedded("webassets/geist/" + name)
		if err != nil {
			return "", err
		}
		uri := "data:font/woff2;base64," + base64.StdEncoding.EncodeToString(data)
		cssStr = strings.ReplaceAll(cssStr, `./geist/`+name, uri)
	}
	return cssStr, nil
}

func isCI() bool {
	return strings.EqualFold(os.Getenv("CI"), "true") || strings.EqualFold(os.Getenv("CI"), "1")
}
