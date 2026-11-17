package cli

import (
	"fmt"

	"github.com/neko233-com/fontcreator233/internal/agentschema"
)

func (a *app) cmdDescribe(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("describe", &common)
	markdown := fs.Bool("markdown", false, "print the agent guide as Markdown")
	if err := parseFlexible(fs, args, nil); err != nil {
		a.json = common.json
		return a.fail("describe", 1, err)
	}
	a.json = common.json
	if *markdown {
		fmt.Fprint(a.io.Out, agentschema.Markdown())
		return 0
	}
	if a.json {
		a.writeJSON(agentschema.Describe())
		return 0
	}
	fmt.Fprint(a.io.Out, agentschema.Markdown())
	return 0
}
