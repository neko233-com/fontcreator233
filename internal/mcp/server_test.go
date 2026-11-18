package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/neko233-com/fontcreator233/internal/recipe"
	"gopkg.in/yaml.v3"
)

func recipeYAML(t *testing.T) string {
	t.Helper()
	r := recipe.Default()
	r.Meta.Family = "MCP Test"
	r.Meta.License = "OFL-1.1"
	r.Coverage.Base.CJK = "sample"
	r.Output.Formats = []string{"ttf"}
	r.Output.Targets = []string{"universal"}
	r.Output.Specimen = false
	r.Output.Open = false
	data, err := yaml.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func call(t *testing.T, outDir, fontPath string) map[float64]map[string]any {
	t.Helper()
	recipeText := recipeYAML(t)
	buildArgs, _ := json.Marshal(map[string]any{"recipe": recipeText, "outDir": outDir, "targets": []string{"universal"}})
	validateArgs, _ := json.Marshal(map[string]any{"recipe": recipeText})
	lines := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"fontcreator_describe","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"fontcreator_validate","arguments":` + string(validateArgs) + `}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"fontcreator_build","arguments":` + string(buildArgs) + `}}`,
		`not json`,
		`{"jsonrpc":"2.0","id":7,"method":"unknown/method"}`,
	}
	if fontPath != "" {
		inspectArgs, _ := json.Marshal(map[string]any{"fontPath": fontPath})
		lines = append(lines, `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"fontcreator_inspect","arguments":`+string(inspectArgs)+`}}`)
	}
	var out bytes.Buffer
	server := &Server{}
	if err := server.Serve(context.Background(), strings.NewReader(strings.Join(lines, "\n")+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	responses := map[float64]map[string]any{}
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Fatalf("response %q: %v", line, err)
		}
		if id, ok := obj["id"].(float64); ok {
			responses[id] = obj
		}
	}
	return responses
}

func content(t *testing.T, resp map[string]any) map[string]any {
	t.Helper()
	result := resp["result"].(map[string]any)
	content := result["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)
	var obj map[string]any
	if err := json.Unmarshal([]byte(text), &obj); err != nil {
		t.Fatalf("content %q: %v", text, err)
	}
	return obj
}

func TestMCPProtocolAndTools(t *testing.T) {
	outDir := t.TempDir()
	res := call(t, outDir, "")
	if res[1] == nil || res[1]["result"].(map[string]any)["protocolVersion"] != protocolVersion {
		t.Fatalf("initialize = %#v", res[1])
	}
	tools := res[2]["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 8 {
		t.Fatalf("tools = %d", len(tools))
	}
	desc := content(t, res[3])
	if desc["recipeSchema"] == nil || desc["workflow"] == nil {
		t.Fatalf("describe = %#v", desc)
	}
	valid := content(t, res[4])
	if valid["ok"] != true {
		t.Fatalf("validate = %#v", valid)
	}
	built := content(t, res[5])
	if built["ok"] != true || built["font"] == "" {
		t.Fatalf("build = %#v", built)
	}
	font := built["font"].(string)
	if !strings.HasSuffix(font, ".ttf") {
		t.Fatalf("font = %q", font)
	}
	if _, err := filepath.Abs(font); err != nil {
		t.Fatal(err)
	}
	if res[7] == nil || res[7]["error"] == nil {
		t.Fatalf("unknown method = %#v", res[7])
	}
}

func TestMCPInspect(t *testing.T) {
	outDir := t.TempDir()
	first := call(t, outDir, "")
	built := content(t, first[5])
	font := built["font"].(string)
	second := call(t, t.TempDir(), font)
	info := content(t, second[8])
	if info["family"] != "MCP Test" || info["glyphs"].(float64) < 100 {
		t.Fatalf("inspect = %#v", info)
	}
}

func TestMCPToolError(t *testing.T) {
	var out bytes.Buffer
	line := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nope","arguments":{}}}` + "\n"
	if err := (&Server{}).Serve(context.Background(), strings.NewReader(line), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"isError":true`) {
		t.Fatalf("out = %s", out.String())
	}
}
