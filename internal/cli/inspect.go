package cli

import (
	"fmt"
	"os"

	"github.com/neko233-com/fontcreator233/internal/inspect"
)

func (a *app) cmdInspect(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("inspect", &common)
	if err := parseFlexible(fs, args, nil); err != nil {
		a.json = common.json
		return a.fail("inspect", 1, err)
	}
	a.json = common.json
	if fs.NArg() != 1 {
		return a.fail("inspect", 1, fmt.Errorf("usage: fontcreator inspect <font.ttf>"))
	}
	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		return a.fail("inspect", 1, err)
	}
	info, err := inspect.Inspect(data)
	if err != nil {
		return a.fail("inspect", 1, err)
	}
	if a.json {
		a.writeJSON(okMap("inspect", map[string]any{"font": info}))
		return 0
	}
	fmt.Fprintf(a.io.Out, "%s %s · %d glyphs · %d UPM · %s\n", info.Family, info.Style, info.Glyphs, info.UnitsPerEm, info.SHA256[:12])
	for _, script := range []string{"latin", "greek", "cyrillic", "cjk", "kana", "hangul", "symbols", "other"} {
		if n := info.Coverage[script]; n > 0 {
			fmt.Fprintf(a.io.Out, "  %-9s %d\n", script, n)
		}
	}
	return 0
}
