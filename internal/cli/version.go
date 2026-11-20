package cli

import (
	"fmt"

	"github.com/neko233-com/fontcreator233/internal/agentschema"
)

func (a *app) cmdVersion(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("version", &common)
	if err := parseFlexible(fs, args, nil); err != nil {
		a.json = common.json
		return a.fail("version", 1, err)
	}
	a.json = common.json
	info := agentschema.Info()
	if a.json {
		a.writeJSON(map[string]any{
			"ok":        true,
			"command":   "version",
			"version":   info.Version,
			"commit":    info.Commit,
			"buildDate": info.BuildDate,
			"goVersion": info.GoVersion,
			"os":        info.OS,
			"arch":      info.Arch,
		})
		return 0
	}
	fmt.Fprintf(a.io.Out, "fontcreator233 %s\n", info.Version)
	fmt.Fprintf(a.io.Out, "  commit:    %s\n", info.Commit)
	fmt.Fprintf(a.io.Out, "  built:     %s\n", info.BuildDate)
	fmt.Fprintf(a.io.Out, "  go:        %s\n", info.GoVersion)
	fmt.Fprintf(a.io.Out, "  platform:  %s/%s\n", info.OS, info.Arch)
	return 0
}
