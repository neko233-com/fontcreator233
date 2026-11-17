package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/neko233-com/fontcreator233/internal/workspace"
)

func (a *app) cmdClean(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("clean", &common)
	out := fs.String("out", "dist", "output directory")
	dry := fs.Bool("dry-run", false, "list what would be deleted")
	all := fs.Bool("all", false, "remove all managed output and state")
	yes := fs.Bool("yes", false, "skip confirmation")
	force := fs.Bool("force", false, "delete modified managed files")
	var targets stringList
	fs.Var(&targets, "target", "limit to universal|web|system|unity|unreal")
	older := fs.String("older-than", "", "only files built before this age, e.g. 7d or 30d")
	values := map[string]bool{"out": true, "target": true, "older-than": true}
	if err := parseFlexible(fs, args, values); err != nil {
		a.json = common.json
		return a.fail("clean", 1, err)
	}
	a.json = common.json
	if !*dry && !*yes {
		return a.fail("clean", 1, fmt.Errorf("refusing to delete without --yes (or use --dry-run)"))
	}
	var age time.Duration
	if *older != "" {
		var err error
		age, err = parseAge(*older)
		if err != nil {
			return a.fail("clean", 1, err)
		}
	}
	ws, err := workspace.Open(*out, workspace.Options{})
	if err != nil {
		return a.fail("clean", 1, err)
	}
	res, err := ws.Clean(workspace.CleanOptions{
		DryRun:    *dry,
		All:       *all,
		Force:     *force,
		Targets:   targets,
		OlderThan: age,
	})
	if err != nil {
		return a.fail("clean", 1, err)
	}
	if a.json {
		a.writeJSON(okMap("clean", map[string]any{"result": res}))
		return 0
	}
	for _, p := range res.Deleted {
		fmt.Fprintf(a.io.Out, "deleted  %s\n", p)
	}
	for _, p := range res.Modified {
		fmt.Fprintf(a.io.Out, "modified %s (kept; use --force)\n", p)
	}
	for _, p := range res.Skipped {
		fmt.Fprintf(a.io.Out, "skipped  %s\n", p)
	}
	fmt.Fprintf(a.io.Out, "%d files, %d bytes%s\n", len(res.Deleted), res.Bytes, map[bool]string{true: " (dry run)"}[res.DryRun])
	return 0
}

func parseAge(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if strings.HasSuffix(s, "d") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil || n < 0 {
			return 0, fmt.Errorf("invalid age %q", s)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < 0 {
		return 0, fmt.Errorf("invalid age %q", s)
	}
	return d, nil
}
