package cli

import (
	"fmt"

	"github.com/neko233-com/fontcreator233/internal/recipe"
)

func (a *app) cmdValidate(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("validate", &common)
	if err := parseFlexible(fs, args, nil); err != nil {
		a.json = common.json
		return a.fail("validate", 1, err)
	}
	a.json = common.json
	if fs.NArg() != 1 {
		return a.fail("validate", 1, fmt.Errorf("usage: fontcreator validate <recipe.yaml>"))
	}
	r, err := recipe.Load(fs.Arg(0))
	if err != nil {
		return a.fail("validate", 1, err)
	}
	if a.json {
		a.writeJSON(okMap("validate", map[string]any{
			"valid":  true,
			"family": r.Meta.Family,
			"style":  r.Meta.Style,
			"path":   fs.Arg(0),
		}))
		return 0
	}
	fmt.Fprintf(a.io.Out, "%s: valid (%s / %s)\n", fs.Arg(0), r.Meta.Family, r.Meta.Style)
	return 0
}
