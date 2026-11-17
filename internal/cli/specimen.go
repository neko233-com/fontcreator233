package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/neko233-com/fontcreator233/internal/inspect"
	"github.com/neko233-com/fontcreator233/internal/specimen"
)

func (a *app) cmdSpecimen(args []string) int {
	var common commonFlags
	fs := a.newFlagSet("specimen", &common)
	out := fs.String("out", "", "output directory (default: font directory)")
	assetsDir := fs.String("assets-dir", "", "shared web assets directory")
	selfContained := fs.Bool("self-contained-html", false, "inline font/CSS/JS")
	values := map[string]bool{"out": true, "assets-dir": true}
	if err := parseFlexible(fs, args, values); err != nil {
		a.json = common.json
		return a.fail("specimen", 1, err)
	}
	a.json = common.json
	if fs.NArg() != 1 {
		return a.fail("specimen", 1, fmt.Errorf("usage: fontcreator specimen <font.ttf> [flags]"))
	}
	fontPath := fs.Arg(0)
	data, err := os.ReadFile(fontPath)
	if err != nil {
		return a.fail("specimen", 1, err)
	}
	info, err := inspect.Inspect(data)
	if err != nil {
		return a.fail("specimen", 1, err)
	}
	glyphs, err := specimen.GlyphsFromFont(data)
	if err != nil {
		return a.fail("specimen", 1, err)
	}
	dir := *out
	if dir == "" {
		dir = filepath.Dir(fontPath)
	}
	path, err := specimen.Build(specimen.Options{
		OutDir:        dir,
		FontPath:      fontPath,
		FontFile:      filepath.Base(fontPath),
		AssetsDir:     *assetsDir,
		SelfContained: *selfContained,
		Open:          !common.noOpen && !isCI(),
		Data: specimen.Data{
			Family:      info.Family,
			Style:       info.Style,
			Description: "",
			Version:     info.Version,
			SHA256:      info.SHA256,
			Accent:      "#5B8CFF",
			SampleText:  "Hamburgefonstiv 字体创造器 0123",
			Glyphs:      glyphs,
			Coverage:    info.Coverage,
		},
	})
	if err != nil {
		return a.fail("specimen", 1, err)
	}
	if a.json {
		a.writeJSON(okMap("specimen", map[string]any{"specimen": path, "font": fontPath}))
		return 0
	}
	fmt.Fprintf(a.io.Out, "specimen: %s\n", path)
	return 0
}
