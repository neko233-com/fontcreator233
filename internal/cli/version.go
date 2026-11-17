package cli

import "fmt"

func (a *app) cmdVersion(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("version", &common)
	if err := parseFlexible(fs, args, nil); err != nil {
		a.json = common.json
		return a.fail("version", 1, err)
	}
	a.json = common.json
	if a.json {
		a.writeJSON(okMap("version", map[string]any{"version": version}))
		return 0
	}
	fmt.Fprintf(a.io.Out, "fontcreator233 %s\n", version)
	return 0
}
