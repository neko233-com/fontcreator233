package recipe

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultIsValid(t *testing.T) {
	r := Default()
	if err := r.Validate(); err != nil {
		t.Fatalf("default invalid: %v", err)
	}
	if r.APIVersion != "fontcreator233/v1" {
		t.Fatalf("apiVersion = %q", r.APIVersion)
	}
	if r.Style.Weight != 400 || r.Style.Width != 100 {
		t.Fatalf("style defaults = %#v", r.Style)
	}
	if r.Coverage.Base.CJK != "common" {
		t.Fatalf("cjk = %q", r.Coverage.Base.CJK)
	}
	if len(r.Output.Targets) == 0 {
		t.Fatal("output targets empty")
	}
}

func TestParseStrictUnknownField(t *testing.T) {
	_, err := Parse([]byte("apiVersion: fontcreator233/v1\nmeta:\n  family: Test\nstyle:\n  weight: 400\n  wat: 1\n"), "test.yaml")
	if err == nil || !strings.Contains(err.Error(), "wat") {
		t.Fatalf("error = %v, want unknown field wat", err)
	}
}

func TestParseDuplicateYAMLKey(t *testing.T) {
	_, err := Parse([]byte("meta:\n  family: A\nmeta:\n  family: B\n"), "test.yaml")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		t.Fatalf("error = %v, want duplicate", err)
	}
}

func TestParseDuplicateJSONKey(t *testing.T) {
	_, err := Parse([]byte(`{"meta":{"family":"A","family":"B"}}`), "test.json")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		t.Fatalf("error = %v, want duplicate", err)
	}
}

func TestParseContextualRules(t *testing.T) {
	data := []byte(`{"apiVersion":"fontcreator233/v1","meta":{"family":"Contextual Font"},"openType":{"contextual":[{"feature":"calt","input":"<-","target":"←"}]}}`)
	r, err := Parse(data, "test.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.OpenType.Contextual) != 1 {
		t.Fatalf("contextual = %#v", r.OpenType.Contextual)
	}
	rule := r.OpenType.Contextual[0]
	if rule.Feature != "calt" || rule.Input != "<-" || rule.Target != "←" {
		t.Fatalf("rule = %#v", rule)
	}
	_, err = Parse([]byte(`{"apiVersion":"fontcreator233/v1","meta":{"family":"Bad"},"openType":{"contextual":[{"input":"<-","target":"ab"}]}}`), "bad.json")
	if err == nil || !strings.Contains(err.Error(), "exactly one character") {
		t.Fatalf("error = %v, want target validation", err)
	}
}

func TestParseColorConfig(t *testing.T) {
	data := []byte(`{"apiVersion":"fontcreator233/v1","meta":{"family":"Color Font"},"openType":{"color":{"palettes":[["#5B8CFF","#FFFFFF"],["#FF6B4A","#FFF3E0"]],"glyphs":[{"codepoint":"E000","advance":700,"layers":[{"shape":"circle","color":0},{"shape":"star","color":1,"scale":0.5}]}]}}}`)
	r, err := Parse(data, "test.json")
	if err != nil {
		t.Fatal(err)
	}
	if r.OpenType.Color == nil || len(r.OpenType.Color.Palettes) != 2 || len(r.OpenType.Color.Glyphs) != 1 {
		t.Fatalf("color config = %#v", r.OpenType.Color)
	}
	if r.OpenType.Color.Glyphs[0].Layers[1].Shape != "star" || r.OpenType.Color.Glyphs[0].Layers[1].Scale != 0.5 {
		t.Fatalf("layer = %#v", r.OpenType.Color.Glyphs[0].Layers[1])
	}
	// Palettes with different lengths must fail validation.
	_, err = Parse([]byte(`{"apiVersion":"fontcreator233/v1","meta":{"family":"Bad"},"openType":{"color":{"palettes":[["#fff","#000"],["#fff"]],"glyphs":[{"codepoint":"E000","layers":[{"shape":"circle","color":0}]}]}}}`), "bad.json")
	if err == nil || !strings.Contains(err.Error(), "same number of colors") {
		t.Fatalf("error = %v, want palette length validation", err)
	}
}

