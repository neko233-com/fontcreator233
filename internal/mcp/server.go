// Package mcp implements a minimal MCP stdio server for fontcreator233.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/neko233-com/fontcreator233/internal/agentschema"
	"github.com/neko233-com/fontcreator233/internal/build"
	"github.com/neko233-com/fontcreator233/internal/inspect"
	"github.com/neko233-com/fontcreator233/internal/recipe"
	"github.com/neko233-com/fontcreator233/internal/skeleton"
	"github.com/neko233-com/fontcreator233/internal/specimen"
	"github.com/neko233-com/fontcreator233/internal/unicodes"
	"github.com/neko233-com/fontcreator233/internal/workspace"
)

const protocolVersion = "2025-06-18"

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// Server serves MCP over a newline-delimited JSON-RPC stream.
type Server struct{}

// Serve processes messages until EOF.
func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	enc := json.NewEncoder(out)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			_ = enc.Encode(rpcResponse{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "parse error"}})
			continue
		}
		resp, notifyOnly := s.handle(ctx, &req)
		if notifyOnly {
			continue
		}
		if err := enc.Encode(resp); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (s *Server) handle(ctx context.Context, req *rpcRequest) (rpcResponse, bool) {
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
	hasID := len(req.ID) > 0 && string(req.ID) != "null"
	switch req.Method {
	case "notifications/initialized", "initialized":
		return resp, true
	case "initialize":
		resp.Result = map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "fontcreator233", "version": agentschema.Version},
		}
	case "ping":
		resp.Result = map[string]any{}
	case "tools/list":
		resp.Result = map[string]any{"tools": toolDefinitions()}
	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &rpcError{Code: -32602, Message: "invalid params: " + err.Error()}
			break
		}
		resp.Result = s.callTool(ctx, params.Name, params.Arguments)
	default:
		resp.Error = &rpcError{Code: -32601, Message: "method not found: " + req.Method}
	}
	if !hasID {
		return resp, true
	}
	return resp, false
}

