package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/neko233-com/fontcreator233/internal/build"
	"github.com/neko233-com/fontcreator233/internal/recipe"
	"github.com/neko233-com/fontcreator233/internal/specimen"
	"github.com/neko233-com/fontcreator233/internal/style"
	"github.com/neko233-com/fontcreator233/internal/ui"
	"github.com/neko233-com/fontcreator233/internal/workspace"
)

func (a *app) cmdBuild(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("build", &common)
	out := fs.String("out", "dist", "output directory")
	var targets stringList
	fs.Var(&targets, "target", "universal|web|system|unity|unreal|all (repeatable)")
	split := fs.Int("split-cjk", -1, "split CJK/Hangul into chunks of N codepoints (overrides recipe)")
	jobs := fs.Int("jobs", 0, "worker count (default CPU count)")
	assetsDir := fs.String("assets-dir", "", "shared web assets directory")
	selfContained := fs.Bool("self-contained-html", false, "inline font/CSS/JS into the HTML")
	noAutoClean := fs.Bool("no-auto-clean", false, "disable automatic workspace cleanup")
	force := fs.Bool("force", false, "overwrite untracked/modified output files")
	values := map[string]bool{"out": true, "target": true, "split-cjk": true, "jobs": true, "assets-dir": true}
	if err := parseFlexible(fs, args, values); err != nil {
		a.json = common.json
		return a.fail("build", 1, err)
	}
	a.json = common.json
	if fs.NArg() != 1 {
		return a.fail("build", 1, fmt.Errorf("usage: fontcreator build <recipe.yaml> [flags]"))
	}
	r, err := recipe.Load(fs.Arg(0))
	if err != nil {
		return a.fail("build", 1, err)
	}
	useTargets := []string(targets)
	if len(useTargets) == 0 {
		useTargets = r.Output.Targets
	}
	ws, err := workspace.Open(*out, workspace.Options{NoAutoClean: *noAutoClean, KeepHistory: 100, Force: *force})
	if err != nil {
		return a.fail("build", 1, err)
	}
	splitValue := r.Output.SplitCJK
	if *split >= 0 {
		splitValue = *split
	}
	styleName := style.ResolveStyle(r)
	meta := workspace.BuildMeta{
		Recipe:  fs.Arg(0),
		Family:  r.Meta.Family,
		Style:   styleName,
		Targets: useTargets,
	}
	opts := build.Options{
		Targets:           useTargets,
		SplitCJK:          splitValue,
		Jobs:              *jobs,
		NoOpen:            common.noOpen || isCI() || a.json,
		NoTUI:             common.noTUI,
		NoAutoClean:       *noAutoClean,
		AssetsDir:         *assetsDir,
		SelfContainedHTML: *selfContained,
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	var res build.Result
	buildFn := func(ctx context.Context, rep ui.Reporter) error {
		var err error
		res, _, err = build.BuildToWorkspace(ctx, ws, meta, r, opts, rep)
		return err
	}
	if a.interactive(common) {
		err = ui.RunTUI(ctx, r.Meta.Family+" / "+r.Meta.Style, buildFn)
	} else {
		err = buildFn(ctx, ui.NewPlain(a.io.Err, common.quiet, common.json))
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return a.fail("build", 130, err)
		}
		return a.fail("build", 1, err)
	}
	if res.SpecimenPath != "" && r.Output.Open && !common.noOpen && !isCI() && !a.json {
		_ = specimen.Open(res.SpecimenPath)
	}
	if a.json {
		extra := map[string]any{
			"font":           res.FontPath,
			"woff":           res.WOFFPath,
			"specimen":       res.SpecimenPath,
			"family":         res.Family,
			"style":          res.Style,
			"requestedStyle": res.RequestedStyle,
			"sha256":         res.SHA256,
			"bytes":          res.Bytes,
			"glyphs":         res.Glyphs,
			"coverage":       res.Coverage,
			"coverageReport": res.CoverageReport,
			"warnings":       res.Warnings,
			"durationMs":     res.Duration.Milliseconds(),
			"targets":        res.Targets,
			"split":          res.Split,
		}
		a.writeJSON(okMap("build", extra))
		return 0
	}
	if !common.quiet {
		fmt.Fprintf(a.io.Out, "%s %s · %d glyphs · %s\n", res.Family, res.Style, res.Glyphs, res.SHA256[:12])
		fmt.Fprintf(a.io.Out, "  font:     %s\n", res.FontPath)
		if res.WOFFPath != "" {
			fmt.Fprintf(a.io.Out, "  woff:     %s\n", res.WOFFPath)
		}
		if res.SpecimenPath != "" {
			fmt.Fprintf(a.io.Out, "  specimen: %s\n", res.SpecimenPath)
		}
		for _, w := range res.Warnings {
			fmt.Fprintf(a.io.Err, "warning: %s\n", w)
		}
	}
	return 0
}

func isCI() bool {
	v := os.Getenv("CI")
	return v == "true" || v == "1"
}
