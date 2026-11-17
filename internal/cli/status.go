package cli

import (
	"fmt"

	"github.com/neko233-com/fontcreator233/internal/workspace"
)

func (a *app) cmdStatus(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("status", &common)
	out := fs.String("out", "dist", "output directory")
	values := map[string]bool{"out": true}
	if err := parseFlexible(fs, args, values); err != nil {
		a.json = common.json
		return a.fail("status", 1, err)
	}
	a.json = common.json
	ws, err := workspace.Open(*out, workspace.Options{})
	if err != nil {
		return a.fail("status", 1, err)
	}
	st, err := ws.Status()
	if err != nil {
		return a.fail("status", 1, err)
	}
	if a.json {
		a.writeJSON(okMap("status", map[string]any{"workspace": st}))
		return 0
	}
	fmt.Fprintf(a.io.Out, "workspace %s\n", st.Dir)
	fmt.Fprintf(a.io.Out, "  managed files: %d (%d bytes)\n", st.Files, st.Bytes)
	fmt.Fprintf(a.io.Out, "  builds:        %d\n", st.Builds)
	fmt.Fprintf(a.io.Out, "  staging:       %d (%d bytes)\n", st.Staging, st.StagingBytes)
	fmt.Fprintf(a.io.Out, "  reclaimable:   %d bytes\n", st.Reclaimable)
	for _, p := range st.Modified {
		fmt.Fprintf(a.io.Out, "  modified:      %s\n", p)
	}
	for _, p := range st.Missing {
		fmt.Fprintf(a.io.Out, "  missing:       %s\n", p)
	}
	return 0
}