func toolDefinitions() []map[string]any {
	return []map[string]any{
		{"name": "fontcreator_describe", "description": "Return the complete machine-readable fontcreator233 contract.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{}}},
		{"name": "fontcreator_validate", "description": "Validate a recipe without building.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"recipe": map[string]any{"type": "string"}, "filename": map[string]any{"type": "string"}}, "required": []string{"recipe"}}},
		{"name": "fontcreator_build", "description": "Build a recipe into fonts, specimen, and platform packages.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"recipe": map[string]any{"type": "string"}, "filename": map[string]any{"type": "string"}, "outDir": map[string]any{"type": "string"}, "targets": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}, "splitCJK": map[string]any{"type": "integer"}, "jobs": map[string]any{"type": "integer"}, "open": map[string]any{"type": "boolean"}}, "required": []string{"recipe", "outDir"}}},
		{"name": "fontcreator_inspect", "description": "Inspect a compiled TTF/OTF.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"fontPath": map[string]any{"type": "string"}}, "required": []string{"fontPath"}}},
		{"name": "fontcreator_specimen", "description": "Regenerate the offline HTML specimen for a font.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"fontPath": map[string]any{"type": "string"}, "outDir": map[string]any{"type": "string"}, "open": map[string]any{"type": "boolean"}}, "required": []string{"fontPath"}}},
		{"name": "fontcreator_library_list", "description": "List the built-in skeleton glyph library.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{}}},
		{"name": "fontcreator_status", "description": "Report workspace managed files, staging, and reclaimable space.", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"outDir": map[string]any{"type": "string"}}, "required": []string{"outDir"}}},
	}
}

func (s *Server) callTool(ctx context.Context, name string, args json.RawMessage) any {
	var payload any
	var err error
	switch name {
	case "fontcreator_describe":
		payload = agentschema.Describe()
	case "fontcreator_validate":
		var a struct {
			Recipe   string `json:"recipe"`
			Filename string `json:"filename"`
		}
		if err = json.Unmarshal(args, &a); err == nil {
			if a.Filename == "" {
				a.Filename = "mcp.recipe.yaml"
			}
			var r *recipe.Recipe
			r, err = recipe.Parse([]byte(a.Recipe), a.Filename)
			if err == nil {
				payload = map[string]any{"ok": true, "family": r.Meta.Family, "style": r.Meta.Style}
			}
		}
	case "fontcreator_build":
		var a struct {
			Recipe   string   `json:"recipe"`
			Filename string   `json:"filename"`
			OutDir   string   `json:"outDir"`
			Targets  []string `json:"targets"`
			SplitCJK int      `json:"splitCJK"`
			Jobs     int      `json:"jobs"`
			Open     bool     `json:"open"`
		}
		if err = json.Unmarshal(args, &a); err == nil {
			if a.Filename == "" {
				a.Filename = "mcp.recipe.yaml"
			}
			var r *recipe.Recipe
			r, err = recipe.Parse([]byte(a.Recipe), a.Filename)
			if err == nil {
				var res build.Result
				res, err = build.Build(ctx, r, build.Options{
					OutDir:   a.OutDir,
					Targets:  a.Targets,
					SplitCJK: a.SplitCJK,
					Jobs:     a.Jobs,
					NoOpen:   !a.Open,
				}, nil)
				if err == nil {
					payload = map[string]any{
						"ok": true, "font": res.FontPath, "woff": res.WOFFPath,
						"specimen": res.SpecimenPath, "family": res.Family, "style": res.Style,
						"sha256": res.SHA256, "glyphs": res.Glyphs, "coverage": res.Coverage,
						"warnings": res.Warnings, "targets": res.Targets, "split": res.Split,
					}
				}
			}
		}
	case "fontcreator_inspect":
		var a struct {
			FontPath string `json:"fontPath"`
		}
		if err = json.Unmarshal(args, &a); err == nil {
			var data []byte
			data, err = os.ReadFile(a.FontPath)
			if err == nil {
				payload, err = inspect.Inspect(data)
			}
		}
	case "fontcreator_specimen":
		var a struct {
			FontPath string `json:"fontPath"`
			OutDir   string `json:"outDir"`
			Open     bool   `json:"open"`
		}
		if err = json.Unmarshal(args, &a); err == nil {
			var data []byte
			data, err = os.ReadFile(a.FontPath)
			if err == nil {
				var info inspect.Info
				info, err = inspect.Inspect(data)
				if err == nil {
					var glyphs []specimen.GlyphInfo
					glyphs, err = specimen.GlyphsFromFont(data)
					if err == nil {
						dir := a.OutDir
						if dir == "" {
							dir = filepath.Dir(a.FontPath)
						}
						var path string
						path, err = specimen.Build(specimen.Options{
							OutDir: dir, FontPath: a.FontPath, FontFile: filepath.Base(a.FontPath),
							Open: a.Open,
							Data: specimen.Data{
								Family: info.Family, Style: info.Style, Version: info.Version,
								SHA256: info.SHA256, Glyphs: glyphs, Coverage: info.Coverage,
								SampleText: "Hamburgefonstiv 字体创造器 0123",
							},
						})
						if err == nil {
							payload = map[string]any{"ok": true, "specimen": path}
						}
					}
				}
			}
		}
	case "fontcreator_library_list":
		counts := map[string]int{}
		runes := skeleton.Available()
		for _, r := range runes {
			counts[unicodes.ScriptOf(r)]++
		}
		payload = map[string]any{"total": len(runes), "scripts": counts}
	case "fontcreator_status":
		var a struct {
			OutDir string `json:"outDir"`
		}
		if err = json.Unmarshal(args, &a); err == nil {
			var ws *workspace.Workspace
			ws, err = workspace.Open(a.OutDir, workspace.Options{})
			if err == nil {
				var st workspace.Status
				st, err = ws.Status()
				if err == nil {
					payload = st
				}
			}
		}
	default:
		return toolError("unknown tool: " + name)
	}
	if err != nil {
		return toolError(err.Error())
	}
	data, _ := json.MarshalIndent(payload, "", "  ")
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": string(data)}},
	}
}

func toolError(message string) map[string]any {
	data, _ := json.Marshal(map[string]string{"error": message})
	return map[string]any{
		"isError": true,
		"content": []map[string]any{{"type": "text", "text": string(data)}},
	}
}
