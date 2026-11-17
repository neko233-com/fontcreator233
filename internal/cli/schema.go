package cli

import "github.com/neko233-com/fontcreator233/internal/recipe"

func (a *app) cmdSchema(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("schema", &common)
	if err := parseFlexible(fs, args, nil); err != nil {
		a.json = common.json
		return a.fail("schema", 1, err)
	}
	a.json = common.json
	a.writeJSON(recipe.Schema())
	return 0
}
