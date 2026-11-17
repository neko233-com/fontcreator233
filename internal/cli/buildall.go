package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/neko233-com/fontcreator233/internal/build"
	"github.com/neko233-com/fontcreator233/internal/parallel"
	"github.com/neko233-com/fontcreator233/internal/recipe"
	"github.com/neko233-com/fontcreator233/internal/specimen"
	"github.com/neko233-com/fontcreator233/internal/ui"
	"github.com/neko233-com/fontcreator233/internal/workspace"
)

func (a *app) cmdBuildAll(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("build-all", &common)
	dir := fs.String("dir", "examples", "directory containing recipe YAML files")
	out := fs.String("out", "dist", "output directory")
	fontJobs := fs.Int("font-jobs", 0, "number of fonts to build concurrently")
	jobs := fs.Int("jobs", 0, "worker count per font")
	var targets stringList
	fs.Var(&targets, "target", "universal|web|system|unity|unreal|all")
	split := fs.Int("split-cjk", 0, "split CJK/Hangul chunks")
	assetsDir := fs.String("assets-dir", "", "shared web assets directory")
	selfContained := fs.Bool("self-contained-html", false, "inline web assets")
	open := fs.Bool("open", false, "open specimens after building")
	noAutoClean := fs.Bool("no-auto-clean", false, "disable automatic workspace cleanup")
	force := fs.Bool("force", false, "overwrite untracked/modified output files")
	values := map[string]bool{"dir": true, "out": true, "font-jobs": true, "jobs": true, "target": true, "split-cjk": true, "assets-dir": true}
	if err := parseFlexible(fs, args, values); err != nil {
		a.json = common.json
		return a.fail("build-all", 1, err)
	}
	a.json = common.json
	entries, err := os.ReadDir(*dir)
	if err != nil {
		return a.fail("build-all", 1, err)
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.ToLower(e.Name())
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".json") {
			paths = append(paths, filepath.Join(*dir, e.Name()))
		}
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return a.fail("build-all", 1, fmt.Errorf("no recipe files in %s", *dir))
	}
	recipes := make([]*recipe.Recipe, len(paths))
	for i, p := range paths {
		r, err := recipe.Load(p)
		if err != nil {
			return a.fail("build-all", 1, err)
		}
		recipes[i] = r
	}
	workers := *fontJobs
	if workers <= 0 {
		workers = len(recipes)
	}
	perFont := *jobs
	if perFont <= 0 {
		perFont = runtime.NumCPU() / workers
		if perFont < 1 {
			perFont = 1
		}
	}
	ws, err := workspace.Open(*out, workspace.Options{NoAutoClean: *noAutoClean, KeepHistory: 100, Force: *force})
	if err != nil {
		return a.fail("build-all", 1, err)
	}
	stage, err := ws.Begin(workspace.BuildMeta{Family: "build-all", Style: "batch", Targets: []string(targets)})
	if err != nil {
		return a.fail("build-all", 1, err)
	}
	defer ws.Abort(stage)
	if _, err := specimen.EnsureAssets(stage.Root, *assetsDir); err != nil {
		return a.fail("build-all", 1, err)
	}
	results, err := parallel.Map(context.Background(), recipes, workers, func(ctx context.Context, i int, r *recipe.Recipe) (build.Result, error) {
		useTargets := []string(targets)
		if len(useTargets) == 0 {
			useTargets = r.Output.Targets
		}
		return build.Build(ctx, r, build.Options{
			OutDir:            stage.Root,
			Targets:           useTargets,
			SplitCJK:          *split,
			Jobs:              perFont,
			NoOpen:            !*open || isCI(),
			AssetsDir:         *assetsDir,
			SelfContainedHTML: *selfContained,
		}, ui.NopReporter{})
	})
	if err != nil {
		return a.fail("build-all", 1, err)
	}
	if _, err := ws.Commit(stage); err != nil {
		return a.fail("build-all", 1, err)
	}
	for i := range results {
		rewriteResultPaths(&results[i], stage.Root, ws.Dir)
	}
	if a.json {
		list := make([]map[string]any, 0, len(results))
		for _, res := range results {
			list = append(list, map[string]any{
				"family":         res.Family,
				"style":          res.Style,
				"font":           res.FontPath,
				"woff":           res.WOFFPath,
				"specimen":       res.SpecimenPath,
				"sha256":         res.SHA256,
				"glyphs":         res.Glyphs,
				"coverage":       res.Coverage,
				"coverageReport": res.CoverageReport,
				"warnings":       res.Warnings,
			})
		}
		a.writeJSON(okMap("build-all", map[string]any{"results": list}))
		return 0
	}
	for _, res := range results {
		fmt.Fprintf(a.io.Out, "%s %s · %d glyphs · %s\n", res.Family, res.Style, res.Glyphs, res.SHA256[:12])
		if res.SpecimenPath != "" {
			fmt.Fprintf(a.io.Out, "  specimen: %s\n", res.SpecimenPath)
		}
	}
	return 0
}
