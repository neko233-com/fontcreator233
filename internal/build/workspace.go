package build

import (
	"context"
	"path/filepath"

	"github.com/neko233-com/fontcreator233/internal/recipe"
	"github.com/neko233-com/fontcreator233/internal/ui"
	"github.com/neko233-com/fontcreator233/internal/workspace"
)

// BuildToWorkspace stages and atomically commits one build. Every writing
// surface (CLI, MCP, bundle rebuild) should use this path so user files and
// previous builds are protected by the same rules.
func BuildToWorkspace(ctx context.Context, ws *workspace.Workspace, meta workspace.BuildMeta, r *recipe.Recipe, o Options, rep ui.Reporter) (Result, []string, error) {
	stage, err := ws.Begin(meta)
	if err != nil {
		return Result{}, nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = ws.Abort(stage)
		}
	}()
	o.OutDir = stage.Root
	if o.FinalOutDir == "" {
		o.FinalOutDir = ws.Dir
	}
	res, err := Build(ctx, r, o, rep)
	if err != nil {
		return res, nil, err
	}
	files, err := ws.Commit(stage)
	if err != nil {
		return res, nil, err
	}
	committed = true
	RewritePaths(&res, stage.Root, ws.Dir)
	return res, files, nil
}

// RewritePaths maps result paths from the staging root to the final output
// directory.
func RewritePaths(res *Result, stageDir, outDir string) {
	rewrite := func(p string) string {
		if p == "" {
			return ""
		}
		rel, err := filepath.Rel(stageDir, p)
		if err != nil {
			return p
		}
		return filepath.Join(outDir, rel)
	}
	res.FontPath = rewrite(res.FontPath)
	res.WOFFPath = rewrite(res.WOFFPath)
	res.SpecimenPath = rewrite(res.SpecimenPath)
	for i := range res.Split {
		res.Split[i] = rewrite(res.Split[i])
	}
	for target, paths := range res.Targets {
		for i := range paths {
			paths[i] = rewrite(paths[i])
		}
		res.Targets[target] = paths
	}
}