func TestParseJSON(t *testing.T) {
	data := []byte(`{"apiVersion":"fontcreator233/v1","meta":{"family":"JSON Font"},"style":{"weight":500}}`)
	r, err := Parse(data, "test.json")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if r.Meta.Family != "JSON Font" || r.Style.Weight != 500 {
		t.Fatalf("r = %#v", r)
	}
	if r.Metrics.Ascender != 800 {
		t.Fatalf("defaults not merged: %#v", r.Metrics)
	}
}

func TestParseJSONTrailing(t *testing.T) {
	_, err := Parse([]byte(`{"apiVersion":"fontcreator233/v1"} {}`), "test.json")
	if err == nil {
		t.Fatal("expected trailing data error")
	}
}

func TestValidationCases(t *testing.T) {
	tests := []struct {
		name  string
		mut   func(*Recipe)
		field string
	}{
		{"weight high", func(r *Recipe) { r.Style.Weight = 1200 }, "style.weight"},
		{"width low", func(r *Recipe) { r.Style.Width = 50 }, "style.width"},
		{"license with base", func(r *Recipe) { r.Meta.License = "MIT" }, "meta.license"},
		{"noto family", func(r *Recipe) { r.Meta.Family = "Noto Sans" }, "meta.family"},
		{"empty family", func(r *Recipe) { r.Meta.Family = "" }, "meta.family"},
		{"nan xheight", func(r *Recipe) { r.Metrics.XHeight = math.NaN() }, "metrics.xHeight"},
		{"cap high", func(r *Recipe) { r.Metrics.CapHeight = 0.9 }, "metrics.capHeight"},
		{"bad target", func(r *Recipe) { r.Output.Targets = []string{"nope"} }, "output.targets"},
		{"bad atlas", func(r *Recipe) { r.Output.TMP.AtlasWidth = 1000 }, "output.tmp.atlasWidth"},
		{"bad format", func(r *Recipe) { r.Output.Formats = []string{"otf"} }, "output.formats"},
		{"negative split", func(r *Recipe) { r.Output.SplitCJK = -1 }, "output.splitCJK"},
		{"bad skeleton", func(r *Recipe) { r.Style.Skeleton = "weird" }, "style.skeleton"},
		{"bad cjk", func(r *Recipe) { r.Coverage.Base.CJK = "all" }, "coverage.base.cjk"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Default()
			r.Meta.Family = "Valid"
			r.Meta.License = "OFL-1.1"
			tt.mut(r)
			err := r.Validate()
			if err == nil {
				t.Fatal("expected error")
			}
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("error type %T: %v", err, err)
			}
			if ve.Field != tt.field {
				t.Fatalf("field = %q, want %q (%v)", ve.Field, tt.field, err)
			}
		})
	}
}

func TestLoadSetsSourcePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "font.recipe.yaml")
	data := "apiVersion: fontcreator233/v1\nmeta:\n  family: Loaded\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if r.SourcePath != path {
		t.Fatalf("source path = %q, want %q", r.SourcePath, path)
	}
}

func TestSerifSizeDefault(t *testing.T) {
	r := Default()
	r.Meta.Family = "Serif"
	r.Meta.License = "OFL-1.1"
	r.Style.Serif = "slab"
	r.Style.SerifSize = 0
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	if r.Style.SerifSize == 0 {
		t.Fatal("serif size should normalize to a non-zero default")
	}
}

func TestSchemaKeyFields(t *testing.T) {
	s := Schema()
	props := s["properties"].(map[string]any)
	style := props["style"].(map[string]any)["properties"].(map[string]any)
	weight := style["weight"].(map[string]any)
	if weight["minimum"].(float64) != 100 || weight["maximum"].(float64) != 900 {
		t.Fatalf("weight schema = %#v", weight)
	}
	out := props["output"].(map[string]any)["properties"].(map[string]any)
	targets := out["targets"].(map[string]any)["items"].(map[string]any)
	enum := targets["enum"].([]any)
	found := false
	for _, v := range enum {
		if v == "unity" {
			found = true
		}
	}
	if !found {
		t.Fatalf("targets enum = %#v", enum)
	}
}
