package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neko233-com/fontcreator233/internal/recipe"
	"gopkg.in/yaml.v3"
)

func run(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(args, IO{In: bytes.NewReader(nil), Out: &out, Err: &errb})
	return out.String(), errb.String(), code
}

func sampleRecipeFile(t *testing.T) string {
	t.Helper()
	r := recipe.Default()
	r.Meta.Family = "CLI Test"
	r.Meta.License = "OFL-1.1"
	r.Coverage.Base.CJK = "sample"
	r.Output.Formats = []string{"ttf"}
	r.Output.Targets = []string{"universal"}
	r.Output.Specimen = true
	r.Output.Open = false
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	data, err := yaml.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "font.recipe.yaml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestVersionJSON(t *testing.T) {
	out, _, code := run(t, "version", "--json")
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if v["ok"] != true || v["version"] == "" {
		t.Fatalf("v = %#v", v)
	}
}

func TestValidateJSON(t *testing.T) {
	path := sampleRecipeFile(t)
	out, _, code := run(t, "validate", path, "--json")
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	if v["valid"] != true {
		t.Fatalf("v = %#v", v)
	}

	out, _, code = run(t, "validate", filepath.Join(t.TempDir(), "missing.yaml"), "--json")
	if code != 1 {
		t.Fatalf("missing code = %d", code)
	}
	if err := json.Unmarshal([]byte(out), &v); err != nil || v["ok"] != false {
		t.Fatalf("error envelope = %s (%v)", out, err)
	}
}

func TestBuildInspectSpecimen(t *testing.T) {
	path := sampleRecipeFile(t)
	dir := t.TempDir()
	out, errText, code := run(t, "build", path, "--out", dir, "--no-open", "--json")
	if code != 0 {
		t.Fatalf("build code=%d err=%s", code, errText)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("json: %v\n%s", err, out)
	}
	font, _ := v["font"].(string)
	if font == "" {
		t.Fatalf("font = %#v", v)
	}
	if _, err := os.Stat(font); err != nil {
		t.Fatal(err)
	}
	spec, _ := v["specimen"].(string)
	if spec == "" {
		t.Fatalf("specimen = %#v", v)
	}
	if _, err := os.Stat(spec); err != nil {
		t.Fatal(err)
	}

	out, _, code = run(t, "inspect", font, "--json")
	if code != 0 {
		t.Fatalf("inspect code=%d", code)
	}
	var info map[string]any
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		t.Fatal(err)
	}
	fontInfo := info["font"].(map[string]any)
	if fontInfo["family"] != "CLI Test" {
		t.Fatalf("info = %#v", fontInfo)
	}

	out, _, code = run(t, "specimen", font, "--out", t.TempDir(), "--no-open", "--json")
	if code != 0 {
		t.Fatalf("specimen code=%d", code)
	}
	var sv map[string]any
	if err := json.Unmarshal([]byte(out), &sv); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(sv["specimen"].(string)); err != nil {
		t.Fatal(err)
	}
}

func TestInit(t *testing.T) {
	dir := t.TempDir()
	out, _, code := run(t, "init", "--dir", dir, "--template", "serif", "--family", "Init Test", "--json")
	if code != 0 {
		t.Fatalf("init code=%d", code)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatal(err)
	}
	path := v["path"].(string)
	if _, err := recipe.Load(path); err != nil {
		t.Fatalf("generated recipe invalid: %v", err)
	}
	if _, _, code := run(t, "init", "--dir", dir); code != 1 {
		t.Fatalf("overwrite code = %d", code)
	}
	if _, _, code := run(t, "init", "--dir", dir, "--force"); code != 0 {
		t.Fatalf("force code = %d", code)
	}
}

func TestDescribeAndSchema(t *testing.T) {
	out, _, code := run(t, "describe", "--json")
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatal(err)
	}
	if v["recipeSchema"] == nil || v["workflow"] == nil {
		t.Fatalf("describe = %#v", v)
	}
	out, _, code = run(t, "describe", "--markdown")
	if code != 0 || !strings.Contains(out, "# fontcreator233") {
		t.Fatalf("markdown = %q", out)
	}
	out, _, code = run(t, "schema")
	if code != 0 || !strings.Contains(out, "fontcreator233/v1") {
		t.Fatalf("schema = %q", out)
	}
}

func TestLibrary(t *testing.T) {
	out, _, code := run(t, "library", "list", "--json")
	if code != 0 {
		t.Fatalf("list code=%d", code)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatal(err)
	}
	if v["total"].(float64) < 100 {
		t.Fatalf("library = %#v", v)
	}
	dir := t.TempDir()
	out, _, code = run(t, "library", "export", "--dir", dir, "--json")
	if code != 0 {
		t.Fatalf("export code=%d", code)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) < 100 {
		t.Fatalf("entries=%d err=%v", len(entries), err)
	}
	if !strings.HasSuffix(entries[0].Name(), ".svg") {
		t.Fatalf("entry = %s", entries[0].Name())
	}
}

func TestUnknownCommandJSON(t *testing.T) {
	out, _, code := run(t, "nope", "--json")
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatal(err)
	}
	if v["ok"] != false {
		t.Fatalf("v = %#v", v)
	}
}
